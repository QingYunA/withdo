package xyz.sriai.adhdtodo.platform

import android.app.AlarmManager
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import xyz.sriai.adhdtodo.MainActivity
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.core.Core_
import xyz.sriai.adhdtodo.data.CoreApi
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.LocalTime
import java.time.ZoneId

/**
 * The end-of-day reminder of the desktop's Notifications: Monday to Friday at a time the user picks
 * (18:30 unless changed), one notification; on Friday it speaks of the week. One inexact alarm is
 * set at the next such moment, fired through ReminderReceiver, which posts it and sets the next.
 * It follows the Reminders switch too: no notifications, no end-of-day reminder.
 */
object EndOfDay {
    const val ACTION = "xyz.sriai.adhdtodo.END_OF_DAY"
    const val CHANNEL_ID = "end_of_day"
    const val DEFAULT_MINUTES = 18 * 60 + 30

    private const val PREFS = "settings"
    private const val KEY_ON = "end_of_day_on"
    private const val KEY_AT = "end_of_day_at"
    private const val NOTIFICATION_ID = 7_000_001

    fun enabled(context: Context): Boolean =
        prefs(context).getBoolean(KEY_ON, true) && Reminders.enabled(context)

    fun switchOn(context: Context): Boolean = prefs(context).getBoolean(KEY_ON, true)

    /** Minutes after midnight. */
    fun minutes(context: Context): Int = prefs(context).getInt(KEY_AT, DEFAULT_MINUTES)

    fun setEnabled(context: Context, on: Boolean) {
        prefs(context).edit().putBoolean(KEY_ON, on).apply()
        schedule(context)
    }

    fun setMinutes(context: Context, minutes: Int) {
        prefs(context).edit().putInt(KEY_AT, minutes.coerceIn(0, 24 * 60 - 1)).apply()
        schedule(context)
    }

    /**
     * The next moment the reminder is due after [now]: today's time if it has not passed and today
     * is a weekday, else the next weekday's.
     */
    fun next(now: LocalDateTime, minutes: Int): LocalDateTime {
        val at = LocalTime.of(minutes / 60, minutes % 60)
        var day: LocalDate = now.toLocalDate()
        if (!now.isBefore(day.atTime(at))) day = day.plusDays(1)
        while (day.dayOfWeek == DayOfWeek.SATURDAY || day.dayOfWeek == DayOfWeek.SUNDAY) day = day.plusDays(1)
        return day.atTime(at)
    }

    /** Sets the one alarm, or cancels it when the reminder is off. */
    fun schedule(context: Context) {
        val manager = context.getSystemService(AlarmManager::class.java) ?: return
        val pending = alarmIntent(context)
        if (!enabled(context)) {
            manager.cancel(pending)
            return
        }
        val zone = ZoneId.systemDefault()
        val at = next(LocalDateTime.now(zone), minutes(context)).atZone(zone).toInstant().toEpochMilli()
        manager.setAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, pending)
    }

    /** Posts the day's (on Friday the week's) words, reading the counts from the core. */
    fun post(context: Context, core: Core_) {
        if (!enabled(context)) return
        val friday = LocalDate.now().dayOfWeek == DayOfWeek.FRIDAY
        val api = CoreApi(core)
        val day = runCatching { api.summary(week = false) }.getOrNull()
        val week = if (friday) runCatching { api.summary(week = true) }.getOrNull() else null
        val finished = day?.total ?: 0
        val merged = day?.merged ?: 0
        val weekFinished = week?.total ?: 0
        val title: Int
        val body: String
        when {
            friday && weekFinished > 0 -> {
                title = R.string.notify_week_done_title
                body = context.getString(R.string.notify_week_done_body, weekFinished)
            }
            !friday && finished > 0 && merged > 0 -> {
                title = R.string.notify_day_done_title
                body = context.getString(R.string.notify_day_done_body_prs, finished, merged)
            }
            !friday && finished > 0 -> {
                title = R.string.notify_day_done_title
                body = context.getString(R.string.notify_day_done_body, finished)
            }
            else -> {
                title = R.string.notify_day_done_title
                body = context.getString(R.string.notify_day_nothing_body)
            }
        }
        ensureChannel(context)
        val open = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
        }
        val content = PendingIntent.getActivity(
            context, 1, open, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = Notification.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(context.getString(title))
            .setContentText(body)
            .setContentIntent(content)
            .setAutoCancel(true)
            .build()
        context.getSystemService(NotificationManager::class.java)?.notify(NOTIFICATION_ID, notification)
    }

    private fun ensureChannel(context: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = context.getSystemService(NotificationManager::class.java) ?: return
        if (manager.getNotificationChannel(CHANNEL_ID) != null) return
        manager.createNotificationChannel(
            NotificationChannel(CHANNEL_ID, context.getString(R.string.notification_channel_end_of_day), NotificationManager.IMPORTANCE_DEFAULT),
        )
    }

    private fun alarmIntent(context: Context): PendingIntent {
        val intent = Intent(context, ReminderReceiver::class.java).apply { action = ACTION }
        return PendingIntent.getBroadcast(
            context, 1, intent, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
    }

    private fun prefs(context: Context) = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
}
