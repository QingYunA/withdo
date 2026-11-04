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
import java.time.Instant
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

/*
 * Withdo Pro on the phone (v45): the plan the Assistant group shows and the upgrade sheet. This
 * file is the state and the wording, without drawing (BillingViews.kt draws them).
 */

/** Where the upgrade sheet is: closed, choosing a plan, asking for the checkout page, or waiting for the payment. */
enum class SheetStage { CLOSED, CHOOSING, STARTING, WAITING }

/** What the app knows of the plan and what the sheet is doing. */
@Stable
class BillingHolder {
    var billing by mutableStateOf(Billing())
    var stage by mutableStateOf(SheetStage.CLOSED)

    /** The segment: Yearly is chosen when the sheet opens. */
    var plan by mutableStateOf(PlanKey.YEARLY)

    /** The checkout page the browser was sent to; Reopen opens it again. */
    var checkoutUrl by mutableStateOf("")

    /** Why the last try failed, shown in the sheet. */
    var failure by mutableStateOf<CheckoutFailure?>(null)

    /** Opens the sheet for a Free account. */
    fun open() {
        if (!billing.canUpgrade) return
        plan = PlanKey.YEARLY
        failure = null
        checkoutUrl = ""
        stage = SheetStage.CHOOSING
    }

    fun close() {
        stage = SheetStage.CLOSED
        failure = null
    }

    /** What the core says now. Once the account is Pro the sheet has nothing left to do. */
    fun update(read: Billing) {
        billing = read
        if (read.pro) close()
    }

    /**
     * The button: asks for the checkout page and hands it to [open] (the browser). A failure stays
     * in the sheet; an account that turns out to be Pro already is read again, which closes it.
     */
    suspend fun start(backend: BillingBackend, lang: String, open: (String) -> Unit) {
        if (stage != SheetStage.CHOOSING) return
        stage = SheetStage.STARTING
        failure = null
        val result = withContext(Dispatchers.IO) { runCatching { backend.checkout(plan, lang) } }
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

/**
 * The line under the quota on a Pro card: `Free trial until Nov 11, then it renews`, `Renews Nov 11,
 * 2027`, `Ends Nov 11` while canceling. Empty when the server named no date.
 */
fun renewalLine(res: Resources, billing: Billing, now: Instant = Instant.now(), zone: ZoneId = ZoneId.systemDefault()): String {
    if (billing.status == "past_due") return res.getString(R.string.billing_past_due)
    val date = billingDate(res, billing.periodEnd, now, zone)
    if (date.isEmpty()) return ""
    return when (billing.status) {
        "trialing" -> res.getString(R.string.billing_trial_until, date)
        "canceling" -> res.getString(R.string.billing_ends, date)
        else -> res.getString(R.string.billing_renews, date)
    }
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
