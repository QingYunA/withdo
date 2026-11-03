package xyz.sriai.adhdtodo

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.test.assertIsSelected
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.longClick
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.swipe
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.CalendarScreen
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.calendarCellTag
import java.time.DayOfWeek
import java.time.LocalDate

/** The Calendar tab's touches (v44), driven on the JVM through Compose's test semantics. */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class CalendarInteractionTest {
    @get:Rule
    val compose = createComposeRule()

    private val categories: List<Category> = Golden.load("categories.golden.json")

    private class Seen {
        var shifted = 0
        var opened = 0L
        var held = 0L
        var completed = 0L
    }

    @Composable
    private fun Calendar(start: LocalDate, seen: Seen, onSelected: (LocalDate) -> Unit = {}) {
        val resources = androidx.compose.ui.platform.LocalContext.current.resources
        val words = remember { Words(resources, FixedTime.clock, FixedTime.zone) }
        var selected by remember { mutableStateOf(start) }
        CalendarScreen(
            page = November.page(categories),
            month = November.month,
            selected = selected,
            today = November.today,
            weekStart = DayOfWeek.MONDAY,
            words = words,
            categories = categories,
            onSelect = { selected = it; onSelected(it) },
            onShift = { seen.shifted += it },
            onOpenTask = { seen.opened = it },
            onLongPress = { seen.held = it },
            onComplete = { seen.completed = it },
            zone = FixedTime.zone,
        )
    }

    @Test
    fun `a tap on a day selects it and lists its rows under the grid`() {
        val seen = Seen()
        var picked: LocalDate? = null
        compose.setThemedContent { Calendar(November.today, seen) { picked = it } }
        compose.onNodeWithTag(calendarCellTag("2026-11-03")).assertIsSelected()
        compose.onNodeWithText("Reply to design feedback").assertExists()
        compose.onNodeWithTag(calendarCellTag("2026-11-06")).performClick()
        assertEquals(LocalDate.of(2026, 11, 6), picked)
        compose.onNodeWithTag(calendarCellTag("2026-11-06")).assertIsSelected()
        compose.onNodeWithText("Fri, Nov 6").assertExists()
        compose.onNodeWithText("v0.13 due").assertExists()
        compose.onNodeWithText("Call the landlord").assertExists()
        compose.onNodeWithText("Write the release notes").assertExists()
        compose.onNodeWithText("Reply to design feedback").assertDoesNotExist()
    }

    @Test
    fun `a day with nothing says so`() {
        compose.setThemedContent { Calendar(LocalDate.of(2026, 11, 5), Seen()) }
        compose.onNodeWithText("Nothing on this day").assertExists()
    }

    @Test
    fun `a row opens its task on a tap and the task menu on a long press`() {
        val seen = Seen()
        compose.setThemedContent { Calendar(LocalDate.of(2026, 11, 6), seen) }
        compose.onNodeWithText("Call the landlord").performClick()
        assertEquals(4L, seen.opened)
        compose.onNodeWithText("Write the release notes").performTouchInput { longClick() }
        assertEquals(5L, seen.held)
    }

    @Test
    fun `a swipe across the grid pages the month`() {
        val seen = Seen()
        compose.setThemedContent { Calendar(November.today, seen) }
        compose.onNodeWithTag(calendarCellTag("2026-11-10")).performTouchInput {
            swipe(Offset(centerX + 100f, centerY), Offset(centerX - 100f, centerY))
        }
        assertEquals(1, seen.shifted)
        compose.onNodeWithTag(calendarCellTag("2026-11-10")).performTouchInput {
            swipe(Offset(centerX - 100f, centerY), Offset(centerX + 100f, centerY))
        }
        assertEquals(0, seen.shifted)
    }
}
