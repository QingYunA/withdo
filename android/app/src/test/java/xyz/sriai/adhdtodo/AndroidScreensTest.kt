package xyz.sriai.adhdtodo

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import androidx.compose.ui.unit.dp
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
import xyz.sriai.adhdtodo.data.AgentInfo
import xyz.sriai.adhdtodo.data.AgentsPage
import xyz.sriai.adhdtodo.data.Device
import xyz.sriai.adhdtodo.data.Link
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.ProjectEntry
import xyz.sriai.adhdtodo.data.ProjectPage
import xyz.sriai.adhdtodo.data.SignInState
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.data.TaskPage
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.ui.TaskBackBar
import xyz.sriai.adhdtodo.ui.TaskMenuPage
import xyz.sriai.adhdtodo.ui.TaskMenuSheet
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.testing.screenshotFile
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.AgentsScreen
import xyz.sriai.adhdtodo.ui.BackBar
import xyz.sriai.adhdtodo.ui.BottomNav
import xyz.sriai.adhdtodo.ui.Composer
import xyz.sriai.adhdtodo.ui.DEFAULT_SYNC_SERVER
import xyz.sriai.adhdtodo.ui.MineScreen
import xyz.sriai.adhdtodo.ui.PageHeader
import xyz.sriai.adhdtodo.ui.PlaceholderScreen
import xyz.sriai.adhdtodo.ui.ProjectScreen
import xyz.sriai.adhdtodo.ui.ProjectsScreen
import xyz.sriai.adhdtodo.ui.CodeFailure
import xyz.sriai.adhdtodo.ui.SignInFlow
import xyz.sriai.adhdtodo.ui.SignInStep
import xyz.sriai.adhdtodo.ui.SyncCloud
import xyz.sriai.adhdtodo.ui.SyncModel
import xyz.sriai.adhdtodo.ui.SyncPhase
import xyz.sriai.adhdtodo.ui.SyncScreen
import xyz.sriai.adhdtodo.ui.Tab
import xyz.sriai.adhdtodo.ui.TaskScreen
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import java.time.Instant
import java.time.temporal.ChronoUnit

/**
 * Every Android screen as a full phone picture, light and dark, under the app's own theme. The
 * screenshots are recorded into android/app/build/screenshots; scripts/android-screens.sh copies
 * them into docs/design/screens/android. A normal testDebugUnitTest renders the same screens but
 * Roborazzi's record flag is off, so it writes nothing and compares nothing.
 *
 * App() needs the Go core, which cannot load on the JVM. The test frame therefore redraws only the
 * shell App() draws around a page: PageHeader or BackBar, the composer, and BottomNav.
 */
