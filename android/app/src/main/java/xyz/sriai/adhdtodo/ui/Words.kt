package xyz.sriai.adhdtodo.ui

import android.content.res.Resources
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.Task
import java.time.Instant
import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.time.temporal.ChronoUnit
import java.util.Locale

/**
 * The words of the screens: relative ages, due labels, dates and agent names, in the language the
 * resources are showing. Times are RFC 3339 strings from the Go core. The time source and the zone are the
 * device's; the screenshot tests pass fixed ones so the pictures do not change from day to day.
 */
class Words(
    private val res: Resources,
    private val timeSource: () -> Instant = Instant::now,
    private val zone: ZoneId = ZoneId.systemDefault(),
) {
    private val locale: Locale = runCatching { res.configuration.locales[0] }.getOrDefault(Locale.getDefault())

    private val monthDay = DateTimeFormatter.ofPattern(if (chinese) "M月d日" else "MMM d", locale)
    private val weekday = DateTimeFormatter.ofPattern("EEE", locale)
    private val headerFormatter = DateTimeFormatter.ofPattern(if (chinese) "M月d日EEEE" else "EEEE, MMM d", locale)
    private val clock = DateTimeFormatter.ofPattern(if (chinese) "ah:mm" else "h:mm a", locale)
    // v30 writes "上午 9:00" with a space, which the plain clock pattern does not add.
    private val remindClock = DateTimeFormatter.ofPattern(if (chinese) "a h:mm" else "h:mm a", locale)

    private val chinese: Boolean get() = locale.language == "zh"

    fun headerDate(now: ZonedDateTime = timeSource().atZone(zone)): String = headerFormatter.format(now)

    fun instant(value: String?): Instant? {
        if (value.isNullOrBlank()) return null
        return runCatching { OffsetDateTime.parse(value).toInstant() }
            .getOrElse { runCatching { Instant.parse(value) }.getOrNull() }
    }

    fun ago(value: String?, now: Instant = timeSource()): String {
        val then = instant(value) ?: return ""
        val minutes = ChronoUnit.MINUTES.between(then, now).coerceAtLeast(0)
        return when {
            minutes < 1 -> res.getString(R.string.age_now)
            minutes < 60 -> res.getString(R.string.age_minutes, minutes)
            minutes < 60 * 24 -> res.getString(R.string.age_hours, minutes / 60)
            minutes < 60 * 24 * 7 -> res.getString(R.string.age_days, minutes / (60 * 24))
            else -> monthDay.format(then.atZone(zone))
        }
    }

    /** A task's due label: Today, Tomorrow, Fri, Oct 20, or the time of day for a reminder today. */
    fun due(item: Item, now: ZonedDateTime = timeSource().atZone(zone)): String =
        dueLabel(item.remindAt, item.dueAt, now)

    fun due(task: Task, now: ZonedDateTime = timeSource().atZone(zone)): String =
        dueLabel(task.remindAt, task.dueAt, now)

    private fun dueLabel(remindAt: String?, dueAt: String?, now: ZonedDateTime): String {
        val today = now.toLocalDate()
        val remind = instant(remindAt)?.atZone(zone)
        if (remind != null && remind.toLocalDate() == today) return clock.format(remind)
        val due = instant(dueAt)?.atZone(zone) ?: return ""
        val day = due.toLocalDate()
        val diff = ChronoUnit.DAYS.between(today, day)
        return when {
            day == today -> res.getString(R.string.due_today)
            day == today.plusDays(1) -> res.getString(R.string.due_tomorrow)
            diff in 2..6 -> weekday.format(day)
            else -> monthDay.format(day)
        }
    }

    fun timeOfDay(value: String?): String {
        val then = instant(value)?.atZone(zone) ?: return ""
        return clock.format(then)
    }

    fun agent(raw: String): String {
        val n = raw.lowercase(Locale.ROOT)
        return when {
            n == "you" -> res.getString(R.string.done_by_you)
            n == "github" -> "GitHub"
            n.contains("claude") -> "Claude Code"
            n.contains("codex") -> "Codex"
            n.contains("cursor") -> "Cursor"
            n.contains("gemini") -> "Gemini CLI"
            n.contains("opencode") -> "opencode"
            n.contains("copilot") || n.contains("visual studio code") -> "GitHub Copilot"
            n.isBlank() -> res.getString(R.string.agent_unnamed)
            else -> raw
        }
    }

    /** The avatar file base for an agent name: one of the sprites, or the robot. */
    fun avatar(raw: String): String {
        val n = raw.lowercase(Locale.ROOT)
        return when {
            n.contains("claude") -> "claude"
            n.contains("codex") -> "codex"
            n.contains("cursor") -> "cursor"
            n.contains("gemini") -> "gemini"
            n.contains("opencode") -> "opencode"
            n.contains("copilot") || n.contains("visual studio code") -> "copilot"
            n.isBlank() -> "bot"
            else -> "bot"
        }
    }

    /** Who finished a Done today row: the agent's name, or You. */
    fun who(by: String): String = if (by.isBlank()) res.getString(R.string.done_by_you) else agent(by)

    fun subtasks(n: Int): String = res.getQuantityString(R.plurals.subtasks, n, n)

    fun statusLabel(status: String): String = when (status) {
        "done" -> res.getString(R.string.pill_done)
        "doing" -> res.getString(R.string.pill_doing)
        "blocked" -> res.getString(R.string.pill_blocked)
        "review" -> res.getString(R.string.pill_review)
        "todo" -> res.getString(R.string.pill_todo)
        "backlog" -> res.getString(R.string.pill_backlog)
        "cancelled" -> res.getString(R.string.pill_cancelled)
        else -> status
    }

    /**
     * The status's name in the task menu and the task page, copied from the desktop's statusBacklog
     * to statusCancelled rather than the row pill's wording (In review, not Handed back).
     */
    fun menuStatus(status: String): String = when (status) {
        "backlog" -> res.getString(R.string.menu_status_backlog)
        "todo" -> res.getString(R.string.menu_status_todo)
        "doing" -> res.getString(R.string.menu_status_doing)
        "blocked" -> res.getString(R.string.menu_status_blocked)
        "review" -> res.getString(R.string.menu_status_review)
        "done" -> res.getString(R.string.menu_status_done)
        "cancelled" -> res.getString(R.string.menu_status_cancelled)
        else -> status
    }

    /** P0 Urgent, P1 High, P2 Normal, P3 Low. */
    fun priorityName(priority: String): String = when (priority) {
        "P0" -> res.getString(R.string.menu_priority_urgent)
        "P1" -> res.getString(R.string.menu_priority_high)
        "P2" -> res.getString(R.string.menu_priority_normal)
        "P3" -> res.getString(R.string.menu_priority_low)
        else -> ""
    }

    /** A category's name; the two built-in ones are stored as Work and Life and worded here. */
    fun categoryName(category: Category): String = when (category.builtin) {
        "work" -> res.getString(R.string.category_work)
        "life" -> res.getString(R.string.category_life)
        else -> category.name
    }

    /**
     * The moment of a ready-made reminder. Later today and Tomorrow already say the day, so they
     * show the time alone; the others show the date and the time, as v30 does.
     */
    fun remindPresetHint(kind: String, atMillis: Long): String {
        val at = Instant.ofEpochMilli(atMillis).atZone(zone)
        val time = remindClock.format(at)
        if (kind == "later" || kind == "tomorrow") return time
        return if (chinese) "${monthDay.format(at)} $time" else "${monthDay.format(at)}, $time"
    }

    /** A set reminder as a row's value: the time today, Tomorrow at 9:00, or Oct 26 at 9:00. */
    fun reminderValue(value: String?): String {
        val at = instant(value)?.atZone(zone) ?: return ""
        val today = timeSource().atZone(zone).toLocalDate()
        val time = remindClock.format(at)
        return when (at.toLocalDate()) {
            today -> time
            today.plusDays(1) -> "${res.getString(R.string.due_tomorrow)} $time"
            else -> if (chinese) "${monthDay.format(at)} $time" else "${monthDay.format(at)}, $time"
        }
    }

    /** The heading of a project's status group, which is worded for the list, not the row pill. */
    fun groupLabel(status: String): String = when (status) {
        "blocked" -> res.getString(R.string.group_blocked)
        "review" -> res.getString(R.string.group_review)
        "doing" -> res.getString(R.string.group_doing)
        "todo" -> res.getString(R.string.group_todo)
        "backlog" -> res.getString(R.string.group_backlog)
        else -> statusLabel(status)
    }

    /** A sync age says "just now" where a task age says "now". */
    fun syncAge(value: String?): String {
        val then = instant(value) ?: return ""
        val minutes = ChronoUnit.MINUTES.between(then, timeSource()).coerceAtLeast(0)
        return if (minutes < 1) res.getString(R.string.time_just_now) else ago(value)
    }

    /** A device that never synced says Never. */
    fun deviceAge(value: String?): String =
        if (value.isNullOrBlank()) res.getString(R.string.sync_never) else syncAge(value)

    /** The time of a failed retry as a line: Retrying soon / in 3 min / in 2 h. */
    fun retryLine(value: String?, now: Instant = timeSource()): String {
        val at = instant(value) ?: return res.getString(R.string.sync_retrying_soon)
        val minutes = ChronoUnit.MINUTES.between(now, at)
        return when {
            minutes < 1 -> res.getString(R.string.sync_retrying_soon)
            minutes < 60 -> res.getString(R.string.sync_retrying_minutes, minutes)
            else -> res.getString(R.string.sync_retrying_hours, minutes / 60)
        }
    }
}
