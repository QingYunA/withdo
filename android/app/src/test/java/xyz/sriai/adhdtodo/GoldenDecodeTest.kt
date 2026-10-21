package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.data.Account
import xyz.sriai.adhdtodo.data.AgentsPage
import xyz.sriai.adhdtodo.data.Device
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.PageJson
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.ProjectPage
import xyz.sriai.adhdtodo.data.Reminder
import xyz.sriai.adhdtodo.data.SignInState
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
        assertEquals("golden files", 11, files.size)
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
                    assertEquals("adhd-todo", page.project.name)
                    assertTrue(page.tasks.isNotEmpty())
                    assertEquals("claude-code", page.agents["1"])
                }
                "projects.golden.json" -> {
                    val projects = PageJson.decodeFromString<List<Project>>(text)
                    assertEquals(3, projects.size)
                    assertEquals("adhd-todo", projects[0].name)
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
    fun `unknown keys are ignored`() {
        val json = """{"FromAgents":[],"Today":[],"Later":[],"Done":[],"MoreLater":0,"MoreDone":0,"FromTheFuture":true}"""
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
