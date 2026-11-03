package xyz.sriai.adhdtodo

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.dp
import com.github.takahirom.roborazzi.captureRoboImage
import com.github.takahirom.roborazzi.captureScreenRoboImage
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.ParameterizedRobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.testing.screenshotFile
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.MineScreen
import xyz.sriai.adhdtodo.ui.PageHeader
import xyz.sriai.adhdtodo.ui.PullToSync
import xyz.sriai.adhdtodo.ui.ShareForm
import xyz.sriai.adhdtodo.ui.ShareSheet
import xyz.sriai.adhdtodo.ui.Tab
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import java.time.LocalDate

/**
 * The phone's own pictures (v43 P2): rows mid-swipe both ways, a list pulled open while it syncs,
 * and the share sheet over another app. They share AndroidScreensTest's frame.
 */
@RunWith(ParameterizedRobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class ScreensPhoneTest(private val case: Case) {
    @get:Rule
    val compose = createComposeRule()

    @Test
    fun screenshot() {
        // The syncing cloud bobs forever; the picture is its first frame.
        compose.mainClock.autoAdvance = case.screen != Screen.PULL
        compose.setThemedContent(night = case.night, zh = case.zh) { PhoneScreen(case.screen) }
        compose.waitForIdle()
        val file = screenshotFile(case.name)
        if (case.screen == Screen.SHARE) captureScreenRoboImage(file) else compose.onRoot().captureRoboImage(file)
    }

    data class Case(val name: String, val night: Boolean, val zh: Boolean, val screen: Screen)

    enum class Screen { SWIPE, PULL, SHARE }

    companion object {
        @JvmStatic
        @ParameterizedRobolectricTestRunner.Parameters(name = "{0}")
        fun cases(): List<Case> = listOf("swipe" to Screen.SWIPE, "pull-sync" to Screen.PULL, "share" to Screen.SHARE)
            .flatMap { (name, screen) ->
                listOf(
                    Case("$name-light", night = false, zh = false, screen = screen),
                    Case("$name-dark", night = true, zh = false, screen = screen),
                    Case("$name-light-zh", night = false, zh = true, screen = screen),
                    Case("$name-dark-zh", night = true, zh = true, screen = screen),
                )
            }
    }
}

@Composable
private fun PhoneScreen(screen: ScreensPhoneTest.Screen) {
    val resources = LocalContext.current.resources
    val words = remember { Words(resources, FixedTime.clock, FixedTime.zone) }
    val mine = remember { Golden.load<MinePage>("mine.golden.json") }
    val categories = remember { Golden.load<List<Category>>("categories.golden.json") }
    when (screen) {
        // Book dentist goes right (Done); the release notes and the activation policy, due today
        // already, go left (Do today, Do tomorrow).
        ScreensPhoneTest.Screen.SWIPE -> TodayFrame(words, mine) {
            MineScreen(
                page = mine,
                agents = mineAgents(mine),
                words = words,
                onAccept = {}, onComplete = {}, onAnswer = {}, onOpenUrl = {}, onOpenTask = {}, onHowToConnect = {},
                onViewAll = null,
                categories = categories,
                onShowAll = {},
                swipePreview = mapOf(8L to 0.3f, 5L to -0.3f, 6L to -0.42f),
            )
        }

        ScreensPhoneTest.Screen.PULL -> TodayFrame(words, mine) {
            PullToSync(enabled = true, syncing = false, onSync = {}, preview = true) {
                MineScreen(
                    page = mine,
                    agents = mineAgents(mine),
                    words = words,
                    onAccept = {}, onComplete = {}, onAnswer = {}, onOpenUrl = {}, onOpenTask = {}, onHowToConnect = {},
                    onViewAll = null,
                    categories = categories,
                    onShowAll = {},
                )
            }
        }

        ScreensPhoneTest.Screen.SHARE -> {
            val projects = remember { Golden.load<List<Project>>("projects.golden.json") }
            val title = "Android: swipe to complete a task"
            OtherApp()
            ShareSheet(
                form = ShareForm(
                    title = TextFieldValue(title, TextRange(title.length)),
                    url = "https://github.com/QingYunA/withdo/issues/219",
                    projectId = projects.firstOrNull { it.name == "withdo" }?.id,
                ),
                projects = projects,
                read = null,
                today = LocalDate.of(2026, 10, 20),
                onChange = {},
                onAdd = {},
                onDismiss = {},
            )
        }
    }
}

@Composable
private fun TodayFrame(words: Words, mine: MinePage, content: @Composable () -> Unit) {
    TestFrame(
        title = words.headerDate(),
        tab = Tab.MINE,
        count = mine.fromAgents.size,
        composer = stringResource(R.string.composer_placeholder_mine),
        topBar = {
            PageHeader(title = words.headerDate(), subtitle = "", syncBlocked = false, onSync = {}, start = 28.dp, onSearch = {})
        },
        content = content,
    )
}

/** The app that shared: a browser's address bar and the grey blocks of a page. */
@Composable
private fun OtherApp() {
    val tokens = LocalTokens.current
    Column(
        Modifier.fillMaxSize().background(tokens.bgSidebar).statusBarsPadding().padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        Box(Modifier.fillMaxWidth().height(40.dp).background(tokens.bgSubtle, RoundedCornerShape(20.dp)).padding(horizontal = 16.dp, vertical = 10.dp)) {
            Text("github.com/QingYunA/withdo/issues/219", style = LocalAppType.current.meta, color = tokens.textSecondary)
        }
        for (width in listOf(0.6f, 1f, 0.8f, 0.7f, 0.9f)) {
            Box(Modifier.fillMaxWidth(width).height(16.dp).background(tokens.bgSelected, RoundedCornerShape(4.dp)))
        }
    }
}
