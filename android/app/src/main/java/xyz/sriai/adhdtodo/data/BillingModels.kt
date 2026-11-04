package xyz.sriai.adhdtodo.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/*
 * Withdo Pro's JSON (mobile/core/billing.go): the plan Billing() says, and the words Checkout()
 * fails with. The core sends data, the app words it.
 */

/** The plans Checkout takes. */
object PlanKey {
    const val MONTHLY = "monthly"
    const val YEARLY = "yearly"
}

/** Who takes the payment (M17): the core's `provider` and what Checkout asks for. */
object ProviderKey {
    const val WAFFO = "waffo"
    const val PAYPAL = "paypal"
}

/**
 * Billing(): what the server sells and the account's plan. [offered] is whether the server sells
 * Withdo Pro (false until known); [plan] is `free` or `pro`, empty while signed out or not read yet;
 * [status] is the subscription's (trialing, active, past_due, canceling, canceled), empty for an
 * account that never subscribed, and [periodEnd] (RFC 3339) is empty then too. [interval] is `monthly`
 * or `yearly`, empty when the server does not say. [manageUrl] is the
 * customer portal, good for 15 minutes. [paypal] is whether the server takes PayPal too (the sheet
 * then shows PayPal's button) and [provider] who bills the subscription, `waffo` or `paypal`.
 */
@Serializable
data class Billing(
    @SerialName("offered") val offered: Boolean = false,
    @SerialName("paypal") val paypal: Boolean = false,
    @SerialName("provider") val provider: String = "",
    @SerialName("plan") val plan: String = "",
    @SerialName("status") val status: String = "",
    @SerialName("interval") val interval: String = "",
    @SerialName("period_end") val periodEnd: String = "",
    @SerialName("trial_used") val trialUsed: Boolean = false,
    @SerialName("monthly") val monthly: String = "",
    @SerialName("yearly") val yearly: String = "",
    @SerialName("manage_url") val manageUrl: String = "",
) {
    /** The plan card shows: the server sells Pro and the account's plan is known. */
    val shown: Boolean get() = offered && plan.isNotEmpty()

    val pro: Boolean get() = plan == "pro"

    /** The subscription is billed by PayPal. */
    val viaPayPal: Boolean get() = provider == ProviderKey.PAYPAL

    /** A Free account may upgrade. */
    val canUpgrade: Boolean get() = shown && !pro
}

/** The core's billing calls the screens use; [CoreApi] implements them, the tests fake them. Each blocks. */
interface BillingBackend {
    fun billing(): Billing

    /** Asks the server for the plan now; [billing] then has the answer. */
    fun refreshBilling()

    /**
     * The checkout page of [plan] ([PlanKey]) at [provider] ([ProviderKey]) in [lang] (`en` or
     * `zh-Hans`); fails with an exception [CheckoutFailure.of] reads.
     */
    fun checkout(plan: String, provider: String, lang: String): String
}

/** Why Checkout failed, from the word the core's error carries. */
enum class CheckoutFailure {
    SIGNED_OUT, OFF, ALREADY_SUBSCRIBED, RATE_LIMITED, UPSTREAM, NETWORK;

    companion object {
        fun of(e: Exception): CheckoutFailure = when (e.message?.trim()) {
            "signed_out" -> SIGNED_OUT
            "off" -> OFF
            "already_subscribed" -> ALREADY_SUBSCRIBED
            "rate_limited" -> RATE_LIMITED
            "upstream" -> UPSTREAM
            else -> NETWORK
        }
    }
}
