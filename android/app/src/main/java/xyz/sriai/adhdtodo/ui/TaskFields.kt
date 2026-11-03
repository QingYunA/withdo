package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import java.time.OffsetDateTime
import java.time.ZoneId
import java.util.Locale

/*
 * The parts the task page gained in v43: the Due value, the Notes field and the Add subtask row.
 */

/**
 * The Due row's value: `Fri, Nov 7` / `11月7日 周五`, with the time of day after it when the due
 * time is not the default end of the working day.
 */
fun dueValue(task: Task, words: Words, locale: Locale, zone: ZoneId = ZoneId.systemDefault()): String {
    val day = dueDay(task.dueAt, zone) ?: return ""
    val local = runCatching { OffsetDateTime.parse(task.dueAt).atZoneSameInstant(zone).toLocalTime() }.getOrNull()
    val date = dueRowDate(day, locale)
    return if (local == null || local == DefaultDueTime) date else "$date ${words.timeOfDay(task.dueAt)}"
}

/**
 * The task's notes, edited in place. Empty it shows `Add notes…`; a tap puts the caret in it; the
 * text is saved when the field loses focus (or the page closes with the caret still in it).
 */
@Composable
fun NotesField(task: Task, onSave: (String) -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    var text by remember(task.id) { mutableStateOf(task.notes) }
    var focused by remember { mutableStateOf(false) }
    // A synced change shows up unless the person is typing.
    LaunchedEffect(task.notes) { if (!focused) text = task.notes }
    val saved by rememberUpdatedState(task.notes)
    val current by rememberUpdatedState(text)
    val save by rememberUpdatedState(onSave)
    DisposableEffect(task.id) {
        onDispose { if (focused && current != saved) save(current) }
    }
    BasicTextField(
        value = text,
        onValueChange = { text = it },
        textStyle = type.body.copy(color = tokens.text),
        cursorBrush = SolidColor(tokens.accent),
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = 44.dp)
            .onFocusChanged {
                if (focused && !it.isFocused && text != task.notes) onSave(text)
                focused = it.isFocused
            },
        decorationBox = { inner ->
            Box(Modifier.padding(vertical = 10.dp), contentAlignment = Alignment.CenterStart) {
                if (text.isEmpty()) {
                    Text(stringResource(R.string.notes_hint), style = type.body, color = tokens.textTertiary)
                }
                inner()
            }
        },
    )
}

/**
 * The last row of Subtasks: `+ Add subtask`. A tap turns it into a field; Enter adds the subtask
 * and keeps the field open for the next one; leaving the field empty puts the row back.
 */
@Composable
fun AddSubtaskRow(onAdd: (String) -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    var open by remember { mutableStateOf(false) }
    var text by remember { mutableStateOf("") }
    val focus = remember { FocusRequester() }
    var wasFocused by remember { mutableStateOf(false) }
    LaunchedEffect(open) { if (open) focus.requestFocus() }
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(44.dp)
            .then(if (open) Modifier else Modifier.clickable { open = true }),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        PixelIcon("plus", size = 18.dp, tint = tokens.textTertiary)
        if (!open) {
            Text(
                stringResource(R.string.add_subtask),
                style = type.body,
                color = tokens.textTertiary,
                modifier = Modifier.padding(start = 12.dp),
            )
        } else {
            BasicTextField(
                value = text,
                onValueChange = { text = it },
                textStyle = type.body.copy(color = tokens.text),
                cursorBrush = SolidColor(tokens.accent),
                singleLine = true,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = {
                    val title = text.trim()
                    if (title.isNotEmpty()) {
                        onAdd(title)
                        text = ""
                    }
                }),
                modifier = Modifier
                    .weight(1f)
                    .padding(start = 12.dp)
                    .focusRequester(focus)
                    .onFocusChanged {
                        if (wasFocused && !it.isFocused && text.isBlank()) open = false
                        wasFocused = it.isFocused
                    },
                decorationBox = { inner ->
                    Box(contentAlignment = Alignment.CenterStart) {
                        if (text.isEmpty()) {
                            Text(stringResource(R.string.add_subtask), style = type.body, color = tokens.textTertiary)
                        }
                        inner()
                    }
                },
            )
        }
    }
}
