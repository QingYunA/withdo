package xyz.sriai.adhdtodo.ui

import android.text.format.DateFormat
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TimePicker
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.material3.rememberTimePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.RemindPreset
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.ui.theme.CategoryIcon
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelGrid
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import xyz.sriai.adhdtodo.ui.theme.StatusGlyph
import java.time.Instant
import java.time.LocalDate
import java.time.LocalTime
import java.time.ZoneId
import java.time.ZoneOffset

/** The main page of the sheet and its three subpages. */
enum class TaskMenuPage { Main, Category, Move, Remind }

/** The seven statuses, in the desktop menu's order. */
private val menuStatuses = listOf("backlog", "todo", "doing", "blocked", "review", "done", "cancelled")

/**
 * The desktop's task menu as a modal bottom sheet (v30): the title and number, the status and
 * priority strips, and the Category, Move to…, Remind me…, Open PR and Delete rows. A subpage
 * replaces the body; back goes to the main page, then closes the sheet. A choice calls its callback,
 * gives a light haptic and closes the sheet.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TaskMenuSheet(
    task: Task,
    words: Words,
    agent: String = "",
    categories: List<Category> = emptyList(),
    projects: List<Project> = emptyList(),
    presets: List<RemindPreset> = emptyList(),
    initialPage: TaskMenuPage = TaskMenuPage.Main,
    onDismiss: () -> Unit,
    onStatus: (String) -> Unit = {},
    onPriority: (String) -> Unit = {},
    onCategory: (Int) -> Unit = {},
    onMove: (Long?) -> Unit = {},
    onRemind: (Long?) -> Unit = {},
    onOpenUrl: (String) -> Unit = {},
    onDelete: () -> Unit = {},
) {
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
        TaskMenuBody(
            task = task,
            words = words,
            agent = agent,
            categories = categories,
            projects = projects,
            presets = presets,
            initialPage = initialPage,
            onDismiss = onDismiss,
            onStatus = onStatus,
            onPriority = onPriority,
            onCategory = onCategory,
            onMove = onMove,
            onRemind = onRemind,
            onOpenUrl = onOpenUrl,
            onDelete = onDelete,
        )
    }
}

@Composable
private fun TaskMenuBody(
    task: Task,
    words: Words,
    agent: String,
    categories: List<Category>,
    projects: List<Project>,
    presets: List<RemindPreset>,
    initialPage: TaskMenuPage,
    onDismiss: () -> Unit,
    onStatus: (String) -> Unit,
    onPriority: (String) -> Unit,
    onCategory: (Int) -> Unit,
    onMove: (Long?) -> Unit,
    onRemind: (Long?) -> Unit,
    onOpenUrl: (String) -> Unit,
    onDelete: () -> Unit,
) {
    var page by remember { mutableStateOf(initialPage) }
    // Back goes from a page to the main page, then closes the sheet. The handler also covers the
    // main page, so a dialog window's own back fallback never closes the sheet behind our back.
    BackHandler {
        if (page == TaskMenuPage.Main) onDismiss() else page = TaskMenuPage.Main
    }
    val haptic = LocalHapticFeedback.current

    fun choose(action: () -> Unit) {
        action()
        onDismiss()
    }

    Column(
        Modifier
            .fillMaxWidth()
            .imePadding()
            .verticalScroll(rememberScrollState())
            .padding(bottom = 18.dp),
    ) {
        when (page) {
            TaskMenuPage.Main -> MainPage(
                task = task,
                words = words,
                agent = agent,
                categories = categories,
                projects = projects,
                onStatus = { status ->
                    haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                    choose { onStatus(status) }
                },
                onPriority = { priority ->
                    haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                    choose { onPriority(priority) }
                },
                onCategory = { page = TaskMenuPage.Category },
                onMove = { page = TaskMenuPage.Move },
                onRemind = { page = TaskMenuPage.Remind },
                onOpenUrl = { url -> choose { onOpenUrl(url) } },
                onDelete = { choose { onDelete() } },
            )
            TaskMenuPage.Category -> CategoryPage(
                task = task,
                categories = categories,
                words = words,
                onBack = { page = TaskMenuPage.Main },
                onPick = { id -> choose { onCategory(id) } },
            )
            TaskMenuPage.Move -> MovePage(
                task = task,
                projects = projects,
                words = words,
                onBack = { page = TaskMenuPage.Main },
                onPick = { id -> choose { onMove(id) } },
            )
            TaskMenuPage.Remind -> RemindPage(
                task = task,
                presets = presets,
                words = words,
                onBack = { page = TaskMenuPage.Main },
                onPick = { at -> choose { onRemind(at) } },
            )
        }
    }
}

@Composable
private fun MainPage(
    task: Task,
    words: Words,
    agent: String,
    categories: List<Category>,
    projects: List<Project>,
    onStatus: (String) -> Unit,
    onPriority: (String) -> Unit,
    onCategory: () -> Unit,
    onMove: () -> Unit,
    onRemind: () -> Unit,
    onOpenUrl: (String) -> Unit,
    onDelete: () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val category = categories.firstOrNull { it.id.toInt() == task.categoryId }
    val project = projects.firstOrNull { it.id == task.projectId }
    val meta = buildString {
        append('#').append(task.id)
        if (agent.isNotEmpty()) append(" · ").append(words.agent(agent))
        append(" · ").append(project?.name ?: stringResource(R.string.title_inbox))
    }

    Column(Modifier.padding(start = 20.dp, end = 20.dp, top = 4.dp, bottom = 10.dp)) {
        Text(
            task.title,
            style = type.bodyStrong.copy(fontSize = 17.sp, lineHeight = 24.sp),
            color = tokens.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        Text(meta, style = type.meta, color = tokens.textTertiary, modifier = Modifier.padding(top = 2.dp))
    }
    MenuLabel(stringResource(R.string.property_status), words.menuStatus(task.status))
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 14.dp),
        horizontalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        for (status in menuStatuses) {
            val selected = status == task.status
            Box(
                Modifier
                    .weight(1f)
                    .height(52.dp)
                    .then(if (selected) Modifier.background(tokens.bgSelected, RoundedCornerShape(12.dp)) else Modifier)
                    .clickable { onStatus(status) }
                    .semantics { contentDescription = words.menuStatus(status) },
                contentAlignment = Alignment.Center,
            ) {
                StatusGlyph(status, size = 24.dp)
            }
        }
    }
    MenuLabel(
        stringResource(R.string.property_priority),
        "${task.priority} ${words.priorityName(task.priority)}".trim(),
        top = 6.dp,
    )
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 14.dp),
        horizontalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        for (priority in listOf("P0", "P1", "P2", "P3")) {
            val selected = priority == task.priority
            val tint = when {
                selected && priority == "P0" -> tokens.statusBlocked.copy(alpha = 0.12f)
                selected -> tokens.bgSelected
                else -> null
            }
            Box(
                Modifier
                    .weight(1f)
                    .height(44.dp)
                    .then(if (tint != null) Modifier.background(tint, RoundedCornerShape(12.dp)) else Modifier)
                    .clickable { onPriority(priority) }
                    .semantics { contentDescription = priority },
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    priority,
                    style = type.bodyStrong.copy(fontSize = 15.sp, lineHeight = 22.sp),
                    color = when {
                        priority == "P0" -> tokens.statusBlocked
                        selected -> tokens.accent
                        else -> tokens.text
                    },
                )
            }
        }
    }
    MenuRule()
    MenuRow(
        icon = "bulletlist",
        label = stringResource(R.string.property_category),
        hint = category?.let { words.categoryName(it) } ?: stringResource(R.string.menu_none),
        chevron = true,
        onClick = onCategory,
    )
    MenuRow(
        icon = "arrow-right",
        label = stringResource(R.string.menu_move_to),
        hint = project?.name ?: stringResource(R.string.title_inbox),
        chevron = true,
        onClick = onMove,
    )
    MenuRow(
        icon = "clock",
        label = stringResource(R.string.menu_remind_me),
        hint = task.remindAt?.takeIf { it.isNotBlank() }?.let { words.reminderValue(it) }
            ?: stringResource(R.string.menu_none),
        chevron = true,
        onClick = onRemind,
    )
    if (task.prNumber > 0 && task.prUrl.isNotEmpty()) {
        MenuRow(
            icon = "external-link",
            label = stringResource(R.string.menu_open_pr, task.prNumber),
            hint = "GitHub",
            onClick = { onOpenUrl(task.prUrl) },
        )
    }
    MenuRule()
    MenuRow(
        icon = "trash",
        label = stringResource(R.string.menu_delete),
        danger = true,
        onClick = onDelete,
    )
}

@Composable
private fun CategoryPage(
    task: Task,
    categories: List<Category>,
    words: Words,
    onBack: () -> Unit,
    onPick: (Int) -> Unit,
) {
    var query by remember { mutableStateOf("") }
    PageHeader(stringResource(R.string.property_category), onBack)
    FilterField(query, { query = it }, stringResource(R.string.menu_find_category))
    val shown = categories.filter { it.name.contains(query, ignoreCase = true) }
    if (shown.isEmpty()) {
        EmptyMatch(stringResource(R.string.menu_no_category_match))
    } else {
        for (category in shown) {
            MenuRow(
                label = words.categoryName(category),
                selected = category.id.toInt() == task.categoryId,
                leading = { CategoryIcon(category, size = 20.dp) },
                trailing = if (category.id.toInt() == task.categoryId) {
                    { PixelIcon("check", size = 20.dp, tint = LocalTokens.current.accent) }
                } else {
                    null
                },
                onClick = { onPick(category.id.toInt()) },
            )
        }
    }
    MenuRow(
        label = stringResource(R.string.menu_none),
        selected = task.categoryId == 0,
        leading = { PixelIcon("close", size = 20.dp, tint = LocalTokens.current.textSecondary) },
        trailing = if (task.categoryId == 0) {
            { PixelIcon("check", size = 20.dp, tint = LocalTokens.current.accent) }
        } else {
            null
        },
        onClick = { onPick(0) },
    )
}

@Composable
private fun MovePage(
    task: Task,
    projects: List<Project>,
    words: Words,
    onBack: () -> Unit,
    onPick: (Long?) -> Unit,
) {
    var query by remember { mutableStateOf("") }
    PageHeader(stringResource(R.string.menu_move_to), onBack)
    FilterField(query, { query = it }, stringResource(R.string.menu_find_project))
    val shown = projects.filter { it.name.contains(query, ignoreCase = true) }
    if (shown.isEmpty()) {
        EmptyMatch(stringResource(R.string.menu_no_project_match))
    } else {
        for (project in shown) {
            MenuRow(
                label = project.name,
                selected = project.id == task.projectId,
                leading = {
                    PixelIcon("folder", size = 20.dp, tint = LocalTokens.current.projectColor(project.name))
                },
                trailing = if (project.id == task.projectId) {
                    { PixelIcon("check", size = 20.dp, tint = LocalTokens.current.accent) }
                } else {
                    null
                },
                onClick = { onPick(project.id) },
            )
        }
    }
    MenuRow(
        label = stringResource(R.string.title_inbox),
        selected = task.projectId == null,
        leading = { PixelIcon("inbox", size = 20.dp, tint = LocalTokens.current.textSecondary) },
        trailing = if (task.projectId == null) {
            { PixelIcon("check", size = 20.dp, tint = LocalTokens.current.accent) }
        } else {
            null
        },
        onClick = { onPick(null) },
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun RemindPage(
    task: Task,
    presets: List<RemindPreset>,
    words: Words,
    onBack: () -> Unit,
    onPick: (Long?) -> Unit,
) {
    var showDate by remember { mutableStateOf(false) }
    var showTime by remember { mutableStateOf(false) }
    var picked by remember { mutableStateOf<LocalDate?>(null) }
    PageHeader(stringResource(R.string.menu_remind_title), onBack)
    for (preset in presets) {
        MenuRow(
            icon = "clock",
            label = when (preset.kind) {
                "later" -> stringResource(R.string.menu_later_today)
                "tomorrow" -> stringResource(R.string.menu_tomorrow)
                "monday" -> stringResource(R.string.menu_next_monday)
                "weekend" -> stringResource(R.string.menu_this_weekend)
                else -> preset.kind
            },
            hint = words.remindPresetHint(preset.kind, preset.at),
            onClick = { onPick(preset.at) },
        )
    }
    MenuRule()
    MenuRow(
        label = stringResource(R.string.menu_no_reminder),
        leading = { Spacer(Modifier.size(20.dp)) },
        onClick = { onPick(null) },
    )
    MenuRow(
        icon = "calendar",
        label = stringResource(R.string.menu_custom),
        chevron = true,
        onClick = { showDate = true },
    )

    if (showDate) {
        val initial = task.remindAt?.let { words.instant(it) }
            ?.atZone(ZoneId.systemDefault())?.toLocalDate()
            ?.atStartOfDay(ZoneOffset.UTC)?.toInstant()?.toEpochMilli()
            ?: LocalDate.now().atStartOfDay(ZoneOffset.UTC).toInstant().toEpochMilli()
        val state = rememberDatePickerState(initialSelectedDateMillis = initial)
        DatePickerDialog(
            onDismissRequest = { showDate = false },
            confirmButton = {
                TextButton(onClick = {
                    picked = state.selectedDateMillis
                        ?.let { Instant.ofEpochMilli(it).atZone(ZoneOffset.UTC).toLocalDate() }
                        ?: LocalDate.now()
                    showDate = false
                    showTime = true
                }) {
                    Text(stringResource(R.string.action_done), color = LocalTokens.current.accent)
                }
            },
            dismissButton = {
                TextButton(onClick = { showDate = false }) {
                    Text(stringResource(R.string.action_cancel), color = LocalTokens.current.textSecondary)
                }
            },
        ) {
            DatePicker(state = state)
        }
    }
    if (showTime) {
        val initial = task.remindAt?.let { words.instant(it) }?.atZone(ZoneId.systemDefault())?.toLocalTime()
            ?: LocalTime.of(9, 0)
        val state = rememberTimePickerState(
            initialHour = initial.hour,
            initialMinute = initial.minute,
            is24Hour = DateFormat.is24HourFormat(LocalContext.current),
        )
        Dialog(onDismissRequest = { showTime = false }) {
            Surface(color = LocalTokens.current.bg, shape = RoundedCornerShape(28.dp)) {
                Column(Modifier.padding(24.dp)) {
                    Text(
                        stringResource(R.string.menu_remind_title),
                        style = LocalAppType.current.bodyStrong,
                        color = LocalTokens.current.text,
                    )
                    TimePicker(state = state, modifier = Modifier.padding(top = 16.dp))
                    Row(
                        Modifier.fillMaxWidth().padding(top = 8.dp),
                        horizontalArrangement = Arrangement.End,
                    ) {
                        TextButton(onClick = { showTime = false }) {
                            Text(
                                stringResource(R.string.action_cancel),
                                color = LocalTokens.current.textSecondary,
                            )
                        }
                        TextButton(onClick = {
                            val date = picked ?: LocalDate.now()
                            val millis = date.atTime(state.hour, state.minute)
                                .atZone(ZoneId.systemDefault()).toInstant().toEpochMilli()
                            showTime = false
                            onPick(millis)
                        }) {
                            Text(stringResource(R.string.action_done), color = LocalTokens.current.accent)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun PageHeader(title: String, onBack: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val back = stringResource(R.string.menu_back)
    Row(
        Modifier.fillMaxWidth().height(52.dp).padding(horizontal = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Box(
            Modifier
                .size(40.dp)
                .clickable(onClick = onBack)
                .semantics { contentDescription = back },
            contentAlignment = Alignment.Center,
        ) {
            PixelIcon("chevron-left", size = 18.dp, tint = tokens.textSecondary)
        }
        Text(
            title,
            style = type.bodyStrong.copy(fontSize = 17.sp, lineHeight = 24.sp),
            color = tokens.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
private fun FilterField(value: String, onValueChange: (String) -> Unit, placeholder: String) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val focus = remember { FocusRequester() }
    val keyboard = LocalSoftwareKeyboardController.current
    LaunchedEffect(Unit) {
        focus.requestFocus()
        keyboard?.show()
    }
    Row(
        Modifier
            .fillMaxWidth()
            .padding(start = 16.dp, end = 16.dp, top = 2.dp, bottom = 8.dp)
            .height(48.dp)
            .background(tokens.bgSubtle, RoundedCornerShape(12.dp))
            .padding(horizontal = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        PixelIcon("search", size = 18.dp, tint = tokens.textTertiary)
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            textStyle = type.input.copy(color = tokens.text),
            cursorBrush = SolidColor(tokens.accent),
            singleLine = true,
            modifier = Modifier
                .weight(1f)
                .focusRequester(focus)
                .semantics { contentDescription = placeholder },
            decorationBox = { inner ->
                Box(contentAlignment = Alignment.CenterStart) {
                    if (value.isEmpty()) {
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
    }
}

@Composable
private fun EmptyMatch(text: String) {
    Text(
        text,
        style = LocalAppType.current.body,
        color = LocalTokens.current.textTertiary,
        modifier = Modifier.padding(horizontal = 20.dp, vertical = 12.dp),
    )
}

@Composable
private fun MenuLabel(label: String, value: String, top: androidx.compose.ui.unit.Dp = 8.dp) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(Modifier.padding(start = 20.dp, end = 20.dp, top = top, bottom = 6.dp)) {
        Text(label, style = type.section, color = tokens.textSecondary)
        if (value.isNotEmpty()) {
            Text(" · $value", style = type.section, color = tokens.textTertiary)
        }
    }
}

@Composable
private fun MenuRule() {
    Box(
        Modifier
            .fillMaxWidth()
            .padding(horizontal = 20.dp, vertical = 10.dp)
            .height(1.dp)
            .background(LocalTokens.current.border),
    )
}

/**
 * One 52 dp row of the sheet: a 20 dp pixelarticon, the label, and at the right a hint, a chevron
 * or a check. The chosen row of a page takes bg.hover; Delete is the one red row.
 */
