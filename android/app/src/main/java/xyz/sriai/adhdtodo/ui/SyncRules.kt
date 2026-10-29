package xyz.sriai.adhdtodo.ui

import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.CodeState
import java.util.Locale

/** The 40 cells of the first upload's bar, as v27 and the desktop's pane draw them. */
const val SYNC_BAR_CELLS = 40

/** Where a phone syncs until the user changes it; mobile/core reports it before sync.json exists. */
const val DEFAULT_SYNC_SERVER = "https://todo.sri-ai.xyz"

/**
 * Whether [server] is [DEFAULT_SYNC_SERVER]. Spaces, a trailing slash and the case of the address
 * do not matter. The pane hides the official address and shows any other; a blank one is not it.
 */
fun isOfficialSyncServer(server: String): Boolean =
    server.trim().trimEnd('/').equals(DEFAULT_SYNC_SERVER, ignoreCase = true)

/** The address to save for what was typed: an empty field means the official server. */
fun syncServerToSave(draft: String): String = draft.trim().ifEmpty { DEFAULT_SYNC_SERVER }

/** How long a failed sync may stay quiet before the header's cloud turns red. */
const val SYNC_ATTENTION_MILLIS = 15 * 60 * 1000L

/** SyncStatus.state of a signed-in account that does not own this phone's device. */
const val SYNC_WRONG_ACCOUNT = "wrong account"

/**
 * Whether the app may (re)start the sync runner. A phone in the wrong account has no retry: the
 * runner stopped on purpose and only signing out (or moving the phone) changes the answer.
 */
fun syncRunnerMayStart(state: String): Boolean = state != SYNC_WRONG_ACCOUNT

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
    if (state == SYNC_WRONG_ACCOUNT) return true
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

/** What a sync error means to the user, from the core's sentence (syncError in mobile/core). */
enum class SyncFailure { AUTH, NETWORK, SERVER, REFUSED, OTHER }

/**
 * Only a real network failure is "the server did not answer"; a 5xx, a refusal and an unknown
 * error each have their own words.
 */
fun syncFailureOf(message: String): SyncFailure {
    val m = message.lowercase(Locale.ROOT)
    return when {
        isAuthFailure(m) || m.contains("sign in again") -> SyncFailure.AUTH
        m.contains("could not reach the server") || m.contains("did not answer") -> SyncFailure.NETWORK
        m.contains("had a problem") -> SyncFailure.SERVER
        m.contains("refused the request") -> SyncFailure.REFUSED
        else -> SyncFailure.OTHER
    }
}

fun syncFailureText(failure: SyncFailure): Int = when (failure) {
    SyncFailure.AUTH -> R.string.sync_failed_auth
    SyncFailure.NETWORK -> R.string.sync_failed_no_answer
    SyncFailure.SERVER -> R.string.sync_failed_server
    SyncFailure.REFUSED -> R.string.sync_failed_refused
    SyncFailure.OTHER -> R.string.sync_failed_other
}

/** What the signed-in card's Google row draws; [shown] is false where the server has no Google. */
data class GoogleLinkRow(
    val available: Boolean = false,
    val linked: Boolean = false,
    /** True while the browser waits for the user. */
    val busy: Boolean = false,
    /** The failure line under the row, a string resource. */
    val message: Int? = null,
) {
    val shown: Boolean get() = available
}

/** The line for a GoogleLinkFinish error kind; cancelled says nothing. */
fun googleLinkFailureText(kind: String): Int? = when (kind.trim()) {
    "cancelled" -> null
    "already_linked" -> R.string.sync_google_link_taken
    "network" -> R.string.sync_sign_in_network
    else -> R.string.sync_google_failed
}

/** The stage the sign-in button says: the server checks the account, then the tasks download. */
/** GOOGLE is the browser open on Google's consent page, the card waiting for it (M13 P3). */
enum class SignInStage { SENDING, CREATING, CHECKING, DOWNLOADING, GOOGLE }

/**
 * The stage to show, from the raw sign_in_stage. Signing in without a stage yet is checking; not
 * signing in has no stage.
 */
