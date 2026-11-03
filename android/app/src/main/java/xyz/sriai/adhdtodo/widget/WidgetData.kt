package xyz.sriai.adhdtodo.widget

import android.content.res.Resources
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.ui.Words

/** One row of the Today widget: the glyph completes the task, the rest opens it. */
data class TodayRow(val id: Long, val status: String, val title: String, val due: String)

/**
 * What the Today widget draws, with every word already in the app's language, so the widget's
 * content is plain data: the screenshots build it from the golden files, the widget from the core.
 * [empty] is the quiet line; it is blank when the core could not be read.
 */
data class TodayData(
    val label: String,
    val count: Int,
    val rows: List<TodayRow>,
    val empty: String,
    val addLabel: String,
    val completeLabel: (String) -> String,
)

/** The first agent card that waits for you: its title and the one line under it. */
data class AgentCard(val id: Long, val title: String, val meta: String)

data class AgentsData(
    val label: String,
    val count: Int,
    val card: AgentCard?,
    val empty: String,
    val replyLabel: String,
)

/** The Today section of the phone's first page: how many, and the first rows. */
fun todayData(page: MinePage?, words: Words, res: Resources): TodayData {
    val today = page?.today.orEmpty()
    return TodayData(
        label = res.getString(R.string.section_today),
        count = today.size,
        rows = today.take(TODAY_ROWS).map { TodayRow(it.id, it.status, it.title, words.due(it)) },
        empty = if (page == null) "" else res.getString(R.string.widget_today_empty),
        addLabel = res.getString(R.string.widget_add_task),
        completeLabel = { res.getString(R.string.widget_complete_task, it) },
    )
}

/**
 * The first of Mine's cards that wait for you, with the age Mine shows beside it. [agent] is the
 * raw agent name behind that card (CoreApi.info), blank when it is not known.
 */
fun agentsData(page: MinePage?, agent: String, words: Words, res: Resources): AgentsData {
    val waiting = page?.fromAgents.orEmpty()
    val first = waiting.firstOrNull()
    return AgentsData(
        label = res.getString(R.string.widget_agent),
        count = waiting.size,
        card = first?.let {
            AgentCard(it.id, it.title, agentMeta(it, words.agent(agent), words, res))
        },
        empty = if (page == null) "" else res.getString(R.string.widget_agents_empty),
        replyLabel = res.getString(R.string.action_answer),
    )
}

/**
 * The line under the title: why it waits, then the age. The age keeps its spaces unbreakable, so
 * a narrow card wraps before "3 days ago" and never inside it.
 */
internal fun agentMeta(item: Item, agentName: String, words: Words, res: Resources): String =
    "${agentReason(item, agentName, words, res)} · ${words.agoFull(item.since).replace(' ', '\u00A0')}"

/**
 * Why the card waits, in MineScreen's words. The widget has room for the short form only, so a
 * blocked card does not quote the agent's question.
 */
internal fun agentReason(item: Item, agentName: String, words: Words, res: Resources): String = when (item.kind) {
    0 -> res.getString(R.string.need_blocked)
    1 -> res.getString(
        R.string.need_reminder,
        words.timeOfDay(item.remindAt ?: item.dueAt).ifEmpty { words.agoFull(item.since) },
    )
    2 -> res.getString(R.string.need_pr_closed, item.prNumber)
    3 -> res.getString(R.string.need_pr_failing, item.prNumber)
    4 -> res.getString(R.string.need_pr_changes, item.prNumber)
    5 -> if (item.prNumber > 0) {
        res.getString(R.string.need_review_pr, item.prNumber)
    } else {
        res.getString(R.string.need_review, agentName)
    }
    else -> res.getString(R.string.need_stale, words.ago(item.since))
}

/** The spec's "up to three rows". */
const val TODAY_ROWS = 3

/** How many rows a widget of [heightDp] has room for: at least one, at most [TODAY_ROWS]. */
fun rowsThatFit(heightDp: Float): Int =
    ((heightDp - WidgetDims.cardChrome - WidgetDims.header) / WidgetDims.row).toInt().coerceIn(1, TODAY_ROWS)
