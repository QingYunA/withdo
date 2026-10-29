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

    /** Makes a plain task and returns its id, which the receipt's Undo deletes. */
    fun createTask(title: String, projectId: Long? = null): Long =
        PageJson.decodeFromString<Task>(
            core.createTask(PageJson.encodeToString(NewTask(title = title, projectId = projectId))),
        ).id

    /**
     * The task menu's Do today: the core moves the due date to today and returns the old one
     * (RFC 3339, empty when there was none), which Undo hands back to updateTask.
     */
    fun doToday(id: Long): String = core.doToday(id)

    /** Undo of doToday: the old due date back, or none. */
    fun restoreDue(id: Long, before: String) {
        updateTask(id, if (before.isEmpty()) TaskPatch(clearDueAt = true) else TaskPatch(dueAt = before))
    }

    fun updateTask(id: Long, patch: TaskPatch) {
        val fields = PageJson.encodeToJsonElement(patch).jsonObject
        val cleared = buildMap {
            if (patch.clearRemindAt) put("remind_at", JsonNull)
            if (patch.clearDueAt) put("due_at", JsonNull)
        }
        core.updateTask(id, JsonObject(fields + cleared).toString())
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

    /**
     * Makes an account with an email and a password (at least 8 characters) and mails its first
     * code; codeState then says step "verify". [lang] is the app's language, "en" or "zh-Hans".
     */
    fun signUp(email: String, password: String, lang: String) = core.signUp(email, password, lang)

    /** Mails a 6-digit code for Forgot password; codeState then says step "reset_code". */
    fun requestResetCode(email: String) = core.requestResetCode(email)

    /** Sets a new password with the reset code and links this phone as signIn does. */
    fun resetPassword(code: String, password: String, deviceName: String): SignInState =
        PageJson.decodeFromString(core.resetPassword(code, password, deviceName))

    /** Checks the code of the verify step, which verifies the email, and links this phone as signIn does. */
    fun signInWithCode(code: String, deviceName: String): SignInState =
        PageJson.decodeFromString(core.signInWithCode(code, deviceName))

    fun resendCode() = core.resendCode()

    /** Whether the server lists a Google provider. Reads the network. */
    fun googleAvailable(): Boolean = core.googleAvailable()

    /** Starts "Continue with Google" and returns the page to open in the browser. */
    fun googleStart(): String = core.googleStart()

    /** Waits for the browser's answer to googleStart (up to 5 minutes), then links as signIn does. */
    fun googleFinish(deviceName: String): SignInState =
        PageJson.decodeFromString(core.googleFinish(deviceName))

    /** Ends the wait of googleFinish. */
    fun googleCancel() = core.googleCancel()

    fun codeState(): CodeState = PageJson.decodeFromString(core.codeState())

    /** The other_account sheet's "Move this phone": links this phone to the signed-in account. */
    fun moveToAccount(deviceName: String): SignInState =
        PageJson.decodeFromString(core.moveToAccount(deviceName))

    /** Whether the Google row shows: signed in and the server offers Google. Reads the network. */
    fun googleLinkAvailable(): Boolean = core.googleLinkAvailable()

    /** Whether the signed-in account has a Google identity. The error's text is a link kind. */
    fun googleLinked(): Boolean = core.googleLinked()

    /** Starts "Link Google" and returns the page for the Custom Tab. */
    fun googleLinkStart(): String = core.googleLinkStart()

    /** Waits (up to 5 minutes) for the browser's answer; the error's text is a link kind. */
    fun googleLinkFinish() = core.googleLinkFinish()

    fun googleLinkCancel() = core.googleLinkCancel()

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
