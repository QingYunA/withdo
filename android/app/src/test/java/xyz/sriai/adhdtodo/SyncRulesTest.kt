package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.data.CodeState
import xyz.sriai.adhdtodo.ui.CodeFailure
import xyz.sriai.adhdtodo.ui.SYNC_ATTENTION_MILLIS
import xyz.sriai.adhdtodo.ui.SignInFlow
import xyz.sriai.adhdtodo.ui.SignInStep
import xyz.sriai.adhdtodo.ui.afterRequest
import xyz.sriai.adhdtodo.ui.codeDigits
import xyz.sriai.adhdtodo.ui.codeFailureOf
import xyz.sriai.adhdtodo.ui.failed
import xyz.sriai.adhdtodo.ui.looksLikeEmail
import xyz.sriai.adhdtodo.ui.resendClock
import xyz.sriai.adhdtodo.ui.SignInStage
import xyz.sriai.adhdtodo.ui.deviceUsesLaptop
import xyz.sriai.adhdtodo.ui.groupThousands
import xyz.sriai.adhdtodo.ui.isAuthFailure
import xyz.sriai.adhdtodo.ui.syncBarFilled
import xyz.sriai.adhdtodo.ui.signInStageOf
import xyz.sriai.adhdtodo.ui.syncCloudBlocked

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
        assertEquals(CodeFailure.OTHER, codeFailureOf("surprise"))
    }

    @Test
    fun `a mailed code moves to the code step and an unsupported server to the old card`() {
        val sent = SignInFlow(email = "a@b.co").afterRequest(null, CodeState("code", "a@b.co", 30, ""))
        assertEquals(SignInStep.CODE, sent.step)
        assertEquals(30, sent.resendIn)
        assertEquals(null, sent.failure)

        val old = SignInFlow(email = "a@b.co").afterRequest("nope", CodeState("", "", 0, "unsupported"))
        assertEquals(SignInStep.LEGACY, old.step)

        val network = SignInFlow(email = "a@b.co").afterRequest("down", CodeState("", "", 0, "network"))
        assertEquals(SignInStep.EMAIL, network.step)
        assertEquals(CodeFailure.NETWORK, network.failure)
    }

    @Test
    fun `a wrong code empties the cells and an expired one restarts the countdown`() {
        val code = SignInFlow(step = SignInStep.CODE, email = "a@b.co", attempt = 2)
        val wrong = code.failed(CodeState("code", "a@b.co", 10, "wrong_code"), "that code isn't right")
        assertEquals(CodeFailure.WRONG_CODE, wrong.failure)
        assertEquals(3, wrong.attempt)
        assertEquals(SignInStep.CODE, wrong.step)
        val expired = code.failed(CodeState("code", "a@b.co", 30, "expired"), "x")
        assertEquals(CodeFailure.EXPIRED, expired.failure)
        assertEquals(30, expired.resendIn)
    }
}
