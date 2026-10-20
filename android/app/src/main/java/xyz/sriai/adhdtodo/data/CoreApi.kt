package xyz.sriai.adhdtodo.data

import kotlinx.serialization.encodeToString
import xyz.sriai.adhdtodo.core.Core_

/**
 * The page reads and writes the app uses, in one place. Every call blocks on the Go core and may
 * touch SQLite, so the screens call these from Dispatchers.IO, never on the UI thread.
 */
class CoreApi(private val core: Core_) {

    fun mine(): MinePage = PageJson.decodeFromString(core.mine())

    fun agents(): AgentsPage = PageJson.decodeFromString(core.agents())

    /**
     * data.Item and data.AgentRow do not name the agent behind a task; Task(id) has its reports.
     * Read the newest report's agent and links for the rows of Mine and Agents. A failure leaves
     * the row with the plain robot, never with a broken page.
     */
    fun info(ids: List<Long>): Map<Long, AgentInfo> {
        val out = HashMap<Long, AgentInfo>()
        for (id in ids) {
            val page = runCatching { PageJson.decodeFromString<TaskPage>(core.task(id)) }.getOrNull() ?: continue
            out[id] = AgentInfo(agent = page.latestAgent, links = page.latestLinks)
        }
        return out
    }

    fun agentInfo(mine: MinePage): Map<Long, AgentInfo> = info(mine.fromAgents.map { it.id })

    fun createTask(title: String) {
        core.createTask(PageJson.encodeToString(NewTask(title = title)))
    }

    fun accept(id: Long) = core.accept(id)

    fun complete(id: Long) = core.complete(id)

    fun reply(id: Long, text: String) = core.reply(id, text)
}
