package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import xyz.sriai.adhdtodo.data.Milestone
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.platform.AppLanguage
import xyz.sriai.adhdtodo.platform.EndOfDay
import xyz.sriai.adhdtodo.ui.BoardLane
import xyz.sriai.adhdtodo.ui.boardLanes
import xyz.sriai.adhdtodo.ui.categoryColorKeys
import xyz.sriai.adhdtodo.ui.hashIndex
import xyz.sriai.adhdtodo.ui.sortedMilestones
import xyz.sriai.adhdtodo.ui.swap
import xyz.sriai.adhdtodo.ui.theme.AppearanceMode
import xyz.sriai.adhdtodo.ui.theme.LightTokens
import java.time.LocalDateTime

/** The pure rules behind the Board, the milestone filter, Categories, Settings and the end-of-day alarm. */
class ProjectsRulesTest {
    private fun task(id: Long, status: String, completedAt: String? = null) =
        Task(id = id, title = "t$id", status = status, completedAt = completedAt, updatedAt = "2026-10-01T00:00:00Z")

    private fun BoardLane.ids() = tasks.map { it.id }

    @Test
    fun `the board has four lanes and leaves backlog and cancelled off`() {
        val lanes = boardLanes(
            listOf(task(1, "todo"), task(2, "backlog"), task(3, "cancelled"), task(4, "review"), task(5, "doing")),
        )
        assertEquals(listOf("todo", "doing", "review", "done"), lanes.map { it.status })
        assertEquals(listOf(1L), lanes[0].ids())
        assertEquals(listOf(5L), lanes[1].ids())
        assertEquals(listOf(4L), lanes[2].ids())
        assertEquals(emptyList<Long>(), lanes[3].ids())
    }

    @Test
    fun `a blocked task sits first in Doing`() {
        val lanes = boardLanes(listOf(task(1, "doing"), task(2, "blocked"), task(3, "doing")))
        assertEquals(listOf(2L, 1L, 3L), lanes[1].ids())
    }

    @Test
    fun `Done is newest first`() {
        val lanes = boardLanes(
            listOf(
                task(1, "done", "2026-10-01T10:00:00Z"),
                task(2, "done", "2026-10-03T10:00:00Z"),
                task(3, "done", "2026-10-02T10:00:00Z"),
            ),
        )
        assertEquals(listOf(2L, 3L, 1L), lanes[3].ids())
    }

    @Test
    fun `milestones list open ones by due date, then closed ones`() {
        val sorted = sortedMilestones(
            listOf(
                Milestone(id = 1, title = "closed", open = false, dueAt = "2026-10-01T00:00:00Z"),
                Milestone(id = 2, title = "no date", open = true),
                Milestone(id = 3, title = "later", open = true, dueAt = "2026-12-01T00:00:00Z"),
                Milestone(id = 4, title = "soon", open = true, dueAt = "2026-11-01T00:00:00Z"),
            ),
        )
        assertEquals(listOf("soon", "later", "no date", "closed"), sorted.map { it.title })
    }

    @Test
    fun `a new category's color is the one its name hashes to, as for a project folder`() {
        for (name in listOf("Reading", "学习", "健康", "withdo", "")) {
            assertEquals(LightTokens.projectColor(name), LightTokens.projectColors[hashIndex(name)])
        }
        assertEquals(8, categoryColorKeys.size)
    }

    @Test
    fun `swap exchanges two places and leaves the list alone`() {
        val before = listOf(1, 2, 3)
        assertEquals(listOf(2, 1, 3), before.swap(0, 1))
        assertEquals(listOf(1, 2, 3), before)
    }

    @Test
    fun `the end-of-day alarm is today's time on a weekday before it, else the next weekday's`() {
        val minutes = 18 * 60 + 30
        // Tuesday 2026-10-20, 09:00: still ahead today.
        assertEquals(LocalDateTime.of(2026, 10, 20, 18, 30), EndOfDay.next(LocalDateTime.of(2026, 10, 20, 9, 0), minutes))
        // Tuesday 19:00: tomorrow.
        assertEquals(LocalDateTime.of(2026, 10, 21, 18, 30), EndOfDay.next(LocalDateTime.of(2026, 10, 20, 19, 0), minutes))
        // Friday evening: Monday.
        assertEquals(LocalDateTime.of(2026, 10, 26, 18, 30), EndOfDay.next(LocalDateTime.of(2026, 10, 23, 19, 0), minutes))
        // Saturday morning: Monday.
        assertEquals(LocalDateTime.of(2026, 10, 26, 18, 30), EndOfDay.next(LocalDateTime.of(2026, 10, 24, 8, 0), minutes))
    }

    @Test
    fun `appearance and language choices read back`() {
        assertEquals(AppearanceMode.Dark, AppearanceMode.of("dark"))
        assertEquals(AppearanceMode.System, AppearanceMode.of("nonsense"))
        assertEquals(AppearanceMode.System, AppearanceMode.of(null))
        assertNull(AppLanguage.localeOf(AppLanguage.SYSTEM))
        assertEquals("zh", AppLanguage.localeOf(AppLanguage.CHINESE)?.language)
        assertEquals("en", AppLanguage.localeOf(AppLanguage.ENGLISH)?.language)
    }
}
