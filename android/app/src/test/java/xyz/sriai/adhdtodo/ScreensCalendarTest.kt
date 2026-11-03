package xyz.sriai.adhdtodo

import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import com.github.takahirom.roborazzi.captureRoboImage
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.ParameterizedRobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.CalendarItem
import xyz.sriai.adhdtodo.data.CalendarPage
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.Milestone
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.testing.screenshotFile
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.CalendarHeader
import xyz.sriai.adhdtodo.ui.CalendarScreen
import xyz.sriai.adhdtodo.ui.Tab
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.calendarShortDay
import xyz.sriai.adhdtodo.ui.calendarTitle
import xyz.sriai.adhdtodo.ui.emptyGrid
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.YearMonth

/**
 * The Calendar tab (v44) as full phone pictures: the month with its dots and the day under it, and a
 * day with a milestone and a reminder, light and dark, English and Chinese. The month is November
 * 2026 with today the 3rd, as in docs/design/mockups/v44.html, and the weeks start on Monday.
 */
@RunWith(ParameterizedRobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class ScreensCalendarTest(private val case: Case) {
    @get:Rule
    val compose = createComposeRule()

    @Test
    fun screenshot() {
        compose.setThemedContent(night = case.night, zh = case.zh) { CalendarPicture(case.selected) }
        compose.waitForIdle()
        compose.onRoot().captureRoboImage(screenshotFile(case.name))
    }

    data class Case(val name: String, val night: Boolean, val zh: Boolean, val selected: LocalDate)

    companion object {
        @JvmStatic
        @ParameterizedRobolectricTestRunner.Parameters(name = "{0}")
        fun cases(): List<Case> = listOf("calendar-month" to November.today, "calendar-day" to LocalDate.of(2026, 11, 6))
            .flatMap { (name, day) ->
                listOf(
                    Case("$name-light", night = false, zh = false, selected = day),
                    Case("$name-dark", night = true, zh = false, selected = day),
                    Case("$name-light-zh", night = false, zh = true, selected = day),
                    Case("$name-dark-zh", night = true, zh = true, selected = day),
                )
            }
    }
}

@Composable
private fun CalendarPicture(selected: LocalDate) {
    val resources = LocalContext.current.resources
    val locale = LocalConfiguration.current.locales[0]
    val words = remember(LocalConfiguration.current) { Words(resources, FixedTime.clock, FixedTime.zone) }
    val mine = remember { Golden.load<MinePage>("mine.golden.json") }
    val categories = remember { Golden.load<List<Category>>("categories.golden.json") }
    TestFrame(
        title = calendarTitle(November.month, locale),
        tab = Tab.CALENDAR,
        count = mine.fromAgents.size,
        composer = stringResource(R.string.calendar_add_to, calendarShortDay(selected, locale)),
        topBar = {
            CalendarHeader(
                title = calendarTitle(November.month, locale),
                syncBlocked = false,
                onSync = {},
                onPrevious = {},
                onToday = {},
                onNext = {},
            )
        },
    ) {
        CalendarScreen(
            page = November.page(categories),
            month = November.month,
            selected = selected,
            today = November.today,
            weekStart = DayOfWeek.MONDAY,
            words = words,
            categories = categories,
            onSelect = {},
            onShift = {},
            onOpenTask = {},
            onLongPress = {},
            onComplete = {},
            zone = FixedTime.zone,
        )
    }
}

/** November 2026 of the v44 mockup: the dots of its month view and the rows of the 3rd and the 6th. */
internal object November {
    val month: YearMonth = YearMonth.of(2026, 11)
    val today: LocalDate = LocalDate.of(2026, 11, 3)

    /** Tasks without a name of their own are numbered from 100; the named ones are 1 to 5. */
    private var nextId = 100L

    private fun at(day: Int, hour: Int, minute: Int = 0): String =
        "2026-11-%02dT%02d:%02d:00+08:00".format(day, hour, minute)

    private fun task(
        title: String,
        id: Long? = null,
        status: String = "todo",
        source: String = "user",
        categoryId: Int = 0,
        project: Long? = null,
        due: String? = null,
        remind: String? = null,
        done: String? = null,
    ) = Task(
        id = id ?: nextId++, title = title, status = status, source = source, categoryId = categoryId, projectId = project,
        dueAt = due, remindAt = remind, completedAt = done,
    )

    private fun open(t: Task, project: String = "", overdue: Boolean = false, kind: String = "due", agent: String = "") =
        CalendarItem(
            kind = kind, at = (if (kind == "reminder") t.remindAt else t.dueAt).orEmpty(), overdue = overdue,
            agent = agent, project = project, task = t,
        )

    private fun finished(day: Int) = CalendarItem(kind = "done", at = at(day, 15), task = task("Done", status = "done", done = at(day, 15)))

    private fun plain(day: Int, hour: Int = 18) = open(task("Task", due = at(day, hour)))

    /** The month's page: what mobile/core's Calendar(2026, 11, Monday) would send for these tasks. */
    fun page(categories: List<Category>): CalendarPage {
        val work = categories.first { it.builtin == "work" }.id.toInt()
        nextId = 100L
        val items = mapOf(
            1 to listOf(finished(1)),
            2 to listOf(finished(2)),
            3 to listOf(
                open(task("Renew the passport", id = 1, due = at(3, 8)), overdue = true),
                open(task("Reply to design feedback", id = 2, categoryId = work, due = at(3, 10))),
                open(task("Decide on the activation policy", id = 3, status = "doing", source = "agent", project = 1, due = at(3, 17)), project = "withdo", agent = "claude-code"),
            ),
            4 to listOf(plain(4)),
            6 to listOf(
                CalendarItem(kind = "milestone", at = at(6, 12), project = "withdo", milestone = Milestone(id = 9, title = "v0.13", open = true, dueAt = at(6, 12))),
                open(task("Call the landlord", id = 4, remind = at(6, 9)), kind = "reminder"),
                open(task("Write the release notes", id = 5, project = 1, due = at(6, 18)), project = "withdo"),
            ),
            7 to listOf(plain(7)),
            10 to listOf(plain(10), plain(10)),
            12 to listOf(plain(12)),
            14 to listOf(CalendarItem(kind = "milestone", at = at(14, 12), milestone = Milestone(id = 10, title = "v0.14", open = true, dueAt = at(14, 12)))),
            18 to listOf(plain(18), plain(18)),
            20 to listOf(plain(20)),
            24 to listOf(plain(24)),
            27 to listOf(plain(27), plain(27), plain(27)),
        )
        val days = emptyGrid(month, DayOfWeek.MONDAY, today).map { day ->
            val date = LocalDate.parse(day.date)
            val here = if (date.month == month.month) items[date.dayOfMonth].orEmpty() else emptyList()
            day.copy(items = here)
        }
        return CalendarPage(year = month.year, month = month.monthValue, weeks = days.size / 7, days = days)
    }
}
