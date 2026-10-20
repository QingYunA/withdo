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
import xyz.sriai.adhdtodo.data.PageJson
import xyz.sriai.adhdtodo.data.Reminder

/**
 * The reminder alarm and its notifications. One alarm is set at Core.NextReminderAt; when it fires
 * ReminderReceiver claims the due reminders, posts one notification each, then sets the next
 * alarm. The alarm is inexact on purpose: there is no exact-alarm permission.
 */
object Reminders {
    const val CHANNEL_ID = "reminders"
    const val ACTION_CLAIM = "xyz.sriai.adhdtodo.CLAIM_REMINDERS"
    const val EXTRA_TASK_ID = "task_id"

    fun ensureChannel(context: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = context.getSystemService(NotificationManager::class.java) ?: return
        if (manager.getNotificationChannel(CHANNEL_ID) != null) return
        val channel = NotificationChannel(
            CHANNEL_ID,
            context.getString(R.string.notification_channel_reminders),
            NotificationManager.IMPORTANCE_DEFAULT,
        )
        channel.description = context.getString(R.string.notification_channel_reminders_description)
        manager.createNotificationChannel(channel)
    }

    /** Sets the one alarm at the next reminder, or cancels it when nothing is due. */
    fun schedule(context: Context, core: Core_) {
        val manager = context.getSystemService(AlarmManager::class.java) ?: return
        val at = runCatching { core.nextReminderAt() }.getOrDefault(0)
        val pending = alarmIntent(context)
        if (at <= 0) {
            manager.cancel(pending)
            return
        }
        manager.setAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, pending)
    }

    /** Claims the due reminders and posts one notification each; returns the claimed count. */
    fun claimAndPost(context: Context, core: Core_): Int {
        val reminders = runCatching {
            PageJson.decodeFromString<List<Reminder>>(core.claimDueReminders())
        }.getOrDefault(emptyList())
        for (reminder in reminders) post(context, reminder)
        return reminders.size
    }

    fun post(context: Context, reminder: Reminder) {
        ensureChannel(context)
        val open = Intent(context, MainActivity::class.java).apply {
            putExtra(EXTRA_TASK_ID, reminder.taskId)
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
        }
        val content = PendingIntent.getActivity(
            context,
            reminder.taskId.toInt(),
            open,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = Notification.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(reminder.title)
            .setContentText(reminder.project?.takeIf { it.isNotBlank() } ?: context.getString(R.string.title_inbox))
            .setContentIntent(content)
            .setAutoCancel(true)
            .setCategory(Notification.CATEGORY_REMINDER)
            .build()
        val manager = context.getSystemService(NotificationManager::class.java) ?: return
        manager.notify(reminder.taskId.toInt(), notification)
    }

    private fun alarmIntent(context: Context): PendingIntent {
        val intent = Intent(context, ReminderReceiver::class.java).apply { action = ACTION_CLAIM }
        return PendingIntent.getBroadcast(
            context,
            0,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
    }
}
