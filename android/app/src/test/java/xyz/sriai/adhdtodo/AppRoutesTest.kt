package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import xyz.sriai.adhdtodo.ui.SignInFailure
import xyz.sriai.adhdtodo.ui.normalizeRoute
import xyz.sriai.adhdtodo.ui.signInFailureOf
import xyz.sriai.adhdtodo.ui.signInFailureSentence
import xyz.sriai.adhdtodo.ui.signInFailureText

/** The debug route and a notification's task_id become stack entries the App can open. */
class AppRoutesTest {

    @Test
    fun `a task route keeps its number and drops the project`() {
        assertEquals("task:7:0", normalizeRoute("task:7"))
        assertEquals("task:7:0", normalizeRoute("task:7:2"))
        assertEquals("task:42:0", normalizeRoute("task:42:0"))
    }

    @Test
    fun `the pages and a malformed route`() {
        assertEquals("home", normalizeRoute("home"))
        assertEquals("tab:projects", normalizeRoute("projects"))
        assertEquals("sync", normalizeRoute("sync"))
        assertEquals("project:3", normalizeRoute("project:3"))
        assertEquals("", normalizeRoute("task:not-a-number"))
        assertEquals("", normalizeRoute("nonsense"))
    }

    @Test
    fun `the Go sign-in sentences become the card's kinds`() {
        assertEquals(
            SignInFailure.CREDENTIALS,
            signInFailureOf("wrong account or password"),
        )
        assertEquals(
            SignInFailure.TOO_MANY,
            signInFailureOf("too many tries; wait a few minutes"),
        )
        assertEquals(
            SignInFailure.SECOND_FACTOR,
            signInFailureOf("this account uses two-step verification, which in-app sign-in does not support yet"),
        )
        assertEquals(
            SignInFailure.LOCKED,
            signInFailureOf("this account is deactivated; ask the admin"),
        )
        assertEquals(
            SignInFailure.UNSUPPORTED,
            signInFailureOf("this server does not support signing in from the app"),
        )
        assertEquals(
            SignInFailure.NETWORK,
            signInFailureOf("could not reach the server; check the connection"),
        )
        assertNull(signInFailureOf("the server answered 500"))
        assertNull(signInFailureOf("   "))
        assertNull(signInFailureOf(null))
    }

    @Test
    fun `every sign-in kind has its own localized sentence`() {
        val ids = SignInFailure.values().map { signInFailureText(it) }
        assertEquals("every kind has a string", ids.size, ids.toSet().size)
    }

    @Test
    fun `an unknown failure keeps the Go sentence`() {
        assertEquals(
            "could not reach the sign-in server",
            signInFailureSentence(Exception("could not reach the sign-in server")),
        )
        assertEquals("sign in again", signInFailureSentence(Exception("  sign in again  ")))
        assertNull(signInFailureSentence(Exception("   ")))
        assertNull(signInFailureSentence(null))
    }
}
