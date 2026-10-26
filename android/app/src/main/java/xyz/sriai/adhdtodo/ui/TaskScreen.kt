package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.Report
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.data.TaskPage
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import xyz.sriai.adhdtodo.ui.theme.StatusGlyph

/**
 * One task, v26 frame 4: the title at 22 sp 700, the property rows with their 18 dp icons and a
 * 104 dp label column, the priority strip, the subtasks and the activity on its rail. The back
 * bar's number stays in the App's back bar; this is the page under it.
 */
@Composable
fun TaskScreen(
    page: TaskPage?,
    projects: List<Project>,
    words: Words,
    onSetPriority: (String) -> Unit,
    onOpenUrl: (String) -> Unit,
    onOpenTask: (Long) -> Unit,
    categories: List<Category> = emptyList(),
    onOpenMenu: (TaskMenuPage) -> Unit = {},
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    if (page == null) return
    val task = page.task
    val project = projects.firstOrNull { it.id == task.projectId }
    val category = categories.firstOrNull { it.id.toInt() == task.categoryId }
    val agent = page.latestAgent.ifEmpty { if (task.source == "agent") "agent" else "user" }
    val fromLabel = if (agent == "user") stringResource(R.string.done_by_you) else words.agent(agent)
    val fromAvatar = if (agent == "user") "you" else words.avatar(agent)

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = Dims.pagePadding, end = Dims.pagePadding, bottom = 8.dp),
    ) {
        item(key = "title") {
            Text(
                task.title,
                style = type.taskTitle,
                color = tokens.text,
                modifier = Modifier.padding(top = 4.dp, bottom = 14.dp),
            )
        }
        item(key = "properties") {
            PropertyRow(
                "invert",
                onClick = { onOpenMenu(TaskMenuPage.Main) },
                leading = { StatusGlyph(task.status, size = Dims.icon) },
            ) {
                Text(words.menuStatus(task.status), style = type.body, color = tokens.text)
            }
            PropertyRow("robot") {
                Avatar(fromAvatar, size = Dims.avatar)
                Text(fromLabel, style = type.body, color = tokens.text)
            }
            PropertyRow("flag") {
                PriorityStrip(task.priority, onSetPriority)
            }
            PropertyRow("bulletlist", onClick = { onOpenMenu(TaskMenuPage.Category) }) {
                Text(
                    category?.let { words.categoryName(it) } ?: stringResource(R.string.menu_none),
                    style = type.body,
                    color = if (category == null) tokens.textTertiary else tokens.text,
                )
            }
            PropertyRow("folder", onClick = { onOpenMenu(TaskMenuPage.Move) }) {
                if (project != null) {
                    PixelIcon("folder", size = Dims.icon, tint = tokens.projectColor(project.name))
                    Text(project.name, style = type.body, color = tokens.text)
                } else {
                    Text(
                        stringResource(R.string.menu_none),
                        style = type.body,
                        color = tokens.textTertiary,
                    )
                }
            }
            PropertyRow("clock", onClick = { onOpenMenu(TaskMenuPage.Remind) }) {
                val remind = task.remindAt?.takeIf { it.isNotBlank() }
                Text(
                    remind?.let { words.reminderValue(it) } ?: stringResource(R.string.menu_none),
                    style = type.body,
                    color = if (remind == null) tokens.textTertiary else tokens.text,
                )
            }
            if (task.branch.isNotEmpty()) {
                PropertyRow("git-branch") {
                    Text(task.branch, style = type.mono, color = tokens.textSecondary)
                }
            }
            if (task.prUrl.isNotEmpty()) {
                PropertyRow("git-pull-request") {
                    Text(
                        "#${task.prNumber}",
                        style = type.body,
                        color = tokens.accentInk,
                        modifier = Modifier.clickable { onOpenUrl(task.prUrl) },
                    )
                    val state = prState(task)
                    if (state.isNotEmpty()) {
                        Text(state, style = type.meta, color = tokens.textTertiary)
                    }
                }
            }
        }
        if (page.subtasks.isNotEmpty()) {
            item(key = "subtasks") {
                SectionHeader(stringResource(R.string.section_subtasks), page.subtasks.size)
                Column {
                    for (sub in page.subtasks) SubtaskRow(sub, words, onOpenTask)
                }
            }
        }
        if (page.reports.isNotEmpty()) {
            item(key = "activity") {
                SectionHeader(stringResource(R.string.section_activity), page.reports.size)
                ActivityRail(page.reports, words)
            }
        }
    }
}

