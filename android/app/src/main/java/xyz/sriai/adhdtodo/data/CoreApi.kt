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

    /** The Calendar tab's month grid; [weekStart] is the first weekday, 0 Sunday to 6 Saturday. */
    fun calendar(year: Int, month: Int, weekStart: Int): CalendarPage =
        PageJson.decodeFromString(core.calendar(year.toLong(), month.toLong(), weekStart.toLong()))

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

    /**
     * The Projects tab's rows in one call: the projects in the sidebar's order (sub-projects follow
     * their parent, [ProjectEntry.depth] says how far in), each with its open-task count and the
     * red dot.
     */
    fun projectEntries(): List<ProjectEntry> =
        PageJson.decodeFromString<List<ProjectRow>>(core.projectList()).map {
            ProjectEntry(project = it.project, open = it.open, total = it.total, blocked = it.blocked, depth = it.depth)
        }

    /**
     * Makes a plain task and returns its id, which the receipt's Undo deletes. [remindAt] and [dueAt]
     * are RFC 3339 (what [parseQuick] read); [priority] "P1" is the composer's star.
     */
    fun createTask(
        title: String,
        projectId: Long? = null,
        remindAt: String? = null,
        dueAt: String? = null,
        categoryId: Long? = null,
        priority: String? = null,
        parentId: Long? = null,
    ): Long =
        PageJson.decodeFromString<Task>(
            core.createTask(
                PageJson.encodeToString(
                    NewTask(
                        title = title, projectId = projectId, remindAt = remindAt, dueAt = dueAt,
                        categoryId = categoryId, priority = priority, parentId = parentId,
                    ),
                ),
            ),
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

    // ---- Completed and Summary --------------------------------------------------------------

    /**
     * The desktop's Completed page: done and cancelled tasks, newest first, in day groups, each with
     * who finished it. [projectId] 0 is everything; [agents] keeps what agents wrote; [limit] 0 is 100.
     */
    fun completed(projectId: Long = 0, agents: Boolean = false, limit: Long = 0): CompletedPage =
        PageJson.decodeFromString(core.completed(projectId, agents, limit))

    /** The desktop's search: tasks matching every word of [query], newest first, any status; blank is empty. */
    fun search(query: String): List<Task> = PageJson.decodeFromString(core.search(query))

    /** The Summary page of a day or week; [back] counts periods back from this one (weeks start on Monday). */
    fun summary(week: Boolean, back: Long = 0): SummaryPage = PageJson.decodeFromString(core.summary(week, back))

    /** The Markdown the share button sends, in [lang] ("en" or "zh-Hans"). */
    fun summaryMarkdown(week: Boolean, back: Long, lang: String): String = core.summaryMarkdown(week, back, lang)

    /** Reads the composer's text without a model; cheap enough to call on every change. */
    fun parseQuick(text: String): QuickParse = PageJson.decodeFromString(core.parseQuick(text))

    // ---- Select: bulk edits and their Undo --------------------------------------------------

    /** Applies [patch] to every task as one change and returns the Undo token for [restoreTaskStates]. */
    fun updateTasks(ids: List<Long>, patch: BulkPatch): String =
        core.updateTasks(PageJson.encodeToString(ids), PageJson.encodeToString(patch))

    /** Undo of [updateTasks]. */
    fun restoreTaskStates(undo: String) = core.restoreTaskStates(undo)

    /** Deletes every task as one change and returns the snapshots [restoreTasks] takes, unchanged. */
    fun deleteTasks(ids: List<Long>): String = core.deleteTasks(PageJson.encodeToString(ids))

    /** Undo of [deleteTasks]. */
    fun restoreTasks(snapshots: String) = core.restoreTasks(snapshots)

    /** Do today for every task of mine that is not finished or due today; returns those it moved. */
    fun doTodayMany(ids: List<Long>): List<MovedToday> =
        PageJson.decodeFromString(core.doTodayMany(PageJson.encodeToString(ids)))

    /** Undo of [doTodayMany]: every moved task gets its old due date back. */
    fun restoreDues(moved: List<MovedToday>) = moved.forEach { restoreDue(it.id, it.before) }

    // ---- Projects and milestones ------------------------------------------------------------

    fun archivedProjects(): List<Project> = PageJson.decodeFromString(core.archivedProjects())

    /** Makes a project, a sub-project of [parentId] unless that is 0. */
    fun createProject(name: String, parentId: Long = 0): Project =
        PageJson.decodeFromString(core.createProject(name, parentId))

    fun renameProject(id: Long, name: String) = core.renameProject(id, name)

    fun archiveProject(id: Long) = core.archiveProject(id)

    fun unarchiveProject(id: Long) = core.unarchiveProject(id)

    /** Deletes a project with its tasks and milestones for good; the counts say what went. */
    fun deleteProject(id: Long): RemovedCounts = PageJson.decodeFromString(core.deleteProject(id))

    /** Adds an open milestone; [dueAt] is ISO 8601, "" for none. */
    fun createMilestone(projectId: Long, title: String, dueAt: String = ""): Milestone =
        PageJson.decodeFromString(core.createMilestone(projectId, title, dueAt))

    fun updateMilestone(id: Long, patch: MilestonePatch): Milestone {
        val fields = PageJson.encodeToJsonElement(patch).jsonObject
        val cleared = if (patch.clearDueAt) mapOf("due_at" to JsonNull) else emptyMap()
        return PageJson.decodeFromString(core.updateMilestone(id, JsonObject(fields + cleared).toString()))
    }

    // ---- Categories -------------------------------------------------------------------------

    fun categoriesWithCounts(): List<CategoryCount> = PageJson.decodeFromString(core.categoriesWithCounts())

    fun createCategory(input: CategoryInput): Category =
        PageJson.decodeFromString(core.createCategory(PageJson.encodeToString(input)))

    fun updateCategory(id: Long, input: CategoryInput): Category =
        PageJson.decodeFromString(core.updateCategory(id, PageJson.encodeToString(input)))

    /** Deletes a category (its tasks become unsorted) and returns the snapshot [restoreCategory] takes. */
    fun deleteCategory(id: Long): String = core.deleteCategory(id)

    fun restoreCategory(snapshot: String) = core.restoreCategory(snapshot)

    /** Puts a category at [index] (0-based) in the list. */
    fun moveCategory(id: Long, index: Long) = core.moveCategory(id, index)

    /** The icon library: the curated groups for a blank [query], else one unnamed group of matches. */
    fun categoryIcons(query: String = ""): IconLibrary = PageJson.decodeFromString(core.categoryIcons(query))

    fun accept(id: Long) = core.accept(id)

    fun complete(id: Long) = core.complete(id)

    fun reply(id: Long, text: String) = core.reply(id, text)

    /** Keeps a page on a task, as the share sheet does with the link it was given; [title] may be empty. */
    fun addLink(id: Long, url: String, title: String) = core.addLink(id, url, title)

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

/**
 * One row of the project list: the project, its count of open top-level tasks, how many tasks it
 * has in all (what Delete would remove), whether one is blocked (the red dot) and how far in it sits.
 */
data class ProjectEntry(
    val project: Project,
    val open: Int,
    val total: Int = 0,
    val blocked: Boolean = false,
    val depth: Int = 0,
)
