package xyz.sriai.adhdtodo.ui

import android.content.res.Resources
import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Billing
import xyz.sriai.adhdtodo.data.BillingBackend
import xyz.sriai.adhdtodo.data.CheckoutFailure
import xyz.sriai.adhdtodo.data.PlanKey
import xyz.sriai.adhdtodo.data.ProviderKey
import java.time.Instant
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

/*
 * Withdo Pro on the phone (v45, v47): the plan the account card shows, the upgrade sheet and the
 * plan sheet. This file is the state and the wording, without drawing (BillingViews.kt draws them).
 */

/** Where the upgrade sheet is: closed, choosing a plan, asking for the checkout page, or waiting for the payment. */
enum class SheetStage { CLOSED, CHOOSING, STARTING, WAITING }

/** What the app knows of the plan and what the sheet is doing. */
@Stable
class BillingHolder {
    var billing by mutableStateOf(Billing())
    var stage by mutableStateOf(SheetStage.CLOSED)

    /** The plan sheet of a Pro account (v47): the plan, the payment method, the date and Manage subscription. */
    var planSheet by mutableStateOf(false)

    /** The segment: Yearly is chosen when the sheet opens. */
    var plan by mutableStateOf(PlanKey.YEARLY)

    /** Who the checkout page in the browser (or being asked for) is at; the wait is worded for it. */
    var provider by mutableStateOf(ProviderKey.WAFFO)

    /** The checkout page the browser was sent to; Reopen opens it again. */
    var checkoutUrl by mutableStateOf("")

    /** Why the last try failed, shown in the sheet. */
    var failure by mutableStateOf<CheckoutFailure?>(null)

    /** Opens the sheet for a Free account. */
    fun open() {
        if (!billing.canUpgrade) return
        plan = PlanKey.YEARLY
        provider = ProviderKey.WAFFO
        failure = null
        checkoutUrl = ""
        stage = SheetStage.CHOOSING
    }

    fun close() {
        stage = SheetStage.CLOSED
        failure = null
    }

    /** The account card's plan row: the plan sheet for a Pro account, the upgrade sheet for a Free one. */
    fun openPlan() {
        if (billing.pro) planSheet = true else open()
    }

    /** What the core says now. Once the account is Pro the upgrade sheet has nothing left to do; once it is not, the plan sheet has not. */
    fun update(read: Billing) {
        billing = read
        if (read.pro) close() else planSheet = false
    }

    /**
     * The button: asks for the checkout page of the chosen plan at [provider] and hands it to [open]
     * (the browser). A failure stays in the sheet; an account that turns out to be Pro already is
     * read again, which closes it.
     */
    suspend fun start(backend: BillingBackend, lang: String, provider: String = ProviderKey.WAFFO, open: (String) -> Unit) {
        if (stage != SheetStage.CHOOSING) return
        stage = SheetStage.STARTING
        failure = null
        this.provider = provider
        val result = withContext(Dispatchers.IO) { runCatching { backend.checkout(plan, provider, lang) } }
        result.exceptionOrNull()?.let { if (it is CancellationException) throw it }
        if (stage != SheetStage.STARTING) return // dismissed while asking: the browser is not wanted
        val url = result.getOrNull()
        if (url != null) {
            checkoutUrl = url
            stage = SheetStage.WAITING
            open(url)
            return
        }
        failure = (result.exceptionOrNull() as? Exception)?.let(CheckoutFailure::of) ?: CheckoutFailure.NETWORK
        stage = SheetStage.CHOOSING
        if (failure == CheckoutFailure.ALREADY_SUBSCRIBED) refresh(backend)
    }

    /** Reads the plan from the server now: coming back from the browser, or before opening the portal. */
    suspend fun refresh(backend: BillingBackend) {
        val read = withContext(Dispatchers.IO) {
            runCatching { backend.refreshBilling(); backend.billing() }
        }
        read.exceptionOrNull()?.let { if (it is CancellationException) throw it }
        read.getOrNull()?.let(::update)
    }

