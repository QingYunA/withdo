package xyz.sriai.adhdtodo

import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.unit.dp
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import com.github.takahirom.roborazzi.ExperimentalRoborazziApi
import com.github.takahirom.roborazzi.captureRoboImage
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.ParameterizedRobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.AgentInfo
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.Link
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.SummaryGood
import xyz.sriai.adhdtodo.data.SummaryOpen
import xyz.sriai.adhdtodo.data.SummaryPage
import xyz.sriai.adhdtodo.data.SummaryPerson
import xyz.sriai.adhdtodo.data.SummaryTask
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.ui.MineScreen
import xyz.sriai.adhdtodo.ui.PageHeader
import xyz.sriai.adhdtodo.testing.screenshotFile
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.SummaryHolder
import xyz.sriai.adhdtodo.ui.SummaryScreen
import xyz.sriai.adhdtodo.ui.Tab
import xyz.sriai.adhdtodo.ui.Words
import java.time.ZonedDateTime

/**
 * The Summary page of the phone (v43) as full phone pictures, light and dark, English and Chinese.
 * The scene is drawn at its last frame (the page's "remove animations" frame), so the pictures
 * do not move from run to run. Run with `scripts/android-screens.sh 'xyz.sriai.adhdtodo.ScreensSummaryTest'`.
 */
