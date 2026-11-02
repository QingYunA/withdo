package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.data.Account
import xyz.sriai.adhdtodo.data.AgentsPage
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.CategoryCount
import xyz.sriai.adhdtodo.data.CompletedPage
import xyz.sriai.adhdtodo.data.BulkPatch
import xyz.sriai.adhdtodo.data.CategoryInput
import xyz.sriai.adhdtodo.data.Device
import xyz.sriai.adhdtodo.data.IconLibrary
import xyz.sriai.adhdtodo.data.MilestonePatch
import xyz.sriai.adhdtodo.data.NewTask
import xyz.sriai.adhdtodo.data.TaskPatch
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.PageJson
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.ProjectPage
import xyz.sriai.adhdtodo.data.ProjectRow
import xyz.sriai.adhdtodo.data.QuickParse
import xyz.sriai.adhdtodo.data.Reminder
import xyz.sriai.adhdtodo.data.SignInState
import xyz.sriai.adhdtodo.data.SummaryPage
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.data.TaskPage
import java.io.File

/**
 * Verification 1 of A2a: every golden file under mobile/core/testdata decodes into the Kotlin data
 * classes with no error, and a few fields pin the shape the screens read.
 */
class GoldenDecodeTest {

    @Test
    fun `every golden file decodes`() {
        val dir = repoFile("mobile/core/testdata")
        val files = dir.listFiles { file -> file.name.endsWith(".golden.json") }.orEmpty().sortedBy { it.name }
        assertEquals("golden files", 18, files.size)
        for (file in files) {
            val text = file.readText()
            when (file.name) {
                "mine.golden.json" -> {
                    val page = PageJson.decodeFromString<MinePage>(text)
                    assertEquals(2, page.fromAgents.size)
                    assertEquals(3L, page.fromAgents[0].id)
                    assertEquals("blocked", page.fromAgents[0].status)
                    assertEquals(0, page.fromAgents[0].kind)
                    assertEquals("Needs a decision on the activation policy", page.fromAgents[0].words)
                    assertEquals(2, page.today.size)
                    assertEquals(2, page.next.size)
                    assertEquals("Book dentist", page.next[0].title)
                    assertEquals(2, page.noDate.size)
                    assertEquals(2, page.noDateTotal)
                    assertEquals(4, page.count)
                    assertEquals(2, page.done.size)
                    assertEquals("claude-code", page.done[0].by)
                }
                "agents.golden.json" -> {
                    val page = PageJson.decodeFromString<AgentsPage>(text)
                    assertEquals(3, page.working.size)
                    assertEquals(3, page.working[0].subs.size)
                    assertEquals("Port the MCP tools", page.working[0].subs[1].title)
                    assertTrue(page.working[0].live)
                    assertEquals(1, page.done.size)
                }
                "project-1.golden.json" -> {
                    val page = PageJson.decodeFromString<ProjectPage>(text)
                    assertEquals("withdo", page.project.name)
                    assertTrue(page.tasks.isNotEmpty())
                    assertEquals("claude-code", page.agents["1"])
                }
                "categories.golden.json" -> {
                    val categories = PageJson.decodeFromString<List<Category>>(text)
                    assertEquals(2, categories.size)
                    assertEquals("work", categories[0].builtin)
                    assertEquals("Work", categories[0].name)
                    assertEquals("blue", categories[0].color)
                    assertEquals(12, categories[0].icon.size)
                    assertEquals(12, categories[0].icon[0].size)
                }
                "projects.golden.json" -> {
                    val projects = PageJson.decodeFromString<List<Project>>(text)
                    assertEquals(3, projects.size)
                    assertEquals("answer-me-with-html", projects[0].name)
                }
                "search.golden.json" -> {
                    val tasks = PageJson.decodeFromString<List<Task>>(text)
                    assertTrue(tasks.isNotEmpty())
                    assertEquals(2L, tasks[0].id)
                    assertEquals("Native UI with mygo", tasks[0].title)
                }
                "task-1.golden.json" -> {
                    val page = PageJson.decodeFromString<TaskPage>(text)
                    assertEquals(1L, page.task.id)
                    assertTrue(page.reports.isNotEmpty())
                    assertEquals("claude-code", page.reports[0].agent)
                    assertEquals("claude-code", page.latestAgent)
                    assertNotNull(page.task.createdAt)
                }
                "account.golden.json" -> {
                    val account = PageJson.decodeFromString<Account>(text)
                    assertEquals("Ada Lovelace", account.name)
                    assertEquals("ada@example.com", account.email)
                    assertEquals("Ada Lovelace", account.display)
                }
                "devices.golden.json" -> {
                    val devices = PageJson.decodeFromString<List<Device>>(text)
                    assertEquals(2, devices.size)
                    assertEquals("d1", devices[0].id)
                    assertEquals("Pixel", devices[0].name)
                    assertTrue(devices[0].isThis)
                    assertNotNull(devices[0].lastSeenAt)
                    assertFalse(devices[1].isThis)
                    assertEquals(null, devices[1].lastSeenAt)
                }
                "reminders.golden.json" -> {
                    val reminders = PageJson.decodeFromString<List<Reminder>>(text)
                    assertEquals(2, reminders.size)
                    assertEquals(7L, reminders[0].taskId)
                    assertEquals("Pay rent", reminders[0].title)
                    assertEquals("Home", reminders[0].project)
                    assertEquals(null, reminders[1].project)
                }
                "signin-linked.golden.json" -> {
                    assertEquals("linked", PageJson.decodeFromString<SignInState>(text).state)
                }
                "signin-choose.golden.json" -> {
                    val answer = PageJson.decodeFromString<SignInState>(text)
                    assertEquals("choose", answer.state)
                    assertEquals(128, answer.localTasks)
                    assertEquals(6, answer.localProjects)
                    assertEquals(2, answer.accountDevices)
                }
                "completed.golden.json" -> {
                    val page = PageJson.decodeFromString<CompletedPage>(text)
                    assertFalse(page.more)
                    assertEquals("2026-10-20", page.days[0].day)
                    val agent = page.days[0].tasks.first { it.id == 20L }
                    assertEquals("claude-code", agent.by)
                    assertEquals("Claude Code", agent.byName)
                    assertEquals("withdo", agent.project)
                    assertEquals("PR #14 merged", agent.summary)
                    val mine = page.days[0].tasks.first { it.id == 21L }
                    assertEquals("", mine.by)
                    assertEquals("you", mine.byName)
                    assertNotNull(mine.completedAt)
                }
                "summary-day.golden.json", "summary-week.golden.json" -> {
                    val page = PageJson.decodeFromString<SummaryPage>(text)
                    assertEquals(file.name == "summary-week.golden.json", page.week)
                    assertTrue(page.total > 0)
                    assertEquals(page.total, page.ledger.sumOf { it.count })
                    assertEquals(page.total, page.goods.size)
                    assertTrue(page.ledger[0].you)
                    assertEquals("you", page.ledger[0].name)
                    assertEquals("Claude Code", page.ledger[1].agent)
                    assertEquals(14, page.ledger[1].tasks[0].prNumber)
                    assertEquals("", page.paragraph)
                    assertEquals(3, page.open.inProgress)
                    if (page.week) {
                        assertEquals("2026-10-19", page.from)
                        assertEquals("2026-10-25", page.to)
                        assertEquals(1, page.goods.last().day)
                    }
                }
                "quick.golden.json" -> {
                    val draft = PageJson.decodeFromString<QuickParse>(text).first!!
                    assertEquals("交房租", draft.title)
                    assertEquals("2026-10-21T09:00:00Z", draft.remindAt)
                    assertEquals("", draft.dueAt)
                    assertEquals(2L, draft.categoryId)
                    assertTrue(draft.important)
                    assertTrue(draft.read)
                    assertEquals(listOf(listOf(0, 7), listOf(11, 14), listOf(15, 16)), draft.ranges)
                }
                "project-list.golden.json" -> {
                    val rows = PageJson.decodeFromString<List<ProjectRow>>(text)
                    assertEquals(3, rows.size)
                    val withdo = rows.first { it.project.name == "withdo" }
                    assertEquals(6, withdo.open)
                    assertEquals(17, withdo.total)
                    assertTrue(withdo.blocked)
                    assertEquals(0, withdo.depth)
                    assertFalse(rows.first { it.project.name == "picpocket" }.blocked)
                }
                "categories-counts.golden.json" -> {
                    val rows = PageJson.decodeFromString<List<CategoryCount>>(text)
                    assertEquals(2, rows.size)
                    assertEquals("work", rows[0].category.builtin)
                    assertEquals(12, rows[0].open)
                    assertEquals(12, rows[0].category.icon.size)
                }
                else -> error("a golden file with no data class: ${file.name}")
            }
        }
    }

