package xyz.sriai.adhdtodo

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import androidx.compose.ui.unit.dp
import com.github.takahirom.roborazzi.captureRoboImage
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.ParameterizedRobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.Account
import xyz.sriai.adhdtodo.data.AssistantDetail
import xyz.sriai.adhdtodo.data.AssistantQuestion
import xyz.sriai.adhdtodo.data.AssistantReceipt
import xyz.sriai.adhdtodo.data.AssistantReply
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.Device
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.ModelPhase
import xyz.sriai.adhdtodo.data.ModelState
import xyz.sriai.adhdtodo.data.QuickDraft
import xyz.sriai.adhdtodo.data.QuickParse
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.data.UpdatePhase
import xyz.sriai.adhdtodo.data.UpdateUi
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.testing.screenshotFile
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.AnswerCard
import xyz.sriai.adhdtodo.ui.AssistantAbove
import xyz.sriai.adhdtodo.ui.BottomNav
import xyz.sriai.adhdtodo.ui.Composer
import xyz.sriai.adhdtodo.ui.MineScreen
import xyz.sriai.adhdtodo.ui.ModelChip
import xyz.sriai.adhdtodo.ui.PageHeader
import xyz.sriai.adhdtodo.ui.QuietNote
import xyz.sriai.adhdtodo.ui.SettingsData
import xyz.sriai.adhdtodo.ui.SettingsScreen
import xyz.sriai.adhdtodo.ui.Tab
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.answerCard
import xyz.sriai.adhdtodo.ui.localCard
import xyz.sriai.adhdtodo.ui.theme.AppearanceMode
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import java.time.Instant
import java.time.ZoneId

/**
 * The composer's assistant (v44 section two) as full phone pictures, with fake replies and no
 * network: the answer with its receipts, a question, thinking, a plain answer, the quota used up,
 * signed out, and Settings' Assistant group. Today is Tuesday 3 November 2026, as in
 * docs/design/mockups/v44.html. Render with
 * `scripts/android-screens.sh 'xyz.sriai.adhdtodo.ScreensAssistantTest'`.
 */
