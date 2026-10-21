package xyz.sriai.adhdtodo.ui

import android.content.res.Resources
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.Task
import java.time.Instant
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
