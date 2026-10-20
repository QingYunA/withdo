package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.AgentInfo
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.StatusGlyph

/**
 * Mine: From your agents, Today, Later and Done today. A section with nothing in it is not drawn;
 * with the first three empty the screen says nothing needs you. Done today is a record, not a
 * to-do, and never makes Mine busy.
 */
@Composable
fun MineScreen(
    page: MinePage?,
    agents: Map<Long, AgentInfo>,
    words: Words,
    onAccept: (Long) -> Unit,
    onComplete: (Long) -> Unit,
    onAnswer: (Item) -> Unit,
    onOpenUrl: (String) -> Unit,
    onOpenTask: (Long) -> Unit,
    onHowToConnect: () -> Unit,
    onComingSoon: () -> Unit,
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = Dims.pagePadding, end = Dims.pagePadding, bottom = 8.dp),
    ) {
        if (page == null) return@LazyColumn
        if (page.empty) {
            item(key = "empty") {
                EmptyState(
                    title = stringResource(R.string.empty_mine_title),
                    body = stringResource(R.string.empty_mine_body),
                    top = 72.dp,
                    link = stringResource(R.string.how_to_connect),
                    onLink = onHowToConnect,
                )
            }
        }
        if (page.fromAgents.isNotEmpty()) {
            item(key = "from-agents") {
                SectionHeader(stringResource(R.string.section_from_agents), page.fromAgents.size)
                FromAgentsBox(
                    items = page.fromAgents,
                    agents = agents,
                    words = words,
                    onAccept = onAccept,
                    onComplete = onComplete,
                    onAnswer = onAnswer,
                    onOpenUrl = onOpenUrl,
                    onOpenTask = onOpenTask,
                )
            }
        }
        if (page.today.isNotEmpty()) {
            item(key = "today") {
                MineSection(
                    title = stringResource(R.string.section_today),
                    items = page.today,
                    words = words,
                    onComplete = onComplete,
                    onOpenTask = onOpenTask,
                )
            }
        }
        if (page.later.isNotEmpty()) {
            item(key = "later") {
                MineSection(
                    title = stringResource(R.string.section_later),
                    items = page.later,
                    words = words,
                    onComplete = onComplete,
                    onOpenTask = onOpenTask,
                )
                if (page.moreLater > 0) {
                    Text(
                        stringResource(R.string.show_all_in_inbox),
                        style = LocalAppType.current.meta,
                        color = LocalTokens.current.accent,
                        modifier = Modifier.padding(start = 8.dp, top = 8.dp).clickable { onComingSoon() },
                    )
                }
            }
        }
        if (page.done.isNotEmpty()) {
            item(key = "done") {
                DoneTodaySection(
                    title = stringResource(R.string.section_done_today),
                    items = page.done,
                    words = words,
                    onOpenTask = onOpenTask,
                )
                if (page.moreDone > 0) {
                    Text(
                        stringResource(R.string.show_all_in_completed),
                        style = LocalAppType.current.meta,
                        color = LocalTokens.current.accent,
                        modifier = Modifier.padding(start = 8.dp, top = 8.dp).clickable { onComingSoon() },
                    )
                }
            }
        }
    }
}

@Composable
private fun FromAgentsBox(
    items: List<Item>,
    agents: Map<Long, AgentInfo>,
    words: Words,
    onAccept: (Long) -> Unit,
    onComplete: (Long) -> Unit,
    onAnswer: (Item) -> Unit,
    onOpenUrl: (String) -> Unit,
    onOpenTask: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val shape = RoundedCornerShape(12.dp)
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clip(shape)
            .border(1.dp, tokens.border, shape),
    ) {
        items.forEachIndexed { index, item ->
            if (index > 0) {
                Box(Modifier.fillMaxWidth().height(1.dp).background(tokens.border))
            }
            HandedRow(
                item = item,
                info = agents[item.id],
                words = words,
                onAccept = onAccept,
                onComplete = onComplete,
                onAnswer = onAnswer,
                onOpenUrl = onOpenUrl,
                onOpenTask = onOpenTask,
            )
        }
    }
}

