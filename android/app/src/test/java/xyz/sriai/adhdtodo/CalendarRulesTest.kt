package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.data.CalendarItem
import xyz.sriai.adhdtodo.data.CalendarPage
import xyz.sriai.adhdtodo.data.QuickDraft
import xyz.sriai.adhdtodo.data.QuickParse
import xyz.sriai.adhdtodo.ui.Dot
import xyz.sriai.adhdtodo.ui.calendarClock
import xyz.sriai.adhdtodo.ui.calendarCount
import xyz.sriai.adhdtodo.ui.calendarDays
import xyz.sriai.adhdtodo.ui.calendarDot
import xyz.sriai.adhdtodo.ui.calendarDots
import xyz.sriai.adhdtodo.ui.calendarDraftDue
import xyz.sriai.adhdtodo.ui.calendarDue
import xyz.sriai.adhdtodo.ui.calendarShortDay
import xyz.sriai.adhdtodo.ui.calendarTitle
import xyz.sriai.adhdtodo.ui.coreWeekStart
import xyz.sriai.adhdtodo.ui.emptyGrid
import xyz.sriai.adhdtodo.ui.gridStart
import xyz.sriai.adhdtodo.ui.gridWeeks
import xyz.sriai.adhdtodo.ui.key
import xyz.sriai.adhdtodo.ui.monthOfKey
import xyz.sriai.adhdtodo.ui.quickReadsDate
import xyz.sriai.adhdtodo.ui.shiftMonth
import xyz.sriai.adhdtodo.ui.swipeMonths
import xyz.sriai.adhdtodo.ui.systemWeekStart
import xyz.sriai.adhdtodo.ui.weekdaysFrom
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.YearMonth
import java.time.ZoneId
import java.util.Locale

/** The Calendar tab's rules (v44): the grid, the dots, where the selection goes, and the default due. */
class CalendarRulesTest {
    private val zone = ZoneId.of("Asia/Shanghai")
    private val november = YearMonth.of(2026, 11)

    private fun item(kind: String, overdue: Boolean = false) = CalendarItem(kind = kind, overdue = overdue)

    @Test
    fun theGridStartsOnTheFirstWeekdayOnOrBeforeThe1stAndNeedsFourToSixWeeks() {
        // November 2026 starts on a Sunday.
        assertEquals(LocalDate.of(2026, 10, 26), gridStart(november, DayOfWeek.MONDAY))
        assertEquals(6, gridWeeks(november, DayOfWeek.MONDAY))
        assertEquals(LocalDate.of(2026, 11, 1), gridStart(november, DayOfWeek.SUNDAY))
        assertEquals(5, gridWeeks(november, DayOfWeek.SUNDAY))
        // February 2027 starts on a Monday and has 28 days.
        assertEquals(LocalDate.of(2027, 2, 1), gridStart(YearMonth.of(2027, 2), DayOfWeek.MONDAY))
        assertEquals(4, gridWeeks(YearMonth.of(2027, 2), DayOfWeek.MONDAY))
        assertEquals(5, gridWeeks(YearMonth.of(2027, 2), DayOfWeek.SUNDAY))
    }

    @Test
    fun theEmptyGridMarksTodayWeekendsAndOtherMonths() {
        val days = emptyGrid(november, DayOfWeek.MONDAY, LocalDate.of(2026, 11, 3))
        assertEquals(42, days.size)
        assertEquals("2026-10-26", days.first().date)
        assertEquals("2026-12-06", days.last().date)
        assertFalse(days.first().inMonth)
        assertTrue(days.single { it.date == "2026-11-03" }.today)
        assertEquals(1, days.count { it.today })
        assertEquals(listOf("2026-10-31", "2026-11-01"), days.take(7).filter { it.weekend }.map { it.date })
        assertEquals(30, days.count { it.inMonth })
    }