@RunWith(ParameterizedRobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class AndroidScreensTest(private val case: ScreenCase) {
    @get:Rule
    val compose = createComposeRule()

    @OptIn(ExperimentalRoborazziApi::class)
    @Test
    fun screenshot() {
        // The resend countdown and the busy indicator would run to their ends under an
        // auto-advancing clock; the pictures show their first frame.
        compose.mainClock.autoAdvance = case.screen !in frozen
        compose.setThemedContent(night = case.night, zh = case.zh) { Screen(case.screen) }
        compose.waitForIdle()
        val file = screenshotFile(case.name)
        if (case.screen in sheets) {
            // A bottom sheet is a second window, so only the screen capture sees it.
            captureScreenRoboImage(file)
        } else {
            compose.onRoot().captureRoboImage(file)
        }
    }

    data class ScreenCase(val name: String, val night: Boolean, val zh: Boolean, val screen: Screen)

    enum class Screen {
        MINE,
        AGENTS,
        PROJECTS,
        PROJECT,
        TASK,
        MINE_EMPTY,
        CALENDAR,
        SYNC_SIGNED_OUT,
        SYNC_SIGNED_OUT_ERROR,
        SYNC_SIGNING_IN_CHECKING,
        SYNC_SIGNING_IN_DOWNLOADING,
        SYNC_SIGNIN_EMAIL,
        SYNC_SIGNIN_GOOGLE_WAITING,
        SYNC_SIGNIN_CODE,
        SYNC_SIGNIN_WRONG_CODE,
        SYNC_SIGNIN_PASSWORD,
        SYNC_SIGNED_IN,
        SYNC_UPLOADING,
        SYNC_FAILED,
        SYNC_CHOOSE,
        TASK_MENU,
        TASK_MENU_MOVE,
        TASK_MENU_REMIND,
    }

    companion object {
        private val frozen = setOf(
            Screen.SYNC_SIGNIN_EMAIL, Screen.SYNC_SIGNIN_GOOGLE_WAITING, Screen.SYNC_SIGNIN_CODE, Screen.SYNC_SIGNIN_WRONG_CODE,
            Screen.SYNC_SIGNIN_PASSWORD, Screen.SYNC_SIGNING_IN_CHECKING, Screen.SYNC_SIGNING_IN_DOWNLOADING,
        )
        private val sheets = setOf(Screen.SYNC_CHOOSE, Screen.TASK_MENU, Screen.TASK_MENU_MOVE, Screen.TASK_MENU_REMIND)

        @JvmStatic
        @ParameterizedRobolectricTestRunner.Parameters(name = "{0}")
        fun cases(): List<ScreenCase> {
            val screens = listOf(
                "mine" to Screen.MINE,
                "agents" to Screen.AGENTS,
                "projects" to Screen.PROJECTS,
                "project" to Screen.PROJECT,
                "task" to Screen.TASK,
                "mine-empty" to Screen.MINE_EMPTY,
                "calendar" to Screen.CALENDAR,
                "sync-signed-out" to Screen.SYNC_SIGNED_OUT,
                "sync-signed-out-error" to Screen.SYNC_SIGNED_OUT_ERROR,
                "sync-signing-in-checking" to Screen.SYNC_SIGNING_IN_CHECKING,
                "sync-signing-in-downloading" to Screen.SYNC_SIGNING_IN_DOWNLOADING,
                "sync-signin-email" to Screen.SYNC_SIGNIN_EMAIL,
                "sync-signin-google-waiting" to Screen.SYNC_SIGNIN_GOOGLE_WAITING,
                "sync-signin-code" to Screen.SYNC_SIGNIN_CODE,
                "sync-signin-wrong-code" to Screen.SYNC_SIGNIN_WRONG_CODE,
                "sync-signin-password" to Screen.SYNC_SIGNIN_PASSWORD,
                "sync-signed-in" to Screen.SYNC_SIGNED_IN,
                "sync-uploading" to Screen.SYNC_UPLOADING,
                "sync-failed" to Screen.SYNC_FAILED,
                "sync-choose" to Screen.SYNC_CHOOSE,
                "task-menu" to Screen.TASK_MENU,
                "task-menu-move" to Screen.TASK_MENU_MOVE,
                "task-menu-remind" to Screen.TASK_MENU_REMIND,
            )
            val chinese = setOf(
                "mine", "task", "sync-signed-out", "sync-signin-email", "sync-signin-google-waiting", "sync-signin-code",
                "sync-signin-wrong-code", "sync-signin-password", "task-menu", "task-menu-remind")
            return screens.flatMap { (name, screen) ->
                buildList {
                    add(ScreenCase("$name-light", night = false, zh = false, screen = screen))
                    add(ScreenCase("$name-dark", night = true, zh = false, screen = screen))
                    if (name in chinese) {
                        add(ScreenCase("$name-light-zh", night = false, zh = true, screen = screen))
                        add(ScreenCase("$name-dark-zh", night = true, zh = true, screen = screen))
                    }
                }
            }
        }
    }
}

@Composable
private fun Screen(screen: AndroidScreensTest.Screen) {
    val resources = LocalContext.current.resources
    val words = remember(resources.configuration) { Words(resources, FixedTime.clock, FixedTime.zone) }
    val mine = remember { Golden.load<MinePage>("mine.golden.json") }
    val agents = remember { Golden.load<AgentsPage>("agents.golden.json") }
    val projects = remember { Golden.load<List<Project>>("projects.golden.json") }
    val project = remember { Golden.load<ProjectPage>("project-1.golden.json") }
    val task = remember { Golden.load<TaskPage>("task-1.golden.json") }
    val categories = remember { Golden.load<List<Category>>("categories.golden.json") }

    when (screen) {
        AndroidScreensTest.Screen.MINE -> TestFrame(
            title = stringResource(R.string.title_mine),
            subtitle = words.headerDate(),
            tab = Tab.MINE,
            count = mine.fromAgents.size,
            composer = stringResource(R.string.composer_placeholder_mine),
        ) {
            MineScreen(
                page = mine,
                agents = mineAgents(mine),
                words = words,
                onAccept = {},
                onComplete = {},
                onAnswer = {},
                onOpenUrl = {},
                onOpenTask = {},
                onHowToConnect = {},
                onComingSoon = {},
            )
        }

        AndroidScreensTest.Screen.AGENTS -> TestFrame(
            title = stringResource(R.string.title_agents),
            subtitle = resources.getString(R.string.agents_working, agents.working.size),
            tab = Tab.AGENTS,
            count = mine.fromAgents.size,
            composer = stringResource(R.string.composer_placeholder_agents),
        ) {
            AgentsScreen(
                page = agents,
                agents = agentInfo(agents),
                words = words,
                onOpenTask = {},
                onHowToConnect = {},
            )
        }

        AndroidScreensTest.Screen.PROJECTS -> TestFrame(
            title = stringResource(R.string.title_projects),
            tab = Tab.PROJECTS,
            count = mine.fromAgents.size,
            composer = stringResource(R.string.composer_placeholder_inbox),
        ) {
            ProjectsScreen(
                entries = projects.map {
                    ProjectEntry(project = it, open = openTasks(it.id, project))
                },
                onOpenProject = {},
                onHowToConnect = {},
            )
        }

        AndroidScreensTest.Screen.PROJECT -> TestFrame(
            title = stringResource(R.string.title_projects),
            tab = Tab.PROJECTS,
            count = mine.fromAgents.size,
            back = stringResource(R.string.title_projects),
            backRight = { SyncCloud(blocked = false, onSync = {}) },
            composer = stringResource(R.string.composer_add_to_project, project.project.name),
        ) {
            ProjectScreen(page = project, words = words, onOpenTask = {})
        }

        AndroidScreensTest.Screen.TASK -> TestFrame(
            title = stringResource(R.string.title_mine),
            tab = Tab.MINE,
            count = mine.fromAgents.size,
            topBar = {
                TaskBackBar(title = stringResource(R.string.title_mine), number = task.task.id, onBack = {}, onMenu = {})
            },
            showNav = false,
            composer = stringResource(R.string.composer_reply_to, words.agent(task.latestAgent)),
        ) {
            TaskScreen(
                page = task,
                projects = projects,
                words = words,
                onSetPriority = {},
                onOpenUrl = {},
                onOpenTask = {},
                categories = categories,
            )
        }

        AndroidScreensTest.Screen.TASK_MENU,
        AndroidScreensTest.Screen.TASK_MENU_MOVE,
        AndroidScreensTest.Screen.TASK_MENU_REMIND,
        -> {
            Screen(AndroidScreensTest.Screen.MINE)
            TaskMenuSheet(
                task = task.task,
                words = words,
                agent = task.latestAgent,
                categories = categories,
                projects = projects,
                presets = FixedTime.remindPresets,
                initialPage = when (screen) {
                    AndroidScreensTest.Screen.TASK_MENU_MOVE -> TaskMenuPage.Move
                    AndroidScreensTest.Screen.TASK_MENU_REMIND -> TaskMenuPage.Remind
                    else -> TaskMenuPage.Main
                },
                onDismiss = {},
            )
        }

        AndroidScreensTest.Screen.MINE_EMPTY -> TestFrame(
            title = stringResource(R.string.title_mine),
            subtitle = words.headerDate(),
            tab = Tab.MINE,
            count = 0,
            composer = stringResource(R.string.composer_placeholder_mine),
        ) {
            MineScreen(
                page = MinePage(),
                agents = emptyMap(),
                words = words,
                onAccept = {},
                onComplete = {},
                onAnswer = {},
                onOpenUrl = {},
                onOpenTask = {},
                onHowToConnect = {},
                onComingSoon = {},
            )
        }

        AndroidScreensTest.Screen.CALENDAR -> TestFrame(
            title = stringResource(R.string.title_calendar),
            tab = Tab.CALENDAR,
            count = mine.fromAgents.size,
            composer = stringResource(R.string.composer_placeholder_inbox),
        ) {
            Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
                PlaceholderScreen(
                    icon = "calendar",
                    title = stringResource(R.string.calendar_placeholder_title),
                    body = stringResource(R.string.calendar_placeholder_body),
                )
            }
        }

        AndroidScreensTest.Screen.SYNC_SIGNED_OUT,
        AndroidScreensTest.Screen.SYNC_SIGNIN_EMAIL,
        AndroidScreensTest.Screen.SYNC_SIGNIN_GOOGLE_WAITING,
        AndroidScreensTest.Screen.SYNC_SIGNIN_CODE,
        AndroidScreensTest.Screen.SYNC_SIGNIN_WRONG_CODE,
        AndroidScreensTest.Screen.SYNC_SIGNIN_PASSWORD,
        AndroidScreensTest.Screen.SYNC_SIGNED_OUT_ERROR,
        AndroidScreensTest.Screen.SYNC_SIGNING_IN_CHECKING,
        AndroidScreensTest.Screen.SYNC_SIGNING_IN_DOWNLOADING,
        AndroidScreensTest.Screen.SYNC_SIGNED_IN,
        AndroidScreensTest.Screen.SYNC_UPLOADING,
        AndroidScreensTest.Screen.SYNC_FAILED,
        AndroidScreensTest.Screen.SYNC_CHOOSE,
        -> TestFrame(
            title = stringResource(R.string.sync_title),
            tab = Tab.MINE,
            back = stringResource(R.string.sync_title),
            showNav = false,
            composer = null,
        ) {
            Column(
                Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = Dims.pagePadding),
            ) {
                SyncScreen(
                    model = syncModel(screen),
                    words = words,
                    serverEditing = false,
                    serverDraft = "",
                    serverError = false,
                    serverSaving = false,
                    onSignIn = { _, _ -> },
                    onSignOut = {},
                    onSyncNow = {},
                    onRetry = {},
                    onMerge = {},
                    onReplace = {},
                    onCancelChoose = {},
                    onStartServerEdit = {},
                    onServerDraft = {},
                    onSaveServer = {},
                    onCancelServerEdit = {},
                )
            }
        }
    }
}

