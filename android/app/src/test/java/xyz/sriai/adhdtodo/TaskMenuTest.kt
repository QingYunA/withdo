package xyz.sriai.adhdtodo

import android.content.Context
import androidx.activity.ComponentActivity
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.unit.dp
import androidx.test.core.app.ApplicationProvider
import androidx.test.espresso.Espresso
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.RemindPreset
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.data.TaskPage
import xyz.sriai.adhdtodo.ui.theme.AdhdTheme
import xyz.sriai.adhdtodo.ui.DeletedSnackbar
import xyz.sriai.adhdtodo.ui.DeletedTask
import xyz.sriai.adhdtodo.ui.LongPressRow
import xyz.sriai.adhdtodo.ui.TaskBackBar
import xyz.sriai.adhdtodo.ui.TaskMenuPage
import xyz.sriai.adhdtodo.ui.TaskMenuSheet
import xyz.sriai.adhdtodo.ui.TaskScreen
import xyz.sriai.adhdtodo.ui.Words
import androidx.compose.material3.Text

/**
 * The task menu's interactions on the JVM: the long press and its movement rule, the strips, the
 * three pages, back, delete and Undo, the PR row and the task page's entry points. The Go core
 * cannot load on the JVM, so the sheet takes fake tasks and callbacks.
 */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w412dp-h915dp")
class TaskMenuTest {

    @get:Rule
    val compose = createAndroidComposeRule<ComponentActivity>()

    private val words: Words by lazy {
        Words(ApplicationProvider.getApplicationContext<Context>().resources)
    }

    private fun task(
        id: Long = 58,
        title: String = "Confirm the sync conflict policy",
        status: String = "doing",
        priority: String = "P1",
        categoryId: Int = 0,
        projectId: Long? = null,
        remindAt: String? = null,
        prNumber: Int = 0,
        prUrl: String = "",
    ) = Task(
        id = id,
        title = title,
        status = status,
        priority = priority,
        categoryId = categoryId,
        projectId = projectId,
        remindAt = remindAt,
        prNumber = prNumber,
        prUrl = prUrl,
    )

    private fun category(id: Long, name: String, color: String = "red") = Category(
        id = id,
        builtin = "",
        name = name,
        icon = List(12) { List(12) { 0 } },
        color = color,
        position = id.toInt(),
    )

    private fun project(id: Long, name: String) = Project(id = id, name = name)

    private fun content(block: @androidx.compose.runtime.Composable () -> Unit) {
        compose.setContent { AdhdTheme { block() } }
    }

    @Test
    fun `a long press opens the sheet, a moving long press does not, and a tap opens the task`() {
        val opened = mutableStateOf(false)
        var tapped = 0
        content {
            Box(Modifier.fillMaxSize()) {
                LongPressRow(onClick = { tapped++ }, onLongPress = { opened.value = true }) { row ->
                    Row(row.height(52.dp)) { Text("A task row") }
                }
                if (opened.value) {
                    TaskMenuSheet(task = task(), words = words, onDismiss = { opened.value = false })
                }
            }
        }
        compose.onNodeWithText("A task row").performTouchInput {
            down(center)
            advanceEventTime(700)
            up()
        }
        compose.waitForIdle()
        compose.onNodeWithText("Move to…").assertIsDisplayed()

        opened.value = false
        compose.waitForIdle()
        compose.onNodeWithText("A task row").performTouchInput {
            down(center)
            advanceEventTime(700)
            moveBy(Offset(120f, 0f))
            up()
        }
        compose.waitForIdle()
        compose.onNodeWithText("Move to…").assertDoesNotExist()

        compose.onNodeWithText("A task row").performClick()
        compose.runOnIdle { assertEquals(1, tapped) }
    }

    @Test
    fun `a child row's tap does not also open its parent row`() {
        var outer = 0
        var inner = 0
        content {
            LongPressRow(onClick = { outer++ }, onLongPress = {}) { outerModifier ->
                androidx.compose.foundation.layout.Column(outerModifier) {
                    LongPressRow(onClick = { inner++ }, onLongPress = {}) { innerModifier ->
                        Row(innerModifier.height(52.dp)) { Text("A subtask row") }
                    }
                }
            }
        }
        compose.onNodeWithText("A subtask row").performClick()
        compose.runOnIdle {
            assertEquals(1, inner)
            assertEquals(0, outer)
        }
    }

