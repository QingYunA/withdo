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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.ProjectEntry
import xyz.sriai.adhdtodo.data.ProjectPage
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import xyz.sriai.adhdtodo.ui.theme.StatusGlyph

/** The status order of a project's groups, as the desktop's project columns draw them. */
private val projectStatusOrder = listOf("blocked", "review", "doing", "todo", "backlog")

/**
 * The Projects tab: one row per project, its folder in the project's color, its name and the
 * number of open top-level tasks. A project with no open task still lists, with a quiet 0.
 */
@Composable
fun ProjectsScreen(
    entries: List<ProjectEntry>?,
    onOpenProject: (Long) -> Unit,
    onHowToConnect: () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = Dims.pagePadding, end = Dims.pagePadding, bottom = 8.dp),
    ) {
        if (entries == null) return@LazyColumn
        if (entries.isEmpty()) {
            item(key = "empty") {
                EmptyState(
                    title = stringResource(R.string.empty_projects_title),
                    body = stringResource(R.string.empty_projects_body),
                    top = 72.dp,
                    link = stringResource(R.string.how_to_connect),
                    onLink = onHowToConnect,
                )
            }
        }
        for (entry in entries) {
            item(key = "project-${entry.project.id}") {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(Dims.rowHeight)
                        .clickable { onOpenProject(entry.project.id) },
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    PixelIcon(
                        "folder",
                        size = Dims.icon,
                        tint = tokens.projectColor(entry.project.name),
                    )
                    Text(
                        entry.project.name,
                        style = type.bodyStrong,
                        color = tokens.text,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f),
                    )
                    Text(
                        entry.open.toString(),
                        style = type.meta,
                        color = tokens.textTertiary,
                    )
                }
            }
        }
    }
}

/**
 * One project, v26 frame 3: the folder and title, the Mine | Agents control, groups by status
 * with the glyph in the header, rows 26 dp in with the author's avatar, and Done folded into one
 * Completed row that unfolds.
 */
