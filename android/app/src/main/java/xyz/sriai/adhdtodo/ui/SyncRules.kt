package xyz.sriai.adhdtodo.ui

import xyz.sriai.adhdtodo.R
import java.util.Locale

/** The 40 cells of the first upload's bar, as v27 and the desktop's pane draw them. */
const val SYNC_BAR_CELLS = 40

/** Where a phone syncs until the user changes it; mobile/core reports it before sync.json exists. */
const val DEFAULT_SYNC_SERVER = "https://todo.sri-ai.xyz"

/** How long a failed sync may stay quiet before the header's cloud turns red. */
const val SYNC_ATTENTION_MILLIS = 15 * 60 * 1000L

/**
 * The laptop or phone icon a device gets. The server does not say what a device is, so the name
 * decides, exactly as the desktop's Settings > Sync guesses it.
 */
fun deviceUsesLaptop(name: String): Boolean {
    val n = name.lowercase(Locale.ROOT)
    return n.contains("mac") || n.contains("book") || n.contains("pc")
}

/**
 * The header's cloud turns status.blocked when a sync has failed for more than 15 minutes, or
 * when a phone that was linked has signed out. A phone that never linked stays quiet.
 */
fun syncCloudBlocked(
    linked: Boolean,
    signedIn: Boolean,
    state: String,
    failedSinceMillis: Long?,
    nowMillis: Long,
): Boolean {
    if (!linked) return false
    if (!signedIn) return true
    return state == "error" && failedSinceMillis != null &&
        nowMillis - failedSinceMillis > SYNC_ATTENTION_MILLIS
}

/** Filled cells of the upload bar: the rounded share, and at least one once anything is done. */
fun syncBarFilled(done: Long, total: Long): Int {
    if (done <= 0 || total <= 0) return 0
    var n = ((done * SYNC_BAR_CELLS + total / 2) / total).toInt()
    if (n < 1) n = 1
    if (n > SYNC_BAR_CELLS) n = SYNC_BAR_CELLS
    return n
}

/** A count with commas, as the upload line says 1,240 / 3,512. */
fun groupThousands(n: Long): String {
    if (n < 0) return "-" + groupThousands(-n)
    val raw = n.toString()
    val out = StringBuilder()
    for ((i, c) in raw.withIndex()) {
        if (i > 0 && (raw.length - i) % 3 == 0) out.append(',')
        out.append(c)
    }
    return out.toString()
}

/** Whether a sync error means the sign-in expired, so the row says "login expired". */
fun isAuthFailure(message: String): Boolean {
    val m = message.lowercase(Locale.ROOT)
    return m.contains("401") || m.contains("unauthorized") || m.contains("signed out") ||
        m.contains("refresh token")
}

/** The stage the sign-in button says: the server checks the account, then the tasks download. */
enum class SignInStage { CHECKING, DOWNLOADING }

/**
 * The stage to show, from the raw sign_in_stage. Signing in without a stage yet is checking; not
 * signing in has no stage.
 */
fun signInStageOf(signingIn: Boolean, raw: String?): SignInStage? = when {
    !signingIn -> null
    raw == "downloading" -> SignInStage.DOWNLOADING
    else -> SignInStage.CHECKING
}

/** Why an in-app sign-in failed, as the Sync card words it. */
enum class SignInFailure { CREDENTIALS, TOO_MANY, SECOND_FACTOR, LOCKED, UNSUPPORTED, NETWORK }

/**
 * The kind of a failed sign-in from mobile/core's sentence. Go keeps these six sentences stable so
 * the app can map them to resources; an unknown failure keeps the raw sentence.
 */
fun signInFailureOf(message: String?): SignInFailure? = when (message?.trim()) {
    "wrong account or password" -> SignInFailure.CREDENTIALS
    "too many tries; wait a few minutes" -> SignInFailure.TOO_MANY
    "this account uses two-step verification, which in-app sign-in does not support yet" ->
        SignInFailure.SECOND_FACTOR
    "this account is deactivated; ask the admin" -> SignInFailure.LOCKED
    "this server does not support signing in from the app" -> SignInFailure.UNSUPPORTED
    "could not reach the server; check the connection" -> SignInFailure.NETWORK
    else -> null
}

/** The localized sentence of a failed sign-in. */
fun signInFailureText(failure: SignInFailure): Int = when (failure) {
    SignInFailure.CREDENTIALS -> R.string.sync_sign_in_invalid
    SignInFailure.TOO_MANY -> R.string.sync_sign_in_too_many
    SignInFailure.SECOND_FACTOR -> R.string.sync_sign_in_second_factor
    SignInFailure.LOCKED -> R.string.sync_sign_in_locked
    SignInFailure.UNSUPPORTED -> R.string.sync_sign_in_unsupported
    SignInFailure.NETWORK -> R.string.sync_sign_in_network
}
