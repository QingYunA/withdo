package xyz.sriai.adhdtodo.widget

import android.content.Context
import android.content.res.Resources
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import xyz.sriai.adhdtodo.core.Core
import xyz.sriai.adhdtodo.data.CoreApi
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.platform.AppLanguage
import xyz.sriai.adhdtodo.platform.KeystoreSecrets
import xyz.sriai.adhdtodo.ui.Words

/**
 * Where the widgets get their data: a short-lived core opened for the read and closed again, as
 * ReminderReceiver does, so a widget never holds the database. A core that cannot be read gives
 * the empty widget without its quiet line (not "nothing for today", which would be untrue).
 */
internal object WidgetSource {
    /** The app's own language (Settings > General), which the widgets' process does not apply itself. */
    fun resources(context: Context): Resources = AppLanguage.wrap(context.applicationContext).resources

    suspend fun today(context: Context): TodayData {
        val res = resources(context)
        val page = withApi(context) { it.mine() }
        return todayData(page, Words(res), res)
    }

    suspend fun agents(context: Context): AgentsData {
        val res = resources(context)
        var agent = ""
        val page = withApi(context) { api ->
            api.mine().also { mine ->
                mine.fromAgents.firstOrNull()?.let { agent = api.info(listOf(it.id))[it.id]?.agent.orEmpty() }
            }
        }
        return agentsData(page, agent, Words(res), res)
    }

    /** Finishes a task from the Today widget; a failure leaves it on the list. */
    suspend fun complete(context: Context, id: Long) {
        withApi(context) { it.complete(id) }
    }

    private suspend fun <T> withApi(context: Context, block: (CoreApi) -> T): T? = withContext(Dispatchers.IO) {
        runCatching {
            val app = context.applicationContext
            val core = Core.open(app.filesDir.absolutePath, KeystoreSecrets(app))
            try {
                block(CoreApi(core))
            } finally {
                runCatching { core.close() }
            }
        }.getOrNull()
    }
}
