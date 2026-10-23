package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.data.PageJson
import xyz.sriai.adhdtodo.data.SignInState
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.ui.GoogleLinkRow
import xyz.sriai.adhdtodo.ui.SyncFailure
import xyz.sriai.adhdtodo.ui.SyncPhase
import xyz.sriai.adhdtodo.ui.googleLinkFailureText
import xyz.sriai.adhdtodo.ui.syncCloudBlocked
import xyz.sriai.adhdtodo.ui.syncFailureOf
import xyz.sriai.adhdtodo.ui.syncFailureText
import xyz.sriai.adhdtodo.ui.syncPhaseOf
import xyz.sriai.adhdtodo.ui.syncRunnerMayStart

/** A phone linked to another account than the one signed in: the wrong-account state and its wording. */
class WrongAccountRulesTest {
    @Test
    fun `the other_account answer decodes with both emails`() {
        val answer = PageJson.decodeFromString<SignInState>(
            """{"state":"other_account","previous_email":"a@x.com","account_email":"b@x.com"}""",
        )
        assertEquals("other_account", answer.state)
        assertEquals("a@x.com", answer.previousEmail)
        assertEquals("b@x.com", answer.accountEmail)
        val unknown = PageJson.decodeFromString<SignInState>(
            """{"state":"other_account","previous_email":"","account_email":"b@x.com"}""",
        )
        assertEquals("", unknown.previousEmail)
        assertEquals("", PageJson.decodeFromString<SignInState>("""{"state":"linked"}""").accountEmail)
    }

    @Test
    fun `the wrong account state has its own phase, no retry and a red cloud`() {
        val wrong = SyncStatus(
            state = "wrong account", linked = true, signedIn = true,
            lastError = "this phone is linked to another account",
        )
        assertEquals(SyncPhase.WRONG_ACCOUNT, syncPhaseOf(wrong, signingIn = false, choosing = false))
        assertFalse(syncRunnerMayStart(wrong.state))
        assertTrue(syncRunnerMayStart("error"))
        assertTrue(syncRunnerMayStart("signed out"))
        assertTrue(syncCloudBlocked(true, true, "wrong account", null, 0L))
    }

    @Test
    fun `the other account sheet outranks every other phase`() {
        val status = SyncStatus(state = "idle", linked = true, signedIn = true)
        assertEquals(
            SyncPhase.OTHER_ACCOUNT,
            syncPhaseOf(status, signingIn = false, choosing = false, otherAccount = true),
        )
    }

    @Test
    fun `only a real network failure says the server did not answer`() {
        assertEquals(SyncFailure.NETWORK, syncFailureOf("could not reach the server; check the connection"))
        assertEquals(SyncFailure.NETWORK, syncFailureOf("the server did not answer"))
        assertEquals(SyncFailure.SERVER, syncFailureOf("the server had a problem; try again later"))
        assertEquals(SyncFailure.REFUSED, syncFailureOf("the server refused the request"))
        assertEquals(SyncFailure.OTHER, syncFailureOf("something went wrong; try again"))
        assertEquals(SyncFailure.OTHER, syncFailureOf("some unknown error"))
        assertEquals(SyncFailure.OTHER, syncFailureOf(""))
        assertEquals(SyncFailure.AUTH, syncFailureOf("sign in again"))
        assertEquals(SyncFailure.AUTH, syncFailureOf("401 Unauthorized"))
        assertEquals(R.string.sync_failed_no_answer, syncFailureText(SyncFailure.NETWORK))
        for (kind in SyncFailure.values().filter { it != SyncFailure.NETWORK }) {
            assertTrue(kind.name, syncFailureText(kind) != R.string.sync_failed_no_answer)
        }
    }

    @Test
    fun `google link failures map to lines and cancelled is quiet`() {
        assertEquals(R.string.sync_google_link_taken, googleLinkFailureText("already_linked"))
        assertEquals(R.string.sync_google_failed, googleLinkFailureText("failed"))
        assertEquals(R.string.sync_sign_in_network, googleLinkFailureText("network"))
        assertEquals(R.string.sync_google_failed, googleLinkFailureText("signed_out"))
        assertEquals(R.string.sync_google_failed, googleLinkFailureText("whatever"))
        assertEquals(null, googleLinkFailureText("cancelled"))
    }

    @Test
    fun `the google row shows only where the server offers it`() {
        assertFalse(GoogleLinkRow().shown)
        assertTrue(GoogleLinkRow(available = true).shown)
        assertFalse(GoogleLinkRow(available = true).linked)
        assertTrue(GoogleLinkRow(available = true, linked = true).linked)
    }
}
