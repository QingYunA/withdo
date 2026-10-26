package xyz.sriai.adhdtodo.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.Transient
import kotlinx.serialization.json.Json

/**
 * The page JSON of mobile/core, decoded with one set of classes. Every field keeps the Go name, so
 * the golden files under mobile/core/testdata are the contract. Unknown keys are ignored, so the
 * core can add a field without breaking an installed app.
 */
val PageJson: Json = Json {
    ignoreUnknownKeys = true
    isLenient = true
    explicitNulls = false
    coerceInputValues = true
}

@Serializable
data class Project(
    @SerialName("ID") val id: Long = 0,
    @SerialName("Name") val name: String = "",
    @SerialName("RepoSlug") val repoSlug: String = "",
    @SerialName("RepoPath") val repoPath: String = "",
    @SerialName("CreatedAt") val createdAt: String = "",
    @SerialName("UpdatedAt") val updatedAt: String = "",
    @SerialName("ArchivedAt") val archivedAt: String? = null,
    @SerialName("ParentID") val parentId: Long? = null,
)

/**
 * One category of the sidebar and the task menu: data.Category. Builtin is "work", "life" or ""
 * for the user's own. Icon is the 12 x 12 picture, one byte a pixel: 0 empty, 35 (the Go '#') in
 * the category's own color, and 97 to 108 for the fixed palette letters a to l.
 */
@Serializable
data class Category(
    @SerialName("ID") val id: Long = 0,
    @SerialName("Builtin") val builtin: String = "",
    @SerialName("Name") val name: String = "",
    @SerialName("Icon") val icon: List<List<Int>> = emptyList(),
    @SerialName("Color") val color: String = "",
    @SerialName("Position") val position: Int = 0,
)

/** One ready-made reminder of the task menu: kind is later, tomorrow, monday or weekend. */
@Serializable
data class RemindPreset(
    @SerialName("kind") val kind: String = "",
    @SerialName("at") val at: Long = 0,
)

@Serializable
data class Milestone(
    @SerialName("ID") val id: Long = 0,
    @SerialName("ProjectID") val projectId: Long = 0,
    @SerialName("Title") val title: String = "",
    @SerialName("DueAt") val dueAt: String? = null,
    @SerialName("Open") val open: Boolean = false,
    @SerialName("CreatedAt") val createdAt: String = "",
)

@Serializable
data class Link(
    @SerialName("URL") val url: String = "",
    @SerialName("Title") val title: String = "",
)

@Serializable
data class Report(
    @SerialName("ID") val id: Long = 0,
    @SerialName("TaskID") val taskId: Long = 0,
    @SerialName("Agent") val agent: String = "",
    @SerialName("Status") val status: String = "",
    @SerialName("Summary") val summary: String = "",
    @SerialName("NextStep") val nextStep: String = "",
    @SerialName("CreatedAt") val createdAt: String = "",
    @SerialName("Cwd") val cwd: String = "",
    @SerialName("Links") val links: List<Link> = emptyList(),
)

/** One task, as the desktop's data.Task. Times are RFC 3339 strings; nil is null. */
@Serializable
data class Task(
    @SerialName("ID") val id: Long = 0,
    @SerialName("ParentID") val parentId: Long? = null,
    @SerialName("ProjectID") val projectId: Long? = null,
    @SerialName("MilestoneID") val milestoneId: Long? = null,
    @SerialName("Title") val title: String = "",
    @SerialName("Notes") val notes: String = "",
    @SerialName("Status") val status: String = "",
    @SerialName("Priority") val priority: String = "",
    @SerialName("CategoryID") val categoryId: Int = 0,
    @SerialName("DueAt") val dueAt: String? = null,
    @SerialName("RemindAt") val remindAt: String? = null,
    @SerialName("RemindedAt") val remindedAt: String? = null,
    @SerialName("Branch") val branch: String = "",
    @SerialName("PRURL") val prUrl: String = "",
    @SerialName("PRNumber") val prNumber: Int = 0,
    @SerialName("Source") val source: String = "",
    @SerialName("CreatedAt") val createdAt: String = "",
    @SerialName("UpdatedAt") val updatedAt: String = "",
    @SerialName("CompletedAt") val completedAt: String? = null,
    @SerialName("PRState") val prState: String = "",
    @SerialName("PRChecks") val prChecks: String = "",
    @SerialName("PRReview") val prReview: String = "",
    @SerialName("PRCheckedAt") val prCheckedAt: String? = null,
    @SerialName("PRClosedSeenAt") val prClosedSeenAt: String? = null,
    @SerialName("HandedTo") val handedTo: String = "",
    @SerialName("HandedAt") val handedAt: String? = null,
)

/**
 * One row of Mine or Agents: data.Item, the task's fields plus why it is here. Only the fields the
 * phone draws are repeated here.
 */