@Composable
private fun MenuRow(
    label: String,
    icon: String? = null,
    hint: String? = null,
    chevron: Boolean = false,
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
            icon != null -> PixelIcon(
                icon,
                size = 20.dp,
                tint = if (danger) tokens.statusBlocked else tokens.textSecondary,
            )
        }
        Text(
            label,
            style = type.body.copy(fontSize = 16.sp, lineHeight = 22.sp),
            color = if (danger) tokens.statusBlocked else tokens.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
        if (hint != null) {
            Text(hint, style = type.body, color = tokens.textTertiary, maxLines = 1)
        }
        if (chevron) {
            PixelIcon("chevron-right", size = 16.dp, tint = tokens.textTertiary)
        }
        trailing?.invoke()
    }
}

/** The three 2 x 2 squares of the task page's ⋯ button, on the 12 x 12 grid. */
@Composable
fun MoreGlyph(size: androidx.compose.ui.unit.Dp = Dims.icon, tint: Color) {
    PixelGrid(size = size) { cell ->
        for (x in listOf(1, 5, 9)) {
            drawRect(
                color = tint,
                topLeft = Offset(x * cell, 5 * cell),
                size = Size(2 * cell, 2 * cell),
            )
        }
    }
}

/** A deleted task: its title for the snackbar, and the snapshot Undo hands back to RestoreTask. */
data class DeletedTask(val title: String, val snapshot: String)

/**
 * The delete's snackbar, above the composer: Deleted “title” and Undo, for six seconds. The App
 * holds the snapshot; the callback gets it back so Undo restores exactly what was deleted.
 */
@Composable
fun DeletedSnackbar(
    deleted: DeletedTask?,
    onUndo: (DeletedTask) -> Unit,
    modifier: Modifier = Modifier,
) {
    if (deleted == null) return
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp)
            .height(48.dp)
            .background(tokens.text, RoundedCornerShape(12.dp))
            .padding(start = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            stringResource(R.string.menu_deleted, deleted.title),
            style = type.body,
            color = tokens.bg,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
        Text(
            stringResource(R.string.menu_undo),
            style = type.bodyStrong,
            color = tokens.accent,
            modifier = Modifier
                .clickable { onUndo(deleted) }
                .padding(horizontal = 10.dp, vertical = 12.dp),
        )
    }
}
