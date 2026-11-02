package xyz.sriai.adhdtodo

import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.down
import androidx.compose.ui.test.moveBy
import androidx.compose.ui.test.up
import androidx.compose.ui.test.advanceEventTime
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.CategoryCount
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.ProjectEntry
import xyz.sriai.adhdtodo.data.ProjectPage
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.data.UpdatePhase
import xyz.sriai.adhdtodo.data.UpdateUi
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.CategoriesScreen
import xyz.sriai.adhdtodo.ui.CategoryActions
import xyz.sriai.adhdtodo.ui.CategorySheet
import xyz.sriai.adhdtodo.ui.ProjectActions
import xyz.sriai.adhdtodo.ui.ProjectScreen
import xyz.sriai.adhdtodo.ui.ProjectView
import xyz.sriai.adhdtodo.ui.ProjectViewPrefs
import xyz.sriai.adhdtodo.ui.ProjectsScreen
import xyz.sriai.adhdtodo.ui.SettingsData
import xyz.sriai.adhdtodo.ui.SettingsScreen
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.theme.AppearanceMode

/** The Projects tab, the Board, Settings and Categories driven on the JVM through Compose's semantics. */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class ProjectsInteractionTest {
    @get:Rule
    val compose = createComposeRule()

    private val page: ProjectPage = Golden.load("project-1.golden.json")
    private val withdo = page.project

    @Composable
    private fun rememberWords(): Words {
        val resources = LocalContext.current.resources
        return remember(LocalConfiguration.current) { Words(resources, FixedTime.clock, FixedTime.zone) }
    }

    private fun longPress(text: String) {
        compose.onNodeWithText(text).performTouchInput {
            down(center)
            advanceEventTime(700)
            up()
        }
        compose.waitForIdle()
    }

    private fun entries() = listOf(
        ProjectEntry(withdo, open = 6, total = 17, blocked = true),
        ProjectEntry(Project(id = 4, name = "android", parentId = withdo.id), open = 3, total = 5, depth = 1),
    )

    @Test
    fun `a long press on a project row opens its sheet with the repository and the task count`() {
        compose.setThemedContent {
            ProjectsScreen(entries = entries(), onOpenProject = {}, onHowToConnect = {})
        }
        longPress("withdo")
        compose.onNodeWithText("QingYunA/withdo · 17 tasks").assertIsDisplayed()
        compose.onNodeWithText("Rename…").assertIsDisplayed()
        compose.onNodeWithText("New sub-project…").assertIsDisplayed()
        compose.onNodeWithText("New milestone…").assertIsDisplayed()
        compose.onNodeWithText("Open on GitHub").assertIsDisplayed()
        compose.onNodeWithText("Archive").assertIsDisplayed()
        compose.onNodeWithText("Delete…").assertIsDisplayed()
    }

    @Test
    fun `a project without a repository has no Open on GitHub`() {
        compose.setThemedContent {
            ProjectsScreen(entries = entries(), onOpenProject = {}, onHowToConnect = {})
        }
        longPress("android")
        compose.onNodeWithText("Rename…").assertIsDisplayed()
        compose.onNodeWithText("Open on GitHub").assertDoesNotExist()
    }

    @Test
    fun `Delete asks with the desktop's words and only then deletes`() {
        val deleted = mutableListOf<Long>()
        compose.setThemedContent {
            ProjectsScreen(
                entries = entries(),
                onOpenProject = {},
                onHowToConnect = {},
                actions = ProjectActions(deleteProject = { deleted += it }),
            )
        }
        longPress("withdo")
        compose.onNodeWithText("Delete…").performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Delete “withdo”?").assertIsDisplayed()
        compose.onNodeWithText(
            "This removes 17 tasks, their reports and the project’s milestones. It can’t be undone. " +
                "To keep the work, archive the project instead.",
        ).assertIsDisplayed()
        assertTrue(deleted.isEmpty())
        compose.onNodeWithText("Delete").performClick()
        compose.waitForIdle()
        assertEquals(listOf(withdo.id), deleted)
    }

    @Test
    fun `Archive asks the core to archive and the fold lists archived projects with Unarchive`() {
        val archived = mutableListOf<Long>()
        val unarchived = mutableListOf<Long>()
        compose.setThemedContent {
            ProjectsScreen(
                entries = entries(),
                onOpenProject = {},
                onHowToConnect = {},
                archived = listOf(ProjectEntry(Project(id = 9, name = "spike"), 0, 2)),
                actions = ProjectActions(archiveProject = { archived += it }, unarchiveProject = { unarchived += it }),
            )
        }
        longPress("withdo")
        compose.onNodeWithText("Archive").performClick()
        compose.waitForIdle()
        assertEquals(listOf(withdo.id), archived)

        compose.onNodeWithText("spike").assertDoesNotExist()
        compose.onNodeWithText("Archived").performClick()
        compose.waitForIdle()
        longPress("spike")
        compose.onNodeWithText("Unarchive").performClick()
        compose.waitForIdle()
        assertEquals(listOf(9L), unarchived)
    }

    @Test
    fun `the board shows the desktop's lanes, with a blocked task first in Doing`() {
        compose.setThemedContent { ProjectScreen(page = page, words = rememberWords(), onOpenTask = {}, initialView = ProjectView.Board) }
        compose.onNodeWithText("To do").assertIsDisplayed()
        compose.onNodeWithText("Doing").assertIsDisplayed()
        compose.onNodeWithText("Write the release notes").assertIsDisplayed()
        compose.onNodeWithText("Fix tray race on launch").assertExists()
    }

    @Test
    fun `a board card opens its task on a tap and its menu on a long press`() {
        var opened = 0L
        var menu = 0L
        compose.setThemedContent {
            ProjectScreen(
                page = page,
                words = rememberWords(),
                onOpenTask = { opened = it },
                onLongPress = { menu = it },
                initialView = ProjectView.Board,
            )
        }
        compose.onNodeWithText("Write the release notes").performClick()
        compose.runOnIdle { assertEquals(5L, opened) }
        longPress("Write the release notes")
        compose.runOnIdle { assertEquals(5L, menu) }
    }

    @Test
    fun `the List and Board control switches the view and remembers it per project`() {
        val context = RuntimeEnvironment.getApplication()
        ProjectViewPrefs.set(context, withdo.id, ProjectView.List)
        compose.setThemedContent { ProjectScreen(page = page, words = rememberWords(), onOpenTask = {}) }
        compose.onNodeWithText("Mine").assertIsDisplayed()
        compose.onNodeWithContentDescription("Board view").performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Doing").assertIsDisplayed()
        assertEquals(ProjectView.Board, ProjectViewPrefs.get(context, withdo.id))
        assertEquals(ProjectView.List, ProjectViewPrefs.get(context, 12345))
    }

    @Test
    fun `the milestone filter lists the milestones and narrows the page`() {
        compose.setThemedContent { ProjectScreen(page = page, words = rememberWords(), onOpenTask = {}) }
        compose.onNodeWithText("Write the release notes").assertExists()
        compose.onNodeWithText("Add a Windows build").assertExists()
        compose.onNodeWithText("All milestones").performClick()
        compose.waitForIdle()
        compose.onNodeWithText("v0.2 native · due Nov 3").assertIsDisplayed()
        compose.onNodeWithText("v0.1 spike · closed").assertIsDisplayed()
        compose.onNodeWithText("New milestone…").assertIsDisplayed()
        compose.onNodeWithText("v0.2 native · due Nov 3").performClick()
        compose.waitForIdle()
        // The filter now reads the milestone's name, and a task outside it is gone.
        compose.onNodeWithText("v0.2 native").assertIsDisplayed()
        compose.onNodeWithText("Add a Windows build").assertDoesNotExist()
        compose.onNodeWithText("Write the release notes").assertExists()
    }

    private fun settings(
        language: String = "",
        signedIn: Boolean = true,
        onLanguage: (String) -> Unit = {},
        onAppearance: (AppearanceMode) -> Unit = {},
        onCheck: () -> Unit = {},
    ) = compose.setThemedContent {
        SettingsScreen(
            data = SettingsData(
                account = xyz.sriai.adhdtodo.data.Account(email = "cyan@example.com"),
                devices = List(3) { xyz.sriai.adhdtodo.data.Device(id = "d$it", name = "d$it") },
                categories = List(4) { CategoryCount(Category(id = it + 1L, name = "c$it")) },
            ),
            status = SyncStatus(state = "idle", linked = signedIn, signedIn = signedIn),
            language = language,
            appearance = AppearanceMode.System,
            update = UpdateUi(UpdatePhase.IDLE, current = "0.13.0"),
            onOpenSync = {},
            onLanguage = onLanguage,
            onAppearance = onAppearance,
            onNotifications = {},
            onCategories = {},
            onCheckUpdate = onCheck,
            onUpdate = {},
            onOpenUrl = {},
        )
    }

    @Test
    fun `settings shows the account, the groups and the version`() {
        settings()
        compose.onNodeWithText("cyan@example.com").assertIsDisplayed()
        compose.onNodeWithText("Synced · 3 devices").assertIsDisplayed()
        compose.onNodeWithText("Language").assertIsDisplayed()
        compose.onNodeWithText("Appearance").assertIsDisplayed()
        compose.onNodeWithText("Notifications").assertIsDisplayed()
        compose.onNodeWithText("Categories").assertIsDisplayed()
        compose.onNodeWithText("4 categories").assertIsDisplayed()
        compose.onNodeWithText("Withdo 0.13.0").assertIsDisplayed()
        compose.onNodeWithText("Check now").assertIsDisplayed()
    }

    @Test
    fun `signed out, the account row says Sign in`() {
        settings(signedIn = false)
        compose.onNodeWithText("Sign in").assertIsDisplayed()
        compose.onNodeWithText("cyan@example.com").assertDoesNotExist()
    }

    @Test
    fun `language and appearance open a choice, and Check now asks the updater`() {
        var language: String? = null
        var appearance: AppearanceMode? = null
        var checked = 0
        settings(onLanguage = { language = it }, onAppearance = { appearance = it }, onCheck = { checked++ })
        compose.onNodeWithText("Language").performClick()
        compose.waitForIdle()
        compose.onNodeWithText("English").assertIsDisplayed()
        compose.onNodeWithText("简体中文").performClick()
        compose.waitForIdle()
        compose.runOnIdle { assertEquals("zh-Hans", language) }

        compose.onNodeWithText("Appearance").performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Dark").performClick()
        compose.waitForIdle()
        compose.runOnIdle { assertEquals(AppearanceMode.Dark, appearance) }

        compose.onNodeWithText("Check now").performClick()
        compose.runOnIdle { assertEquals(1, checked) }
    }

    private val counts = listOf(
        CategoryCount(Category(id = 1, builtin = "work", name = "Work", color = "blue"), 12),
        CategoryCount(Category(id = 2, builtin = "life", name = "Life", color = "green"), 7),
        CategoryCount(Category(id = 3, name = "Study", color = "yellow"), 3),
    )

    @Test
    fun `categories list their open tasks and end with New category`() {
        compose.setThemedContent { CategoriesScreen(categories = counts, actions = CategoryActions(), onChanged = {}) }
        compose.onNodeWithText("Work").assertIsDisplayed()
        compose.onNodeWithText("12 tasks").assertIsDisplayed()
        compose.onNodeWithText("Study").assertIsDisplayed()
        compose.onNodeWithText("New category…").assertIsDisplayed()
    }

    @Test
    fun `a long press then a drag moves a category to its new place`() {
        val moves = mutableListOf<Pair<Long, Int>>()
        compose.setThemedContent {
            CategoriesScreen(
                categories = counts,
                actions = CategoryActions(move = { id, index -> moves += id to index }),
                onChanged = {},
            )
        }
        compose.onNodeWithText("Work").performTouchInput {
            down(center)
            advanceEventTime(700)
            // Two rows down, in a few steps like a finger.
            repeat(8) { moveBy(Offset(0f, 40f)); advanceEventTime(16) }
            up()
        }
        compose.waitForIdle()
        assertEquals(listOf(1L to 2), moves)
    }

    @Test
    fun `Work and Life keep their names and cannot be deleted, a category of the user's can`() {
        compose.setThemedContent {
            CategorySheet(category = counts[0].category, actions = CategoryActions(), onDismiss = {})
        }
        compose.onNodeWithText("Work and Life keep their names.").assertIsDisplayed()
        // Only the icon search is a field: the name is not editable.
        assertEquals(1, compose.onAllTextFields().fetchSemanticsNodes().size)
        compose.onNodeWithText("Delete category").assertDoesNotExist()
    }

    @Test
    fun `a category of the user's has Delete, which asks first`() {
        var snapshot: String? = null
        compose.setThemedContent {
            CategorySheet(
                category = counts[2].category,
                actions = CategoryActions(delete = { "snap-$it" }),
                onDismiss = {},
                onDeleted = { _, s -> snapshot = s },
            )
        }
        compose.onNodeWithText("Delete category").performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Delete “Study”?").assertIsDisplayed()
        compose.onNodeWithText("Its tasks become unsorted.").assertIsDisplayed()
        compose.onNodeWithText("Delete").performClick()
        compose.waitForIdle()
        compose.runOnIdle { assertEquals("snap-3", snapshot) }
    }

    private fun androidx.compose.ui.test.junit4.ComposeContentTestRule.onAllTextFields() = onAllNodes(hasSetTextAction())
}
