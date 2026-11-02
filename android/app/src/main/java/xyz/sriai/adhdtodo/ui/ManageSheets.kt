package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneOffset

/*
 * The small modal pieces the Projects, Board and Categories pages share: the bottom sheet frame of
 * the task menu (28 dp top corners, a 32 x 4 dp handle), its 52 dp rows, a one-field sheet, a date
 * picker and the desktop's confirmation as a Material dialog.
 */

/** The task menu's sheet: `bg`, 28 dp top corners, a handle, the scrim at 32 % black. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SheetFrame(onDismiss: () -> Unit, content: @Composable ColumnScope.() -> Unit) {
    val tokens = LocalTokens.current
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        containerColor = tokens.bg,
        scrimColor = tokens.scrim,
        shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp),
        dragHandle = {
            Box(Modifier.fillMaxWidth().height(22.dp), contentAlignment = Alignment.Center) {
                Box(
                    Modifier
                        .size(width = 32.dp, height = 4.dp)
                        .background(tokens.textTertiary.copy(alpha = 0.7f), RoundedCornerShape(2.dp)),
                )
            }
        },
    ) {
        Column(
            Modifier
                .fillMaxWidth()
                .imePadding()
                .verticalScroll(rememberScrollState())
                .padding(bottom = 18.dp),
            content = content,
        )
    }
}

/** The sheet's top: the name in 17 sp 600 and a quiet `meta` line under it. */
@Composable
fun SheetTitle(title: String, meta: String? = null) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Column(Modifier.padding(start = 20.dp, end = 20.dp, top = 4.dp, bottom = 10.dp)) {
        Text(
            title,
            style = type.bodyStrong.copy(fontSize = 17.sp, lineHeight = 24.sp),
            color = tokens.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        if (!meta.isNullOrEmpty()) {
            Text(
                meta,
                style = type.meta,
                color = tokens.textTertiary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.padding(top = 2.dp),
            )
        }
    }
}

@Composable
fun SheetRule() {
    Box(
        Modifier
            .fillMaxWidth()
            .padding(horizontal = 20.dp, vertical = 10.dp)
            .height(Dims.hairline)
            .background(LocalTokens.current.border),
    )
}

/**
 * One 52 dp row: a 20 dp pixelarticon (or [leading]), the label, and at the right a hint, a
 * chevron, a check or [trailing]. The chosen row takes `bg.hover`; a danger row is blocked red.
 */
@Composable
fun SheetRow(
    label: String,
    icon: String? = null,
    hint: String? = null,
    chevron: Boolean = false,
    checked: Boolean = false,
    danger: Boolean = false,
    selected: Boolean = false,
    leading: (@Composable () -> Unit)? = null,
    trailing: (@Composable () -> Unit)? = null,
    onClick: (() -> Unit)? = null,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        Modifier
            .fillMaxWidth()
            .height(52.dp)
            .then(if (selected) Modifier.background(tokens.bgHover) else Modifier)
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier)
            .padding(horizontal = 20.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        when {
            leading != null -> leading()
            icon != null -> PixelIcon(icon, size = 20.dp, tint = if (danger) tokens.statusBlocked else tokens.textSecondary)
        }
        Text(
            label,
            style = type.body.copy(fontSize = 16.sp),
            color = if (danger) tokens.statusBlocked else tokens.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
        if (!hint.isNullOrEmpty()) {
            Text(hint, style = type.meta, color = tokens.textTertiary, maxLines = 1)
        }
        if (checked) PixelIcon("check", size = 20.dp, tint = tokens.accent)
        trailing?.invoke()
        if (chevron) PixelIcon("chevron-right", size = 14.dp, tint = tokens.textTertiary)
    }
}

/** A line of explanation or a problem under a field, never red (the desktop's rule). */
@Composable
fun SheetNote(text: String, modifier: Modifier = Modifier) {
    Text(
        text,
        style = LocalAppType.current.meta,
        color = LocalTokens.current.textSecondary,
        modifier = modifier.padding(horizontal = 20.dp, vertical = 4.dp),
    )
}

/** A 48 dp field on `bg.subtle`, radius 12, the keyboard up; Enter runs [onDone]. */
@Composable
fun SheetField(
    value: TextFieldValue,
    onValueChange: (TextFieldValue) -> Unit,
    placeholder: String,
    onDone: () -> Unit,
    modifier: Modifier = Modifier,
    singleLine: Boolean = true,
    focus: Boolean = true,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val requester = remember { FocusRequester() }
    LaunchedEffect(Unit) { if (focus) runCatching { requester.requestFocus() } }
    Row(
        modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 4.dp)
            .height(48.dp)
            .background(tokens.bgSubtle, RoundedCornerShape(12.dp))
            .padding(horizontal = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            textStyle = type.input.copy(color = tokens.text),
            cursorBrush = SolidColor(tokens.accent),
            singleLine = singleLine,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
            keyboardActions = KeyboardActions(onDone = { onDone() }),
            modifier = Modifier.weight(1f).focusRequester(requester),
            decorationBox = { inner ->
                Box(contentAlignment = Alignment.CenterStart) {
                    if (value.text.isEmpty()) {
                        Text(placeholder, style = type.input, color = tokens.textTertiary, maxLines = 1)
                    }
                    inner()
                }
            },
        )
    }
}

