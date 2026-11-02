package xyz.sriai.adhdtodo

import android.content.Context
import androidx.activity.ComponentActivity
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.height
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performImeAction
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.unit.dp
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.Milestone
import xyz.sriai.adhdtodo.data.QuickDraft
import xyz.sriai.adhdtodo.data.QuickParse
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.ui.AddSubtaskRow
import xyz.sriai.adhdtodo.ui.BulkBar
import xyz.sriai.adhdtodo.ui.Composer
import xyz.sriai.adhdtodo.ui.LocalSelection
import xyz.sriai.adhdtodo.ui.LongPressRow
import xyz.sriai.adhdtodo.ui.NotesField
import xyz.sriai.adhdtodo.ui.QuickPreview
import xyz.sriai.adhdtodo.ui.SelectionUi
import xyz.sriai.adhdtodo.ui.TaskMenuPage
import xyz.sriai.adhdtodo.ui.TaskMenuSheet
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.quickPreview
import xyz.sriai.adhdtodo.ui.quickReceipt
import xyz.sriai.adhdtodo.ui.theme.AdhdTheme
import xyz.sriai.adhdtodo.testing.FixedTime
import java.time.LocalDate

/**
 * The parts of "change one task" that answer a touch: the Due and Milestone pages, Select in the
 * menu, the notes and subtask fields, the composer's preview line, a tap in select mode and the
 * bulk bar. The Go core cannot load on the JVM, so everything takes fake data and callbacks.
 */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w412dp-h915dp")
class TaskChangeInteractionTest {

    @get:Rule
    val compose = createAndroidComposeRule<ComponentActivity>()

    private val words: Words by lazy {
        Words(ApplicationProvider.getApplicationContext<Context>().resources, FixedTime.clock, FixedTime.zone)
    }

    private fun content(block: @Composable () -> Unit) {
        compose.setContent { AdhdTheme { block() } }
    }

    private val today = LocalDate.of(2026, 11, 2)

    private fun task(milestoneId: Long? = null, dueAt: String? = null) =
        Task(id = 12, title = "Write the release notes", status = "todo", projectId = 1, milestoneId = milestoneId, dueAt = dueAt)

    @Test
    fun `the Due page sets a day and clears it`() {
        var picked: LocalDate? = LocalDate.MIN
        content {
            TaskMenuSheet(
                task = task(),
                words = words,
                today = today,
                initialPage = TaskMenuPage.Due,
                onDismiss = {},
                onDue = { picked = it },
            )
        }
        compose.onNodeWithText("This Friday").assertIsDisplayed()
        compose.onNodeWithText("Fri, Nov 6").assertIsDisplayed()
        compose.onNodeWithText("Next Monday").performClick()
        compose.runOnIdle { assertEquals(LocalDate.of(2026, 11, 9), picked) }

        compose.onNodeWithText("No due date").performClick()
        compose.runOnIdle { assertNull(picked) }
    }

    @Test
    fun `the Milestone page lists open milestones, None, and names a new one`() {
        var picked = -1L
        var created = ""
        val milestones = listOf(
            Milestone(id = 1, projectId = 1, title = "v0.13", dueAt = "2026-11-14T09:00:00+08:00", open = true),
            Milestone(id = 2, projectId = 1, title = "v0.12", open = false),
        )
        content {
            TaskMenuSheet(
                task = task(milestoneId = 1),
                words = words,
                milestones = milestones,
                initialPage = TaskMenuPage.Milestone,
                onDismiss = {},
                onMilestone = { picked = it },
                onNewMilestone = { created = it },
            )
        }
        compose.onNodeWithText("v0.13 · due Nov 14").assertIsDisplayed()
        compose.onNodeWithText("v0.12").assertDoesNotExist()
        compose.onNodeWithText("None").performClick()
        compose.runOnIdle { assertEquals(0L, picked) }

        compose.onNodeWithText("New milestone…").performClick()
        compose.onNodeWithText("Create").performClick() // Create waits for a name
        compose.runOnIdle { assertEquals("", created) }
        compose.onNode(hasSetTextAction()).performTextInput("v0.14")
        compose.onNodeWithText("Create").performClick()
        compose.runOnIdle { assertEquals("v0.14", created) }
    }

    @Test
    fun `Select shows under Remind me only where a list can be selected`() {
        var selected = 0
        var shown by mutableStateOf(true)
        content {
            TaskMenuSheet(
                task = task(),
                words = words,
                showSelect = shown,
                onDismiss = {},
                onSelect = { selected++ },
            )
        }
        compose.onNodeWithText("Select").assertIsDisplayed()
        compose.onNodeWithText("Select").performClick()
        compose.runOnIdle { assertEquals(1, selected) }

        compose.runOnIdle { shown = false }
        compose.onNodeWithText("Select").assertDoesNotExist()
    }

