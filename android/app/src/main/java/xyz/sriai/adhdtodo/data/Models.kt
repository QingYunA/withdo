package xyz.sriai.adhdtodo.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
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
    @SerialName("Later") val later: List<Item> = emptyList(),
    @SerialName("MoreLater") val moreLater: Int = 0,
    @SerialName("Done") val done: List<Item> = emptyList(),
    @SerialName("MoreDone") val moreDone: Int = 0,
) {
    /** data.Mine.Empty: what agents handed over, today and later are all empty. */
    val empty: Boolean get() = fromAgents.isEmpty() && today.isEmpty() && later.isEmpty()

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