/** The shell App draws around a page, without the core-backed routing. */
@Composable
private fun TestFrame(
    title: String,
    tab: Tab,
    count: Int = 0,
    subtitle: String = "",
    back: String? = null,
    backRight: @Composable RowScope.() -> Unit = {},
    topBar: (@Composable () -> Unit)? = null,
    showNav: Boolean = true,
    composer: String? = null,
    content: @Composable () -> Unit,
) {
    val tokens = LocalTokens.current
    Column(Modifier.fillMaxSize().background(tokens.bg)) {
        Column(Modifier.weight(1f).statusBarsPadding().imePadding()) {
            if (topBar != null) {
                topBar()
            } else if (back == null) {
                PageHeader(title = title, subtitle = subtitle, syncBlocked = false, onSync = {})
            } else {
                BackBar(title = back, onBack = {}, right = backRight)
            }
            Box(Modifier.weight(1f)) { content() }
            if (composer != null) {
                Composer(
                    value = "",
                    placeholder = composer,
                    notice = null,
                    sending = false,
                    onValueChange = {},
                    onSend = {},
                )
            }
        }
        if (showNav) BottomNav(tab = tab, count = count, onSelect = {})
    }
}

private fun mineAgents(page: MinePage): Map<Long, AgentInfo> =
    page.fromAgents.associate { item ->
        val links = if (item.prUrl.isNotEmpty()) listOf(Link(item.prUrl, "PR #${item.prNumber}")) else emptyList()
        item.id to AgentInfo("claude-code", links)
    }