@Composable
private fun PropertyRow(
    icon: String,
    onClick: (() -> Unit)? = null,
    leading: (@Composable () -> Unit)? = null,
    content: @Composable () -> Unit,
) {
    val tokens = LocalTokens.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(40.dp)
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Row(
            modifier = Modifier.width(104.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            if (leading != null) {
                leading()
            } else {
                PixelIcon(icon, size = Dims.icon, tint = tokens.textTertiary)
            }
            Text(
                stringResource(
                    when (icon) {
                        "invert" -> R.string.property_status
                        "robot" -> R.string.property_from
                        "flag" -> R.string.property_priority
                        "folder" -> R.string.property_project
                        "bulletlist" -> R.string.property_category
                        "clock" -> R.string.property_remind
                        "git-branch" -> R.string.property_branch
                        else -> R.string.property_pr
                    },
                ),
                style = LocalAppType.current.body,
                color = tokens.textSecondary,
            )
        }
        Row(
            modifier = Modifier.weight(1f),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            content()
        }
        if (onClick != null) {
            PixelIcon("chevron-right", size = 16.dp, tint = tokens.textTertiary)
        }
    }
}

/**
 * A task's back bar: the page it came from, the ⋯ that opens the whole task menu, and the number
 * at the right in mono (v30 frame 4).
 */
@Composable
fun TaskBackBar(title: String, number: Long, onBack: () -> Unit, onMenu: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val menuLabel = stringResource(R.string.task_menu)
    BackBar(
        title = title,
        onBack = onBack,
        right = {
            Box(
                Modifier
                    .size(40.dp)
                    .clickable(onClick = onMenu)
                    .semantics { contentDescription = menuLabel },
                contentAlignment = Alignment.Center,
            ) {
                MoreGlyph(size = Dims.icon, tint = tokens.textSecondary)
            }
            Text(
                "#$number",
                style = type.mono,
                color = tokens.textTertiary,
                modifier = Modifier.padding(end = 8.dp),
            )
        },
    )
}

@Composable
private fun PriorityStrip(priority: String, onSetPriority: (String) -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        for (value in listOf("P0", "P1", "P2", "P3")) {
            val selected = value == priority
            Box(
                modifier = Modifier
                    .height(22.dp)
                    .background(
                        if (selected) tokens.bgSelected else tokens.bgSubtle,
                        RoundedCornerShape(5.dp),
                    )
                    .clickable { onSetPriority(value) }
                    .padding(horizontal = 8.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    value,
                    style = type.badge,
                    color = if (selected) tokens.text else tokens.textSecondary,
                )
            }
        }
    }
}

/** The words after the PR number: checks or review state, as the poller last found them. */
@Composable
private fun prState(task: Task): String = when {
    task.prState == "merged" -> stringResource(R.string.pr_state_merged)
    task.prState == "closed" -> stringResource(R.string.pr_state_closed)
    task.prChecks == "failing" -> stringResource(R.string.pr_state_failing)
    task.prChecks == "pending" -> stringResource(R.string.pr_state_pending)
    task.prChangesRequested -> stringResource(R.string.pr_state_changes)
    task.prReview == "approved" -> stringResource(R.string.pr_state_approved)
    task.prReview == "required" -> stringResource(R.string.pr_state_review)
    else -> ""
}

/** The task's pull request asks for changes; the Go side calls this PRChangesRequested. */
private val Task.prChangesRequested: Boolean
    get() = prState == "open" && prReview == "changes_requested"

@Composable
private fun SubtaskRow(sub: Task, words: Words, onOpenTask: (Long) -> Unit) {
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
        Text(words.ago(sub.updatedAt), style = type.meta, color = tokens.textTertiary, maxLines = 1)
    }
}

@Composable
private fun ActivityRail(reports: List<Report>, words: Words) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .drawBehind {
                val x = 9.dp.toPx()
                drawLine(
                    color = tokens.border,
                    start = Offset(x, 10.dp.toPx()),
                    end = Offset(x, size.height - 6.dp.toPx()),
                    strokeWidth = Dims.hairline.toPx(),
                )
            },
    ) {
        for (report in reports) {
            Row(
                modifier = Modifier.fillMaxWidth().padding(bottom = 10.dp),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.Top,
            ) {
                Box(modifier = Modifier.width(28.dp), contentAlignment = Alignment.TopStart) {
                    ReportAvatar(report, words)
                }
                Column(Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            words.agent(report.agent),
                            style = type.meta,
                            color = if (report.status == "blocked") tokens.statusBlocked else tokens.textSecondary,
                        )
                        Text(
                            words.ago(report.createdAt),
                            style = type.meta,
                            color = tokens.textTertiary,
                            modifier = Modifier.padding(start = 6.dp),
                        )
                    }
                    if (report.summary.isNotEmpty()) {
                        Text(
                            report.summary,
                            style = type.body,
                            color = tokens.text,
                            modifier = Modifier.padding(top = 2.dp),
                        )
                    }
                    if (report.nextStep.isNotEmpty()) {
                        Text(
                            stringResource(R.string.activity_next, report.nextStep),
                            style = type.meta,
                            color = tokens.textSecondary,
                            modifier = Modifier.padding(top = 2.dp),
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ReportAvatar(report: Report, words: Words) {
    val tokens = LocalTokens.current
    when (report.agent) {
        "github" -> PixelIcon("github", size = Dims.avatar, tint = tokens.textSecondary)
        "you" -> Avatar("you", size = Dims.avatar)
        else -> Avatar(words.avatar(report.agent), size = Dims.avatar)
    }
}
