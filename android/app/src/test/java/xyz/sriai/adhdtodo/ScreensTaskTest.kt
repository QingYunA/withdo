package xyz.sriai.adhdtodo

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import androidx.compose.ui.unit.dp
import com.github.takahirom.roborazzi.ExperimentalRoborazziApi
import com.github.takahirom.roborazzi.captureRoboImage
import com.github.takahirom.roborazzi.captureScreenRoboImage
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.ParameterizedRobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.Milestone
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.ProjectPage
import xyz.sriai.adhdtodo.data.QuickDraft
import xyz.sriai.adhdtodo.data.QuickParse
import xyz.sriai.adhdtodo.data.TaskPage
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.testing.screenshotFile
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.BulkBar
import xyz.sriai.adhdtodo.ui.Composer
import xyz.sriai.adhdtodo.ui.LocalSelection
import xyz.sriai.adhdtodo.ui.MineScreen
import xyz.sriai.adhdtodo.ui.PageHeader
import xyz.sriai.adhdtodo.ui.SelectHeader
import xyz.sriai.adhdtodo.ui.SelectionUi
import xyz.sriai.adhdtodo.ui.TaskBackBar
import xyz.sriai.adhdtodo.ui.TaskMenuPage
import xyz.sriai.adhdtodo.ui.TaskMenuSheet
import xyz.sriai.adhdtodo.ui.TaskScreen
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.mineOrder
import xyz.sriai.adhdtodo.ui.quickHighlights
import xyz.sriai.adhdtodo.ui.quickPreview
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import java.time.LocalDate
import java.util.TimeZone

/**
 * The "change one task" screens of v43 as phone pictures, light and dark, English and Chinese: the
 * whole task page, its Due and Milestone pages, the composer reading a date, and select mode.
 * scripts/android-screens.sh 'xyz.sriai.adhdtodo.ScreensTaskTest' records them.
 */
