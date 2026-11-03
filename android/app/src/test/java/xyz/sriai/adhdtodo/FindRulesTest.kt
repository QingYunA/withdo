package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.ui.CompletedScope
import xyz.sriai.adhdtodo.ui.SearchLimit
import xyz.sriai.adhdtodo.ui.completedRoute
import xyz.sriai.adhdtodo.ui.effectiveCategory
import xyz.sriai.adhdtodo.ui.filterMine
import xyz.sriai.adhdtodo.ui.groupSearch
import xyz.sriai.adhdtodo.ui.parentIds
import xyz.sriai.adhdtodo.ui.parseCompletedRoute
import xyz.sriai.adhdtodo.ui.repoProjectIds
import xyz.sriai.adhdtodo.ui.todayChips
import xyz.sriai.adhdtodo.ui.workCategoryId

/** The pure rules of Search, Today's category chips and the Completed routes. */
class FindRulesTest {
    private val withdo = Project(id = 1, name = "withdo", repoSlug = "QingYunA/withdo")
    private val notes = Project(id = 2, name = "notes")
    private val work = Category(id = 1, builtin = "work", name = "Work")
    private val life = Category(id = 2, builtin = "life", name = "Life")
    private val study = Category(id = 3, name = "Study")

    private fun task(id: Long, project: Long? = null, parent: Long? = null) =
        Task(id = id, projectId = project, parentId = parent, title = "t$id")

    private fun item(id: Long, category: Int = 0, project: Long? = null) =
        Item(id = id, categoryId = category, projectId = project)

    @Test
    fun `results group by project in the order of their newest task and no project last`() {
        val tasks = listOf(task(1), task(2, project = 2), task(3, project = 1), task(4, project = 2), task(5, project = 1))
        val grouped = groupSearch("t", tasks, listOf(withdo, notes))
        assertEquals(listOf(2L, 1L, null), grouped.groups.map { it.projectId })
        assertEquals(listOf("notes", "withdo", ""), grouped.groups.map { it.name })
        assertEquals(listOf(2L, 4L), grouped.groups[0].tasks.map { it.id })
        assertEquals(5, grouped.shown)
        assertFalse(grouped.more)
    }

    @Test
    fun `a task of a project the list does not know counts as no project`() {
        val grouped = groupSearch("t", listOf(task(1, project = 99)), listOf(withdo))
        assertEquals(listOf<Long?>(null), grouped.groups.map { it.projectId })
    }

    @Test
    fun `at most 100 results are shown and the rest is announced`() {
        val tasks = (1L..130L).map { task(it, project = 1) }
        val grouped = groupSearch("t", tasks, listOf(withdo))
        assertEquals(SearchLimit, grouped.shown)
        assertEquals(SearchLimit, grouped.groups.single().tasks.size)
        assertTrue(grouped.more)
        assertFalse(groupSearch("t", tasks.take(100), listOf(withdo)).more)
    }

    @Test
    fun `no results is no groups`() {
        val grouped = groupSearch("xyz", emptyList(), listOf(withdo))
        assertTrue(grouped.groups.isEmpty())
        assertEquals(0, grouped.shown)
    }

    @Test
    fun `the parents to name are the distinct parents of the shown subtasks`() {
        assertEquals(listOf(7L, 9L), parentIds(listOf(task(1, parent = 7), task(2), task(3, parent = 7), task(4, parent = 9))))
    }

    @Test
    fun `a task counts under its own category, else Work in a repository, else none`() {
        val repos = repoProjectIds(listOf(withdo, notes))
        assertEquals(setOf(1L), repos)
        assertEquals(2L, effectiveCategory(item(1, category = 2, project = 1), repos, workId = 1))
        assertEquals(1L, effectiveCategory(item(2, project = 1), repos, workId = 1))
        assertEquals(0L, effectiveCategory(item(3, project = 2), repos, workId = 1))
        assertEquals(0L, effectiveCategory(item(4), repos, workId = 1))
        assertEquals(1L, workCategoryId(listOf(life, work)))
        assertEquals(0L, workCategoryId(listOf(life)))
    }

    private val page = MinePage(
        fromAgents = listOf(item(10, project = 1)),
        today = listOf(item(1, category = 1), item(2, project = 1)),
        next = listOf(item(3, category = 2)),
        noDate = listOf(item(4, category = 2), item(5)),
        noDateTotal = 2,
        done = listOf(item(6, category = 2), item(7, category = 1)),
        moreDone = 3,
    )

    @Test
    fun `chips list the categories my open tasks count under, in the user's order`() {
        val repos = repoProjectIds(listOf(withdo))
        val chips = todayChips(page, listOf(work, life, study), repos, workId = 1)
        assertEquals(listOf(1L, 2L), chips.map { it.id })
    }

    @Test
    fun `fewer than two categories gives no chips`() {
        val one = page.copy(next = emptyList(), noDate = listOf(item(5)), today = listOf(item(1, category = 1)))
        assertTrue(todayChips(one, listOf(work, life), emptySet(), workId = 1).isEmpty())
        assertTrue(todayChips(MinePage(), listOf(work, life), emptySet(), workId = 1).isEmpty())
    }

    @Test
    fun `a chip narrows every list with the same rule and All leaves the page alone`() {
        val repos = repoProjectIds(listOf(withdo))
        assertEquals(page, filterMine(page, 0, repos, workId = 1))
        val workOnly = filterMine(page, 1, repos, workId = 1)
        assertEquals(listOf(10L), workOnly.fromAgents.map { it.id })
        assertEquals(listOf(1L, 2L), workOnly.today.map { it.id })
        assertTrue(workOnly.next.isEmpty())
        assertTrue(workOnly.noDate.isEmpty())
        assertEquals(0, workOnly.noDateTotal)
        assertEquals(listOf(7L), workOnly.done.map { it.id })
        val lifeOnly = filterMine(page, 2, repos, workId = 1)
        assertEquals(listOf(3L), lifeOnly.next.map { it.id })
        assertEquals(listOf(4L), lifeOnly.noDate.map { it.id })
        assertEquals(1, lifeOnly.noDateTotal)
    }

    @Test
    fun `a completed route names its scope and back`() {
        for (scope in listOf(
            CompletedScope(),
            CompletedScope(agents = true),
            CompletedScope(projectId = 7),
            CompletedScope(projectId = 7, agents = true),
        )) {
            assertEquals(scope, parseCompletedRoute(completedRoute(scope)))
        }
        assertEquals("completed:all", completedRoute(CompletedScope()))
        assertEquals("completed:agents", completedRoute(CompletedScope(agents = true)))
        assertEquals("completed:project:7", completedRoute(CompletedScope(projectId = 7)))
    }

    @Test
    fun `other routes and malformed ones are not completed routes`() {
        assertNull(parseCompletedRoute("home"))
        assertNull(parseCompletedRoute("project:3"))
        assertNull(parseCompletedRoute("completed:"))
        assertNull(parseCompletedRoute("completed:project:x"))
        assertNull(parseCompletedRoute("completed:project:0"))
    }
}
