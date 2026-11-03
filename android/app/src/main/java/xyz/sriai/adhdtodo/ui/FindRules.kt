package xyz.sriai.adhdtodo.ui

import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.Task

/*
 * The rules of the Find group (v43): how search results group, how Today narrows to a category,
 * and how a Completed route names its scope. Plain functions so JUnit can pin them.
 */

/** Search shows at most this many tasks; a quiet last line says when there are more. */
const val SearchLimit = 100

/** One project's results; [projectId] is null for the group of tasks with no project. */
data class SearchGroup(val projectId: Long?, val name: String, val tasks: List<Task>)

/** What the Search page draws: the groups, how many tasks they hold, and whether more exist. */
data class SearchResults(
    val query: String,
    val groups: List<SearchGroup>,
    val shown: Int,
    val more: Boolean,
    /** The title of a subtask's parent by the parent's id, where a result keeps its project name. */
    val parents: Map<Long, String> = emptyMap(),
)

/**
 * Groups the core's results (newest first) by project, at most [SearchLimit] of them: the groups
 * come in the order of their newest task, the tasks with no project last. A task of a project the
 * list does not know (an archived one) counts as no project.
 */
fun groupSearch(
    query: String,
    tasks: List<Task>,
    projects: List<Project>,
    parents: Map<Long, String> = emptyMap(),
): SearchResults {
    val capped = tasks.take(SearchLimit)
    val names = projects.associate { it.id to it.name }
    val byProject = LinkedHashMap<Long?, MutableList<Task>>()
    for (task in capped) {
        val id = task.projectId?.takeIf { it in names }
        byProject.getOrPut(id) { mutableListOf() }.add(task)
    }
    val groups = byProject.entries
        .filter { it.key != null }
        .map { (id, list) -> SearchGroup(id, names.getValue(id!!), list) } +
        listOfNotNull(byProject[null]?.let { SearchGroup(null, "", it) })
    return SearchResults(query, groups, capped.size, tasks.size > SearchLimit, parents)
}

/** The ids of the parents of the subtasks among the results, which the rows name. */
fun parentIds(tasks: List<Task>): List<Long> =
    tasks.take(SearchLimit).mapNotNull { it.parentId }.distinct()

// ---- Category chips of Today ---------------------------------------------------------------

/** What an item's category counts as: its own, else Work in a repository's project, else 0 (data.CategoryOf). */
fun effectiveCategory(item: Item, repoProjects: Set<Long>, workId: Long): Long = when {
    item.categoryId != 0 -> item.categoryId.toLong()
    item.projectId != null && item.projectId in repoProjects -> workId
    else -> 0L
}

/** The ids of the projects that have a repository, where an uncategorized task counts as Work. */
fun repoProjectIds(projects: List<Project>): Set<Long> =
    projects.filter { it.repoSlug.isNotEmpty() || it.repoPath.isNotEmpty() }.map { it.id }.toSet()

/** The id of the built-in Work, 0 when the list has none. */
fun workCategoryId(categories: List<Category>): Long = categories.firstOrNull { it.builtin == "work" }?.id ?: 0L

/**
 * The chips under Today's date: each category (in the user's order) that my open tasks of Today,
 * Next and No date count under; none when fewer than two, since one chip would filter nothing.
 */
fun todayChips(page: MinePage, categories: List<Category>, repoProjects: Set<Long>, workId: Long): List<Category> {
    val used = (page.today + page.next + page.noDate)
        .map { effectiveCategory(it, repoProjects, workId) }
        .filter { it != 0L }
        .toSet()
    val chips = categories.filter { it.id in used }
    return if (chips.size >= 2) chips else emptyList()
}

/**
 * Narrows the page to the category as the desktop's builder does with its keep function: every
 * list keeps the items counting under it, and the No date count follows. The Done today count
 * beyond the rows the core sent is not known per category, so it is dropped.
 */
fun filterMine(page: MinePage, category: Long, repoProjects: Set<Long>, workId: Long): MinePage {
    if (category == 0L) return page
    fun keep(list: List<Item>) = list.filter { effectiveCategory(it, repoProjects, workId) == category }
    val noDate = keep(page.noDate)
    return page.copy(
        fromAgents = keep(page.fromAgents),
        today = keep(page.today),
        next = keep(page.next),
        noDate = noDate,
        noDateTotal = noDate.size,
        done = keep(page.done),
        moreDone = 0,
    )
}

// ---- Completed routes ------------------------------------------------------------------------

/** What a Completed page lists: everything (both 0 and false), what agents wrote, or one project. */
data class CompletedScope(val projectId: Long = 0, val agents: Boolean = false)

const val CompletedPrefix = "completed:"

/** The stack entry of a scope: `completed:all`, `completed:agents`, `completed:project:7[:agents]`. */
fun completedRoute(scope: CompletedScope): String = CompletedPrefix + when {
    scope.projectId > 0 -> "project:${scope.projectId}" + if (scope.agents) ":agents" else ""
    scope.agents -> "agents"
    else -> "all"
}

/** The scope a `completed:` route names, or null for any other route or a malformed one. */
fun parseCompletedRoute(route: String): CompletedScope? {
    if (!route.startsWith(CompletedPrefix)) return null
    val parts = route.removePrefix(CompletedPrefix).split(':')
    return when (parts.firstOrNull()) {
        "all" -> CompletedScope()
        "agents" -> CompletedScope(agents = true)
        "project" -> {
            val id = parts.getOrNull(1)?.toLongOrNull()?.takeIf { it > 0 } ?: return null
            CompletedScope(projectId = id, agents = parts.getOrNull(2) == "agents")
        }
        else -> null
    }
}