    /** Reads the cached plan (cheap); the core refreshes it in the background. */
    suspend fun read(backend: BillingBackend) {
        val read = withContext(Dispatchers.IO) { runCatching { backend.billing() } }
        read.exceptionOrNull()?.let { if (it is CancellationException) throw it }
        read.getOrNull()?.let(::update)
    }

    /** `Manage subscription`: the portal link is good for 15 minutes, so it is made fresh and opened at once. */
    suspend fun manage(backend: BillingBackend, open: (String) -> Unit) {
        refresh(backend)
        billing.manageUrl.takeIf { it.isNotBlank() }?.let(open)
    }
}

/** The language the checkout page and the core's mail speak: `zh-Hans` or `en`. */
fun checkoutLanguage(res: Resources): String =
    if (runCatching { res.configuration.locales[0].language }.getOrNull() == "zh") "zh-Hans" else "en"

// ---- Wording -----------------------------------------------------------------------------------

/**
 * The yearly price over twelve months, in the price's own currency and notation: `$58.72` gives
 * `$4.89`. Null when the price has no number to divide.
 */
fun perMonth(yearly: String): String? {
    val match = Regex("""^(\D*?)(\d+(?:[.,]\d+)?)(\D*)$""").find(yearly.trim()) ?: return null
    val (prefix, number, suffix) = match.destructured
    val value = number.replace(',', '.').toDoubleOrNull() ?: return null
    val comma = ',' in number
    val text = String.format(Locale.ROOT, "%.2f", value / 12).let { if (comma) it.replace('.', ',') else it }
    return prefix + text + suffix
}

/** `$58.72 / year, about $4.89 / month`, or `$6.99 / month`: what is added after the price in 28 sp. */
fun priceNote(res: Resources, billing: Billing, plan: String): String {
    val yearly = billing.yearly.ifBlank { DefaultYearly }
    if (plan == PlanKey.MONTHLY) return res.getString(R.string.billing_price_monthly)
    val each = perMonth(yearly)
    return if (each != null) res.getString(R.string.billing_price_yearly_each, each) else res.getString(R.string.billing_price_yearly)
}

/** The price in 28 sp: `$58.72` or `$6.99`. */
fun priceOf(billing: Billing, plan: String): String =
    if (plan == PlanKey.MONTHLY) billing.monthly.ifBlank { DefaultMonthly } else billing.yearly.ifBlank { DefaultYearly }

private const val DefaultMonthly = "$6.99"
private const val DefaultYearly = "$58.72"

/** What the Pro subscription is doing, as the plan row and the plan sheet tell it. An unknown status reads as active. */
enum class PlanState { TRIALING, ACTIVE, CANCELING, PAST_DUE }

fun Billing.planState(): PlanState = when (status) {
    "trialing" -> PlanState.TRIALING
    "canceling" -> PlanState.CANCELING
    "past_due" -> PlanState.PAST_DUE
    else -> PlanState.ACTIVE
}

/** The tag after `Withdo Pro`: `Free trial`, `Pro`, `Canceled` or `Payment failed`. */
fun planTag(res: Resources, billing: Billing): String = res.getString(
    when (billing.planState()) {
        PlanState.TRIALING -> R.string.billing_tag_trialing
        PlanState.ACTIVE -> R.string.billing_tag_active
        PlanState.CANCELING -> R.string.billing_tag_canceling
        PlanState.PAST_DUE -> R.string.billing_tag_past_due
    },
)

/**
 * The plan row's `meta` line: `Yearly · First charge Nov 12` while trialing, `Yearly · Renews Nov 12,
 * 2027` when active, `Pro until Nov 12` while canceling, `The renewal didn't go through` when past
 * due. The interval leads only where the server says it; empty when there is nothing to say.
 */
fun planMeta(res: Resources, billing: Billing, now: Instant = Instant.now(), zone: ZoneId = ZoneId.systemDefault()): String {
    val state = billing.planState()
    if (state == PlanState.PAST_DUE) return res.getString(R.string.billing_meta_past_due)
    val date = billingDate(res, billing.periodEnd, now, zone)
    val tail = when {
        date.isEmpty() -> null
        state == PlanState.TRIALING -> res.getString(R.string.billing_meta_first_charge, date)
        state == PlanState.CANCELING -> res.getString(R.string.billing_meta_until, date)
        else -> res.getString(R.string.billing_renews, date)
    }
    val lead = if (state == PlanState.CANCELING) null else intervalName(res, billing)
    return listOfNotNull(lead, tail).joinToString(" · ")
}

