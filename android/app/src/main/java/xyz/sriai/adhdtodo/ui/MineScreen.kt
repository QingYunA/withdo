package xyz.sriai.adhdtodo.ui

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Box
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.AgentInfo
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import xyz.sriai.adhdtodo.ui.theme.StarGlyph
import xyz.sriai.adhdtodo.ui.theme.StatusGlyph

/** The inset of a list's text from its container: titles, headers and glyphs share this edge. */
private val TextInset = 12.dp

/**
 * Today (v35): the Agents tray with its cards first, then Today, Next, the No date fold and Done
 * today, with the desktop's v34 rules. A section with nothing in it is not drawn; with the first
 * four empty the screen says nothing needs you. Done today is a record, not a to-do, and never
 * makes the page busy.
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
    onViewAll: () -> Unit,
    onLongPress: (Long) -> Unit = {},
    categories: List<Category> = emptyList(),
    noDateOpen: Boolean = false,
    onToggleNoDate: () -> Unit = {},
    highlightId: Long = 0,
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = 16.dp, end = 16.dp, bottom = 8.dp),
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
                AgentsTray(
                    items = page.fromAgents,
                    agents = agents,
                    words = words,
                    onAccept = onAccept,
                    onComplete = onComplete,
                    onAnswer = onAnswer,
                    onOpenUrl = onOpenUrl,
                    onOpenTask = onOpenTask,
                    onLongPress = onLongPress,
                )
            }
        }
        if (page.today.isNotEmpty()) {
            item(key = "today") {
                MineSection(
                    title = stringResource(R.string.section_today),
                    items = page.today,
                    categories = categories,
                    words = words,
                    highlightId = highlightId,
                    onComplete = onComplete,
                    onOpenTask = onOpenTask,
                    onLongPress = onLongPress,
                )
            }
        }
        if (page.next.isNotEmpty()) {
            item(key = "next") {
                MineSection(
                    title = stringResource(R.string.section_next),
                    items = page.next,
                    categories = categories,
                    words = words,
                    highlightId = highlightId,
                    onComplete = onComplete,
                    onOpenTask = onOpenTask,
                    onLongPress = onLongPress,
                )
            }
        }
        if (page.noDate.isNotEmpty()) {
            item(key = "no-date") {
                NoDateFold(
                    items = page.noDate,
                    total = maxOf(page.noDateTotal, page.noDate.size),
                    open = noDateOpen,
                    categories = categories,
                    words = words,
                    highlightId = highlightId,
                    onToggle = onToggleNoDate,
                    onComplete = onComplete,
                    onOpenTask = onOpenTask,
                    onLongPress = onLongPress,
                )
            }
        }
        if (page.done.isNotEmpty()) {
            item(key = "done") {
                DoneTodaySection(
                    items = page.done,
                    more = page.moreDone,
                    words = words,
                    onViewAll = onViewAll,
                    onOpenTask = onOpenTask,
                    onLongPress = onLongPress,
                )
            }
        }
    }
}

/**
 * What agents handed over: slate 2, radius 12, a 4 dp inset and a 0.5 dp edge, its header the bot
 * avatar, `Agent` and the count; one card per row on bg, radius 8 (12 less the inset).
 */
