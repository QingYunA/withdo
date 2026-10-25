package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.data.UPDATE_CHECK_INTERVAL_MILLIS
import xyz.sriai.adhdtodo.data.UpdatePhase
import xyz.sriai.adhdtodo.data.UpdateUi
import xyz.sriai.adhdtodo.data.parseSemver
import xyz.sriai.adhdtodo.data.pickUpdate
import xyz.sriai.adhdtodo.data.sha256For
import xyz.sriai.adhdtodo.data.updateCheckDue

/** The phone updater's pure rules: versions, the release answer, the checksum line, the daily check. */
class UpdateRulesTest {
    private fun release(
        tag: String = "v0.9.0",
        apk: Boolean = true,
        sums: Boolean = true,
        draft: Boolean = false,
        prerelease: Boolean = false,
    ): String {
        val assets = buildList {
            add("""{"name":"ADHD-Todo-0.9.0-macos-universal.dmg","browser_download_url":"https://example.com/dmg"}""")
            if (apk) add("""{"name":"adhd-todo-${tag.removePrefix("v")}.apk","browser_download_url":"https://example.com/apk"}""")
            if (sums) add("""{"name":"SHA256SUMS.txt","browser_download_url":"https://example.com/sums"}""")
        }
        return """{"tag_name":"$tag","draft":$draft,"prerelease":$prerelease,"assets":[${assets.joinToString(",")}],"body":"x"}"""
    }

    @Test
    fun `versions compare number by number, not as text`() {
        assertTrue(parseSemver("v0.10.0")!! > parseSemver("0.9.9")!!)
        assertTrue(parseSemver("1.0.0")!! > parseSemver("0.99.99")!!)
        assertEquals(parseSemver("v0.8.1"), parseSemver("0.8.1"))
        assertEquals("0.8.1", parseSemver(" v0.8.1 ").toString())
    }

    @Test
    fun `text that is not x y z is not a version`() {
        for (raw in listOf("", "v", "0.9", "0.9.0.1", "0.9.0-rc1", "latest", "v1.x.0", "-1.0.0")) {
            assertNull(raw, parseSemver(raw))
        }
    }

    @Test
    fun `a newer release with its apk and checksums is an update`() {
        val update = pickUpdate(release(), "0.8.1")!!
        assertEquals("0.9.0", update.version)
        assertEquals("https://example.com/apk", update.apkUrl)
        assertEquals("https://example.com/sums", update.sumsUrl)
    }

    @Test
    fun `the same or an older release is no update`() {
        assertNull(pickUpdate(release("v0.8.1"), "0.8.1"))
        assertNull(pickUpdate(release("v0.8.0"), "0.8.1"))
    }

    @Test
    fun `a release without the apk or the checksums waits`() {
        assertNull(pickUpdate(release(apk = false), "0.8.1"))
        assertNull(pickUpdate(release(sums = false), "0.8.1"))
    }

    @Test
    fun `drafts, pre-releases and answers that are not releases are ignored`() {
        assertNull(pickUpdate(release(draft = true), "0.8.1"))
        assertNull(pickUpdate(release(prerelease = true), "0.8.1"))
        assertNull(pickUpdate(release("nightly"), "0.8.1"))
        assertNull(pickUpdate("""{"message":"API rate limit exceeded"}""", "0.8.1"))
        assertNull(pickUpdate("not json", "0.8.1"))
        assertNull(pickUpdate(release(), "debug"))
    }

    @Test
    fun `an apk url that is not https is refused`() {
        val body = release().replace("https://example.com/apk", "http://example.com/apk")
        assertNull(pickUpdate(body, "0.8.1"))
    }

    @Test
    fun `the checksum is the line of the file's name`() {
        val a = "a".repeat(64)
        val b = "B".repeat(64)
        val sums = "$a  ADHD-Todo-0.9.0-macos-universal.dmg\n$b *adhd-todo-0.9.0.apk\n\n"
        assertEquals("b".repeat(64), sha256For(sums, "adhd-todo-0.9.0.apk"))
        assertEquals(a, sha256For(sums, "ADHD-Todo-0.9.0-macos-universal.dmg"))
        assertNull(sha256For(sums, "adhd-todo-0.8.1.apk"))
        assertNull(sha256For("${"z".repeat(64)}  adhd-todo-0.9.0.apk", "adhd-todo-0.9.0.apk"))
        assertNull(sha256For("abc  adhd-todo-0.9.0.apk", "adhd-todo-0.9.0.apk"))
    }

    @Test
    fun `a check is due once a day`() {
        val day = UPDATE_CHECK_INTERVAL_MILLIS
        assertTrue(updateCheckDue(0, 5 * day))
        assertFalse(updateCheckDue(5 * day, 5 * day + day - 1))
        assertTrue(updateCheckDue(5 * day, 5 * day + day))
        assertTrue("a clock set back", updateCheckDue(5 * day, 4 * day))
    }

    @Test
    fun `a closed banner stays closed except while it works`() {
        val offer = UpdateUi(UpdatePhase.AVAILABLE, current = "0.8.1", version = "0.9.0")
        assertTrue(offer.bannerShown)
        assertFalse(offer.copy(dismissed = true).bannerShown)
        assertTrue(offer.copy(phase = UpdatePhase.DOWNLOADING, dismissed = true).bannerShown)
        assertFalse(UpdateUi(UpdatePhase.UP_TO_DATE).bannerShown)
        assertFalse(UpdateUi(UpdatePhase.CHECKING).bannerShown)
    }
}