@Composable
private fun HandedRow(
    item: Item,
    info: AgentInfo?,
    words: Words,
    onAccept: (Long) -> Unit,
    onComplete: (Long) -> Unit,
    onAnswer: (Item) -> Unit,
    onOpenUrl: (String) -> Unit,
    onOpenTask: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val agent = info?.agent.orEmpty()
    val links = info?.links.orEmpty()
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onOpenTask(item.id) }
            .padding(horizontal = 14.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Avatar(words.avatar(agent), size = Dims.avatar)
        Column(Modifier.weight(1f)) {
            Text(
                item.title,
                style = type.bodyStrong,
                color = tokens.text,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                reason(item, info, words),
                style = type.meta,
                color = tokens.textSecondary,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            Row(
                modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                when (item.kind) {
                    0 -> ActionButton(stringResource(R.string.action_answer)) { onAnswer(item) }
                    1 -> ActionButton(stringResource(R.string.action_done)) { onComplete(item.id) }
                    2 -> if (item.prUrl.isNotEmpty()) {
                        ActionButton(stringResource(R.string.action_open_pr)) { onOpenUrl(item.prUrl) }
                    }
                    3, 4 -> if (item.prUrl.isNotEmpty()) {
                        ActionButton(stringResource(R.string.action_open_pr)) { onOpenUrl(item.prUrl) }
                    }
                    5 -> {
                        if (links.isNotEmpty()) {
                            ActionButton(stringResource(R.string.action_open_page)) { onOpenUrl(links.first().url) }
                        }
                        if (item.prUrl.isNotEmpty()) {
                            ActionButton(stringResource(R.string.action_open_pr)) { onOpenUrl(item.prUrl) }
                        } else {
                            ActionButton(stringResource(R.string.action_review)) { onOpenTask(item.id) }
                        }
                        ActionButton(stringResource(R.string.action_accept)) { onAccept(item.id) }
                    }
                    else -> ActionButton(stringResource(R.string.action_open)) { onOpenTask(item.id) }
                }
                Spacer(Modifier.weight(1f))
                Text(
                    words.ago(item.since),
                    style = type.meta,
                    color = if (item.stale) tokens.stale else tokens.textTertiary,
                    maxLines = 1,
                )
            }
        }
    }
}

@Composable
private fun ActionButton(label: String, onClick: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Box(
        modifier = Modifier
            .clip(RoundedCornerShape(6.dp))
            .background(tokens.bgSelected)
            .clickable { onClick() }
            .padding(horizontal = 12.dp, vertical = 5.dp),
    ) {
        Text(label, style = type.meta, color = tokens.accent)
    }
}

@Composable
private fun MineSection(
    title: String,
    items: List<Item>,
    words: Words,
    onComplete: (Long) -> Unit,
    onOpenTask: (Long) -> Unit,
) {
    SectionHeader(title, items.size)
    Column {
        for (item in items) {
            TaskRow(item, words, onComplete, onOpenTask)
        }
    }
}

@Composable
private fun TaskRow(item: Item, words: Words, onComplete: (Long) -> Unit, onOpenTask: (Long) -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(Dims.rowHeight)
            .clickable { onOpenTask(item.id) },
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        StatusGlyph(
            status = item.status,
            onClick = {
                if (item.source == "user" && item.status != "done" && item.status != "cancelled") {
                    onComplete(item.id)
                } else {
                    onOpenTask(item.id)
                }
            },
        )
        Text(
            item.title,
            style = type.bodyStrong,
            color = tokens.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.widthIn(max = 230.dp),
        )
        if (item.project.isNotEmpty()) {
            Text(
                item.project,
                style = type.meta,
                color = tokens.textTertiary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
        } else {
            Spacer(Modifier.weight(1f))
        }
        val due = words.due(item)
        if (item.priority == "P0" || item.priority == "P1" || due.isNotEmpty()) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                PriorityBadge(item.priority)
                if (due.isNotEmpty()) Text(due, style = type.meta, color = tokens.textTertiary, maxLines = 1)
            }
        }
    }
}

@Composable
private fun DoneTodaySection(
    title: String,
    items: List<Item>,
    words: Words,
    onOpenTask: (Long) -> Unit,
) {
    SectionHeader(title, items.size)
    Column {
        for (item in items) {
            val tokens = LocalTokens.current
            val type = LocalAppType.current
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(Dims.rowHeight)
                    .clickable { onOpenTask(item.id) },
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                StatusGlyph(status = "done")
                Text(
                    item.title,
                    style = type.body,
                    color = tokens.textSecondary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    "${words.who(item.by)} · ${words.timeOfDay(item.since)}",
                    style = type.meta,
                    color = tokens.textTertiary,
                    maxLines = 1,
                )
            }
        }
    }
}

/** The one line under an agent's title: why the row is in From your agents, then its project. */
@Composable
private fun reason(item: Item, info: AgentInfo?, words: Words): String {
    val agent = words.agent(info?.agent.orEmpty())
    val reason = when (item.kind) {
        0 -> if (item.words.isNotEmpty()) {
            stringResource(R.string.need_blocked_with_words, agent, item.words)
        } else {
            stringResource(R.string.need_blocked, agent)
        }
        1 -> {
            val at = words.timeOfDay(item.remindAt ?: item.dueAt)
            if (at.isNotEmpty()) {
                stringResource(R.string.need_reminder, at)
            } else {
                stringResource(R.string.need_reminder, words.ago(item.since))
            }
        }
        2 -> stringResource(R.string.need_pr_closed, item.prNumber)
        3 -> stringResource(R.string.need_pr_failing, item.prNumber)
        4 -> stringResource(R.string.need_pr_changes, item.prNumber)
        5 -> if (item.prNumber > 0) {
            stringResource(R.string.need_review_pr, item.prNumber)
        } else {
            stringResource(R.string.need_review, agent)
        }
        else -> stringResource(R.string.need_stale, words.ago(item.since))
    }
    return if (item.project.isNotEmpty()) "$reason · ${item.project}" else reason
}
