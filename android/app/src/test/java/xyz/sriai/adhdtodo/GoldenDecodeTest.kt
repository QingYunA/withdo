package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.data.AgentsPage
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.PageJson
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.ProjectPage
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
        assertEquals("golden files", 6, files.size)
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
                else -> error("a golden file with no data class: ${file.name}")
            }
        }
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
