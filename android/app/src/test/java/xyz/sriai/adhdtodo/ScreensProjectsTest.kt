package xyz.sriai.adhdtodo

import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import com.github.takahirom.roborazzi.ExperimentalRoborazziApi
import com.github.takahirom.roborazzi.captureRoboImage
import com.github.takahirom.roborazzi.captureScreenRoboImage
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.ParameterizedRobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.Account
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.CategoryCount
import xyz.sriai.adhdtodo.data.Device
import xyz.sriai.adhdtodo.data.IconLibrary
import xyz.sriai.adhdtodo.data.PageJson
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.ProjectEntry
import xyz.sriai.adhdtodo.data.ProjectPage
import xyz.sriai.adhdtodo.data.ProjectRow
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.data.TaskPage
import xyz.sriai.adhdtodo.data.UpdatePhase
import xyz.sriai.adhdtodo.data.UpdateUi
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.testing.screenshotFile
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.CategoriesScreen
import xyz.sriai.adhdtodo.ui.CategoryActions
import xyz.sriai.adhdtodo.ui.CategorySheet
import xyz.sriai.adhdtodo.ui.HeaderIconButton
import xyz.sriai.adhdtodo.ui.MilestoneFilterSheet
import xyz.sriai.adhdtodo.ui.NewProjectSheet
import xyz.sriai.adhdtodo.ui.NotificationSettingsScreen
import xyz.sriai.adhdtodo.ui.PageHeader
import xyz.sriai.adhdtodo.ui.ProjectActions
import xyz.sriai.adhdtodo.ui.ProjectScreen
import xyz.sriai.adhdtodo.ui.ProjectSheet
import xyz.sriai.adhdtodo.ui.ProjectView
import xyz.sriai.adhdtodo.ui.ProjectsScreen
import xyz.sriai.adhdtodo.ui.SettingsData
import xyz.sriai.adhdtodo.ui.SettingsScreen
import xyz.sriai.adhdtodo.ui.SyncCloud
import xyz.sriai.adhdtodo.ui.Tab
import xyz.sriai.adhdtodo.ui.TaskMenuPage
import xyz.sriai.adhdtodo.ui.TaskMenuSheet
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.theme.AppearanceMode
import java.io.File

/**
 * The v43 pictures of the Projects tab, the project sheet, the Board, Settings and Categories, light
 * and dark (Chinese where the copy matters). Render with
 * `scripts/android-screens.sh 'xyz.sriai.adhdtodo.ScreensProjectsTest'`.
 */
