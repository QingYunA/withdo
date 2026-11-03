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
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import xyz.sriai.adhdtodo.ui.theme.StatusGlyph

/** The inset of a result's text from the list's edge: headers and titles share it with Today. */
private val ResultInset = 12.dp

/** The header's `search` button: a 40 dp target left of the sync cloud. */
@Composable
fun SearchButton(onClick: () -> Unit) {
    val tokens = LocalTokens.current
    val label = stringResource(R.string.search_button)
    Box(
        modifier = Modifier
            .size(40.dp)
            .clip(RoundedCornerShape(10.dp))
            .clickable(onClick = onClick)
            .semantics { contentDescription = label },
        contentAlignment = Alignment.Center,
    ) {
        PixelIcon("search", size = Dims.icon, tint = tokens.textTertiary)
    }
}

/**
 * The full-screen Search page (v43): a focused 44 dp field with the clear `x` and Cancel, the quiet
 * count line, then the results grouped by project. [results] is null until the first answer for a
 * non-blank [query]; scrolling drops the keyboard.
 */
@Composable
fun SearchScreen(
    query: String,
    onQueryChange: (String) -> Unit,
    results: SearchResults?,
    words: Words,
    onCancel: () -> Unit,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
    onToggle: (Task) -> Unit,
    focusOnStart: Boolean = true,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val keyboard = LocalSoftwareKeyboardController.current
    val focusManager = LocalFocusManager.current
    val listState = rememberLazyListState()
    LaunchedEffect(listState.isScrollInProgress) {
        if (listState.isScrollInProgress) {
            keyboard?.hide()
            focusManager.clearFocus()
        }
    }
    Column(Modifier.fillMaxSize()) {
        SearchField(query, onQueryChange, onCancel, focusOnStart) {
            keyboard?.hide()
            focusManager.clearFocus()
        }
        // The last answer stays while the next one is asked for, so the page does not flash.
        val shown = results?.takeIf { query.isNotBlank() }
        LazyColumn(
            state = listState,
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(start = 16.dp, end = 16.dp, bottom = 16.dp),
        ) {
            if (shown != null && shown.shown > 0) {
                item(key = "count") {
                    Text(
                        pluralStringResource(R.plurals.search_count, shown.shown, shown.shown, shown.query),
                        style = type.meta,
                        color = tokens.textSecondary,
                        modifier = Modifier.padding(start = ResultInset, top = 4.dp, bottom = 2.dp),
                    )
                }
                for (group in shown.groups) {
                    item(key = "group-${group.projectId}") {
                        GroupHeader(group)
                    }
                    items(group.tasks.size, key = { "task-${group.tasks[it].id}" }) { index ->
                        ResultRow(group.tasks[index], shown, words, onOpenTask, onLongPress, onToggle)
                    }
                }
                if (shown.more) {
                    item(key = "more") {
                        Text(
                            stringResource(R.string.search_more, SearchLimit),
                            style = type.meta,
                            color = tokens.textTertiary,
                            modifier = Modifier.padding(start = ResultInset, top = 16.dp),
                        )
                    }
                }
            } else if (shown != null) {
                item(key = "none") {
                    EmptyState(
                        title = stringResource(R.string.search_none, shown.query),
                        body = stringResource(R.string.search_none_body),
                        top = 56.dp,
                    )
                }
            }
        }
    }
}

