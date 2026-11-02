package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.widget.agentsData
import xyz.sriai.adhdtodo.widget.rowsThatFit
import xyz.sriai.adhdtodo.widget.todayData

/** What the widgets show: which rows, how many, and the words under the agent's title. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class WidgetDataTest {
    private val res get() = RuntimeEnvironment.getApplication().resources
    private val words get() = Words(res, FixedTime.clock, FixedTime.zone)
    private val mine = Golden.load<MinePage>("mine.golden.json")

    private fun item(id: Long) = Item(id = id, title = "Task $id", status = "todo")

    @Test
    fun `Today counts every task and shows the first three`() {
        val page = MinePage(today = (1L..5L).map(::item))
        val data = todayData(page, words, res)
        assertEquals(5, data.count)
        assertEquals(listOf(1L, 2L, 3L), data.rows.map { it.id })
    }

    @Test
    fun `Today does not borrow from next when empty and says so quietly`() {
        val data = todayData(MinePage(next = listOf(item(8))), words, res)
        assertEquals(0, data.count)
        assertTrue(data.rows.isEmpty())
        assertEquals("Nothing for today", data.empty)
    }

    @Test
    fun `a core that could not be read leaves no quiet line`() {
        assertEquals("", todayData(null, words, res).empty)
        assertEquals("", agentsData(null, "", words, res).empty)
    }

    @Test
    fun `Today rows carry the due words of the screens`() {
        val rows = todayData(mine, words, res).rows
        assertEquals(listOf(7L, 6L), rows.map { it.id })
        assertEquals("5:00 PM", rows[0].due)
        assertEquals("Today", rows[1].due)
    }

    @Test
    fun `Agents shows the first card with its reason and age`() {
        val data = agentsData(mine, "", words, res)
        assertEquals(2, data.count)
        assertEquals(3L, data.card?.id)
        // The age's spaces do not break, so a narrow card wraps before the whole age.
        assertEquals("Waiting for your reply · 3\u00A0days\u00A0ago", data.card?.meta)
        assertEquals("Reply", data.replyLabel)
    }

    @Test
    fun `Agents with nothing waiting has no card`() {
        val data = agentsData(MinePage(), "", words, res)
        assertNull(data.card)
        assertEquals("No agent is waiting for you", data.empty)
    }

    @Test
    fun `a review card names the agent when it has no PR`() {
        val review = mine.fromAgents.first { it.kind == 5 }.copy(prNumber = 0)
        val data = agentsData(MinePage(fromAgents = listOf(review)), "Claude Code", words, res)
        assertTrue(data.card!!.meta.startsWith("Claude Code finished and is waiting for you"))
    }

    @Test
    fun `the height decides how many rows fit, one to three`() {
        assertEquals(3, rowsThatFit(176f))
        assertEquals(2, rowsThatFit(130f))
        assertEquals(1, rowsThatFit(110f))
        assertEquals(1, rowsThatFit(40f))
        assertEquals(3, rowsThatFit(400f))
    }
}