@Composable
fun ProjectScreen(
    page: ProjectPage?,
    words: Words,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit = {},
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    var agents by remember(page?.project?.id) { mutableStateOf(false) }
    var completedOpen by remember(page?.project?.id) { mutableStateOf(false) }
    if (page == null) return

    val top = page.tasks.filter { it.parentId == null }
    val mine = top.filter { it.source != "agent" }
    val agentTasks = top.filter { it.source == "agent" }
    val shown = if (agents) agentTasks else mine
    val active = shown.filter { it.status != "done" && it.status != "cancelled" }
    val done = shown.filter { it.status == "done" || it.status == "cancelled" }
    val mineActive = mine.count { it.status != "done" && it.status != "cancelled" }
    val agentsActive = agentTasks.count { it.status != "done" && it.status != "cancelled" }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = Dims.pagePadding, end = Dims.pagePadding, bottom = 8.dp),
    ) {
        item(key = "title") {
            Row(
                modifier = Modifier.padding(top = 2.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                PixelIcon(
                    "folder",
                    size = Dims.icon,
                    tint = tokens.projectColor(page.project.name),
                )
                Text(
                    page.project.name,
                    style = type.title,
                    color = tokens.text,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            SegmentControl(
                mine = mineActive,
                agents = agentsActive,
                agentsSelected = agents,
                onSelect = { agents = it },
            )
        }

        if (top.isEmpty()) {
            item(key = "empty") {
                EmptyState(
                    title = stringResource(R.string.empty_project_title),
                    body = stringResource(R.string.empty_project_body),
                    top = 56.dp,
                )
            }
        }

        for (status in projectStatusOrder) {
            val tasks = active.filter { it.status == status }
            if (tasks.isEmpty()) continue
            item(key = "group-$status") {
                StatusGroupHeader(status, tasks.size, words)
                Column {
                    for (task in tasks) {
                        ProjectTaskRow(task, page.agents[task.id.toString()].orEmpty(), words, onOpenTask, onLongPress)
                    }
                }
            }
        }

        if (done.isNotEmpty()) {
            item(key = "completed") {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(44.dp)
                        .clickable { completedOpen = !completedOpen },
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    PixelIcon(
                        if (completedOpen) "chevron-down" else "chevron-right",
                        size = 12.dp,
                        tint = tokens.textTertiary,
                    )
                    Text(
                        stringResource(R.string.project_completed, done.size),
                        style = type.meta,
                        color = tokens.textTertiary,
                    )
                }
            }
            if (completedOpen) {
                item(key = "completed-rows") {
                    Column {
                        for (task in done) {
                            DoneProjectRow(task, page.agents[task.id.toString()].orEmpty(), words, onOpenTask, onLongPress)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun SegmentControl(
    mine: Int,
    agents: Int,
    agentsSelected: Boolean,
    onSelect: (Boolean) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 14.dp, bottom = 4.dp)
            .clip(RoundedCornerShape(8.dp))
            .background(tokens.bgSubtle)
            .padding(3.dp),
    ) {
        Segment(
            label = stringResource(R.string.segment_mine),
            count = mine,
            selected = !agentsSelected,
            modifier = Modifier.weight(1f),
            onClick = { onSelect(false) },
        )
        Segment(
            label = stringResource(R.string.nav_agents),
            count = agents,
            selected = agentsSelected,
            modifier = Modifier.weight(1f),
            onClick = { onSelect(true) },
        )
    }
}

@Composable
private fun Segment(
    label: String,
    count: Int,
    selected: Boolean,
    modifier: Modifier,
    onClick: () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = modifier
            .height(30.dp)
            .clip(RoundedCornerShape(6.dp))
            .then(
                if (selected) {
                    Modifier.background(tokens.bg).border(Dims.hairline, tokens.border, RoundedCornerShape(6.dp))
                } else {
                    Modifier
                },
            )
            .clickable(onClick = onClick),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            label,
            style = if (selected) type.section else type.meta,
            color = if (selected) tokens.text else tokens.textSecondary,
        )
        Text(
            count.toString(),
            style = type.meta,
            color = tokens.textTertiary,
            modifier = Modifier.padding(start = 4.dp),
        )
    }
}

@Composable
private fun StatusGroupHeader(status: String, count: Int, words: Words) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier.padding(top = 18.dp, bottom = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        StatusGlyph(status, size = 16.dp)
        Text(words.groupLabel(status), style = type.section, color = tokens.textSecondary)
        Text(count.toString(), style = type.meta, color = tokens.textTertiary)
    }
}

@Composable
private fun ProjectTaskRow(
    task: Task,
    agent: String,
    words: Words,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    LongPressRow(
        onClick = { onOpenTask(task.id) },
        onLongPress = { onLongPress(task.id) },
    ) { row ->
        Row(
            modifier = row.height(Dims.rowHeight).padding(start = 26.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
        Avatar(rowAvatar(task, agent, words), size = Dims.avatar)
        Text(
            task.title,
            style = type.bodyStrong,
            color = tokens.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
        PriorityBadge(task.priority)
        val due = words.due(task)
        if (due.isNotEmpty()) {
            Text(due, style = type.meta, color = tokens.textTertiary, maxLines = 1)
        }
        Spacer(Modifier.width(2.dp))
        }
    }
}

@Composable
private fun DoneProjectRow(
    task: Task,
    agent: String,
    words: Words,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    LongPressRow(
        onClick = { onOpenTask(task.id) },
        onLongPress = { onLongPress(task.id) },
    ) { row ->
        Row(
            modifier = row.height(Dims.rowHeight).padding(start = 26.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
        Avatar(rowAvatar(task, agent, words), size = Dims.avatar)
        Text(
            task.title,
            style = type.body,
            color = tokens.textSecondary,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
        Text(
            words.ago(task.completedAt ?: task.updatedAt),
            style = type.meta,
            color = tokens.textTertiary,
            maxLines = 1,
        )
        }
    }
}

/** The author of a project row: you for your tasks; for an agent's, the agent of its latest report. */
private fun rowAvatar(task: Task, agent: String, words: Words): String = when {
    task.source != "agent" -> "you"
    agent.isNotEmpty() && agent != "you" -> words.avatar(agent)
    else -> "bot"
}