    @Test
    fun `a status tap and a priority tap call back and close the sheet`() {
        val open = mutableStateOf(true)
        var status = ""
        var priority = ""
        content {
            if (open.value) {
                TaskMenuSheet(
                    task = task(),
                    words = words,
                    onDismiss = { open.value = false },
                    onStatus = { status = it },
                    onPriority = { priority = it },
                )
            }
        }
        compose.onNodeWithContentDescription("Blocked").performClick()
        compose.waitForIdle()
        compose.runOnIdle {
            assertEquals("blocked", status)
            assertEquals(false, open.value)
        }
        compose.onNodeWithText("Move to…").assertDoesNotExist()

        open.value = true
        compose.waitForIdle()
        compose.onNodeWithText("P2").performClick()
        compose.waitForIdle()
        compose.runOnIdle {
            assertEquals("P2", priority)
            assertEquals(false, open.value)
        }
    }

    @Test
    fun `move to filters by name and returns a project id, and the inbox is null`() {
        val open = mutableStateOf(true)
        var moved: Long? = -1L
        val projects = listOf(project(1, "adhd-todo"), project(2, "adhd-landing"))
        content {
            if (open.value) {
                TaskMenuSheet(
                    task = task(projectId = 1),
                    words = words,
                    projects = projects,
                    onDismiss = { open.value = false },
                    onMove = { moved = it },
                )
            }
        }
        compose.onNodeWithText("Move to…").performClick()
        compose.waitForIdle()
        compose.onNodeWithContentDescription("Find a project").performTextInput("land")
        compose.waitForIdle()
        compose.onNodeWithText("adhd-todo").assertDoesNotExist()
        compose.onNodeWithText("adhd-landing").performClick()
        compose.waitForIdle()
        compose.runOnIdle {
            assertEquals(2L, moved)
            assertEquals(false, open.value)
        }

        open.value = true
        moved = -1L
        compose.waitForIdle()
        compose.onNodeWithText("Move to…").performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Inbox").performClick()
        compose.waitForIdle()
        compose.runOnIdle {
            assertNull(moved)
            assertEquals(false, open.value)
        }
    }

    @Test
    fun `the category page sends 0 for none`() {
        val open = mutableStateOf(true)
        var picked = -1
        val categories = listOf(category(1, "Work"), category(2, "Reading"))
        content {
            if (open.value) {
                TaskMenuSheet(
                    task = task(categoryId = 2),
                    words = words,
                    categories = categories,
                    onDismiss = { open.value = false },
                    onCategory = { picked = it },
                )
            }
        }
        compose.onNodeWithText("Category").performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Reading").assertIsDisplayed()
        compose.onNodeWithText("None").performClick()
        compose.waitForIdle()
        compose.runOnIdle {
            assertEquals(0, picked)
            assertEquals(false, open.value)
        }
    }

    @Test
    fun `a reminder preset sends its moment and no reminder sends null`() {
        val open = mutableStateOf(true)
        var at: Long? = -1L
        val preset = RemindPreset("later", 1_792_480_800_000L)
        content {
            if (open.value) {
                TaskMenuSheet(
                    task = task(),
                    words = words,
                    presets = listOf(preset),
                    onDismiss = { open.value = false },
                    onRemind = { at = it },
                )
            }
        }
        compose.onNodeWithText("Remind me…").performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Later today").performClick()
        compose.waitForIdle()
        compose.runOnIdle {
            assertEquals(preset.at, at)
            assertEquals(false, open.value)
        }

        open.value = true
        at = -1L
        compose.waitForIdle()
        compose.onNodeWithText("Remind me…").performClick()
        compose.waitForIdle()
        compose.onNodeWithText("No reminder").performClick()
        compose.waitForIdle()
        compose.runOnIdle {
            assertNull(at)
            assertEquals(false, open.value)
        }
    }