@RunWith(ParameterizedRobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class ScreensSummaryTest(private val case: Case) {
    @get:Rule
    val compose = createComposeRule()

    @OptIn(ExperimentalRoborazziApi::class)
    @Test
    fun screenshot() {
        compose.setThemedContent(night = case.night, zh = case.zh) { Frame(case.kind, case.zh) }
        compose.waitForIdle()
        compose.onRoot().captureRoboImage(screenshotFile(case.name))
    }

    data class Case(val name: String, val night: Boolean, val zh: Boolean, val kind: String)

    companion object {
        @JvmStatic
        @ParameterizedRobolectricTestRunner.Parameters(name = "{0}")
        fun cases(): List<Case> = listOf("summary-day", "summary-week", "summary-empty", "summary-entry", "summary-throw").flatMap { name ->
            listOf(
                Case("$name-light", night = false, zh = false, kind = name),
                Case("$name-dark", night = true, zh = false, kind = name),
                Case("$name-light-zh", night = false, zh = true, kind = name),
                Case("$name-dark-zh", night = true, zh = true, kind = name),
            )
        }
    }
}

@Composable
private fun Frame(kind: String, zh: Boolean) {
    val configuration = LocalConfiguration.current
    val resources = LocalContext.current.resources
    val words = remember(configuration) { Words(resources, FixedTime.clock, FixedTime.zone) }
    if (kind == "summary-entry") {
        // Today, with `See summary ›` ending the Done today header.
        val mine = remember { Golden.load<MinePage>("mine.golden.json") }
        val categories = remember { Golden.load<List<Category>>("categories.golden.json") }
        TestFrame(title = words.headerDate(), tab = Tab.MINE, count = mine.fromAgents.size, headerStart = 28.dp, composer = stringResource(R.string.composer_placeholder_mine)) {
            MineScreen(
                page = mine,
                agents = mine.fromAgents.associate { item ->
                    val links = if (item.prUrl.isNotEmpty()) listOf(Link(item.prUrl, "PR #${item.prNumber}")) else emptyList()
                    item.id to AgentInfo("claude-code", links)
                },
                words = words,
                onAccept = {}, onComplete = {}, onAnswer = {}, onOpenUrl = {}, onOpenTask = {}, onHowToConnect = {},
                onViewAll = null,
                categories = categories,
                onSummary = {},
            )
        }
        return
    }
    val holder = remember(kind, zh) {
        SummaryHolder().apply {
            week = kind == "summary-week"
            page = when (kind) {
                "summary-day", "summary-throw" -> dayPage(zh)
                "summary-week" -> weekPage(zh)
                else -> emptyPage()
            }
        }
    }
    TestFrame(
        title = stringResource(R.string.title_mine),
        tab = Tab.MINE,
        count = 2,
        back = stringResource(R.string.title_mine),
    ) {
        SummaryScreen(
            holder = holder,
            words = words,
            onOpenTask = {},
            onShare = {},
            zone = FixedTime.zone,
            now = ZonedDateTime.ofInstant(FixedTime.now, FixedTime.zone).withHour(10),
            // The throw picture is the scene 1.7 s in, the third good in the air.
            calm = kind != "summary-throw",
            frozenMs = if (kind == "summary-throw") 1700 else null,
        )
    }
}

private fun task(id: Long, title: String, at: String, kind: String, project: String = "", pr: Int = 0, merged: Boolean = false, said: String = "") =
    SummaryTask(id = id, title = title, project = project, at = at, kind = kind, merged = merged, prNumber = pr, said = said)

private fun person(agent: String, vararg tasks: SummaryTask) = SummaryPerson(
    agent = agent, name = agent.ifEmpty { "you" }, you = agent.isEmpty(), count = tasks.size,
    merged = tasks.count { it.merged }, tasks = tasks.toList(),
)

/** The mockup's day: you answered, replied and booked; Claude Code merged two and Codex one. */
internal fun dayPage(zh: Boolean): SummaryPage {
    val mine = person(
        "",
        task(1, if (zh) "回复打包的问题" else "Send the packaging answer", "2026-10-20T06:00:00Z", "mail"),
        task(2, if (zh) "回复设计反馈" else "Reply to design feedback", "2026-10-20T03:10:00Z", "mail"),
        task(3, if (zh) "预约牙医" else "Book the dentist", "2026-10-20T01:30:00Z", "scroll"),
    )
    val claude = person(
        "Claude Code",
        task(4, if (zh) "移植原生界面" else "Port the native UI", "2026-10-20T04:40:00Z", "gem", "withdo", 21, true, if (zh) "合并了" else "Merged"),
        task(5, if (zh) "移植 MCP 服务器" else "Port the MCP server", "2026-10-20T07:20:00Z", "flask", "withdo", 14, true, if (zh) "八个工具连测试一起移过来了" else "Eight tools ported with their tests"),
        task(6, if (zh) "修登录跳转" else "Fix the login redirect", "2026-10-20T02:50:00Z", "jar", "withdo"),
    )
    val codex = person(
        "Codex",
        task(7, if (zh) "写监视器" else "Write the watcher", "2026-10-20T05:30:00Z", "gear", "withdo", 19, true),
        task(8, if (zh) "整理发布脚本" else "Tidy the release script", "2026-10-20T08:10:00Z", "book"),
    )
    val everyone = listOf(mine, claude, codex)
    val goods = listOf(
        SummaryGood(0, "mail", false, 0, 2), SummaryGood(1, "jar", false, 0, 6), SummaryGood(0, "scroll", false, 0, 3),
        SummaryGood(1, "gem", true, 0, 4), SummaryGood(2, "gear", true, 0, 7), SummaryGood(0, "mail", false, 0, 1),
        SummaryGood(1, "flask", true, 0, 5), SummaryGood(2, "book", false, 0, 8),
    )
    return SummaryPage(
        from = "2026-10-20", to = "2026-10-20", total = 8, merged = 3, ledger = everyone,
        open = SummaryOpen(inProgress = 3, inReview = 1),
        paragraph = if (zh) {
            "完成 8 件。你回了打包的问题、预约了牙医；Claude Code 合并了原生界面和 MCP 服务器，Codex 合并了监视器。原生界面的评审还在等。"
        } else {
            "Eight finished. You answered the packaging question and booked the dentist; Claude Code merged the native UI and the MCP server, and Codex merged the watcher."
        },
        goods = goods,
    )
}

/** A week: goods on four days, a busy Wednesday so its stack is full and counted. */
internal fun weekPage(zh: Boolean): SummaryPage {
    val days = listOf(
        listOf("book", "mail"), listOf("gem", "flask", "jar"), listOf("gem", "gear", "mail", "scroll", "coin", "flask", "book", "gem", "jar", "crate"),
        emptyList(), listOf("flask", "gem"), emptyList(), emptyList(),
    )
    val goods = days.flatMapIndexed { d, kinds ->
        kinds.mapIndexed { i, kind -> SummaryGood(actor = if (i % 3 == 0) 0 else 1, kind = kind, starred = i == 1, day = d, taskId = (d * 20 + i).toLong()) }
    }
    fun tasks(actor: Int) = goods.filter { it.actor == actor }.map {
        task(it.taskId, (if (zh) "任务 " else "Task ") + it.taskId, "2026-10-${19 + it.day}T0${it.actor + 6}:00:00Z", it.kind)
    }
    val everyone = listOf(
        person("", *tasks(0).toTypedArray()),
        person("Claude Code", *tasks(1).toTypedArray()),
    )
    return SummaryPage(
        week = true, from = "2026-10-19", to = "2026-10-25", total = goods.size, merged = 0, ledger = everyone,
        paragraph = if (zh) "这周出了 17 件，周三最忙。" else "Seventeen shipped this week, and Wednesday was the busy one.",
        goods = goods,
    )
}

internal fun emptyPage() = SummaryPage(from = "2026-10-20", to = "2026-10-20")
