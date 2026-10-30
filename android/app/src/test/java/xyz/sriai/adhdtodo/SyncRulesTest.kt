package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.data.CodeState
import xyz.sriai.adhdtodo.ui.CodeFailure
import xyz.sriai.adhdtodo.ui.DEFAULT_SYNC_SERVER
import xyz.sriai.adhdtodo.ui.SYNC_ATTENTION_MILLIS
import xyz.sriai.adhdtodo.ui.SignInFlow
import xyz.sriai.adhdtodo.ui.SignInStep
import xyz.sriai.adhdtodo.ui.afterResetRequest
import xyz.sriai.adhdtodo.ui.afterSignUp
import xyz.sriai.adhdtodo.ui.afterVerifyAnswer
import xyz.sriai.adhdtodo.ui.isTab
import xyz.sriai.adhdtodo.ui.passwordLongEnough
import xyz.sriai.adhdtodo.ui.passwordStrength
import xyz.sriai.adhdtodo.ui.syncLang
import java.util.Locale
import xyz.sriai.adhdtodo.ui.codeDigits
import xyz.sriai.adhdtodo.ui.codeFailureOf
import xyz.sriai.adhdtodo.ui.codeFailureText
import xyz.sriai.adhdtodo.ui.failed
import xyz.sriai.adhdtodo.ui.looksLikeEmail
import xyz.sriai.adhdtodo.ui.resendClock
import xyz.sriai.adhdtodo.ui.SignInStage
import xyz.sriai.adhdtodo.ui.deviceUsesLaptop
import xyz.sriai.adhdtodo.ui.groupThousands
import xyz.sriai.adhdtodo.ui.isAuthFailure
import xyz.sriai.adhdtodo.ui.isOfficialSyncServer
import xyz.sriai.adhdtodo.ui.syncServerToSave
import xyz.sriai.adhdtodo.ui.syncBarFilled
import xyz.sriai.adhdtodo.ui.signInStageOf
import xyz.sriai.adhdtodo.ui.syncCloudBlocked

/** The Google sign-in's stage and failure kinds (M13 P3). */
class GoogleRulesTest {
    @Test
    fun `the google stage is the browser wait`() {
        assertEquals(SignInStage.GOOGLE, signInStageOf(true, "google"))
        assertEquals(null, signInStageOf(false, "google"))
    }

    @Test
    fun `google failures have their own kinds and lines`() {
        assertEquals(CodeFailure.GOOGLE_CANCELLED, codeFailureOf("google_cancelled"))
        assertEquals(CodeFailure.GOOGLE_FAILED, codeFailureOf("google_failed"))
        assertEquals(R.string.sync_google_cancelled, codeFailureText(CodeFailure.GOOGLE_CANCELLED))
        assertEquals(R.string.sync_google_failed, codeFailureText(CodeFailure.GOOGLE_FAILED))
    }
}

/** Verification 1 of A2b-UI: the device icon, the header cloud's red rule and the upload bar. */
class SyncRulesTest {

    @Test
    fun `a Mac, MacBook or PC is a laptop and a phone is not`() {
        for (name in listOf("Cyan 的 MacBook Pro", "公司的 Mac mini", "WORK-PC", "mac studio")) {
            assertTrue(name, deviceUsesLaptop(name))
        }
        for (name in listOf("Pixel 8", "iPhone 15", "Android tablet", "Cyan 的手机")) {
            assertFalse(name, deviceUsesLaptop(name))
        }
    }

    @Test
    fun `the cloud is red only for the desktop's two reasons`() {
        val now = 1_000_000_000L
        // Linked and signed in: never red, whatever the runner is doing.
        assertFalse(syncCloudBlocked(true, true, "idle", null, now))
        assertFalse(syncCloudBlocked(true, true, "error", now - 14 * 60 * 1000, now))
        // A failure older than fifteen minutes is red.
        assertTrue(syncCloudBlocked(true, true, "error", now - SYNC_ATTENTION_MILLIS - 1, now))
        // A linked phone that signed out is red; one that never linked is quiet.
        assertTrue(syncCloudBlocked(true, false, "signed out", null, now))
        assertFalse(syncCloudBlocked(false, false, "signed out", null, now))
        // Offline is not a failure.
        assertFalse(syncCloudBlocked(true, true, "offline", now - 60 * 60 * 1000, now))
    }

