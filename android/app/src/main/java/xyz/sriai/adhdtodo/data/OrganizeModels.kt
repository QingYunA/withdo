package xyz.sriai.adhdtodo.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.Transient

/*
 * The pages and inputs of the organizing calls (v43): Completed, Summary, the composer's parser,
 * bulk edits, the Projects tab and Settings > Categories. The Go side documents every shape on its
 * method in mobile/core; the golden files under mobile/core/testdata pin them.
 */

/** One finished task of Completed: data.Task's fields the phone draws, plus who finished it. */
@Serializable
data class CompletedTask(
    @SerialName("ID") val id: Long = 0,
    @SerialName("ParentID") val parentId: Long? = null,
    @SerialName("ProjectID") val projectId: Long? = null,
    @SerialName("Title") val title: String = "",
    @SerialName("Status") val status: String = "",
    @SerialName("Priority") val priority: String = "",
    @SerialName("CategoryID") val categoryId: Int = 0,
    @SerialName("Source") val source: String = "",
    @SerialName("PRNumber") val prNumber: Int = 0,
    @SerialName("CreatedAt") val createdAt: String = "",
    @SerialName("CompletedAt") val completedAt: String? = null,
    /** The project's name, "" for the Inbox. */
    @SerialName("Project") val project: String = "",
    /** The agent that finished it as it reported, "" when it was you (the avatar key). */
    @SerialName("By") val by: String = "",
    /** "you" or the agent's name as people write it. */
    @SerialName("ByName") val byName: String = "",
    /** What the agent's closing report said, "" when it was you. */
    @SerialName("Summary") val summary: String = "",
)

/** The tasks finished on one local day; [day] is `2026-10-20`. */
@Serializable
data class CompletedDay(
    @SerialName("Day") val day: String = "",
    @SerialName("Tasks") val tasks: List<CompletedTask> = emptyList(),
)

/** Completed()'s page: day groups, newest first; [more] says older ones exist (ask with a larger limit). */
@Serializable
data class CompletedPage(
    @SerialName("Days") val days: List<CompletedDay> = emptyList(),
    @SerialName("More") val more: Boolean = false,
)

/** One finished task of a ledger row. [at] is RFC 3339. */
@Serializable
data class SummaryTask(
    @SerialName("id") val id: Long = 0,
    @SerialName("title") val title: String = "",
    @SerialName("project") val project: String = "",
    @SerialName("at") val at: String = "",
    @SerialName("kind") val kind: String = "",
    @SerialName("merged") val merged: Boolean = false,
    @SerialName("pr_number") val prNumber: Int = 0,
    @SerialName("said") val said: String = "",
)

/** One row of the ledger: you ([you], [agent] empty) or an agent. */
@Serializable
data class SummaryPerson(
    @SerialName("agent") val agent: String = "",
    @SerialName("name") val name: String = "",
    @SerialName("you") val you: Boolean = false,
    @SerialName("count") val count: Int = 0,
    @SerialName("merged") val merged: Int = 0,
    @SerialName("tasks") val tasks: List<SummaryTask> = emptyList(),
)

/** One good the scene throws. [actor] indexes the ledger; [day] is 0 to 6 in a week, 0 in a day. */
@Serializable
data class SummaryGood(
    @SerialName("actor") val actor: Int = 0,
    @SerialName("kind") val kind: String = "",
    @SerialName("starred") val starred: Boolean = false,
    @SerialName("day") val day: Int = 0,
    @SerialName("task_id") val taskId: Long = 0,
)

/** What is still open at the end of the Markdown and the ledger. */
@Serializable
data class SummaryOpen(
    @SerialName("in_progress") val inProgress: Int = 0,
    @SerialName("in_review") val inReview: Int = 0,
    @SerialName("blocked") val blocked: Int = 0,
)

/**
 * Summary()'s page. [from] and [to] are the first and last day (`2026-10-19`); [paragraph] is the
 * assistant's text, "" until a summary row syncs; the ledger is you first, then the agents.
 */
@Serializable
data class SummaryPage(
    @SerialName("week") val week: Boolean = false,
    @SerialName("back") val back: Long = 0,
    @SerialName("from") val from: String = "",
    @SerialName("to") val to: String = "",
    @SerialName("over") val over: Boolean = false,
    @SerialName("total") val total: Int = 0,
    @SerialName("merged") val merged: Int = 0,
    @SerialName("ledger") val ledger: List<SummaryPerson> = emptyList(),
    @SerialName("open") val open: SummaryOpen = SummaryOpen(),
    @SerialName("paragraph") val paragraph: String = "",
    @SerialName("goods") val goods: List<SummaryGood> = emptyList(),
)

