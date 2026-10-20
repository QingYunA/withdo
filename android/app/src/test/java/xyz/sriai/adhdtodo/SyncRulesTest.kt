package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.ui.SYNC_ATTENTION_MILLIS
import xyz.sriai.adhdtodo.ui.deviceUsesLaptop
import xyz.sriai.adhdtodo.ui.groupThousands
import xyz.sriai.adhdtodo.ui.isAuthFailure
import xyz.sriai.adhdtodo.ui.syncBarFilled
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
    fun `a failure that means the sign-in expired is worded as such`() {
        assertTrue(isAuthFailure("401 Unauthorized"))
        assertTrue(isAuthFailure("the refresh token was rejected"))
        assertFalse(isAuthFailure("the server did not answer"))
    }
}