    @Test
    fun `the upload bar lights the rounded share and at least one cell`() {
        assertEquals(0, syncBarFilled(0, 3512))
        assertEquals(0, syncBarFilled(1240, 0))
        assertEquals(14, syncBarFilled(1240, 3512))
        assertEquals(1, syncBarFilled(1, 3512))
        assertEquals(40, syncBarFilled(3512, 3512))
        assertEquals(40, syncBarFilled(4000, 3512))
    }

    @Test
    fun `counts are grouped in thousands`() {
        assertEquals("0", groupThousands(0))
        assertEquals("999", groupThousands(999))
        assertEquals("1,000", groupThousands(1000))
        assertEquals("3,512", groupThousands(3512))
        assertEquals("-12,345", groupThousands(-12345))
    }

    @Test
    fun `a sign-in without a stage says checking and then downloading`() {
        assertEquals(null, signInStageOf(false, "downloading"))
        assertEquals(SignInStage.CHECKING, signInStageOf(true, null))
        assertEquals(SignInStage.CHECKING, signInStageOf(true, "checking"))
        assertEquals(SignInStage.DOWNLOADING, signInStageOf(true, "downloading"))
        assertEquals(SignInStage.CHECKING, signInStageOf(true, "later"))
    }

    @Test
    fun `a failure that means the sign-in expired is worded as such`() {
        assertTrue(isAuthFailure("401 Unauthorized"))
        assertTrue(isAuthFailure("the refresh token was rejected"))
        assertFalse(isAuthFailure("the server did not answer"))
    }

    @Test
    fun `a pasted code keeps its six digits only`() {
        assertEquals("482913", codeDigits("482 913"))
        assertEquals("482913", codeDigits("48-29-13-77"))
        assertEquals("", codeDigits("abc"))
        assertEquals("0:24", resendClock(24))
        assertEquals("0:05", resendClock(5))
    }

    @Test
    fun `an email needs an at sign and a dot after it`() {
        assertTrue(looksLikeEmail(" cyan@example.com "))
        assertFalse(looksLikeEmail("cyan"))
        assertFalse(looksLikeEmail("cyan@example"))
        assertFalse(looksLikeEmail("@example.com"))
        assertFalse(looksLikeEmail("cyan @example.com"))
    }

    @Test
    fun `failure kinds map and unknown ones are other`() {
        assertEquals(null, codeFailureOf(""))
        assertEquals(CodeFailure.WRONG_CODE, codeFailureOf("wrong_code"))
        assertEquals(CodeFailure.EXPIRED, codeFailureOf("expired"))
        assertEquals(CodeFailure.EMAIL_TAKEN, codeFailureOf("email_taken"))
        assertEquals(CodeFailure.INVALID_EMAIL, codeFailureOf("invalid_email"))
        assertEquals(CodeFailure.WEAK_PASSWORD, codeFailureOf("weak_password"))
        assertEquals(CodeFailure.OTHER, codeFailureOf("surprise"))
        assertEquals(R.string.sync_email_taken, codeFailureText(CodeFailure.EMAIL_TAKEN))
        assertEquals(R.string.sync_invalid_email, codeFailureText(CodeFailure.INVALID_EMAIL))
        assertEquals(R.string.sync_password_min, codeFailureText(CodeFailure.WEAK_PASSWORD))
    }

    @Test
    fun `a sign-up moves to verify and a failure stays on sign up`() {
        val up = SignInFlow(step = SignInStep.SIGN_UP, email = "a@b.co")
        val sent = up.copy(creating = true).afterSignUp(null, CodeState("verify", "a@b.co", 30, ""))
        assertEquals(SignInStep.VERIFY, sent.step)
        assertEquals(30, sent.resendIn)
        assertEquals(null, sent.failure)
        assertFalse(sent.creating)

        val taken = up.afterSignUp("this email already has an account", CodeState("", "a@b.co", 0, "email_taken"))
        assertEquals(SignInStep.SIGN_UP, taken.step)
        assertEquals(CodeFailure.EMAIL_TAKEN, taken.failure)

        // A code that the core did not report as the verify step is not a success.
        val odd = up.afterSignUp(null, CodeState("reset_code", "a@b.co", 30, ""))
        assertEquals(SignInStep.SIGN_UP, odd.step)
    }

    @Test
    fun `forgot password moves to the reset code step`() {
        val flow = SignInFlow(step = SignInStep.RESET_EMAIL, email = "a@b.co")
        val sent = flow.afterResetRequest(null, CodeState("reset_code", "a@b.co", 30, ""))
        assertEquals(SignInStep.RESET_CODE, sent.step)
        val network = flow.afterResetRequest("down", CodeState("", "", 0, "network"))
        assertEquals(SignInStep.RESET_EMAIL, network.step)
        assertEquals(CodeFailure.NETWORK, network.failure)
    }

