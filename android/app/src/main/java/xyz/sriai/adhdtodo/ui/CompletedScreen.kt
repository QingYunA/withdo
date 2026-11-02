package xyz.sriai.adhdtodo.ui

import android.util.Log
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
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.CompletedPage
import xyz.sriai.adhdtodo.data.CompletedTask
import xyz.sriai.adhdtodo.data.CoreApi
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.StatusGlyph

/** The inset of a row's text from the list's edge, as on Today. */
private val RowInset = 12.dp

/** How many tasks one `Show older` adds, and the core's default page. */
private const val PageSize = 100

/**
 * The Completed route: it loads the scope's page, 100 tasks at a time, and again when the data
 * changes. [scopeName] is the project's name (empty for everything and for Agents' own scope word).
 */
@Composable
fun CompletedRoute(
    api: CoreApi,
    scope: CompletedScope,
    scopeName: String,
    words: Words,
    revision: Int,
    tick: Int,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
    onReopen: (Long) -> Unit,
) {
    var limit by rememberSaveable(scope) { mutableIntStateOf(PageSize) }
    var page by remember(scope) { mutableStateOf<CompletedPage?>(null) }
    LaunchedEffect(scope, limit, revision, tick) {
        try {
            page = withContext(Dispatchers.IO) { api.completed(scope.projectId, scope.agents, limit.toLong()) }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            Log.w("adhd-todo", "completed failed", e)
        }
    }
    CompletedScreen(
        page = page,
        subtitle = if (scopeName.isEmpty()) {
            stringResource(R.string.completed_subtitle)
        } else {
            stringResource(R.string.completed_subtitle_in, scopeName)
        },
        words = words,
        onOpenTask = onOpenTask,
        onLongPress = onLongPress,
        onReopen = onReopen,
        onShowOlder = { limit += PageSize },
    )
}

/**
 * Completed: the title and its quiet line, then the day groups newest first, each row a glyph that
 * reopens, the title, the project, and at the right who finished it and when. `Show older` ends it.
 */
@Composable
fun CompletedScreen(
    page: CompletedPage?,
    subtitle: String,
    words: Words,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
    onReopen: (Long) -> Unit,
    onShowOlder: () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = 16.dp, end = 16.dp, bottom = 16.dp),
    ) {
        item(key = "title") {
            Column(Modifier.padding(start = RowInset, top = 4.dp)) {
                Text(stringResource(R.string.completed_title), style = type.title, color = tokens.text)
                Text(
                    subtitle,
                    style = type.meta,
                    color = tokens.textSecondary,
                    modifier = Modifier.padding(top = 2.dp),
                )
            }
        }
        if (page == null) return@LazyColumn
        if (page.days.isEmpty()) {
            item(key = "empty") {
                EmptyState(
                    title = stringResource(R.string.completed_empty_title),
                    body = stringResource(R.string.completed_empty_body),
                    top = 56.dp,
                )
            }
        }
        for (day in page.days) {
            item(key = "day-${day.day}") {
                SectionHeader(words.dayLabel(day.day), day.tasks.size, start = RowInset)
            }
            items(day.tasks.size, key = { "task-${day.tasks[it].id}" }) { index ->
                CompletedRow(day.tasks[index], words, onOpenTask, onLongPress, onReopen)
            }
        }
        if (page.more) {
            item(key = "older") {
                TextLink(stringResource(R.string.completed_older), start = RowInset, onClick = onShowOlder)
            }
        }
    }
}

/** A finished task, 48 dp: the glyph (a button that reopens), title, project; who finished it and the time at the right. */
@Composable
private fun CompletedRow(
    task: CompletedTask,
    words: Words,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
    onReopen: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    LongPressRow(
        onClick = { onOpenTask(task.id) },
        onLongPress = { onLongPress(task.id) },
    ) { row ->
        Row(
            modifier = row.height(Dims.rowHeight).padding(horizontal = RowInset),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            StatusGlyph(status = task.status, onClick = { onReopen(task.id) })
            Row(
                modifier = Modifier.weight(1f),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(
                    task.title,
                    style = type.body,
                    color = tokens.textSecondary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
                if (task.project.isNotEmpty()) {
                    Text(
                        task.project,
                        style = type.meta,
                        color = tokens.textTertiary,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.widthIn(max = 88.dp),
                    )
                }
            }
            Avatar(if (task.by.isBlank()) "you" else words.avatar(task.by), size = 14.dp)
            Text(words.timeOfDay(task.completedAt), style = type.meta, color = tokens.textSecondary, maxLines = 1)
        }
    }
}

/** An accent text link at the end of a list: `Show all`, `Show older`. */
@Composable
fun TextLink(label: String, start: Dp, onClick: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Box(Modifier.fillMaxWidth().padding(top = 4.dp)) {
        Text(
            label,
            style = type.body,
            color = tokens.accentInk,
            modifier = Modifier
                .padding(start = start)
                .clip(RoundedCornerShape(8.dp))
                .clickable(onClick = onClick)
                .padding(vertical = 10.dp),
        )
    }
}