    @Test
    fun weekdaysStartOnTheFirstWeekday() {
        assertEquals(DayOfWeek.MONDAY, weekdaysFrom(DayOfWeek.MONDAY).first())
        assertEquals(DayOfWeek.SUNDAY, weekdaysFrom(DayOfWeek.MONDAY).last())
        assertEquals(DayOfWeek.SATURDAY, weekdaysFrom(DayOfWeek.SATURDAY).first())
        assertEquals(DayOfWeek.FRIDAY, weekdaysFrom(DayOfWeek.SATURDAY).last())
    }

    @Test
    fun theCoreCountsWeekdaysFromSunday() {
        assertEquals(0, coreWeekStart(DayOfWeek.SUNDAY))
        assertEquals(1, coreWeekStart(DayOfWeek.MONDAY))
        assertEquals(6, coreWeekStart(DayOfWeek.SATURDAY))
    }

    @Test
    fun theSystemsFirstWeekdayFollowsTheLocale() {
        assertEquals(DayOfWeek.SUNDAY, systemWeekStart(Locale.US))
        assertEquals(DayOfWeek.MONDAY, systemWeekStart(Locale.UK))
        // The user's own choice in Android's settings rides on the locale and wins.
        assertEquals(DayOfWeek.MONDAY, systemWeekStart(Locale.forLanguageTag("en-US-u-fw-mon")))
        assertEquals(DayOfWeek.SATURDAY, systemWeekStart(Locale.forLanguageTag("zh-CN-u-fw-sat")))
    }

    @Test
    fun aCellHasAtMostThreeDotsInTheCoresOrder() {
        assertEquals(emptyList<Dot>(), calendarDots(emptyList()))
        assertEquals(
            listOf(Dot.Milestone, Dot.Open, Dot.Overdue),
            calendarDots(listOf(item("milestone"), item("reminder"), item("due", overdue = true), item("done"))),
        )
        assertEquals(
            listOf(Dot.Overdue, Dot.Open, Dot.Finished),
            calendarDots(listOf(item("due", overdue = true), item("due"), item("done"), item("done"))),
        )
        assertEquals(3, calendarDots(List(9) { item("due") }).size)
    }

    @Test
    fun aDotIsOpenOverdueFinishedOrAMilestone() {
        assertEquals(Dot.Open, calendarDot(item("due")))
        assertEquals(Dot.Open, calendarDot(item("reminder")))
        assertEquals(Dot.Overdue, calendarDot(item("reminder", overdue = true)))
        assertEquals(Dot.Finished, calendarDot(item("done")))
        // A finished task is never overdue, whatever the flag says.
        assertEquals(Dot.Finished, calendarDot(item("done", overdue = true)))
        assertEquals(Dot.Milestone, calendarDot(item("milestone")))
    }

    @Test
    fun theDayCountIsItsTasksNotItsMilestones() {
        assertEquals(2, calendarCount(listOf(item("milestone"), item("due"), item("done"))))
        assertEquals(0, calendarCount(listOf(item("milestone"))))
    }

    @Test
    fun aPageForAnotherMonthOrWeekStartIsNotDrawn() {
        val grid = emptyGrid(november, DayOfWeek.MONDAY, LocalDate.of(2026, 11, 3))
        val page = CalendarPage(year = 2026, month = 11, weeks = 6, days = grid)
        val today = LocalDate.of(2026, 11, 3)
        assertEquals(grid, calendarDays(page, november, DayOfWeek.MONDAY, today))
        // The page of October is not November's grid; neither is the same month with Sunday first.
        assertEquals("2026-09-28", calendarDays(page, YearMonth.of(2026, 10), DayOfWeek.MONDAY, today).first().date)
        assertEquals(35, calendarDays(page, november, DayOfWeek.SUNDAY, today).size)
        assertEquals(42, calendarDays(null, november, DayOfWeek.MONDAY, today).size)
    }