    @Test
    fun `a sign-in to an unverified account shows verify`() {
        val flow = SignInFlow(step = SignInStep.SIGN_IN, email = "a@b.co", attempt = 1)
            .afterVerifyAnswer(CodeState("verify", "a@b.co", 30, ""))
        assertEquals(SignInStep.VERIFY, flow.step)
        assertEquals(30, flow.resendIn)
        assertEquals(2, flow.attempt)
        // Without a readable state the countdown starts over and the address stays.
        val blind = SignInFlow(email = "a@b.co").afterVerifyAnswer(null)
        assertEquals(SignInStep.VERIFY, blind.step)
        assertEquals("a@b.co", blind.email)
        assertEquals(30, blind.resendIn)
    }

    @Test
    fun `a wrong code empties the cells and an expired one restarts the countdown`() {
        val code = SignInFlow(step = SignInStep.VERIFY, email = "a@b.co", attempt = 2)
        val wrong = code.failed(CodeState("verify", "a@b.co", 10, "wrong_code"), "that code isn't right")
        assertEquals(CodeFailure.WRONG_CODE, wrong.failure)
        assertEquals(3, wrong.attempt)
        assertEquals(SignInStep.VERIFY, wrong.step)
        val expired = code.failed(CodeState("verify", "a@b.co", 30, "expired"), "x")
        assertEquals(CodeFailure.EXPIRED, expired.failure)
        assertEquals(30, expired.resendIn)
    }

    @Test
    fun `only the two tabs are tabs`() {
        assertTrue(SignInStep.SIGN_IN.isTab())
        assertTrue(SignInStep.SIGN_UP.isTab())
        assertFalse(SignInStep.VERIFY.isTab())
        assertFalse(SignInStep.RESET_EMAIL.isTab())
        assertFalse(SignInStep.RESET_CODE.isTab())
    }

    @Test
    fun `a password needs eight characters and the bar has four steps`() {
        assertFalse(passwordLongEnough("short"))
        assertTrue(passwordLongEnough("12345678"))
        assertTrue(passwordLongEnough("短短短短短短短短"))
        assertEquals(0, passwordStrength(""))
        assertEquals(0, passwordStrength("short"))
        assertEquals(1, passwordStrength("lowercas"))
        assertEquals(1, passwordStrength("12345678"))
        assertEquals(1, passwordStrength("短短短短短短短短"))
        assertEquals(2, passwordStrength("alllowercase"))
        assertEquals(2, passwordStrength("abcdefgh1"))
        assertEquals(3, passwordStrength("Lowercase1"))
        assertEquals(3, passwordStrength("correct horse battery"))
        assertEquals(4, passwordStrength("Correct-Horse-9"))
    }

    @Test
    fun `the code mails follow the language of the app`() {
        assertEquals("zh-Hans", syncLang(Locale.SIMPLIFIED_CHINESE))
        assertEquals("zh-Hans", syncLang(Locale.forLanguageTag("zh-Hant-TW")))
        assertEquals("en", syncLang(Locale.US))
        assertEquals("en", syncLang(Locale.GERMANY))
    }
}

/** The server row: the official address is hidden, any other is shown, and empty means official. */
class SyncServerRulesTest {
    @Test
    fun `the official server compares without spaces, a trailing slash or case`() {
        assertTrue(isOfficialSyncServer(DEFAULT_SYNC_SERVER))
        assertTrue(isOfficialSyncServer("$DEFAULT_SYNC_SERVER/"))
        assertTrue(isOfficialSyncServer("  $DEFAULT_SYNC_SERVER/ "))
        assertTrue(isOfficialSyncServer("HTTPS://TODO.SRI-AI.XYZ"))
    }

    @Test
    fun `any other address is a custom server`() {
        assertFalse(isOfficialSyncServer("https://todo.example.com"))
        assertFalse(isOfficialSyncServer("http://todo.sri-ai.xyz"))
        assertFalse(isOfficialSyncServer("$DEFAULT_SYNC_SERVER/v1"))
        assertFalse(isOfficialSyncServer(""))
    }

    @Test
    fun `an empty field saves the official server`() {
        assertEquals(DEFAULT_SYNC_SERVER, syncServerToSave(""))
        assertEquals(DEFAULT_SYNC_SERVER, syncServerToSave("   "))
        assertEquals("https://todo.example.com", syncServerToSave("  https://todo.example.com "))
    }
}