    @Test
    fun `the sync status decodes the settings the screen reads`() {
        val json = """{"state":"error","last_error":"the server did not answer","pending":3,"retry_at":"2026-10-20T09:02:00Z","failed_since":"2026-10-20T08:40:00Z","seed_total":3512,"server":"https://todo.sri-ai.xyz","linked":true,"signed_in":true}"""
        val status = PageJson.decodeFromString<SyncStatus>(json)
        assertEquals("error", status.state)
        assertEquals(3L, status.pending)
        assertEquals(3512L, status.seedTotal)
        assertEquals(3509L, status.seedDone)
        assertEquals("https://todo.sri-ai.xyz", status.server)
        assertTrue(status.linked)
        assertTrue(status.signedIn)
        assertNotNull(status.retryAt)
        assertNotNull(status.failedSince)
    }

    @Test
    fun `a signed-out status decodes for the sign-in card`() {
        val json = """{"state":"idle","pending":0,"server":"https://todo.sri-ai.xyz","linked":false,"signed_in":false}"""
        val status = PageJson.decodeFromString<SyncStatus>(json)
        assertEquals("idle", status.state)
        assertEquals("https://todo.sri-ai.xyz", status.server)
        assertFalse(status.linked)
        assertFalse(status.signedIn)
    }

    @Test
    fun `a signing-in status carries the stage`() {
        val json = """{"state":"idle","pending":0,"server":"https://todo.sri-ai.xyz","linked":false,"signed_in":true,"sign_in_stage":"downloading"}"""
        val status = PageJson.decodeFromString<SyncStatus>(json)
        assertEquals("downloading", status.signInStage)
    }