@Serializable
data class Item(
    @SerialName("ID") val id: Long = 0,
    @SerialName("ParentID") val parentId: Long? = null,
    @SerialName("ProjectID") val projectId: Long? = null,
    @SerialName("MilestoneID") val milestoneId: Long? = null,
    @SerialName("Title") val title: String = "",
    @SerialName("Status") val status: String = "",
    @SerialName("Priority") val priority: String = "",
    @SerialName("CategoryID") val categoryId: Int = 0,
    @SerialName("DueAt") val dueAt: String? = null,
    @SerialName("RemindAt") val remindAt: String? = null,
    @SerialName("Branch") val branch: String = "",
    @SerialName("PRURL") val prUrl: String = "",
    @SerialName("PRNumber") val prNumber: Int = 0,
    @SerialName("Source") val source: String = "",
    @SerialName("CreatedAt") val createdAt: String = "",
    @SerialName("UpdatedAt") val updatedAt: String = "",
    @SerialName("CompletedAt") val completedAt: String? = null,
    @SerialName("Project") val project: String = "",
    @SerialName("Kind") val kind: Int = 0,
    @SerialName("Summary") val summary: String = "",
    @SerialName("NextStep") val nextStep: String = "",
    @SerialName("Words") val words: String = "",
    @SerialName("Since") val since: String = "",
    @SerialName("Stale") val stale: Boolean = false,
    @SerialName("Hours") val hours: Int = 0,
    @SerialName("Live") val live: Boolean = false,
    @SerialName("By") val by: String = "",
)

@Serializable
data class AgentRow(
    @SerialName("ID") val id: Long = 0,
    @SerialName("ParentID") val parentId: Long? = null,
    @SerialName("ProjectID") val projectId: Long? = null,
    @SerialName("MilestoneID") val milestoneId: Long? = null,
    @SerialName("Title") val title: String = "",
    @SerialName("Status") val status: String = "",
    @SerialName("Priority") val priority: String = "",
    @SerialName("DueAt") val dueAt: String? = null,
    @SerialName("RemindAt") val remindAt: String? = null,
    @SerialName("Branch") val branch: String = "",
    @SerialName("PRURL") val prUrl: String = "",
    @SerialName("PRNumber") val prNumber: Int = 0,
    @SerialName("Source") val source: String = "",
    @SerialName("CreatedAt") val createdAt: String = "",
    @SerialName("UpdatedAt") val updatedAt: String = "",
    @SerialName("CompletedAt") val completedAt: String? = null,
    @SerialName("Project") val project: String = "",
    @SerialName("Kind") val kind: Int = 0,
    @SerialName("Summary") val summary: String = "",
    @SerialName("NextStep") val nextStep: String = "",
    @SerialName("Words") val words: String = "",
    @SerialName("Since") val since: String = "",
    @SerialName("Stale") val stale: Boolean = false,
    @SerialName("Hours") val hours: Int = 0,
    @SerialName("Live") val live: Boolean = false,
    @SerialName("By") val by: String = "",
    @SerialName("Subs") val subs: List<Item> = emptyList(),
)

@Serializable
data class MinePage(
    @SerialName("FromAgents") val fromAgents: List<Item> = emptyList(),
    @SerialName("Today") val today: List<Item> = emptyList(),
    /** My tasks dated within the next week, by date. */
    @SerialName("Next") val next: List<Item> = emptyList(),
    /** My undated tasks that have no project: the fold's rows. */
    @SerialName("NoDate") val noDate: List<Item> = emptyList(),
    @SerialName("NoDateTotal") val noDateTotal: Int = 0,
    @SerialName("Done") val done: List<Item> = emptyList(),
    @SerialName("MoreDone") val moreDone: Int = 0,
) {
    /** data.Mine.Empty: what agents handed over, today, next and the undated are all empty. */
    val empty: Boolean
        get() = fromAgents.isEmpty() && today.isEmpty() && next.isEmpty() && noDate.isEmpty() && noDateTotal == 0

    /** data.Mine.Count: the plain count beside Mine once nothing is unseen (the badge is fromAgents). */
    val count: Int get() = fromAgents.size + today.size
}

@Serializable
data class AgentsPage(
    @SerialName("Working") val working: List<AgentRow> = emptyList(),
    @SerialName("Done") val done: List<AgentRow> = emptyList(),
    @SerialName("MoreDone") val moreDone: Int = 0,
) {
    val empty: Boolean get() = working.isEmpty() && done.isEmpty()
}

@Serializable
data class ProjectPage(
    @SerialName("Project") val project: Project = Project(),
    @SerialName("Tasks") val tasks: List<Task> = emptyList(),
    @SerialName("Milestones") val milestones: List<Milestone> = emptyList(),
    /** Task id → the agent of its latest report: the row's avatar. */
    @SerialName("Agents") val agents: Map<String, String> = emptyMap(),
)

