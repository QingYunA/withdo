package xyz.sriai.adhdtodo.data

import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import xyz.sriai.adhdtodo.core.Core_

/**
 * The page reads and writes the app uses, in one place. Every call blocks on the Go core and may
 * touch SQLite or the network, so the screens call these from Dispatchers.IO, never on the UI
 * thread.
 */
class CoreApi(private val core: Core_) {

    fun mine(): MinePage = PageJson.decodeFromString(core.mine())

    fun agents(): AgentsPage = PageJson.decodeFromString(core.agents())

    fun projects(): List<Project> = PageJson.decodeFromString(core.projects())

    fun categories(): List<Category> = PageJson.decodeFromString(core.categories())

    /** The task menu's ready-made reminders for now, in the order the menu shows them. */
    fun remindPresets(): List<RemindPreset> = PageJson.decodeFromString(core.remindPresets())

    fun project(id: Long): ProjectPage = PageJson.decodeFromString(core.project(id))

    fun task(id: Long): TaskPage = PageJson.decodeFromString(core.task(id))

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

    /** The project list with its open-task count, read one project page at a time. */
    fun projectEntries(): List<ProjectEntry> = projects().map { project ->
        val open = runCatching {
            project(project.id).tasks.count { task ->
                task.parentId == null && task.status != "done" && task.status != "cancelled"
            }
        }.getOrDefault(0)
        ProjectEntry(project = project, open = open)
    }

    fun createTask(title: String, projectId: Long? = null) {
        core.createTask(PageJson.encodeToString(NewTask(title = title, projectId = projectId)))
    }

    fun updateTask(id: Long, patch: TaskPatch) {
        val fields = PageJson.encodeToJsonElement(patch).jsonObject
        val body = if (patch.clearRemindAt) JsonObject(fields + ("remind_at" to JsonNull)) else fields
        core.updateTask(id, body.toString())
    }

    /** Deletes a task and returns the snapshot RestoreTask takes, unchanged. */
    fun deleteTask(id: Long): String = core.deleteTask(id)

    /** Puts back what DeleteTask removed. */
    fun restoreTask(snapshot: String) = core.restoreTask(snapshot)

    fun accept(id: Long) = core.accept(id)

    fun complete(id: Long) = core.complete(id)

    fun reply(id: Long, text: String) = core.reply(id, text)

    fun account(): Account = PageJson.decodeFromString(core.account())

    fun devices(): List<Device> = PageJson.decodeFromString(core.devices())

    fun syncStatus(): SyncStatus = PageJson.decodeFromString(core.syncStatus())

    fun signIn(login: String, password: String, deviceName: String): SignInState =
        PageJson.decodeFromString(core.signIn(login, password, deviceName))

    /** Mails a 6-digit sign-in code to the address; an address with no account gets one. */
    fun requestCode(email: String) = core.requestCode(email)

    /** Checks the code and links this phone as signIn does. */
    fun signInWithCode(code: String, deviceName: String): SignInState =
        PageJson.decodeFromString(core.signInWithCode(code, deviceName))

    fun resendCode() = core.resendCode()

    fun requestPasswordReset(email: String) = core.requestPasswordReset(email)

    fun codeState(): CodeState = PageJson.decodeFromString(core.codeState())

    fun linkMerge(deviceName: String) = core.linkMerge(deviceName)

    fun linkReplace(deviceName: String): String = core.linkReplace(deviceName)

    fun signOut() = core.signOut()

    fun syncNow() = core.syncNow()

    fun startSync() = core.startSync()

    fun stopSync() = core.stopSync()

    fun syncSetServer(server: String) = core.syncSetServer(server)

    fun claimDueReminders(): List<Reminder> = PageJson.decodeFromString(core.claimDueReminders())

    fun nextReminderAt(): Long = core.nextReminderAt()
}

/** One row of the project list: the project and its count of open, top-level tasks. */
data class ProjectEntry(val project: Project, val open: Int)
