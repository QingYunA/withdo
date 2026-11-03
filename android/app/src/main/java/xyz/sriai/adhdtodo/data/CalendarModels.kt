package xyz.sriai.adhdtodo.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * One thing on a day of the Calendar tab (mobile/core's Calendar). [kind] is `milestone`,
 * `reminder`, `due` or `done`; a milestone carries [milestone] and no [task], anything else the
 * reverse. [at] is the moment that put it on the day, RFC 3339.
 */
@Serializable
data class CalendarItem(
    @SerialName("Kind") val kind: String = "due",
    @SerialName("At") val at: String = "",
    @SerialName("Overdue") val overdue: Boolean = false,
    @SerialName("Agent") val agent: String = "",
    @SerialName("Project") val project: String = "",
    @SerialName("Task") val task: Task? = null,
    @SerialName("Milestone") val milestone: Milestone? = null,
)

/** One cell of the month grid. [date] is `2026-11-03`. */
@Serializable
data class CalendarDay(
    @SerialName("Date") val date: String = "",
    @SerialName("InMonth") val inMonth: Boolean = false,
    @SerialName("Weekend") val weekend: Boolean = false,
    @SerialName("Today") val today: Boolean = false,
    @SerialName("Items") val items: List<CalendarItem> = emptyList(),
)

/** The month grid of Calendar(year, month, weekStart): the weeks the month needs, 7 days each. */
@Serializable
data class CalendarPage(
    @SerialName("Year") val year: Int = 0,
    @SerialName("Month") val month: Int = 0,
    @SerialName("Weeks") val weeks: Int = 0,
    @SerialName("Days") val days: List<CalendarDay> = emptyList(),
)