    @Test
    fun previousAndNextKeepTheDayNumberAndClampToAShorterMonth() {
        val (month, day) = shiftMonth(november, LocalDate.of(2026, 11, 3), 1)
        assertEquals(YearMonth.of(2026, 12), month)
        assertEquals(LocalDate.of(2026, 12, 3), day)
        val (feb, last) = shiftMonth(YearMonth.of(2027, 1), LocalDate.of(2027, 1, 31), 1)
        assertEquals(YearMonth.of(2027, 2), feb)
        assertEquals(LocalDate.of(2027, 2, 28), last)
        val (back, same) = shiftMonth(YearMonth.of(2027, 1), LocalDate.of(2027, 1, 15), -1)
        assertEquals(YearMonth.of(2026, 12), back)
        assertEquals(LocalDate.of(2026, 12, 15), same)
    }

    @Test
    fun theMonthSurvivesAsAKey() {
        for (month in listOf(YearMonth.of(2026, 1), YearMonth.of(2026, 11), YearMonth.of(2026, 12), YearMonth.of(2027, 1))) {
            assertEquals(month, monthOfKey(month.key()))
        }
        assertEquals(1, YearMonth.of(2027, 1).key() - YearMonth.of(2026, 12).key())
    }

    @Test
    fun aHorizontalSwipePagesTheMonthPastTheThreshold() {
        assertEquals(1, swipeMonths(-120f, 100f))
        assertEquals(-1, swipeMonths(120f, 100f))
        assertEquals(0, swipeMonths(-99f, 100f))
        assertEquals(0, swipeMonths(0f, 100f))
    }

    @Test
    fun aTaskTypedUnderADayIsDueThatDayAt1800() {
        assertEquals("2026-11-06T18:00:00+08:00", calendarDue(LocalDate.of(2026, 11, 6), zone))
    }

    @Test
    fun theDefaultDueYieldsToWhatTheTextSaid() {
        val fallback = calendarDue(LocalDate.of(2026, 11, 6), zone)
        // Nothing read, or only a category: the day's 18:00.
        assertEquals(fallback, calendarDraftDue(QuickDraft(title = "Pay rent"), fallback))
        assertEquals(fallback, calendarDraftDue(QuickDraft(title = "Pay rent", category = "Life"), fallback))
        // A due date or a time in the text wins; a reminder time is its own date, so no due is added.
        assertEquals("2026-11-09T18:00:00+08:00", calendarDraftDue(QuickDraft(dueAt = "2026-11-09T18:00:00+08:00"), fallback))
        assertNull(calendarDraftDue(QuickDraft(remindAt = "2026-11-04T09:00:00+08:00"), fallback))
        // Outside the Calendar there is no default.
        assertNull(calendarDraftDue(QuickDraft(title = "Pay rent"), null))
    }

    @Test
    fun theReceiptNamesTheDayOnlyWhenTheTextGaveNone() {
        assertFalse(quickReadsDate(QuickParse(listOf(QuickDraft(title = "Pay rent", category = "Life")))))
        assertTrue(quickReadsDate(QuickParse(listOf(QuickDraft(dueAt = "2026-11-09T18:00:00+08:00")))))
        assertTrue(quickReadsDate(QuickParse(listOf(QuickDraft(remindAt = "2026-11-04T09:00:00+08:00")))))
    }

    @Test
    fun theWordsAreWrittenInTheLanguage() {
        val day = LocalDate.of(2026, 11, 3)
        assertEquals("2026年11月", calendarTitle(november, Locale.SIMPLIFIED_CHINESE))
        assertEquals("November 2026", calendarTitle(november, Locale.US))
        assertEquals("11月3日", calendarShortDay(day, Locale.SIMPLIFIED_CHINESE))
        assertEquals("Nov 3", calendarShortDay(day, Locale.US))
        assertEquals("下午 5:00", calendarClock("2026-11-03T17:00:00+08:00", Locale.SIMPLIFIED_CHINESE, zone))
        assertEquals("5:00 PM", calendarClock("2026-11-03T17:00:00+08:00", Locale.US, zone))
        assertEquals("", calendarClock("", Locale.US, zone))
    }
}
