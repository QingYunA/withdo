package xyz.sriai.adhdtodo.ui

import xyz.sriai.adhdtodo.data.CalendarDay
import xyz.sriai.adhdtodo.data.CalendarItem
import xyz.sriai.adhdtodo.data.CalendarPage
import xyz.sriai.adhdtodo.data.QuickDraft
import xyz.sriai.adhdtodo.data.QuickParse
import java.time.DayOfWeek
import java.time.Instant
import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.YearMonth
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.temporal.WeekFields
import java.util.Locale

/*
 * The Calendar tab's rules, without any drawing (v44): what a cell's dots are, the weeks a month
 * needs, where Previous, Next and Today move the selection, and when a task typed under a day is
 * due. The day rules themselves (what counts on a day, in what order) are the core's: the desktop's
 * data.BuildCalendar.
 */

/** A dot of a cell: open, overdue, finished, or a milestone. */
enum class Dot { Open, Overdue, Finished, Milestone }

/** A cell holds at most this many dots. */
const val MaxDots = 3

/**
 * The dots of a day, in the order the core listed its items (milestones, open ones by time,
 * finished ones), the first [MaxDots] of them.
 */
fun calendarDots(items: List<CalendarItem>): List<Dot> = items.take(MaxDots).map(::calendarDot)

/** One item's dot: a milestone, a finished task (never overdue), an overdue one, or an open one. */
fun calendarDot(item: CalendarItem): Dot = when {
    item.kind == "milestone" -> Dot.Milestone
    item.kind == "done" -> Dot.Finished
    item.overdue -> Dot.Overdue
    else -> Dot.Open
}

/** What the list under the grid counts: the day's tasks; its milestones are chips, not rows. */
fun calendarCount(items: List<CalendarItem>): Int = items.count { it.kind != "milestone" }

/**
 * The system's first weekday for [locale]: the one the user chose in Android's settings (the locale's
 * `-u-fw-` extension), else the locale's own (Sunday for en-US, Monday for en-GB).
 */
fun systemWeekStart(locale: Locale): DayOfWeek =
    when (locale.getUnicodeLocaleType("fw")) {
        "mon" -> DayOfWeek.MONDAY
        "tue" -> DayOfWeek.TUESDAY
        "wed" -> DayOfWeek.WEDNESDAY
        "thu" -> DayOfWeek.THURSDAY
        "fri" -> DayOfWeek.FRIDAY
        "sat" -> DayOfWeek.SATURDAY
        "sun" -> DayOfWeek.SUNDAY
        else -> WeekFields.of(locale).firstDayOfWeek
    }

/** The core's weekStart for a weekday: 0 is Sunday, 1 Monday, up to 6 Saturday (Go's time.Weekday). */
fun coreWeekStart(day: DayOfWeek): Int = day.value % 7

/** The week's seven weekdays in order from [weekStart], for the row above the grid. */
fun weekdaysFrom(weekStart: DayOfWeek): List<DayOfWeek> = List(7) { weekStart.plus(it.toLong()) }

/** The first cell of the grid of [month]: the week start on or before the 1st (data.MonthGrid). */
fun gridStart(month: YearMonth, weekStart: DayOfWeek): LocalDate {
    val first = month.atDay(1)
    return first.minusDays(((first.dayOfWeek.value - weekStart.value + 7) % 7).toLong())
}

/** How many weeks the month needs, four to six. */
fun gridWeeks(month: YearMonth, weekStart: DayOfWeek): Int {
    val lead = (month.atDay(1).dayOfWeek.value - weekStart.value + 7) % 7
    return (lead + month.lengthOfMonth() + 6) / 7
}

/**
 * The month's cells with nothing on them, for the moment before the core's page arrives (or when it
 * cannot be read): the grid is never blank.
 */