fun signInStageOf(signingIn: Boolean, raw: String?): SignInStage? = when {
    !signingIn -> null
    raw == "downloading" -> SignInStage.DOWNLOADING
    raw == "sending" -> SignInStage.SENDING
    raw == "creating" -> SignInStage.CREATING
    raw == "google" -> SignInStage.GOOGLE
    else -> SignInStage.CHECKING
}

/**
 * The steps of the signed-out card (v42): the Sign in and Sign up tabs, Verify after a sign-up (or
 * a sign-in to an account whose email was never verified), and the two steps of Forgot password.
 */
enum class SignInStep { SIGN_IN, SIGN_UP, VERIFY, RESET_EMAIL, RESET_CODE }

/** Whether [step] is one of the card's two tabs. */
fun SignInStep.isTab(): Boolean = this == SignInStep.SIGN_IN || this == SignInStep.SIGN_UP

/** Why a sign-up, sign-in, code or reset failed, from CodeState's failure kinds. */
enum class CodeFailure {
    EMAIL_TAKEN, INVALID_EMAIL, WEAK_PASSWORD,
    WRONG_CODE, EXPIRED, TOO_MANY, WRONG_PASSWORD, DEACTIVATED, NETWORK, UNSUPPORTED,
    GOOGLE_CANCELLED, GOOGLE_FAILED, OTHER,
}

/** The failure kind of CodeState's "failure" field; blank is no failure. */
fun codeFailureOf(kind: String?): CodeFailure? = when (kind?.trim()) {
    null, "" -> null
    "email_taken" -> CodeFailure.EMAIL_TAKEN
    "invalid_email" -> CodeFailure.INVALID_EMAIL
    "weak_password" -> CodeFailure.WEAK_PASSWORD
    "wrong_code" -> CodeFailure.WRONG_CODE
    "expired" -> CodeFailure.EXPIRED
    "too_many" -> CodeFailure.TOO_MANY
    "wrong_password" -> CodeFailure.WRONG_PASSWORD
    "deactivated" -> CodeFailure.DEACTIVATED
    "network" -> CodeFailure.NETWORK
    "unsupported" -> CodeFailure.UNSUPPORTED
    "google_cancelled" -> CodeFailure.GOOGLE_CANCELLED
    "google_failed" -> CodeFailure.GOOGLE_FAILED
    else -> CodeFailure.OTHER
}

/** The localized line of a failure; OTHER has none (the core's own sentence is shown). */
fun codeFailureText(failure: CodeFailure): Int? = when (failure) {
    CodeFailure.EMAIL_TAKEN -> R.string.sync_email_taken
    CodeFailure.INVALID_EMAIL -> R.string.sync_invalid_email
    CodeFailure.WEAK_PASSWORD -> R.string.sync_password_min
    CodeFailure.WRONG_CODE -> R.string.sync_code_wrong
    CodeFailure.EXPIRED -> R.string.sync_code_expired
    CodeFailure.TOO_MANY -> R.string.sync_too_many_tries
    CodeFailure.WRONG_PASSWORD -> R.string.sync_wrong_password
    CodeFailure.DEACTIVATED -> R.string.sync_sign_in_locked
    CodeFailure.NETWORK -> R.string.sync_sign_in_network
    CodeFailure.UNSUPPORTED -> R.string.sync_sign_in_unsupported
    CodeFailure.GOOGLE_CANCELLED -> R.string.sync_google_cancelled
    CodeFailure.GOOGLE_FAILED -> R.string.sync_google_failed
    CodeFailure.OTHER -> null
}

/** Everything the sign-in card draws between the email and the linked account. */
data class SignInFlow(
    val step: SignInStep = SignInStep.SIGN_IN,
    val email: String = "",
    /** Seconds before another code may be asked for, as of when the step was drawn. */
    val resendIn: Int = 0,
    val failure: CodeFailure? = null,
    /** The core's sentence for a failure with no localized line. */
    val failureText: String? = null,
    /** Counts failed attempts, so the cells and the password empty even when the failure repeats. */
    val attempt: Int = 0,
    /** True while a code is being mailed. */
    val sending: Boolean = false,
    /** True while an account is being made. */
    val creating: Boolean = false,
)

/** A plausible address: something, an @, a dot after it. The server decides the rest. */
fun looksLikeEmail(text: String): Boolean {
    val t = text.trim()
    val at = t.indexOf('@')
    return at > 0 && t.indexOf('.', at + 2) > at + 1 && !t.endsWith(".") && !t.contains(' ')
}