@Composable
private fun AgentsTray(
    items: List<Item>,
    agents: Map<Long, AgentInfo>,
    words: Words,
    onAccept: (Long) -> Unit,
    onComplete: (Long) -> Unit,
    onAnswer: (Item) -> Unit,
    onOpenUrl: (String) -> Unit,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val shape = RoundedCornerShape(12.dp)
    Column(
        modifier = Modifier
            .padding(top = 4.dp)
            .fillMaxWidth()
            .background(tokens.bgTray, shape)
            .border(Dims.hairline, tokens.border, shape)
            .padding(4.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(start = 8.dp, end = 8.dp, top = 6.dp, bottom = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Avatar("bot", size = Dims.avatar)
            Text(stringResource(R.string.agents_tray_title), style = type.section, color = tokens.textSecondary)
            Box(Modifier.weight(1f))
            Text(items.size.toString(), style = type.meta, color = tokens.textSecondary)
        }
        for (item in items) {
            AgentCard(
                item = item,
                info = agents[item.id],
                words = words,
                onAccept = onAccept,
                onComplete = onComplete,
                onAnswer = onAnswer,
                onOpenUrl = onOpenUrl,
                onOpenTask = onOpenTask,
                onLongPress = onLongPress,
            )
        }
    }
}

@Composable
private fun AgentCard(
    item: Item,
    info: AgentInfo?,
    words: Words,
    onAccept: (Long) -> Unit,
    onComplete: (Long) -> Unit,
    onAnswer: (Item) -> Unit,
    onOpenUrl: (String) -> Unit,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val shape = RoundedCornerShape(8.dp)
    val agent = info?.agent.orEmpty()
    Box(Modifier.fillMaxWidth().clip(shape).background(tokens.bg).border(Dims.hairline, tokens.border, shape)) {
        LongPressRow(
            onClick = { onOpenTask(item.id) },
            onLongPress = { onLongPress(item.id) },
        ) { row ->
            Row(
                modifier = row.padding(horizontal = 12.dp, vertical = 12.dp),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.Top,
            ) {
                Avatar(words.avatar(agent), size = Dims.avatar, modifier = Modifier.padding(top = 2.dp))
                Column(Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(
                            item.title,
                            style = type.bodyStrong,
                            color = tokens.text,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.weight(1f),
                        )
                        Text(
                            words.agoFull(item.since),
                            style = type.meta,
                            color = if (item.stale) tokens.stale else tokens.textSecondary,
                            maxLines = 1,
                        )
                    }
                    Text(
                        reason(item, words.agent(agent), words),
                        style = type.meta,
                        color = tokens.textSecondary,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                    Row(
                        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        CardActions(item, onAccept, onComplete, onAnswer, onOpenUrl, onOpenTask)
                    }
                }
            }
        }
    }
}

/** One tinted button, and where there is a second action, a slate 11 text action. */
@Composable
private fun CardActions(
    item: Item,
    onAccept: (Long) -> Unit,
    onComplete: (Long) -> Unit,
    onAnswer: (Item) -> Unit,
    onOpenUrl: (String) -> Unit,
    onOpenTask: (Long) -> Unit,
) {
    val openPr = stringResource(R.string.action_open_pr)
    when (item.kind) {
        0 -> TintButton(stringResource(R.string.action_answer)) { onAnswer(item) }
        1 -> TintButton(stringResource(R.string.action_done)) { onComplete(item.id) }
        2, 3, 4 -> if (item.prUrl.isNotEmpty()) {
            TintButton(openPr) { onOpenUrl(item.prUrl) }
        } else {
            TintButton(stringResource(R.string.action_open)) { onOpenTask(item.id) }
        }
        5 -> {
            if (item.prUrl.isNotEmpty()) {
                TintButton(openPr) { onOpenUrl(item.prUrl) }
            } else {
                TintButton(stringResource(R.string.action_review)) { onOpenTask(item.id) }
            }
            TextAction(stringResource(R.string.action_accept)) { onAccept(item.id) }
        }
        else -> TintButton(stringResource(R.string.action_open)) { onOpenTask(item.id) }
    }
}

@Composable
private fun TintButton(label: String, onClick: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Box(
        modifier = Modifier
            .clip(RoundedCornerShape(6.dp))
            .background(tokens.accentSoft)
            .clickable { onClick() }
            .padding(horizontal = 12.dp, vertical = 7.dp),
    ) {
        Text(label, style = type.meta, color = tokens.accentInk)
    }
}

@Composable
private fun TextAction(label: String, onClick: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Box(
        modifier = Modifier
            .clip(RoundedCornerShape(6.dp))
            .clickable { onClick() }
            .padding(horizontal = 8.dp, vertical = 7.dp),
    ) {
        Text(label, style = type.meta, color = tokens.textSecondary)
    }
}

@Composable
private fun MineSection(
    title: String,
    items: List<Item>,
    categories: List<Category>,
    words: Words,
    highlightId: Long,
    onComplete: (Long) -> Unit,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
) {
    SectionHeader(title, items.size, start = TextInset)
    for (item in items) {
        TaskRow(item, categories, words, highlightId, onComplete, onOpenTask, onLongPress)
    }
}

/** The fold of my undated tasks that have no project: closed by default, with its count. */
@Composable
private fun NoDateFold(
    items: List<Item>,
    total: Int,
    open: Boolean,
    categories: List<Category>,
    words: Words,
    highlightId: Long,
    onToggle: () -> Unit,
    onComplete: (Long) -> Unit,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier
            .padding(top = 12.dp)
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .clickable(onClick = onToggle)
            .padding(start = TextInset, top = 10.dp, bottom = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(stringResource(R.string.section_no_date), style = type.section, color = tokens.textSecondary)
        Text(total.toString(), style = type.meta, color = tokens.textSecondary)
        PixelIcon(if (open) "chevron-down" else "chevron-right", size = 12.dp, tint = tokens.textTertiary)
    }
    if (open) {
        for (item in items) {
            TaskRow(item, categories, words, highlightId, onComplete, onOpenTask, onLongPress)
        }
    }
}

/**
 * A row of mine, 48 dp: the status glyph (which ticks it off), the title, the category as a
 * colored word, the project; at the right the star of an important task and its date. Doing is
 * shown by its glyph alone. A task the composer just made takes the iris 3 fill for 2 s.
 */
@Composable
private fun TaskRow(
    item: Item,
    categories: List<Category>,
    words: Words,
    highlightId: Long,
    onComplete: (Long) -> Unit,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val category = if (item.source == "user" && item.categoryId != 0) {
        categories.firstOrNull { it.id.toInt() == item.categoryId }
    } else {
        null
    }
    val fill = animateColorAsState(
        if (item.id == highlightId) tokens.accentSoft else Color.Transparent,
        animationSpec = tween(250),
        label = "new-row",
    )
    Box(Modifier.fillMaxWidth().clip(RoundedCornerShape(8.dp)).background(fill.value)) {
        LongPressRow(
            onClick = { onOpenTask(item.id) },
            onLongPress = { onLongPress(item.id) },
        ) { row ->
            Row(
                modifier = row.height(Dims.rowHeight).padding(horizontal = TextInset),
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
                Row(
                    modifier = Modifier.weight(1f),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Text(
                        item.title,
                        style = type.bodyStrong,
                        color = tokens.text,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f, fill = false),
                    )
                    if (category != null) {
                        Text(
                            words.categoryName(category),
                            style = type.meta.copy(fontWeight = FontWeight.Medium),
                            color = tokens.categoryInk(category.color),
                            maxLines = 1,
                        )
                    }
                    if (item.project.isNotEmpty()) {
                        Text(
                            item.project,
                            style = type.meta,
                            color = tokens.textSecondary,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.widthIn(max = 96.dp),
                        )
                    }
                }
                val due = words.due(item)
                val starred = item.source == "user" && (item.priority == "P0" || item.priority == "P1")
                if (due.isNotEmpty() || starred) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        if (starred) StarGlyph(size = 12.dp, tint = tokens.statusReview)
                        if (due.isNotEmpty()) Text(due, style = type.meta, color = tokens.textSecondary, maxLines = 1)
                    }
                }
            }
        }
    }
}