    @Test
    fun `notes save when the field loses focus and not before`() {
        val saved = mutableListOf<String>()
        val elsewhere = FocusRequester()
        content {
            androidx.compose.foundation.layout.Column {
                NotesField(task(), onSave = { saved += it })
                androidx.compose.foundation.text.BasicTextField(
                    value = "",
                    onValueChange = {},
                    modifier = Modifier.height(40.dp).focusRequester(elsewhere),
                )
            }
        }
        compose.onNodeWithText("Add notes…").assertIsDisplayed()
        compose.onAllNodes(hasSetTextAction())[0].performClick()
        compose.onAllNodes(hasSetTextAction())[0].performTextInput("Cover the two ways in")
        compose.runOnIdle { assertTrue(saved.isEmpty()) }
        // Another field takes the focus: the notes are saved, once.
        compose.runOnIdle { elsewhere.requestFocus() }
        compose.runOnIdle { assertEquals(listOf("Cover the two ways in"), saved) }
    }

    @Test
    fun `Add subtask turns into a field, Enter adds and keeps it open`() {
        val added = mutableListOf<String>()
        content { AddSubtaskRow(onAdd = { added += it }) }
        compose.onNodeWithText("Add subtask").performClick()
        compose.waitForIdle()
        compose.onNode(hasSetTextAction()).performTextInput("Collect the merged PRs")
        compose.onNode(hasSetTextAction()).performImeAction()
        compose.runOnIdle { assertEquals(listOf("Collect the merged PRs"), added) }
        // The field is still there, empty, for the next one.
        compose.onNodeWithText("Add subtask").assertIsDisplayed()
        compose.onNode(hasSetTextAction()).performTextInput("Second")
        compose.onNode(hasSetTextAction()).performImeAction()
        compose.runOnIdle { assertEquals(listOf("Collect the merged PRs", "Second"), added) }
    }

    @Test
    fun `the composer's preview line says what Send stores and a tap ignores it`() {
        var ignored = 0
        content {
            Composer(
                value = "call the landlord tomorrow at 9",
                placeholder = "",
                notice = "Added to No date",
                sending = false,
                onValueChange = {},
                onSend = {},
                highlights = listOf(18 until 31),
                preview = QuickPreview("Tomorrow 9:00 AM reminder", "Inbox"),
                onIgnorePreview = { ignored++ },
            )
        }
        compose.onNodeWithText("Tomorrow 9:00 AM reminder").assertIsDisplayed()
        compose.onNodeWithText("· Inbox").assertIsDisplayed()
        // The preview takes the receipt's place.
        compose.onNodeWithText("Added to No date").assertDoesNotExist()
        compose.onNodeWithText("Tomorrow 9:00 AM reminder").performClick()
        compose.runOnIdle { assertEquals(1, ignored) }
    }

    @Test
    fun `the preview words and the receipt use the app's date words`() {
        val res = ApplicationProvider.getApplicationContext<Context>().resources
        val parse = QuickParse(
            listOf(
                QuickDraft(
                    title = "Call the landlord",
                    remindAt = "2026-10-21T09:00:00+08:00",
                    category = "Work",
                    important = true,
                    ranges = listOf(listOf(0, 3)),
                ),
            ),
        )
        assertEquals(
            "Tomorrow 9:00 AM reminder, Work, important",
            quickPreview(res, words, parse, "Inbox")?.phrase,
        )
        assertEquals("Added · Tomorrow 9:00 AM reminder, Work, important", quickReceipt(res, words, parse))
        val due = QuickParse(listOf(QuickDraft(title = "Ship it", dueAt = "2026-10-23T18:00:00+08:00", ranges = listOf(listOf(0, 3)))))
        assertEquals("due Fri", quickPreview(res, words, due, "Inbox")?.phrase)
        val two = QuickParse(listOf(QuickDraft(title = "a", ranges = listOf(listOf(0, 1))), QuickDraft(title = "b")))
        assertEquals("Added 2 tasks", quickReceipt(res, words, two))
        assertNull(quickPreview(res, words, QuickParse(listOf(QuickDraft(title = "plain"))), "Inbox"))
    }

    @Test
    fun `in select mode a tap ticks the row instead of opening it`() {
        var opened = 0
        var held = 0
        val ticked = mutableListOf<Long>()
        content {
            val selection = SelectionUi(emptySet(), listOf(7), toggle = { ticked += it }, report = {})
            CompositionLocalProvider(LocalSelection provides selection) {
                LongPressRow(onClick = { opened++ }, onLongPress = { held++ }, taskId = 7) { row ->
                    Row(row.height(52.dp)) { Text("A task row") }
                }
            }
        }
        compose.onNodeWithText("A task row").performClick()
        compose.runOnIdle {
            assertEquals(listOf(7L), ticked)
            assertEquals(0, opened)
            assertEquals(0, held)
        }
    }

    @Test
    fun `the bulk bar calls back for each of its five actions`() {
        val called = mutableListOf<String>()
        content {
            BulkBar(
                enabled = true,
                onDone = { called += "done" },
                onDoToday = { called += "today" },
                onMove = { called += "move" },
                onPriority = { called += "priority" },
                onDelete = { called += "delete" },
            )
        }
        for (label in listOf("Done", "Do today", "Move to…", "Priority", "Delete")) {
            compose.onNodeWithText(label).performClick()
        }
        compose.runOnIdle { assertEquals(listOf("done", "today", "move", "priority", "delete"), called) }
        compose.onNodeWithContentDescription("Selected").assertDoesNotExist()
    }
}
