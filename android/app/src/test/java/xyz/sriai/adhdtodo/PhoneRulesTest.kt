package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.ui.DueKind
import xyz.sriai.adhdtodo.ui.DueOption
import xyz.sriai.adhdtodo.ui.RowSwipes
import xyz.sriai.adhdtodo.ui.ShareDraft
import xyz.sriai.adhdtodo.ui.ShareWhen
import xyz.sriai.adhdtodo.ui.SwipeAction
import xyz.sriai.adhdtodo.ui.linkLabel
import xyz.sriai.adhdtodo.ui.pullSyncs
import xyz.sriai.adhdtodo.ui.rowSwipes
import xyz.sriai.adhdtodo.ui.shareDraft
import xyz.sriai.adhdtodo.ui.shareWhen
import xyz.sriai.adhdtodo.ui.swipeOnRelease
import java.time.LocalDate
import java.time.ZoneId

/** The rules of the phone's own gestures and surfaces (v43 P2): swipes, pull to sync and sharing. */
class PhoneRulesTest {
    private val zone = ZoneId.of("Asia/Shanghai")
    private val today = LocalDate.of(2026, 10, 20)

    @Test
    fun aTaskOfMineSwipesRightToDoneAndLeftToToday() {
        assertEquals(
            RowSwipes(SwipeAction.Done, SwipeAction.Today),
            rowSwipes("user", "todo", "2026-10-21T09:00:00Z", today, zone),
        )
        assertEquals(RowSwipes(SwipeAction.Done, SwipeAction.Today), rowSwipes("user", "todo", null, today, zone))
        // Overdue moves to today too.
        assertEquals(SwipeAction.Today, rowSwipes("user", "doing", "2026-10-18T09:00:00Z", today, zone)?.left)
    }

    @Test
    fun aTaskDueTodayAlreadySwipesLeftToTomorrow() {
        // 22:00 in Shanghai on the 20th.
        assertEquals(SwipeAction.Tomorrow, rowSwipes("user", "doing", "2026-10-20T14:00:00Z", today, zone)?.left)
    }

    @Test
    fun anAgentsTaskOnlySwipesRightAndAFinishedOneNotAtAll() {
        assertEquals(RowSwipes(SwipeAction.Done, null), rowSwipes("agent", "review", null, today, zone))
        assertNull(rowSwipes("user", "done", null, today, zone))
        assertNull(rowSwipes("agent", "cancelled", null, today, zone))
    }

    @Test
    fun theActionRunsOnlyPastHalfTheRow() {
        val both = RowSwipes(SwipeAction.Done, SwipeAction.Today)
        assertNull(swipeOnRelease(both, 199f, 400))
        assertEquals(SwipeAction.Done, swipeOnRelease(both, 200f, 400))
        assertEquals(SwipeAction.Today, swipeOnRelease(both, -260f, 400))
        assertNull(swipeOnRelease(RowSwipes(SwipeAction.Done, null), -300f, 400))
        assertNull(swipeOnRelease(both, 300f, 0))
    }

    @Test
    fun aPullSyncsOnlyWhenSignedInAndLinked() {
        assertTrue(pullSyncs(SyncStatus(signedIn = true, linked = true)))
        assertFalse(pullSyncs(SyncStatus(signedIn = false, linked = true)))
        assertFalse(pullSyncs(SyncStatus(signedIn = true, linked = false)))
    }

    @Test
    fun aBrowsersShareIsThePageTitleAndItsLink() {
        assertEquals(
            ShareDraft("Android: swipe to complete a task", "https://github.com/QingYunA/withdo/issues/219"),
            shareDraft("https://github.com/QingYunA/withdo/issues/219", "Android: swipe to complete a task"),
        )
    }

    @Test
    fun sharedTextKeepsItsWordsAndLosesTheLink() {
        assertEquals(
            ShareDraft("Look at this: tomorrow", "https://example.com/a?b=1"),
            shareDraft("Look at this: https://example.com/a?b=1 tomorrow", null),
        )
        assertEquals(ShareDraft("Call the landlord at 9", null), shareDraft("  Call the landlord at 9 ", ""))
        // A link alone is its own title; trailing punctuation is not part of it.
        assertEquals(ShareDraft("https://example.com/x", "https://example.com/x"), shareDraft("https://example.com/x.", null))
        assertEquals(ShareDraft("看看这个 很好", "https://x.com/a"), shareDraft("看看这个https://x.com/a，很好", null))
        assertNull(shareDraft("   ", null))
        assertNull(shareDraft(null, null))
    }

    @Test
    fun theChipNamesTheHostAndTheEndOfThePath() {
        assertEquals("github.com › issues/219", linkLabel("https://github.com/QingYunA/withdo/issues/219"))
        assertEquals("example.com › a", linkLabel("https://www.example.com/a"))
        assertEquals("example.com", linkLabel("https://example.com/"))
    }

    @Test
    fun whenFollowsTheParserUntilThePersonPicks() {
        val friday = ShareWhen.Day(DueOption(DueKind.Friday, LocalDate.of(2026, 10, 23)))
        assertEquals(ShareWhen.NoDate, shareWhen(null, parserRead = false))
        assertEquals(ShareWhen.Read, shareWhen(null, parserRead = true))
        assertEquals(friday, shareWhen(friday, parserRead = true))
        assertEquals(ShareWhen.NoDate, shareWhen(ShareWhen.NoDate, parserRead = true))
        assertEquals(ShareWhen.NoDate, shareWhen(ShareWhen.Read, parserRead = false))
    }
}