    @Test
    fun `back goes from a page to the main page, then closes the sheet`() {
        val open = mutableStateOf(true)
        content {
            if (open.value) {
                TaskMenuSheet(task = task(), words = words, onDismiss = { open.value = false })
            }
        }
        compose.onNodeWithText("Move to…").performClick()
        compose.waitForIdle()
        compose.onNodeWithContentDescription("Find a project").assertIsDisplayed()

        Espresso.pressBack()
        compose.waitForIdle()
        compose.onNodeWithText("Move to…").assertIsDisplayed()
        compose.onNodeWithContentDescription("Find a project").assertDoesNotExist()

        Espresso.pressBack()
        compose.waitForIdle()
        compose.runOnIdle { assertEquals(false, open.value) }
        compose.mainClock.advanceTimeBy(1000)
        compose.waitForIdle()
        compose.onNodeWithText("Move to…").assertDoesNotExist()
    }

    @Test
    fun `delete calls back and undo receives the snapshot`() {
        val open = mutableStateOf(true)
        val deleted = mutableStateOf<DeletedTask?>(null)
        var deleteCalls = 0
        var restored: DeletedTask? = null
        val snapshot = """{"Task":{"ID":58},"Reports":[],"Images":[],"Subtasks":[]}"""
        content {
            if (open.value) {
                TaskMenuSheet(
                    task = task(title = "Buy milk"),
                    words = words,
                    onDismiss = { open.value = false },
                    onDelete = {
                        deleteCalls++
                        deleted.value = DeletedTask("Buy milk", snapshot)
                    },
                )
            }
            DeletedSnackbar(deleted = deleted.value, onUndo = { restored = it; deleted.value = null })
        }
        compose.onNodeWithText("Delete").performClick()
        compose.waitForIdle()
        compose.runOnIdle { assertEquals(1, deleteCalls) }
        compose.onNodeWithText("Deleted “Buy milk”").assertIsDisplayed()
        compose.onNodeWithText("Undo").performClick()
        compose.waitForIdle()
        compose.runOnIdle {
            assertEquals(snapshot, restored?.snapshot)
            assertEquals("Buy milk", restored?.title)
        }
        compose.onNodeWithText("Deleted “Buy milk”").assertDoesNotExist()
    }

    @Test
    fun `open PR is only there when the task has a pull request`() {
        var url = ""
        content {
            TaskMenuSheet(
                task = task(prNumber = 73, prUrl = "https://github.com/example/pull/73"),
                words = words,
                onDismiss = {},
                onOpenUrl = { url = it },
            )
        }
        compose.onNodeWithText("Open PR #73").assertIsDisplayed()
        compose.onNodeWithText("GitHub").assertIsDisplayed()
        compose.onNodeWithText("Open PR #73").performClick()
        compose.runOnIdle { assertEquals("https://github.com/example/pull/73", url) }
    }

    @Test
    fun `a task without a pull request has no open PR row`() {
        content {
            TaskMenuSheet(task = task(), words = words, onDismiss = {})
        }
        compose.onNodeWithText("Open PR #73").assertDoesNotExist()
    }

    @Test
    fun `the task page's more opens the main page of the menu`() {
        val open = mutableStateOf(false)
        content {
            TaskBackBar(title = "Mine", number = 58, onBack = {}, onMenu = { open.value = true })
            if (open.value) {
                TaskMenuSheet(task = task(), words = words, onDismiss = { open.value = false })
            }
        }
        compose.onNodeWithContentDescription("Task menu").performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Move to…").assertIsDisplayed()
        compose.onNodeWithText("Doing", substring = true).assertIsDisplayed()
    }

    @Test
    fun `a task page's project row opens the move page`() {
        var page: TaskMenuPage? = null
        content {
            Box(Modifier.fillMaxSize()) {
                TaskScreen(
                    page = TaskPage(task = task(projectId = 1)),
                    projects = listOf(project(1, "adhd-todo")),
                    categories = listOf(category(1, "Work")),
                    words = words,
                    onSetPriority = {},
                    onOpenUrl = {},
                    onOpenTask = {},
                    onOpenMenu = { page = it },
                )
            }
        }
        compose.onNodeWithText("Project").performClick()
        compose.runOnIdle { assertEquals(TaskMenuPage.Move, page) }
        compose.onNodeWithText("Category").performClick()
        compose.runOnIdle { assertEquals(TaskMenuPage.Category, page) }
        compose.onNodeWithText("Remind").performClick()
        compose.runOnIdle { assertEquals(TaskMenuPage.Remind, page) }
    }
}