fun emptyGrid(month: YearMonth, weekStart: DayOfWeek, today: LocalDate): List<CalendarDay> {
    val start = gridStart(month, weekStart)
    return List(gridWeeks(month, weekStart) * 7) { i ->
        val day = start.plusDays(i.toLong())
        CalendarDay(
            date = day.toString(),
            inMonth = YearMonth.from(day) == month,
            weekend = day.dayOfWeek == DayOfWeek.SATURDAY || day.dayOfWeek == DayOfWeek.SUNDAY,
            today = day == today,
        )
    }
}

/**
 * The cells to draw: the core's page when it is the one asked for (the month and the first weekday),
 * else the empty grid.
 */
fun calendarDays(page: CalendarPage?, month: YearMonth, weekStart: DayOfWeek, today: LocalDate): List<CalendarDay> =
    page?.takeIf {
        it.year == month.year && it.month == month.monthValue &&
            it.days.firstOrNull()?.date == gridStart(month, weekStart).toString()
    }?.days ?: emptyGrid(month, weekStart, today)

/**
 * Previous or Next moves the month by [by] and the selection along to the same day number of it
 * (the last day of a shorter month), as the desktop does.
 */
fun shiftMonth(month: YearMonth, selected: LocalDate, by: Int): Pair<YearMonth, LocalDate> {
    val next = month.plusMonths(by.toLong())
    return next to next.atDay(minOf(selected.dayOfMonth, next.lengthOfMonth()))
}

/** A horizontal drag of [dx] px pages the month once it passes [threshold]: left is the next month. */
fun swipeMonths(dx: Float, threshold: Float): Int = when {
    dx <= -threshold -> 1
    dx >= threshold -> -1
    else -> 0
}

/** The RFC 3339 time a task typed under [day] is due when its text gives none: 18:00 that day. */
fun calendarDue(day: LocalDate, zone: ZoneId): String = dueStamp(day, null, zone)

/**
 * The due date a draft typed in the calendar is stored with: the one its text gave, none when the
 * text gave a reminder time instead (that is its date), else the day's [fallback] (18:00).
 */
fun calendarDraftDue(draft: QuickDraft, fallback: String?): String? = when {
    draft.dueAt.isNotBlank() -> draft.dueAt
    draft.remindAt.isNotBlank() -> null
    else -> fallback
}

/** `2026年11月` or `November 2026`: the page's title. */
fun calendarTitle(month: YearMonth, locale: Locale): String =
    DateTimeFormatter.ofPattern(if (locale.language == "zh") "yyyy年M月" else "MMMM yyyy", locale).format(month)

/** `11月3日` or `Nov 3`: the day as the composer's placeholder names it. */
fun calendarShortDay(day: LocalDate, locale: Locale): String =
    DateTimeFormatter.ofPattern(if (locale.language == "zh") "M月d日" else "MMM d", locale).format(day)

/** `11月3日 周二` or `Tue, Nov 3`: the heading of the selected day. */
fun calendarDayHeading(day: LocalDate, locale: Locale): String = dueRowDate(day, locale)

/** `下午 5:00` or `5:00 PM`: a moment's clock time in [zone], as the Today rows write it. */
fun calendarClock(value: String, locale: Locale, zone: ZoneId): String {
    val at = runCatching { OffsetDateTime.parse(value).toInstant() }
        .getOrElse { runCatching { Instant.parse(value) }.getOrNull() } ?: return ""
    return DateTimeFormatter.ofPattern(if (locale.language == "zh") "a h:mm" else "h:mm a", locale)
        .format(at.atZone(zone))
}

/** The month as the one number the screen state keeps (it survives a rotation as an Int). */
fun YearMonth.key(): Int = year * 12 + monthValue - 1

/** The month a [key] names. */
fun monthOfKey(key: Int): YearMonth = YearMonth.of(key / 12, key % 12 + 1)

/** Whether the parser read a day or a time out of the text, so its own receipt says when. */
fun quickReadsDate(parse: QuickParse): Boolean =
    parse.drafts.any { it.dueAt.isNotBlank() || it.remindAt.isNotBlank() }
