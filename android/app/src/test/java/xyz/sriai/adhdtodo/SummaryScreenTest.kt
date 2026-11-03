package xyz.sriai.adhdtodo

import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.SummaryHolder
import xyz.sriai.adhdtodo.ui.SummaryScreen
import xyz.sriai.adhdtodo.ui.Words
import java.time.ZonedDateTime

/** The Summary page's taps: the ledger opens inline, the segment and the paging ask for another period. */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class SummaryScreenTest {
    @get:Rule
    val compose = createComposeRule()

    private fun show(holder: SummaryHolder, onOpenTask: (Long) -> Unit = {}, onShare: () -> Unit = {}) {
        compose.setThemedContent {
            val configuration = LocalConfiguration.current
            val resources = LocalContext.current.resources
            val words = remember(configuration) { Words(resources, FixedTime.clock, FixedTime.zone) }
            SummaryScreen(
                holder = holder, words = words, onOpenTask = onOpenTask, onShare = onShare, zone = FixedTime.zone,
                now = ZonedDateTime.ofInstant(FixedTime.now, FixedTime.zone), calm = true,
            )
        }
    }

    @Test
    fun `a ledger row opens its tasks inline and a task opens its page`() {
        val holder = SummaryHolder().apply { page = dayPage(zh = false) }
        var opened = 0L
        show(holder, onOpenTask = { opened = it })
        compose.onNodeWithText("Port the MCP server").assertDoesNotExist()
        compose.onNodeWithText("Claude Code").performClick()
        compose.onNodeWithText("Port the MCP server").assertIsDisplayed()
        compose.onNodeWithText("Port the MCP server").performClick()
        assertEquals(5L, opened)
        compose.onNodeWithText("Claude Code").performClick()
        compose.onNodeWithText("Port the MCP server").assertDoesNotExist()
    }

    @Test
    fun `the paragraph shows only when one synced, the empty line only for an empty day`() {
        val withText = SummaryHolder().apply { page = dayPage(zh = false) }
        show(withText)
        compose.onNodeWithText("Today · written by the assistant").assertIsDisplayed()
        compose.onNodeWithText("Shipped today").assertIsDisplayed()
    }

    @Test
    fun `no paragraph means no bubble but the ledger stays`() {
        val holder = SummaryHolder().apply { page = dayPage(zh = false).copy(paragraph = "") }
        show(holder)
        compose.onNodeWithText("Today · written by the assistant").assertDoesNotExist()
        compose.onNodeWithText("Shipped today").assertIsDisplayed()
    }

    @Test
    fun `an empty day says so and has no ledger`() {
        val holder = SummaryHolder().apply { page = emptyPage() }
        show(holder)
        compose.onNodeWithText("Nothing finished yet today").assertIsDisplayed()
        compose.onNodeWithText("Shipped today").assertDoesNotExist()
    }

    @Test
    fun `the segment and the paging change the period and forward stops at today`() {
        val holder = SummaryHolder().apply { page = dayPage(zh = false) }
        show(holder)
        compose.onNodeWithText("This week").performClick()
        assertTrue(holder.week)
        assertEquals(0L, holder.back)
        compose.onNodeWithText("Today").performClick()
        assertFalse(holder.week)
        compose.onNodeWithContentDescription("Previous").performClick()
        assertEquals(1L, holder.back)
        compose.onNodeWithContentDescription("Next").performClick()
        assertEquals(0L, holder.back)
        compose.onNodeWithContentDescription("Next").performClick()
        assertEquals(0L, holder.back)
    }

    @Test
    fun `share calls back`() {
        var shared = 0
        show(SummaryHolder().apply { page = dayPage(zh = false) }, onShare = { shared++ })
        compose.onNodeWithContentDescription("Share").performClick()
        assertEquals(1, shared)
    }
}