/**
 * What was finished since midnight, by agents and by me. A row an agent finished shows the agent's
 * avatar and what its closing report said before the time; one I finished says nothing.
 */
@Composable
private fun DoneTodaySection(
    items: List<Item>,
    more: Int,
    words: Words,
    onViewAll: () -> Unit,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    SectionHeader(
        stringResource(R.string.section_done_today),
        items.size + more,
        start = TextInset,
        end = {
            Text(
                stringResource(R.string.view_all),
                style = type.meta,
                color = tokens.accentInk,
                modifier = Modifier
                    .clip(RoundedCornerShape(6.dp))
                    .clickable(onClick = onViewAll)
                    .padding(horizontal = 12.dp, vertical = 6.dp),
            )
        },
    )
    for (item in items) {
        LongPressRow(
            onClick = { onOpenTask(item.id) },
            onLongPress = { onLongPress(item.id) },
        ) { row ->
            Row(
                modifier = row.height(Dims.rowHeight).padding(horizontal = TextInset),
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
                if (item.by.isNotBlank()) {
                    Row(
                        modifier = Modifier.widthIn(max = 150.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        Avatar(words.avatar(item.by), size = 14.dp)
                        Text(
                            item.summary.ifEmpty { words.agent(item.by) },
                            style = type.meta,
                            color = tokens.textSecondary,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
                Text(words.timeOfDay(item.since), style = type.meta, color = tokens.textSecondary, maxLines = 1)
            }
        }
    }
}

/** The one short line under an agent's title: why the card is in the tray. */
@Composable
private fun reason(item: Item, agent: String, words: Words): String = when (item.kind) {
    0 -> if (item.words.isNotEmpty()) {
        stringResource(R.string.need_blocked_with_words, item.words)
    } else {
        stringResource(R.string.need_blocked)
    }
    1 -> {
        val at = words.timeOfDay(item.remindAt ?: item.dueAt)
        stringResource(R.string.need_reminder, at.ifEmpty { words.agoFull(item.since) })
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