/**
 * A sheet with one name field and a primary button: New project, Rename…, New sub-project, New
 * milestone, Rename milestone. [onConfirm] may throw; its message reads under the field. With
 * [withDue] the sheet also offers the milestone's optional due date.
 */
@Composable
fun PromptSheet(
    title: String,
    placeholder: String,
    confirm: String,
    onDismiss: () -> Unit,
    onConfirm: suspend (name: String, due: LocalDate?) -> Unit,
    initial: String = "",
    subtitle: String? = null,
    withDue: Boolean = false,
) {
    val scope = rememberCoroutineScope()
    var field by remember { mutableStateOf(TextFieldValue(initial, TextRange(0, initial.length))) }
    var due by remember { mutableStateOf<LocalDate?>(null) }
    var picking by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val ready = field.text.isNotBlank() && !busy

    fun submit() {
        if (!ready) return
        busy = true
        scope.launch {
            try {
                onConfirm(field.text.trim(), due)
                onDismiss()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                error = e.message
                busy = false
            }
        }
    }

    SheetFrame(onDismiss) {
        SheetTitle(title)
        if (!subtitle.isNullOrEmpty()) SheetNote(subtitle)
        SheetField(field, { field = it; error = null }, placeholder, onDone = ::submit)
        error?.let { SheetNote(it) }
        if (withDue) {
            SheetRow(
                label = stringResource(R.string.milestone_due_row),
                icon = "calendar",
                hint = due?.let { dueLabel(it) } ?: stringResource(R.string.milestone_no_due),
                chevron = true,
                onClick = { picking = true },
            )
        }
        PrimaryButton(
            label = confirm,
            enabled = ready,
            onClick = ::submit,
            modifier = Modifier.fillMaxWidth().padding(start = 16.dp, end = 16.dp, top = 12.dp),
        )
    }
    if (picking) {
        DueDateDialog(
            initial = due,
            onPick = { due = it; picking = false },
            onClear = { due = null; picking = false },
            onDismiss = { picking = false },
        )
    }
}

/** Material's date picker with a Clear action; the date is a plain day, not an instant. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DueDateDialog(
    initial: LocalDate?,
    onPick: (LocalDate) -> Unit,
    onClear: () -> Unit,
    onDismiss: () -> Unit,
) {
    val tokens = LocalTokens.current
    val state = rememberDatePickerState(
        initialSelectedDateMillis = (initial ?: LocalDate.now()).atStartOfDay(ZoneOffset.UTC).toInstant().toEpochMilli(),
    )
    DatePickerDialog(
        onDismissRequest = onDismiss,
        confirmButton = {
            TextButton(onClick = {
                state.selectedDateMillis?.let { onPick(Instant.ofEpochMilli(it).atZone(ZoneOffset.UTC).toLocalDate()) }
            }) { Text(stringResource(R.string.action_done), color = tokens.accentInk) }
        },
        dismissButton = {
            TextButton(onClick = onClear) { Text(stringResource(R.string.milestone_clear_due), color = tokens.textSecondary) }
        },
    ) { DatePicker(state = state) }
}

/** The desktop's confirmation as a Material dialog: `Cancel` and `Delete` (accent, not red). */
@Composable
fun DeleteDialog(title: String, body: String, onConfirm: () -> Unit, onDismiss: () -> Unit) {
    val tokens = LocalTokens.current
    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = tokens.bg,
        titleContentColor = tokens.text,
        textContentColor = tokens.textSecondary,
        title = { Text(title) },
        text = { Text(body) },
        confirmButton = {
            TextButton(onClick = onConfirm) { Text(stringResource(R.string.manage_delete_confirm), color = tokens.accentInk) }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text(stringResource(R.string.action_cancel), color = tokens.textSecondary) }
        },
    )
}

/** A header icon button: 40 dp target, an 18 dp pixelarticon in `text.secondary`. */
@Composable
fun HeaderIconButton(icon: String, description: String, onClick: () -> Unit, size: Dp = 40.dp) {
    val tokens = LocalTokens.current
    Box(
        Modifier
            .size(size)
            .clickable(onClick = onClick)
            .semanticsLabel(description),
        contentAlignment = Alignment.Center,
    ) {
        PixelIcon(icon, size = Dims.icon, tint = tokens.textSecondary)
    }
}

private fun Modifier.semanticsLabel(label: String): Modifier =
    this.semantics { contentDescription = label }

/** A due date as the sheets and the filter say it: `Nov 14` / `11月14日`. */
fun dueLabel(day: LocalDate, locale: java.util.Locale = java.util.Locale.getDefault()): String =
    java.time.format.DateTimeFormatter
        .ofPattern(android.text.format.DateFormat.getBestDateTimePattern(locale, "MMMd"), locale)
        .format(day)