@RunWith(ParameterizedRobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class ScreensProjectsTest(private val case: ProjectsCase) {
    @get:Rule
    val compose = createComposeRule()

    @OptIn(ExperimentalRoborazziApi::class)
    @Test
    fun screenshot() {
        compose.setThemedContent(night = case.night, zh = case.zh) { Frame(case.screen) }
        compose.waitForIdle()
        val file = screenshotFile(case.name)
        if (case.screen in sheets) captureScreenRoboImage(file) else compose.onRoot().captureRoboImage(file)
    }

    data class ProjectsCase(val name: String, val night: Boolean, val zh: Boolean, val screen: Shot)

    enum class Shot {
        PROJECTS, PROJECT_SHEET, PROJECT_NEW, PROJECT_LIST, BOARD, BOARD_MILESTONES,
        SETTINGS, NOTIFICATIONS, CATEGORIES, CATEGORY_SHEET, CATEGORY_SHEET_NEW, TASK_MENU_CATEGORY,
    }

    companion object {
        private val sheets = setOf(
            Shot.PROJECT_SHEET, Shot.PROJECT_NEW, Shot.BOARD_MILESTONES, Shot.CATEGORY_SHEET,
            Shot.CATEGORY_SHEET_NEW, Shot.TASK_MENU_CATEGORY,
        )

        @JvmStatic
        @ParameterizedRobolectricTestRunner.Parameters(name = "{0}")
        fun cases(): List<ProjectsCase> {
            val shots = listOf(
                Triple("projects", Shot.PROJECTS, true),
                Triple("project-sheet", Shot.PROJECT_SHEET, true),
                Triple("project-new", Shot.PROJECT_NEW, false),
                Triple("project", Shot.PROJECT_LIST, false),
                Triple("board", Shot.BOARD, true),
                Triple("board-milestones", Shot.BOARD_MILESTONES, true),
                Triple("settings", Shot.SETTINGS, true),
                Triple("settings-notifications", Shot.NOTIFICATIONS, false),
                Triple("categories", Shot.CATEGORIES, true),
                Triple("category-sheet", Shot.CATEGORY_SHEET, true),
                Triple("category-sheet-new", Shot.CATEGORY_SHEET_NEW, false),
                Triple("task-menu-category", Shot.TASK_MENU_CATEGORY, false),
            )
            return shots.flatMap { (name, shot, zh) ->
                buildList {
                    add(ProjectsCase("$name-light", false, false, shot))
                    add(ProjectsCase("$name-dark", true, false, shot))
                    if (zh) {
                        add(ProjectsCase("$name-light-zh", false, true, shot))
                        add(ProjectsCase("$name-dark-zh", true, true, shot))
                    }
                }
            }
        }
    }
}

private val icons: IconLibrary by lazy {
    PageJson.decodeFromString(File(Golden.root, "android/app/src/test/resources/category-icons.json").readText())
}

/** The Projects tab as the mockup lists it: a blocked withdo with android and site under it. */
private fun projectEntries(): List<ProjectEntry> {
    val rows = Golden.load<List<ProjectRow>>("project-list.golden.json")
    fun entry(name: String) = rows.first { it.project.name == name }.let {
        ProjectEntry(it.project, it.open, it.total, it.blocked, it.depth)
    }
    val withdo = entry("withdo")
    fun sub(id: Long, name: String, open: Int) = ProjectEntry(
        project = withdo.project.copy(id = id, name = name, repoSlug = "", parentId = withdo.project.id),
        open = open, total = open + 2, depth = 1,
    )
    return listOf(entry("answer-me-with-html"), entry("picpocket"), withdo, sub(4, "android", 3), sub(5, "site", 0))
}

private fun archived(): List<ProjectEntry> {
    val base = Golden.load<List<Project>>("projects.golden.json").first()
    return listOf(
        ProjectEntry(base.copy(id = 6, name = "old-landing", repoSlug = ""), 0, 4),
        ProjectEntry(base.copy(id = 7, name = "spike", repoSlug = ""), 0, 1),
    )
}

private fun categories(): List<CategoryCount> {
    val golden = Golden.load<List<CategoryCount>>("categories-counts.golden.json")
    fun pick(name: String) = icons.groups.flatMap { it.icons }.first { it.name == name }.icon
    fun own(id: Long, name: String, color: String, icon: String, open: Int, position: Int) = CategoryCount(
        Category(id = id, name = name, color = color, icon = pick(icon), position = position), open,
    )
    return listOf(
        golden[0].copy(open = 12),
        golden[1].copy(open = 7),
        own(3, "学习", "yellow", "library", 3, 2),
        own(4, "健康", "red", "heart", 1, 3),
    )
}

@Composable
private fun Frame(shot: ScreensProjectsTest.Shot) {
    val resources = LocalContext.current.resources
    val words = remember(LocalConfiguration.current) { Words(resources, FixedTime.clock, FixedTime.zone) }
    val project = remember { Golden.load<ProjectPage>("project-1.golden.json") }
    val task = remember { Golden.load<TaskPage>("task-1.golden.json") }
    val cats = remember { categories() }
    val entries = remember { projectEntries() }

    when (shot) {
        ScreensProjectsTest.Shot.PROJECTS,
        ScreensProjectsTest.Shot.PROJECT_SHEET,
        ScreensProjectsTest.Shot.PROJECT_NEW,
        -> {
            ProjectsFrame(entries)
            if (shot == ScreensProjectsTest.Shot.PROJECT_SHEET) {
                ProjectSheet(
                    entry = entries.first { it.project.name == "withdo" },
                    archived = false,
                    actions = ProjectActions(),
                    onOpenUrl = {},
                    onDismiss = {},
                )
            }
            if (shot == ScreensProjectsTest.Shot.PROJECT_NEW) NewProjectSheet(ProjectActions(), onDismiss = {})
        }

        ScreensProjectsTest.Shot.PROJECT_LIST,
        ScreensProjectsTest.Shot.BOARD,
        ScreensProjectsTest.Shot.BOARD_MILESTONES,
        -> {
            TestFrame(
                title = stringResource(R.string.title_projects),
                tab = Tab.PROJECTS,
                back = stringResource(R.string.title_projects),
                backRight = { SyncCloud(blocked = false, onSync = {}) },
                composer = stringResource(R.string.composer_add_to_project, project.project.name),
            ) {
                ProjectScreen(
                    page = project,
                    words = words,
                    onOpenTask = {},
                    initialView = if (shot == ScreensProjectsTest.Shot.PROJECT_LIST) ProjectView.List else ProjectView.Board,
                )
            }
            if (shot == ScreensProjectsTest.Shot.BOARD_MILESTONES) {
                MilestoneFilterSheet(
                    projectId = project.project.id,
                    milestones = project.milestones,
                    selected = null,
                    actions = ProjectActions(),
                    onSelect = {},
                    onDismiss = {},
                )
            }
        }

        ScreensProjectsTest.Shot.SETTINGS -> TestFrame(
            title = stringResource(R.string.settings_title),
            tab = Tab.PROJECTS,
            back = stringResource(R.string.title_projects),
        ) {
            SettingsScreen(
                data = SettingsData(
                    account = Golden.load<Account>("account.golden.json").copy(email = "cyan@example.com", name = ""),
                    devices = Golden.load<List<Device>>("devices.golden.json").take(3),
                    categories = cats,
                ),
                status = SyncStatus(state = "idle", linked = true, signedIn = true),
                language = "",
                appearance = AppearanceMode.System,
                update = UpdateUi(UpdatePhase.UP_TO_DATE, current = "0.13.0"),
                onOpenSync = {},
                onLanguage = {},
                onAppearance = {},
                onNotifications = {},
                onCategories = {},
                onCheckUpdate = {},
                onUpdate = {},
                onOpenUrl = {},
            )
        }

        ScreensProjectsTest.Shot.NOTIFICATIONS -> TestFrame(
            title = stringResource(R.string.settings_notifications),
            tab = Tab.PROJECTS,
            back = stringResource(R.string.settings_title),
        ) {
            NotificationSettingsScreen(
                reminders = true,
                endOfDay = true,
                minutes = 18 * 60 + 30,
                systemAllows = true,
                onReminders = {},
                onEndOfDay = {},
                onMinutes = {},
                onOpenSystemSettings = {},
            )
        }

        ScreensProjectsTest.Shot.CATEGORIES,
        ScreensProjectsTest.Shot.CATEGORY_SHEET,
        -> {
            TestFrame(
                title = stringResource(R.string.settings_categories),
                tab = Tab.PROJECTS,
                back = stringResource(R.string.settings_title),
            ) {
                CategoriesScreen(
                    categories = cats,
                    actions = CategoryActions(),
                    onChanged = {},
                    libraryForPreview = icons,
                    startWith = if (shot == ScreensProjectsTest.Shot.CATEGORY_SHEET) cats[2].category else null,
                )
            }
        }

        ScreensProjectsTest.Shot.CATEGORY_SHEET_NEW -> {
            TestFrame(
                title = stringResource(R.string.settings_categories),
                tab = Tab.PROJECTS,
                back = stringResource(R.string.settings_title),
            ) { CategoriesScreen(categories = cats, actions = CategoryActions(), onChanged = {}) }
            CategorySheet(category = null, actions = CategoryActions(), onDismiss = {}, initialLibrary = icons)
        }

        ScreensProjectsTest.Shot.TASK_MENU_CATEGORY -> {
            ProjectsFrame(entries)
            TaskMenuSheet(
                task = task.task.copy(source = "user", dueAt = null),
                words = words,
                agent = task.latestAgent,
                categories = cats.map { it.category },
                projects = Golden.load<List<Project>>("projects.golden.json"),
                presets = FixedTime.remindPresets,
                initialPage = TaskMenuPage.Category,
                onDismiss = {},
            )
        }
    }
}

/** The Projects tab with its header: the title, plus, gear and the cloud. */
@Composable
private fun ProjectsFrame(entries: List<ProjectEntry>) {
    TestFrame(
        title = stringResource(R.string.title_projects),
        tab = Tab.PROJECTS,
        topBar = {
            PageHeader(
                title = stringResource(R.string.title_projects),
                subtitle = "",
                syncBlocked = false,
                onSync = {},
                actions = {
                    HeaderIconButton("plus", stringResource(R.string.manage_new_project), {})
                    HeaderIconButton("gear", stringResource(R.string.settings_title), {})
                },
            )
        },
    ) {
        ProjectsScreen(entries = entries, onOpenProject = {}, onHowToConnect = {}, archived = archived())
    }
}