/**
 * One task read out of the composer's text. [ranges] are `[start, end)` offsets in UTF-16 units,
 * which are Kotlin String indexes; an empty list means nothing was read. [remindAt] and [dueAt] are
 * RFC 3339 or "".
 */
@Serializable
data class QuickDraft(
    @SerialName("title") val title: String = "",
    @SerialName("remind_at") val remindAt: String = "",
    @SerialName("due_at") val dueAt: String = "",
    @SerialName("category_id") val categoryId: Long = 0,
    @SerialName("category") val category: String = "",
    @SerialName("important") val important: Boolean = false,
    @SerialName("ranges") val ranges: List<List<Int>> = emptyList(),
) {
    /** True when the parser read something: a time, a category, a star. */
    val read: Boolean get() = ranges.isNotEmpty()
}

/** ParseQuick()'s answer: one draft per task of the text, none for blank text. */
@Serializable
data class QuickParse(
    @SerialName("drafts") val drafts: List<QuickDraft> = emptyList(),
) {
    val first: QuickDraft? get() = drafts.firstOrNull()
}

/** The Projects tab's row as the core sends it (ProjectList). */
@Serializable
data class ProjectRow(
    @SerialName("Project") val project: Project = Project(),
    @SerialName("Open") val open: Int = 0,
    @SerialName("Total") val total: Int = 0,
    @SerialName("Blocked") val blocked: Boolean = false,
    @SerialName("Depth") val depth: Int = 0,
)

/** What DeleteProject took: the project's tasks, their reports and its milestones. */
@Serializable
data class RemovedCounts(
    @SerialName("Tasks") val tasks: Int = 0,
    @SerialName("Reports") val reports: Int = 0,
    @SerialName("Milestones") val milestones: Int = 0,
)

/** The patch of UpdateMilestone; an absent key is left alone. */
@Serializable
data class MilestonePatch(
    @SerialName("title") val title: String? = null,
    @SerialName("due_at") val dueAt: String? = null,
    @SerialName("closed") val closed: Boolean? = null,
    /** True writes `"due_at": null`, which clears the date. */
    @Transient val clearDueAt: Boolean = false,
)

/** One row of Settings > Categories: the category and its open task count. */
@Serializable
data class CategoryCount(
    @SerialName("Category") val category: Category = Category(),
    @SerialName("Open") val open: Int = 0,
)

/**
 * The input of CreateCategory and UpdateCategory; an absent key is left alone. [icon] is the name of
 * a library icon ([LibraryIcon.name]); [color] is indigo, yellow, green, red, blue, pink, teal or
 * orange.
 */
@Serializable
data class CategoryInput(
    @SerialName("name") val name: String? = null,
    @SerialName("color") val color: String? = null,
    @SerialName("icon") val icon: String? = null,
)

/** One icon of the library; [icon] is the 12 x 12 picture in the form Category.icon has. */
@Serializable
data class LibraryIcon(
    @SerialName("name") val name: String = "",
    @SerialName("keywords") val keywords: String = "",
    @SerialName("icon") val icon: List<List<Int>> = emptyList(),
)

/** A section of the library: the curated groups (a stable [name] to localize), or "" for a search. */
@Serializable
data class IconGroup(
    @SerialName("name") val name: String = "",
    @SerialName("icons") val icons: List<LibraryIcon> = emptyList(),
)

@Serializable
data class IconLibrary(
    @SerialName("groups") val groups: List<IconGroup> = emptyList(),
)

/** The patch of UpdateTasks; an absent key is left alone. projectId 0 moves the tasks to the Inbox. */
@Serializable
data class BulkPatch(
    @SerialName("status") val status: String? = null,
    @SerialName("priority") val priority: String? = null,
    @SerialName("project_id") val projectId: Long? = null,
)

/** One task DoTodayMany moved, with the due date it had ([before] is RFC 3339, "" for none). */
@Serializable
data class MovedToday(
    @SerialName("id") val id: Long = 0,
    @SerialName("before") val before: String = "",
)
