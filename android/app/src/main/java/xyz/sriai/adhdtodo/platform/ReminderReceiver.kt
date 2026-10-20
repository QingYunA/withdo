package xyz.sriai.adhdtodo.platform

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import xyz.sriai.adhdtodo.core.Core

/**
 * The alarm and boot receiver: it claims the reminders that came due, posts them, and sets the
 * next alarm. A receiver may not block the main thread, so the work runs on its own thread with
 * goAsync; a failure leaves the next alarm to the next change or boot.
 */
class ReminderReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val app = context.applicationContext
        val pending = goAsync()
        Thread {
            try {
                Reminders.ensureChannel(app)
                val core = Core.open(app.filesDir.absolutePath, KeystoreSecrets(app))
                try {
                    val claim = intent.action == Reminders.ACTION_CLAIM ||
                        intent.action == Intent.ACTION_BOOT_COMPLETED ||
                        intent.action == Intent.ACTION_MY_PACKAGE_REPLACED
                    if (claim) Reminders.claimAndPost(app, core)
                    Reminders.schedule(app, core)
                } finally {
                    runCatching { core.close() }
                }
            } catch (_: Throwable) {
                // A reminder that cannot be read waits for the next alarm, change or boot.
            } finally {
                pending.finish()
            }
        }.start()
    }
}