@Serializable
data class TaskPage(
    @SerialName("Task") val task: Task = Task(),
    @SerialName("Subtasks") val subtasks: List<Task> = emptyList(),
    @SerialName("Reports") val reports: List<Report> = emptyList(),
) {
    /**
     * The task's latest report in the desktop's sense: the newest one that is neither GitHub's nor
     * a reply the user wrote. A reply must not replace what the agent said.
     */
    private val latestReport: Report?
        get() = reports.firstOrNull { it.agent != "github" && it.agent != "you" }

    /** The agent of the latest report, or "" when the task has none. */
    val latestAgent: String get() = latestReport?.agent.orEmpty()

    /** The links the latest report carries, for Open page. */
    val latestLinks: List<Link> get() = latestReport?.links.orEmpty()
}

/** The newest report's agent and links of a From your agents row, loaded with Task(id). */
data class AgentInfo(val agent: String, val links: List<Link>)

/** The shape CreateTask takes; every field but title is optional. */
@Serializable
data class NewTask(
    @SerialName("title") val title: String,
    @SerialName("project_id") val projectId: Long? = null,
)

/** The shape UpdateTask takes. An absent key is left alone. */
@Serializable
data class TaskPatch(
    @SerialName("title") val title: String? = null,
    @SerialName("notes") val notes: String? = null,
    @SerialName("status") val status: String? = null,
    @SerialName("priority") val priority: String? = null,
    @SerialName("due_at") val dueAt: String? = null,
    @SerialName("remind_at") val remindAt: String? = null,
    @SerialName("category") val category: Long? = null,
    @SerialName("project_id") val projectId: Long? = null,
    /** True writes `"due_at": null`, which the core reads as clear: the Undo of a first due date. */
    @Transient val clearDueAt: Boolean = false,
    /**
     * True writes `"remind_at": null`, which the Go core reads as clear. The plain null above is
     * absent from the JSON (explicitNulls = false) and leaves the reminder alone.
     */
    @Transient val clearRemindAt: Boolean = false,
)

/** Account()'s JSON: who this phone signed in as. Both are "" when signed out. */
@Serializable
data class Account(
    @SerialName("name") val name: String = "",
    @SerialName("email") val email: String = "",
) {
    val display: String get() = name.ifBlank { email }
}

/** One device of the account, as Devices() lists it. */
@Serializable
data class Device(
    @SerialName("id") val id: String = "",
    @SerialName("name") val name: String = "",
    @SerialName("last_seen_at") val lastSeenAt: String? = null,
    @SerialName("this") val isThis: Boolean = false,
)

/** One due reminder, as ClaimDueReminders returns it. Project is null for an Inbox task. */
@Serializable
data class Reminder(
    @SerialName("task_id") val taskId: Long = 0,
    @SerialName("title") val title: String = "",
    @SerialName("project") val project: String? = null,
)

/**
 * SignIn()'s answer: "linked" or "choose". The choose answer carries the counts the
 * both-have-tasks sheet shows: this phone's tasks and projects and the account's synced devices.
 */
@Serializable
data class SignInState(
    @SerialName("state") val state: String = "",
    @SerialName("local_tasks") val localTasks: Int = 0,
    @SerialName("local_projects") val localProjects: Int = 0,
    @SerialName("account_devices") val accountDevices: Int = 0,
    /** The other_account answer: the account this phone was synced with, "" when it is not remembered. */
    @SerialName("previous_email") val previousEmail: String = "",
    /** The other_account answer: the account that just signed in. */
    @SerialName("account_email") val accountEmail: String = "",
)

/**
 * CodeState()'s JSON: where the emailed-code sign-in is. step is "code" once a code was mailed;
 * resend_in is the seconds before another may be asked for; failure is the kind of the last
 * failed attempt (wrong_code, expired, too_many, wrong_password, deactivated, network,
 * unsupported, other) or "".
 */
@Serializable
data class CodeState(
    @SerialName("step") val step: String = "",
    @SerialName("email") val email: String = "",
    @SerialName("resend_in") val resendIn: Int = 0,
    @SerialName("failure") val failure: String = "",
)

/**
 * SyncStatus()'s JSON: the runner's state plus the settings the Sync screen reads. A missing
 * retry_at, failed_since or seed_total means the runner is not in that state.
 */
@Serializable
data class SyncStatus(
    @SerialName("state") val state: String = "idle",
    @SerialName("last_synced") val lastSynced: String? = null,
    @SerialName("last_error") val lastError: String = "",
    @SerialName("pending") val pending: Long = 0,
    @SerialName("retry_at") val retryAt: String? = null,
    @SerialName("failed_since") val failedSince: String? = null,
    @SerialName("seed_total") val seedTotal: Long = 0,
    @SerialName("server") val server: String = "",
    @SerialName("linked") val linked: Boolean = false,
    @SerialName("signed_in") val signedIn: Boolean = false,
    @SerialName("sign_in_stage") val signInStage: String? = null,
) {
    val seedDone: Long get() = (seedTotal - pending).coerceAtLeast(0)
}
