package xyz.sriai.adhdtodo

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import com.github.takahirom.roborazzi.captureRoboImage
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.ParameterizedRobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.CompletedPage
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.testing.screenshotFile
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.BackBar
import xyz.sriai.adhdtodo.ui.CategoryChips
import xyz.sriai.adhdtodo.ui.CompletedScreen
import xyz.sriai.adhdtodo.ui.MineScreen
import xyz.sriai.adhdtodo.ui.PageHeader
import xyz.sriai.adhdtodo.ui.SearchResults
import xyz.sriai.adhdtodo.ui.SearchScreen
import xyz.sriai.adhdtodo.ui.Tab
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.filterMine
import xyz.sriai.adhdtodo.ui.groupSearch
import xyz.sriai.adhdtodo.ui.repoProjectIds
import xyz.sriai.adhdtodo.ui.todayChips
import xyz.sriai.adhdtodo.ui.workCategoryId
import xyz.sriai.adhdtodo.ui.theme.LocalTokens

/**
 * The Find group's pictures (v43): Search with results and with none, Today narrowed by a category
 * chip, and Completed. They share AndroidScreensTest's frame, so the shell reads the same.
 */
@RunWith(ParameterizedRobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class ScreensFindTest(private val case: Case) {
    @get:Rule
    val compose = createComposeRule()

    @Test
    fun screenshot() {
        compose.setThemedContent(night = case.night, zh = case.zh) { FindScreen(case.screen) }
        compose.waitForIdle()
        compose.onRoot().captureRoboImage(screenshotFile(case.name))
    }

    data class Case(val name: String, val night: Boolean, val zh: Boolean, val screen: Screen)

    enum class Screen { SEARCH_RESULTS, SEARCH_NONE, MINE_CHIPS, COMPLETED }

    companion object {
        @JvmStatic
        @ParameterizedRobolectricTestRunner.Parameters(name = "{0}")
        fun cases(): List<Case> {
            val screens = listOf(
                "search-results" to Screen.SEARCH_RESULTS,
                "search-none" to Screen.SEARCH_NONE,
                "mine-chips" to Screen.MINE_CHIPS,
                "completed" to Screen.COMPLETED,
            )
            val chinese = setOf("search-results", "mine-chips", "completed")
            return screens.flatMap { (name, screen) ->
                buildList {
                    add(Case("$name-light", night = false, zh = false, screen = screen))
                    add(Case("$name-dark", night = true, zh = false, screen = screen))
                    if (name in chinese) {
                        add(Case("$name-light-zh", night = false, zh = true, screen = screen))
                        add(Case("$name-dark-zh", night = true, zh = true, screen = screen))
                    }
                }
            }
        }
    }
}

private fun searchTask(
    id: Long,
    title: String,
    status: String,
    project: Long?,
    branch: String = "",
    pr: Int = 0,
    updated: String,
): Task = Task(
    id = id, projectId = project, title = title, status = status, branch = branch, prNumber = pr,
    source = "agent", updatedAt = updated,
)

/** What a search for `login` finds in the fixture: three tasks of withdo and one with no project. */
private fun loginResults(projects: List<Project>): SearchResults {
    val withdo = projects.first { it.name == "withdo" }.id
    return groupSearch(
        query = "login",
        tasks = listOf(
            searchTask(41, "Fix the login redirect", "doing", withdo, branch = "fix/login", updated = "2026-10-20T08:58:00Z"),
            searchTask(42, "Google login on Android", "review", withdo, pr = 171, updated = "2026-10-19T08:00:00Z"),
            searchTask(43, "Port the login screen", "done", withdo, updated = "2026-10-10T08:00:00Z"),
            searchTask(44, "Change the bank login password", "todo", null, updated = "2026-10-17T08:00:00Z")
                .copy(source = "user"),
        ),
        projects = projects,
    )
}

@Composable
private fun FindScreen(screen: ScreensFindTest.Screen) {
    val resources = LocalContext.current.resources
    val tokens = LocalTokens.current
    val words = remember { Words(resources, FixedTime.clock, FixedTime.zone) }
    val mine = remember { Golden.load<MinePage>("mine.golden.json") }
    val projects = remember { Golden.load<List<Project>>("projects.golden.json") }
    val categories = remember { Golden.load<List<Category>>("categories.golden.json") }

    when (screen) {
        ScreensFindTest.Screen.SEARCH_RESULTS, ScreensFindTest.Screen.SEARCH_NONE -> {
            val found = screen == ScreensFindTest.Screen.SEARCH_RESULTS
            Column(Modifier.fillMaxSize().background(tokens.bg).statusBarsPadding()) {
                SearchScreen(
                    query = if (found) "login" else "xyz",
                    onQueryChange = {},
                    results = if (found) loginResults(projects) else SearchResults("xyz", emptyList(), 0, false),
                    words = words,
                    onCancel = {},
                    onOpenTask = {},
                    onLongPress = {},
                    onToggle = {},
                    focusOnStart = false,
                )
            }
        }

        ScreensFindTest.Screen.MINE_CHIPS -> {
            val repos = repoProjectIds(projects)
            val workId = workCategoryId(categories)
            val chips = todayChips(mine, categories, repos, workId)
            TestFrame(
                title = words.headerDate(),
                tab = Tab.MINE,
                count = mine.fromAgents.size,
                composer = stringResource(R.string.composer_placeholder_mine),
                topBar = {
                    PageHeader(
                        title = words.headerDate(),
                        subtitle = "",
                        syncBlocked = false,
                        onSync = {},
                        start = 28.dp,
                        onSearch = {},
                    )
                    CategoryChips(chips = chips, selected = workId, words = words, onSelect = {})
                },
            ) {
                MineScreen(
                    page = filterMine(mine, workId, repos, workId),
                    agents = mineAgents(mine),
                    words = words,
                    onAccept = {},
                    onComplete = {},
                    onAnswer = {},
                    onOpenUrl = {},
                    onOpenTask = {},
                    onHowToConnect = {},
                    onViewAll = null,
                    categories = categories.filterNot { it.id == workId },
                    onShowAll = {},
                )
            }
        }

        ScreensFindTest.Screen.COMPLETED -> {
            val page = remember { Golden.load<CompletedPage>("completed.golden.json") }
            TestFrame(
                title = stringResource(R.string.title_mine),
                tab = Tab.MINE,
                count = mine.fromAgents.size,
                topBar = { BackBar(title = stringResource(R.string.title_mine), onBack = {}) },
            ) {
                CompletedScreen(
                    page = page,
                    subtitle = stringResource(R.string.completed_subtitle),
                    words = words,
                    onOpenTask = {},
                    onLongPress = {},
                    onReopen = {},
                    onShowOlder = {},
                )
            }
        }
    }
}
