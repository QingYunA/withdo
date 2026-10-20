package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import xyz.sriai.adhdtodo.ui.normalizeRoute
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
    fun `a failed sign-in shows the Go error's sentence`() {
        assertEquals(
            "could not reach the sign-in server",
            signInFailureText(Exception("could not reach the sign-in server")),
        )
        assertEquals("sign in again", signInFailureText(Exception("  sign in again  ")))
        assertNull(signInFailureText(Exception("   ")))
        assertNull(signInFailureText(null))
    }
}