    @Test
    fun `the inputs of the organizing calls carry the Go keys`() {
        val task = PageJson.encodeToString(
            NewTask(title = "x", remindAt = "2026-10-21T09:00:00Z", categoryId = 2, priority = "P1"),
        )
        assertEquals("""{"title":"x","remind_at":"2026-10-21T09:00:00Z","category_id":2,"priority":"P1"}""", task)
        assertEquals("""{"milestone_id":0}""", PageJson.encodeToString(TaskPatch(milestoneId = 0)))
        assertEquals(
            """{"status":"done","project_id":0}""",
            PageJson.encodeToString(BulkPatch(status = "done", projectId = 0)),
        )
        assertEquals(
            """{"title":"v0.14","closed":true}""",
            PageJson.encodeToString(MilestonePatch(title = "v0.14", closed = true)),
        )
        assertEquals(
            """{"name":"Health","icon":"heart"}""",
            PageJson.encodeToString(CategoryInput(name = "Health", icon = "heart")),
        )
    }

    @Test
    fun `the icon library and a draft that read nothing decode`() {
        val library = PageJson.decodeFromString<IconLibrary>(
            """{"groups":[{"name":"Work","icons":[{"name":"briefcase","keywords":"工作","icon":[[0,35],[35,0]]}]},{"name":"","icons":[]}]}""",
        )
        assertEquals("briefcase", library.groups[0].icons[0].name)
        assertEquals(listOf(35, 0), library.groups[0].icons[0].icon[1])
        assertTrue(library.groups[1].icons.isEmpty())
        val plain = PageJson.decodeFromString<QuickParse>(
            """{"drafts":[{"title":"buy milk","remind_at":"","due_at":"","category_id":0,"category":"","important":false,"ranges":[]}]}""",
        ).first!!
        assertFalse(plain.read)
        assertEquals(null, PageJson.decodeFromString<QuickParse>("""{"drafts":[]}""").first)
    }

    @Test
    fun `unknown keys are ignored`() {
        val json = """{"FromAgents":[],"Today":[],"Next":[],"NoDate":[],"NoDateTotal":0,"Done":[],"MoreDone":0,"FromTheFuture":true}"""
        val page = PageJson.decodeFromString<MinePage>(json)
        assertTrue(page.empty)
    }

    private fun repoFile(path: String): File {
        var dir: File? = File("").absoluteFile
        while (dir != null) {
            val candidate = File(dir, path)
            if (candidate.exists()) return candidate
            dir = dir.parentFile
        }
        error("could not find $path above ${File("").absolutePath}")
    }
}