/**
 * The plan sheet's box: `Plan` with `Yearly $58.72` (`Pro` when the server names no interval),
 * `Payment` with `Waffo` or `PayPal`, then the date by status (`First charge`, `Renews`, `Pro
 * until`), left out when the server named no date.
 */
fun planRows(res: Resources, billing: Billing, now: Instant = Instant.now(), zone: ZoneId = ZoneId.systemDefault()): List<Pair<String, String>> {
    val interval = intervalName(res, billing)
    val price = when (billing.interval) {
        PlanKey.MONTHLY -> priceOf(billing, PlanKey.MONTHLY)
        PlanKey.YEARLY -> priceOf(billing, PlanKey.YEARLY)
        else -> null
    }
    val plan = if (interval != null && price != null) res.getString(R.string.billing_plan_value, interval, price) else res.getString(R.string.billing_plan_pro)
    val date = billingDate(res, billing.periodEnd, now, zone)
    return listOfNotNull(
        res.getString(R.string.billing_row_plan) to plan,
        res.getString(R.string.billing_row_payment) to if (billing.viaPayPal) "PayPal" else "Waffo",
        if (date.isEmpty()) null else res.getString(
            when (billing.planState()) {
                PlanState.TRIALING -> R.string.billing_row_first_charge
                PlanState.CANCELING -> R.string.billing_row_until
                else -> R.string.billing_row_renews
            },
        ) to date,
    )
}

/** The one paragraph under the plan sheet's box, by status. */
fun planNote(res: Resources, billing: Billing): String = res.getString(
    when (billing.planState()) {
        PlanState.TRIALING -> R.string.billing_note_trialing
        PlanState.ACTIVE -> R.string.billing_note_active
        PlanState.CANCELING -> R.string.billing_note_canceling
        PlanState.PAST_DUE -> R.string.billing_note_past_due
    },
)

/** `Monthly` or `Yearly` when the server says the interval. */
private fun intervalName(res: Resources, billing: Billing): String? = when (billing.interval) {
    PlanKey.MONTHLY -> res.getString(R.string.billing_monthly)
    PlanKey.YEARLY -> res.getString(R.string.billing_yearly)
    else -> null
}

/** `Nov 11`, or `Nov 11, 2027` in another year than [now]'s; `11月11日` and `2027年11月11日` in Chinese. */
fun billingDate(res: Resources, value: String, now: Instant, zone: ZoneId): String {
    if (value.isBlank()) return ""
    val at = runCatching { OffsetDateTime.parse(value).toInstant() }
        .getOrElse { runCatching { Instant.parse(value) }.getOrNull() } ?: return ""
    val locale = runCatching { res.configuration.locales[0] }.getOrDefault(Locale.getDefault())
    val chinese = locale.language == "zh"
    val day = at.atZone(zone)
    val sameYear = day.year == now.atZone(zone).year
    val pattern = when {
        chinese && sameYear -> "M月d日"
        chinese -> "yyyy年M月d日"
        sameYear -> "MMM d"
        else -> "MMM d, yyyy"
    }
    return DateTimeFormatter.ofPattern(pattern, locale).format(day)
}

/** What the sheet says when [failure] stopped the try, in the `status.blocked` line. */
fun failureLine(res: Resources, failure: CheckoutFailure): String = res.getString(
    when (failure) {
        CheckoutFailure.RATE_LIMITED -> R.string.billing_error_rate_limited
        CheckoutFailure.UPSTREAM -> R.string.billing_error_upstream
        CheckoutFailure.SIGNED_OUT -> R.string.billing_error_signed_out
        CheckoutFailure.ALREADY_SUBSCRIBED -> R.string.billing_error_already
        else -> R.string.billing_error_open
    },
)