private fun openTasks(projectId: Long, page: ProjectPage): Int =
    if (page.project.id != projectId) {
        0
    } else {
        page.tasks.count {
            it.parentId == null && it.source != "agent" && it.status != "done" && it.status != "cancelled"
        }
    }

private fun agentInfo(page: AgentsPage): Map<Long, AgentInfo> =
    (page.working + page.done).associate { it.id to AgentInfo("claude-code", emptyList()) }

private fun syncModel(screen: AndroidScreensTest.Screen): SyncModel {
    val now = FixedTime.now
    val devices = Golden.load<List<Device>>("devices.golden.json")
    val account = Golden.load<Account>("account.golden.json")
    val base = SyncStatus(
        state = "idle",
        lastSynced = now.minus(3, ChronoUnit.MINUTES).toString(),
        server = DEFAULT_SYNC_SERVER,
        linked = true,
        signedIn = true,
    )
    return when (screen) {
        AndroidScreensTest.Screen.SYNC_SIGNED_OUT -> SyncModel(
            SyncPhase.SIGNED_OUT,
            SyncStatus(server = DEFAULT_SYNC_SERVER),
            flow = SignInFlow(step = SignInStep.LEGACY),
        )
        AndroidScreensTest.Screen.SYNC_SIGNIN_EMAIL -> SyncModel(
            SyncPhase.SIGNED_OUT,
            SyncStatus(server = DEFAULT_SYNC_SERVER),
            flow = SignInFlow(email = "cyan@example.com"),
            googleAvailable = true,
        )
        AndroidScreensTest.Screen.SYNC_SIGNIN_GOOGLE_WAITING -> SyncModel(
            SyncPhase.SIGNING_IN,
            SyncStatus(server = DEFAULT_SYNC_SERVER, signedIn = true, signInStage = "google"),
            flow = SignInFlow(email = "cyan@example.com"),
            googleAvailable = true,
        )
        AndroidScreensTest.Screen.SYNC_SIGNIN_CODE -> SyncModel(
            SyncPhase.SIGNED_OUT,
            SyncStatus(server = DEFAULT_SYNC_SERVER),
            flow = SignInFlow(step = SignInStep.CODE, email = "cyan@example.com", resendIn = 24),
            typedCode = "48",
        )
        AndroidScreensTest.Screen.SYNC_SIGNIN_WRONG_CODE -> SyncModel(
            SyncPhase.SIGNED_OUT,
            SyncStatus(server = DEFAULT_SYNC_SERVER),
            flow = SignInFlow(
                step = SignInStep.CODE,
                email = "cyan@example.com",
                failure = CodeFailure.WRONG_CODE,
                attempt = 1,
            ),
        )
        AndroidScreensTest.Screen.SYNC_SIGNIN_PASSWORD -> SyncModel(
            SyncPhase.SIGNED_OUT,
            SyncStatus(server = DEFAULT_SYNC_SERVER),
            flow = SignInFlow(step = SignInStep.PASSWORD, email = "cyan@example.com"),
        )
        AndroidScreensTest.Screen.SYNC_SIGNED_OUT_ERROR -> SyncModel(
            SyncPhase.SIGNED_OUT,
            SyncStatus(server = DEFAULT_SYNC_SERVER),
            flow = SignInFlow(step = SignInStep.LEGACY),
            signInError = "Wrong account or password",
            signInFailure = xyz.sriai.adhdtodo.ui.SignInFailure.CREDENTIALS,
        )
        AndroidScreensTest.Screen.SYNC_SIGNING_IN_CHECKING -> SyncModel(
            SyncPhase.SIGNING_IN,
            SyncStatus(server = DEFAULT_SYNC_SERVER, signedIn = true, signInStage = "checking"),
            flow = SignInFlow(step = SignInStep.CODE, email = "cyan@example.com"),
            typedCode = "482913",
        )
        AndroidScreensTest.Screen.SYNC_SIGNING_IN_DOWNLOADING -> SyncModel(
            SyncPhase.SIGNING_IN,
            SyncStatus(server = DEFAULT_SYNC_SERVER, signedIn = true, signInStage = "downloading"),
            flow = SignInFlow(step = SignInStep.CODE, email = "cyan@example.com"),
            typedCode = "482913",
        )
        AndroidScreensTest.Screen.SYNC_SIGNED_IN -> SyncModel(SyncPhase.SYNCED, base, account, devices)
        AndroidScreensTest.Screen.SYNC_UPLOADING -> SyncModel(
            SyncPhase.UPLOADING,
            base.copy(state = "seeding", lastSynced = null, seedTotal = 3512, pending = 2272),
            account,
            devices,
        )
        AndroidScreensTest.Screen.SYNC_FAILED -> SyncModel(
            SyncPhase.FAILED,
            base.copy(
                state = "error",
                lastError = "the server did not answer",
                retryAt = now.plus(2, ChronoUnit.MINUTES).toString(),
                failedSince = now.minus(20, ChronoUnit.MINUTES).toString(),
                pending = 3,
            ),
            account,
            devices,
        )
        AndroidScreensTest.Screen.SYNC_CHOOSE -> SyncModel(
            SyncPhase.CHOOSE,
            base.copy(linked = false, signedIn = true),
            account,
            devices.filterNot { it.isThis },
            choose = SignInState(state = "choose", localTasks = 128, localProjects = 6, accountDevices = 2),
        )
        else -> error("not a sync screen: $screen")
    }
}
