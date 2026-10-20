package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.AgentInfo
import xyz.sriai.adhdtodo.data.AgentRow
import xyz.sriai.adhdtodo.data.AgentsPage
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens

/**
 * Agents: what is at work with its subtasks on the rail, then what was finished today. The header
 * says how many are working; an agent's latest report supplies its name, branch and age.
 */
@Composable
fun AgentsScreen(
    page: AgentsPage?,
    agents: Map<Long, AgentInfo>,
    words: Words,
    onOpenTask: (Long) -> Unit,
    onHowToConnect: () -> Unit,
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = Dims.pagePadding, end = Dims.pagePadding, bottom = 8.dp),
    ) {
        if (page == null) return@LazyColumn
        if (page.empty) {
            item(key = "empty") {
                EmptyState(
                    title = stringResource(R.string.empty_agents_title),
                    body = stringResource(R.string.empty_agents_body),
                    top = 72.dp,
                    link = stringResource(R.string.how_to_connect),
                    onLink = onHowToConnect,
                )
            }
        }
        if (page.working.isNotEmpty()) {
            item(key = "working") {
                SectionHeader(stringResource(R.string.section_working), page.working.size)
                Column {
                    for (row in page.working) {
                        WorkingRow(row, agents[row.id], words, onOpenTask)
                    }
                }
            }
        }
        if (page.done.isNotEmpty()) {
            item(key = "done") {
                SectionHeader(stringResource(R.string.section_done_today), page.done.size)
                Column {
                    for (row in page.done) {
                        DoneRow(row, agents[row.id], words, onOpenTask)
                    }
                }
            }
        }
    }
}

@Composable
private fun WorkingRow(
    row: AgentRow,
    info: AgentInfo?,
    words: Words,
    onOpenTask: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val agent = info?.agent.orEmpty()
    val nextPrefix = stringResource(R.string.next_step_prefix)
    val meta = buildAnnotatedString {
        append(words.agent(agent))
        if (row.branch.isNotEmpty()) {
            append(" · ")
            withStyle(SpanStyle(fontFamily = type.mono.fontFamily, fontSize = type.mono.fontSize)) { append(row.branch) }
        }
        if (row.summary.isNotEmpty()) {
            append(" · ")
            append(row.summary)
        }
        if (row.nextStep.isNotEmpty()) {
            append(nextPrefix)
            append(row.nextStep)
        }
    }
    Column(
        modifier = Modifier.fillMaxWidth().clickable { onOpenTask(row.id) },
    ) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(vertical = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.Top,
        ) {
            Avatar(words.avatar(agent), size = Dims.avatar)
            Column(Modifier.weight(1f)) {
                Text(
                    row.title,
                    style = type.bodyStrong,
                    color = tokens.text,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    meta,
                    style = type.meta,
                    color = tokens.textSecondary,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Column(
                horizontalAlignment = Alignment.End,
                verticalArrangement = Arrangement.spacedBy(4.dp),
                modifier = Modifier.padding(top = 2.dp),
            ) {
                if (row.status == "blocked") {
                    StatusPill("blocked", stringResource(R.string.pill_blocked))
                }
                Row(verticalAlignment = Alignment.CenterVertically) {
                    if (row.live && row.status != "blocked") {
                        Box(Modifier.size(6.dp).background(tokens.statusDoing))
                        Spacer(Modifier.size(6.dp))
                    }
                    Text(
                        words.ago(row.since),
                        style = type.meta,
                        color = if (row.stale) tokens.stale else tokens.textTertiary,
                        maxLines = 1,
                    )
                }
            }
        }
        if (row.subs.isNotEmpty()) {
            SubtaskRail(row.subs, words, onOpenTask)
        }
    }
}

@Composable
private fun SubtaskRail(subs: List<Item>, words: Words, onOpenTask: (Long) -> Unit) {
    val tokens = LocalTokens.current
    Column(
        modifier = Modifier
            .padding(start = 8.dp, top = 2.dp, bottom = 6.dp)
            .drawBehind {
                drawLine(
                    color = tokens.border,
                    start = Offset(0.5.dp.toPx(), 0f),
                    end = Offset(0.5.dp.toPx(), size.height),
                    strokeWidth = 1.dp.toPx(),
                )
            }
            .padding(start = 18.dp),
    ) {
        for (sub in subs) {
            SubtaskRow(sub, words, onOpenTask)
        }
    }
}

@Composable
private fun SubtaskRow(sub: Item, words: Words, onOpenTask: (Long) -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(40.dp)
            .clickable { onOpenTask(sub.id) },
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Avatar("bot", size = Dims.avatarSmall)
        Text(
            sub.title,
            style = type.body,
            color = tokens.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
        StatusPill(sub.status, words.statusLabel(sub.status))
        Text(words.ago(sub.since), style = type.meta, color = tokens.textTertiary, maxLines = 1)
    }
}

@Composable
private fun DoneRow(row: AgentRow, info: AgentInfo?, words: Words, onOpenTask: (Long) -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val agent = row.by.ifEmpty { info?.agent.orEmpty() }
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(Dims.rowHeight)
            .clickable { onOpenTask(row.id) },
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Avatar(if (agent.isEmpty()) "bot" else words.avatar(agent), size = Dims.avatar)
        Text(
            row.title,
            style = type.body,
            color = tokens.textSecondary,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
        val subs = if (row.subs.isNotEmpty()) "${words.subtasks(row.subs.size)} · " else ""
        Text(
            subs + words.timeOfDay(row.since),
            style = type.meta,
            color = tokens.textTertiary,
            maxLines = 1,
        )
    }
}