const val CODE_LENGTH = 6

/** The digits of what was typed or pasted, at most six. */
fun codeDigits(text: String): String = text.filter { it in '0'..'9' }.take(CODE_LENGTH)

/** 24 seconds as 0:24, the way the resend line counts down. */
fun resendClock(seconds: Int): String = "%d:%02d".format(seconds / 60, seconds % 60)

/**
 * The flow after a failed code or password sign-in: the kind comes from CodeState, the core's
 * sentence is kept for a kind with no line of its own, and the attempt counter empties the cells.
 */
fun SignInFlow.failed(state: CodeState?, message: String?): SignInFlow = copy(
    failure = codeFailureOf(state?.failure) ?: CodeFailure.OTHER,
    failureText = message?.trim()?.takeIf { it.isNotEmpty() },
    attempt = attempt + 1,
    resendIn = state?.resendIn ?: resendIn,
    sending = false,
    creating = false,
)

/**
 * The flow once SignUp answered. A mailed code moves on to Verify; a failure (a taken email, a
 * malformed one, a short password) stays on Sign up with its line.
 */
fun SignInFlow.afterSignUp(errorMessage: String?, state: CodeState?): SignInFlow =
    afterCodeMailed(errorMessage, state, "verify", SignInStep.VERIFY)

/** The flow once RequestResetCode answered: the code step of Forgot password, or the failure. */
fun SignInFlow.afterResetRequest(errorMessage: String?, state: CodeState?): SignInFlow =
    afterCodeMailed(errorMessage, state, "reset_code", SignInStep.RESET_CODE)

private fun SignInFlow.afterCodeMailed(errorMessage: String?, state: CodeState?, kind: String, next: SignInStep): SignInFlow {
    if (errorMessage == null && state?.step == kind) {
        return copy(
            step = next,
            email = state.email.ifBlank { email },
            resendIn = state.resendIn,
            failure = null,
            failureText = null,
            attempt = attempt + 1,
            sending = false,
            creating = false,
        )
    }
    return failed(state, errorMessage)
}

/**
 * The flow once SignIn answered {"state":"verify"}: the password was right but the email was never
 * verified, a code was mailed, and the card shows Verify.
 */
fun SignInFlow.afterVerifyAnswer(state: CodeState?): SignInFlow = copy(
    step = SignInStep.VERIFY,
    email = state?.email?.ifBlank { null } ?: email,
    resendIn = state?.resendIn ?: 30,
    failure = null,
    failureText = null,
    attempt = attempt + 1,
    sending = false,
    creating = false,
)

/** The shortest password the server takes. */
const val PASSWORD_MIN = 8

/** The steps of the strength bar. */
const val STRENGTH_STEPS = 4

/** Whether [password] is long enough to create an account with. */
fun passwordLongEnough(password: String): Boolean = password.codePointCount(0, password.length) >= PASSWORD_MIN

/**
 * 0 for a password under 8 characters, else one step for the length, one more from 12, one for two
 * kinds of characters (lower case, upper case, digits, the rest) and one for three kinds from 10
 * characters. The desktop's `passwordStrength` is the same rule.
 */
fun passwordStrength(password: String): Int {
    val n = password.codePointCount(0, password.length)
    if (n < PASSWORD_MIN) return 0
    val kinds = listOf(
        password.any { it.isLowerCase() },
        password.any { it.isUpperCase() },
        password.any { it.isDigit() },
        password.any { !it.isLowerCase() && !it.isUpperCase() && !it.isDigit() },
    ).count { it }
    var score = 1
    if (n >= 12) score++
    if (kinds >= 2) score++
    if (kinds >= 3 && n >= 10) score++
    return score
}

/** The language of the code mails: the app's, English or Simplified Chinese. */
fun syncLang(locale: Locale): String = if (locale.language.equals("zh", ignoreCase = true)) "zh-Hans" else "en"

/** The terms and the privacy policy a sign-up agrees to, on the website. */
const val TERMS_URL = "https://qingyuna.github.io/withdo/terms.html"
const val PRIVACY_URL = "https://qingyuna.github.io/withdo/privacy.html"