@RunWith(ParameterizedRobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class ScreensAssistantTest(private val case: Case) {
    @get:Rule
    val compose = createComposeRule()

    @Test
    fun screenshot() {
        // The spinner turns forever; the picture is its first frame.
        compose.mainClock.autoAdvance = false
        compose.setThemedContent(night = case.night, zh = case.zh) { Picture(case.shot) }
        compose.waitForIdle()
        compose.onRoot().captureRoboImage(screenshotFile(case.name))
    }

    data class Case(val name: String, val night: Boolean, val zh: Boolean, val shot: Shot)

    enum class Shot { ANSWER_RECEIPT, ANSWER_QUESTION, THINKING, ANSWER_PLAIN, QUOTA_USED_UP, SIGNED_OUT, SETTINGS }

    companion object {
        @JvmStatic
        @ParameterizedRobolectricTestRunner.Parameters(name = "{0}")
        fun cases(): List<Case> = listOf(
            "answer-receipt" to Shot.ANSWER_RECEIPT,
            "answer-question" to Shot.ANSWER_QUESTION,
            "thinking" to Shot.THINKING,
            "answer-plain" to Shot.ANSWER_PLAIN,
            "quota-used-up" to Shot.QUOTA_USED_UP,
            "signed-out" to Shot.SIGNED_OUT,
            "settings-assistant" to Shot.SETTINGS,
        ).flatMap { (name, shot) ->
            listOf(
                Case("$name-light", night = false, zh = false, shot = shot),
                Case("$name-dark", night = true, zh = false, shot = shot),
                Case("$name-light-zh", night = false, zh = true, shot = shot),
                Case("$name-dark-zh", night = true, zh = true, shot = shot),
            )
        }
    }
}

/** Tuesday 3 November 2026, 9:41 in Shanghai. */
private val Tuesday: Instant = Instant.parse("2026-11-03T01:41:00Z")
private val Shanghai: ZoneId = ZoneId.of("Asia/Shanghai")

private fun receipt(id: Long, taskId: Long, title: String, detail: AssistantDetail, bell: Boolean = false) =
    AssistantReceipt(id = id, taskId = taskId, title = title, status = "todo", bell = bell, detail = detail)

private val callTheLandlord = receipt(
    1, 2, "打给房东", AssistantDetail(kind = "reminder", at = "2026-11-04T09:00:00+08:00"), bell = true,
)

@Composable
private fun Picture(shot: ScreensAssistantTest.Shot) {
    val resources = LocalContext.current.resources
    val configuration = LocalConfiguration.current
    val words = remember(configuration) { Words(resources, { Tuesday }, Shanghai) }
    val categories = remember { Golden.load<List<Category>>("categories.golden.json") }
    val zh = configuration.locales[0].language == "zh"
    if (shot == ScreensAssistantTest.Shot.SETTINGS) {
        SettingsPicture(ModelState("official", ModelPhase.READY, left = 86, limit = 100))
        return
    }
    val model = when (shot) {
        ScreensAssistantTest.Shot.QUOTA_USED_UP -> ModelState("none", ModelPhase.USED_UP, left = 0, limit = 100)
        ScreensAssistantTest.Shot.SIGNED_OUT -> ModelState("none", ModelPhase.SIGNED_OUT)
        else -> ModelState("official", ModelPhase.READY, left = 86, limit = 100)
    }
    val card: AnswerCard? = when (shot) {
        ScreensAssistantTest.Shot.ANSWER_RECEIPT -> answerCard(
            resources, words,
            AssistantReply(
                receipts = listOf(
                    callTheLandlord,
                    receipt(2, 7, "Fix the login bug", AssistantDetail(kind = "added", project = "withdo")),
                ),
            ),
        )
        ScreensAssistantTest.Shot.ANSWER_QUESTION -> AnswerCard(question = questionFor(zh))
        ScreensAssistantTest.Shot.ANSWER_PLAIN -> AnswerCard(text = plainAnswer(zh))
        ScreensAssistantTest.Shot.QUOTA_USED_UP, ScreensAssistantTest.Shot.SIGNED_OUT -> localCard(
            resources, words,
            QuickParse(listOf(QuickDraft(title = "打给房东", remindAt = "2026-11-04T09:00:00+08:00", ranges = listOf(listOf(0, 5))))),
            titles = listOf("打给房东"), ids = listOf(2),
            note = if (shot == ScreensAssistantTest.Shot.SIGNED_OUT) QuietNote.SIGNED_OUT else QuietNote.USED_UP,
        )
        else -> null
    }
    val thinking = shot == ScreensAssistantTest.Shot.THINKING
    val draft = when {
        thinking -> if (zh) "这周 Claude 做完了什么？" else "What did Claude finish this week?"
        shot == ScreensAssistantTest.Shot.ANSWER_QUESTION -> ""
        else -> if (zh) "明天 9 点打给房东，再让 Claude 修 bug" else "Call the landlord at 9 tomorrow, have Claude fix the bug"
    }
    AssistantFrame(
        words = words,
        categories = categories,
        card = card,
        thinking = thinking,
        model = model,
        draft = draft,
        placeholder = if (shot == ScreensAssistantTest.Shot.ANSWER_QUESTION) stringResource(R.string.assistant_answer_hint) else stringResource(R.string.composer_placeholder_mine),
    )
}

private fun questionFor(zh: Boolean) = if (zh) {
    AssistantQuestion("“修 bug”是哪个项目的？", listOf("withdo", "picpocket", "answer-me-with-html"))
} else {
    AssistantQuestion("Which project is “fix the bug” for?", listOf("withdo", "picpocket", "answer-me-with-html"))
}

private fun plainAnswer(zh: Boolean) = if (zh) {
    "这周 Claude Code 做完 5 件：MCP 服务器、原生界面、同步服务器部署，还有两个小修复。原生界面的 PR #12 还在等你审。"
} else {
    "Claude Code finished 5 things this week: the MCP server, the native UI, the sync server deploy and two small fixes. The native UI's PR #12 is waiting for your review."
}

/** Today at 9:41 with its three rows, and the composer with whatever the assistant put around it. */
@Composable
private fun AssistantFrame(
    words: Words,
    categories: List<Category>,
    card: AnswerCard?,
    thinking: Boolean,
    model: ModelState,
    draft: String,
    placeholder: String,
) {
    val tokens = LocalTokens.current
    val work = categories.first { it.builtin == "work" }.id.toInt()
    val page = MinePage(
        today = listOf(
            Item(id = 1, title = "Reply to design feedback", status = "todo", source = "user", categoryId = work, remindAt = "2026-11-03T10:00:00+08:00"),
            Item(id = 2, title = "打给房东", status = "todo", source = "user", remindAt = "2026-11-04T09:00:00+08:00"),
            Item(id = 3, title = "Decide on the activation policy", status = "doing", source = "agent", project = "withdo", dueAt = "2026-11-03T17:00:00+08:00"),
        ),
    )
    Column(Modifier.fillMaxSize().background(tokens.bg)) {
        Column(Modifier.weight(1f).statusBarsPadding().imePadding()) {
            PageHeader(title = words.headerDate(), subtitle = "", syncBlocked = false, onSync = {}, start = 28.dp)
            Box(Modifier.weight(1f)) {
                MineScreen(
                    page = page,
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
                    highlightId = if (card != null && card.rows.isNotEmpty()) 2 else 0,
                )
            }
            Composer(
                value = draft,
                placeholder = placeholder,
                notice = null,
                sending = thinking,
                onValueChange = {},
                onSend = {},
                onStop = if (thinking) ({}) else null,
                thinking = thinking,
                above = card?.let { c -> { AssistantAbove(c, onUndo = {}, onPick = {}, onSignIn = {}) } },
                chip = if (model.official) ({ ModelChip {} }) else null,
            )
        }
        BottomNav(tab = Tab.MINE, count = 2, onSelect = {})
    }
}

/** Settings with the Assistant group under the account (the pushed page, over Projects' bar). */
@Composable
private fun SettingsPicture(model: ModelState) {
    val tokens = LocalTokens.current
    Column(Modifier.fillMaxSize().background(tokens.bg)) {
        Column(Modifier.weight(1f).statusBarsPadding()) {
            xyz.sriai.adhdtodo.ui.BackBar(title = stringResource(R.string.title_projects), onBack = {})
            Box(Modifier.weight(1f)) {
                SettingsScreen(
                    data = SettingsData(
                        account = Golden.load<Account>("account.golden.json").copy(email = "cyan@example.com", name = ""),
                        devices = Golden.load<List<Device>>("devices.golden.json").take(3),
                        model = model,
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
        }
        BottomNav(tab = Tab.PROJECTS, count = 2, onSelect = {})
    }
}
