package xyz.sriai.adhdtodo.widget

import android.content.Context
import androidx.glance.appwidget.updateAll
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * Keeps the home-screen widgets in step with the data. The core calls onChange on every write
 * (a sync can send dozens in a burst), so [refresh] waits for a quiet second before it redraws.
 */
object Widgets {
    private const val QUIET_MS = 1000L

    // The process's own scope: a redraw is short, idempotent and not tied to any screen, so it
    // must outlive the Activity whose onChange asked for it (lifecycleScope would cancel it).
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private var pending: Job? = null

    /** Redraws the widgets once changes have stopped for a second. Safe from any thread. */
    @Synchronized
    fun refresh(context: Context) {
        val app = context.applicationContext
        pending?.cancel()
        pending = scope.launch {
            delay(QUIET_MS)
            refreshNow(app)
        }
    }

    /** Redraws at once, one widget after the other: each opens its own short-lived core. */
    suspend fun refreshNow(context: Context) {
        val app = context.applicationContext
        runCatching { TodayWidget().updateAll(app) }
        runCatching { AgentsWidget().updateAll(app) }
        runCatching { NoteWidget().updateAll(app) }
    }
}