/** The 44 dp field (radius 12, `bg.hover`): magnifier, text, the clear `x`; Cancel in accent at its right. */
@Composable
private fun SearchField(
    query: String,
    onQueryChange: (String) -> Unit,
    onCancel: () -> Unit,
    focusOnStart: Boolean,
    onSearch: () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val keyboard = LocalSoftwareKeyboardController.current
    val focus = remember { FocusRequester() }
    val placeholder = stringResource(R.string.search_placeholder)
    val clear = stringResource(R.string.search_clear)
    LaunchedEffect(Unit) {
        if (focusOnStart) {
            focus.requestFocus()
            keyboard?.show()
        }
    }
    Row(
        modifier = Modifier.fillMaxWidth().padding(start = 16.dp, end = 4.dp, top = 8.dp, bottom = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Row(
            modifier = Modifier
                .weight(1f)
                .height(44.dp)
                .background(tokens.bgHover, RoundedCornerShape(12.dp))
                .padding(start = 12.dp, end = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            PixelIcon("search", size = Dims.icon, tint = tokens.textTertiary)
            BasicTextField(
                value = query,
                onValueChange = onQueryChange,
                textStyle = type.input.copy(color = tokens.text),
                cursorBrush = SolidColor(tokens.accent),
                singleLine = true,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                keyboardActions = KeyboardActions(onSearch = { onSearch() }),
                modifier = Modifier
                    .weight(1f)
                    .focusRequester(focus)
                    .semantics { contentDescription = placeholder },
                decorationBox = { inner ->
                    Box(contentAlignment = Alignment.CenterStart) {
                        if (query.isEmpty()) {
                            Text(
                                placeholder,
                                style = type.input,
                                color = tokens.textTertiary,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        inner()
                    }
                },
            )
            if (query.isNotEmpty()) {
                Box(
                    modifier = Modifier
                        .size(36.dp)
                        .clip(RoundedCornerShape(8.dp))
                        .clickable { onQueryChange("") }
                        .semantics { contentDescription = clear },
                    contentAlignment = Alignment.Center,
                ) {
                    PixelIcon("close", size = 14.dp, tint = tokens.textTertiary)
                }
            }
        }
        Box(
            modifier = Modifier
                .height(44.dp)
                .clip(RoundedCornerShape(10.dp))
                .clickable(onClick = onCancel)
                .padding(horizontal = 12.dp),
            contentAlignment = Alignment.Center,
        ) {
            Text(stringResource(R.string.action_cancel), style = type.body, color = tokens.accentInk)
        }
    }
}

/** A project's name with its folder, or the tray and `No project`; the count is quiet. */
@Composable
private fun GroupHeader(group: SearchGroup) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier.fillMaxWidth().padding(start = ResultInset, top = 20.dp, bottom = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        if (group.projectId != null) {
            PixelIcon("folder", size = 14.dp, tint = tokens.projectColor(group.name))
        } else {
            PixelIcon("inbox", size = 14.dp, tint = tokens.textSecondary)
        }
        Text(
            if (group.projectId != null) group.name else stringResource(R.string.search_no_project),
            style = type.bodyStrong,
            color = tokens.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f, fill = false),
        )
        Text(group.tasks.size.toString(), style = type.meta, color = tokens.textTertiary)
    }
}

/**
 * A result, 48 dp: the status glyph (a button, so a finished task can be reopened from here), the
 * title, the branch pill and `#PR`, and at the right the age of the last change. A finished row
 * keeps its glyph and reads quieter.
 */
@Composable
private fun ResultRow(
    task: Task,
    results: SearchResults,
    words: Words,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
    onToggle: (Task) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val finished = task.status == "done" || task.status == "cancelled"
    LongPressRow(
        onClick = { onOpenTask(task.id) },
        onLongPress = { onLongPress(task.id) },
    ) { row ->
        Row(
            modifier = row.height(Dims.rowHeight).padding(horizontal = ResultInset),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            StatusGlyph(status = task.status, onClick = { onToggle(task) })
            Row(
                modifier = Modifier.weight(1f),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(
                    task.title,
                    style = if (finished) type.body else type.bodyStrong,
                    color = if (finished) tokens.textSecondary else tokens.text,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
                val parent = task.parentId?.let { results.parents[it] }
                if (parent != null) {
                    Text(
                        parent,
                        style = type.meta,
                        color = tokens.textSecondary,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.widthIn(max = 96.dp),
                    )
                }
                if (task.branch.isNotEmpty()) {
                    Text(
                        task.branch,
                        style = type.mono,
                        color = tokens.textSecondary,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier
                            .widthIn(max = 104.dp)
                            .background(tokens.bgSubtle, RoundedCornerShape(5.dp))
                            .padding(horizontal = 6.dp),
                    )
                }
                if (task.prNumber > 0) {
                    Text("#${task.prNumber}", style = type.meta, color = tokens.textTertiary, maxLines = 1)
                }
            }
            Text(words.ago(task.updatedAt), style = type.meta, color = tokens.textSecondary, maxLines = 1)
        }
    }
}