@RunWith(ParameterizedRobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class ScreensTaskTest(private val case: ScreenCase) {
    @get:Rule
    val compose = createComposeRule()

    @Before
    fun fixZone() {
        TimeZone.setDefault(TimeZone.getTimeZone(FixedTime.zone))
    }

    @OptIn(ExperimentalRoborazziApi::class)
    @Test
    fun screenshot() {
        compose.setThemedContent(night = case.night, zh = case.zh) { TaskScreenCase(case.screen) }
        compose.waitForIdle()
        val file = screenshotFile(case.name)
        if (case.screen in sheets) captureScreenRoboImage(file) else compose.onRoot().captureRoboImage(file)
    }

    data class ScreenCase(val name: String, val night: Boolean, val zh: Boolean, val screen: Screen)

    enum class Screen { TASK_FULL, MENU_DUE, MENU_MILESTONE, COMPOSER_PARSED, SELECT }

    companion object {
        private val sheets = setOf(Screen.MENU_DUE, Screen.MENU_MILESTONE)

        @JvmStatic
        @ParameterizedRobolectricTestRunner.Parameters(name = "{0}")
        fun cases(): List<ScreenCase> = listOf(
            "task-full" to Screen.TASK_FULL,
            "task-menu-due" to Screen.MENU_DUE,
            "task-menu-milestone" to Screen.MENU_MILESTONE,
            "composer-parsed" to Screen.COMPOSER_PARSED,
            "select" to Screen.SELECT,
        ).flatMap { (name, screen) ->
            listOf(
                ScreenCase("$name-light", night = false, zh = false, screen = screen),
                ScreenCase("$name-dark", night = true, zh = false, screen = screen),
                ScreenCase("$name-light-zh", night = false, zh = true, screen = screen),
                ScreenCase("$name-dark-zh", night = true, zh = true, screen = screen),
            )
        }
    }
}

/** The mockup's task: mine, P1, Work, in withdo's v0.2 milestone, due Friday, with notes and a subtask. */
private fun fullTask(): TaskPage {
    val golden = Golden.load<TaskPage>("task-1.golden.json")
    return TaskPage(
        task = golden.task.copy(
            title = "Write the release notes",
            source = "user",
            status = "todo",
            priority = "P1",
            categoryId = 1,
            dueAt = "2026-10-23T18:00:00+08:00",
            notes = "Cover the two ways in, the reset flow and the localized emails. Link the v42 mockup.",
        ),
        subtasks = golden.subtasks.take(1),
        reports = emptyList(),
    )
}

private fun milestones(): List<Milestone> =
    Golden.load<ProjectPage>("project-1.golden.json").milestones + Milestone(
        id = 3, projectId = 1, title = "v0.3 sync", dueAt = "2026-11-20T09:00:00Z", open = true,
    )

@Composable
private fun TaskScreenCase(screen: ScreensTaskTest.Screen) {
    val resources = LocalContext.current.resources
    val configuration = LocalConfiguration.current
    val words = remember(configuration) { Words(resources, FixedTime.clock, FixedTime.zone) }
    val zh = configuration.locales[0].language == "zh"
    val mine = remember { Golden.load<MinePage>("mine.golden.json") }
    val projects = remember { Golden.load<List<Project>>("projects.golden.json") }
    val categories = remember { Golden.load<List<Category>>("categories.golden.json") }
    val page = remember { fullTask() }
    val tokens = LocalTokens.current

    when (screen) {
        ScreensTaskTest.Screen.TASK_FULL,
        ScreensTaskTest.Screen.MENU_DUE,
        ScreensTaskTest.Screen.MENU_MILESTONE,
        -> {
            Column(Modifier.fillMaxSize().background(tokens.bg)) {
                Column(Modifier.weight(1f).statusBarsPadding().imePadding()) {
                    TaskBackBar(title = stringResource(R.string.title_mine), number = page.task.id, onBack = {}, onMenu = {})
                    Box(Modifier.weight(1f)) {
                        TaskScreen(
                            page = page,
                            projects = projects,
                            words = words,
                            onSetPriority = {},
                            onOpenUrl = {},
                            onOpenTask = {},
                            categories = categories,
                            milestones = milestones(),
                        )
                    }
                    Composer(
                        value = "",
                        placeholder = stringResource(R.string.composer_add_note),
                        notice = null,
                        sending = false,
                        onValueChange = {},
                        onSend = {},
                    )
                }
            }
            if (screen != ScreensTaskTest.Screen.TASK_FULL) {
                TaskMenuSheet(
                    task = page.task,
                    words = words,
                    categories = categories,
                    projects = projects,
                    milestones = milestones(),
                    today = LocalDate.of(2026, 10, 20),
                    initialPage = if (screen == ScreensTaskTest.Screen.MENU_DUE) TaskMenuPage.Due else TaskMenuPage.Milestone,
                    onDismiss = {},
                )
            }
        }

        ScreensTaskTest.Screen.COMPOSER_PARSED -> {
            val text = if (zh) "明天 9 点打给房东" else "Call the landlord tomorrow at 9"
            val phrase = if (zh) "明天 9 点" else "tomorrow at 9"
            val start = text.indexOf(phrase)
            val parse = QuickParse(
                listOf(
                    QuickDraft(
                        title = if (zh) "打给房东" else "Call the landlord",
                        remindAt = "2026-10-21T09:00:00+08:00",
                        ranges = listOf(listOf(start, start + phrase.length)),
                    ),
                ),
            )
            Column(Modifier.fillMaxSize().background(tokens.bg)) {
                Column(Modifier.weight(1f).statusBarsPadding().imePadding()) {
                    PageHeader(title = words.headerDate(), subtitle = "", syncBlocked = false, onSync = {}, start = 28.dp)
                    Box(Modifier.weight(1f)) {
                        MineScreen(
                            page = mine,
                            agents = emptyMap(),
                            words = words,
                            onAccept = {},
                            onComplete = {},
                            onAnswer = {},
                            onOpenUrl = {},
                            onOpenTask = {},
                            onHowToConnect = {},
                            onViewAll = null,
                            categories = categories,
                        )
                    }
                    Composer(
                        value = text,
                        placeholder = "",
                        notice = null,
                        sending = false,
                        onValueChange = {},
                        onSend = {},
                        highlights = quickHighlights(parse, text.length),
                        preview = quickPreview(resources, words, parse, stringResource(R.string.title_inbox)),
                    )
                }
            }
        }

        ScreensTaskTest.Screen.SELECT -> {
            val ticked = mine.today.map { it.id }.toSet()
            val selection = SelectionUi(ticked, mineOrder(mine, false), toggle = {}, report = {})
            CompositionLocalProvider(LocalSelection provides selection) {
                Column(Modifier.fillMaxSize().background(tokens.bg)) {
                    Column(Modifier.weight(1f).statusBarsPadding()) {
                        SelectHeader(count = ticked.size, onSelectAll = {}, onDone = {})
                        Box(Modifier.weight(1f)) {
                            MineScreen(
                                page = mine.copy(fromAgents = emptyList()),
                                agents = emptyMap(),
                                words = words,
                                onAccept = {},
                                onComplete = {},
                                onAnswer = {},
                                onOpenUrl = {},
                                onOpenTask = {},
                                onHowToConnect = {},
                                onViewAll = null,
                                categories = categories,
                            )
                        }
                    }
                    BulkBar(
                        enabled = true,
                        onDone = {},
                        onDoToday = {},
                        onMove = {},
                        onPriority = {},
                        onDelete = {},
                    )
                }
            }
        }
    }
}
