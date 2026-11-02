package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberDatePickerState
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
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalLocale
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Milestone
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import java.time.DayOfWeek
import java.time.Instant
import java.time.LocalDate
import java.time.LocalTime
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.ZoneOffset
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.time.temporal.TemporalAdjusters
import java.util.Locale

/*
 * The Due and Milestone pages of the task menu sheet (v43). TaskMenuSheet only dispatches to them;
 * the pages and the date rules live here.
 */

/** The ready-made due days of the Due page. */
enum class DueKind { Today, Tomorrow, Friday, Monday }

/** One ready-made day: what it is called and the date it means. */
data class DueOption(val kind: DueKind, val date: LocalDate)

/**
 * Today, Tomorrow, This Friday and Next Monday as of [today]. A Friday or Monday that falls on
 * Today or Tomorrow is left out, so no two rows name the same day. "This Friday" is the Friday on
 * or after today; "Next Monday" is the Monday after today.
 */
fun dueOptions(today: LocalDate): List<DueOption> {
    val tomorrow = today.plusDays(1)
    val friday = today.with(TemporalAdjusters.nextOrSame(DayOfWeek.FRIDAY))
    val monday = today.with(TemporalAdjusters.next(DayOfWeek.MONDAY))
    return buildList {
        add(DueOption(DueKind.Today, today))
        add(DueOption(DueKind.Tomorrow, tomorrow))
        if (friday != today && friday != tomorrow) add(DueOption(DueKind.Friday, friday))
        if (monday != today && monday != tomorrow && monday != friday) add(DueOption(DueKind.Monday, monday))
    }
}

/** A ready-made day's name: Today, Tomorrow, This Friday, Next Monday. */
@Composable
fun dueKindLabel(kind: DueKind): String = stringResource(
    when (kind) {
        DueKind.Today -> R.string.due_today
        DueKind.Tomorrow -> R.string.menu_tomorrow
        DueKind.Friday -> R.string.menu_this_friday
        DueKind.Monday -> R.string.menu_next_monday
    },
)

/** A deadline given as a bare day is the end of the working day, as the desktop's defaultDueHour. */
val DefaultDueTime: LocalTime = LocalTime.of(18, 0)

/**
 * The RFC 3339 time a due day is stored as: the task's own time of day when it has a due date
 * already (a time set stays), else 18:00, in [zone]. Seconds are always written.
 */
fun dueStamp(date: LocalDate, existing: String?, zone: ZoneId): String {
    val time = existing?.takeIf { it.isNotBlank() }
        ?.let { runCatching { OffsetDateTime.parse(it).atZoneSameInstant(zone).toLocalTime() }.getOrNull() }
        ?: DefaultDueTime
    return DateTimeFormatter.ISO_OFFSET_DATE_TIME.format(ZonedDateTime.of(date, time, zone).toOffsetDateTime())
}

/** The local day a due time falls on, or null when it has none. */
fun dueDay(value: String?, zone: ZoneId): LocalDate? =
    value?.takeIf { it.isNotBlank() }
        ?.let { runCatching { OffsetDateTime.parse(it).atZoneSameInstant(zone).toLocalDate() }.getOrNull() }

/** `Fri, Nov 7` or `11月7日 周五`: the date at the right of a Due row. */
fun dueRowDate(date: LocalDate, locale: Locale): String {
    val pattern = if (locale.language == "zh") "M月d日 EEE" else "EEE, MMM d"
    return DateTimeFormatter.ofPattern(pattern, locale).format(date)
}

/** `Nov 14` or `11月14日`: a milestone's date. */
fun milestoneDate(value: String?, locale: Locale, zone: ZoneId): String {
    val day = dueDay(value, zone) ?: return ""
    val pattern = if (locale.language == "zh") "M月d日" else "MMM d"
    return DateTimeFormatter.ofPattern(pattern, locale).format(day)
}

/** `v0.13 · due Nov 14`, or the title alone when the milestone has no date. */
@Composable
fun milestoneLabel(milestone: Milestone, locale: Locale, zone: ZoneId): String {
    val date = milestoneDate(milestone.dueAt, locale, zone)
    return if (date.isEmpty()) milestone.title else stringResource(R.string.milestone_with_due, milestone.title, date)
}

@Composable
private fun currentLocale(): Locale = LocalLocale.current.platformLocale

/**
 * The Due page: four ready-made days with their dates, a hairline, No due date and a date picker.
 * [onPick] gets the chosen day, or null for No due date.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun DuePage(
    task: Task,
    today: LocalDate = LocalDate.now(),
    zone: ZoneId = ZoneId.systemDefault(),
    onBack: () -> Unit,
    onPick: (LocalDate?) -> Unit,
) {
    val tokens = LocalTokens.current
    val locale = currentLocale()
    val current = dueDay(task.dueAt, zone)
    var picking by remember { mutableStateOf(false) }
    PageHeader(stringResource(R.string.menu_due_title), onBack)
    for (option in dueOptions(today)) {
        MenuRow(
            label = dueKindLabel(option.kind),
            hint = dueRowDate(option.date, locale),
            selected = option.date == current,
            trailing = if (option.date == current) {
                { PixelIcon("check", size = 20.dp, tint = tokens.accent) }
            } else {
                null
            },
            onClick = { onPick(option.date) },
        )
    }
    MenuRule()
    MenuRow(
        label = stringResource(R.string.menu_no_due),
        selected = current == null,
        onClick = { onPick(null) },
    )
    MenuRow(label = stringResource(R.string.menu_pick_date), onClick = { picking = true })

    if (picking) {
        val initial = (current ?: today).atStartOfDay(ZoneOffset.UTC).toInstant().toEpochMilli()
        val state = rememberDatePickerState(initialSelectedDateMillis = initial)
        DatePickerDialog(
            onDismissRequest = { picking = false },
            confirmButton = {
                TextButton(onClick = {
                    val day = state.selectedDateMillis
                        ?.let { Instant.ofEpochMilli(it).atZone(ZoneOffset.UTC).toLocalDate() }
                        ?: today
                    picking = false
                    onPick(day)
                }) {
                    Text(stringResource(R.string.action_done), color = tokens.accentInk)
                }
            },
            dismissButton = {
                TextButton(onClick = { picking = false }) {
                    Text(stringResource(R.string.action_cancel), color = tokens.textSecondary)
                }
            },
        ) {
            DatePicker(state = state)
        }
    }
}

/**
 * The Milestone page: the project's open milestones, None, and New milestone… which asks for a
 * name. [onPick] gets a milestone id, or 0 for None; [onCreate] gets the new name.
 */
@Composable
internal fun MilestonePage(
    task: Task,
    milestones: List<Milestone>,
    zone: ZoneId = ZoneId.systemDefault(),
    onBack: () -> Unit,
    onPick: (Long) -> Unit,
    onCreate: (String) -> Unit,
) {
    val tokens = LocalTokens.current
    val locale = currentLocale()
    var naming by remember { mutableStateOf(false) }
    if (naming) {
        NewMilestoneForm(onBack = { naming = false }, onCreate = onCreate)
        return
    }
    PageHeader(stringResource(R.string.menu_milestone_title), onBack)
    for (milestone in milestones.filter { it.open }) {
        val selected = milestone.id == task.milestoneId
        MenuRow(
            label = milestoneLabel(milestone, locale, zone),
            selected = selected,
            trailing = if (selected) {
                { PixelIcon("check", size = 20.dp, tint = tokens.accent) }
            } else {
                null
            },
            onClick = { onPick(milestone.id) },
        )
    }
    MenuRow(
        label = stringResource(R.string.menu_none),
        selected = task.milestoneId == null,
        trailing = if (task.milestoneId == null) {
            { PixelIcon("check", size = 20.dp, tint = tokens.accent) }
        } else {
            null
        },
        onClick = { onPick(0) },
    )
    MenuRow(label = stringResource(R.string.menu_new_milestone), onClick = { naming = true })
}

/** The small name sheet: a page of the task menu with one field and Create. Enter creates too. */
@Composable
private fun NewMilestoneForm(onBack: () -> Unit, onCreate: (String) -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    var name by remember { mutableStateOf("") }
    val focus = remember { FocusRequester() }
    val keyboard = LocalSoftwareKeyboardController.current
    LaunchedEffect(Unit) {
        focus.requestFocus()
        keyboard?.show()
    }
    val ready = name.isNotBlank()
    PageHeader(stringResource(R.string.menu_new_milestone).removeSuffix("…"), onBack)
    Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Box(
            Modifier
                .fillMaxWidth()
                .height(48.dp)
                .background(tokens.bgSubtle, RoundedCornerShape(12.dp))
                .padding(horizontal = 14.dp),
            contentAlignment = Alignment.CenterStart,
        ) {
            BasicTextField(
                value = name,
                onValueChange = { name = it },
                textStyle = type.input.copy(color = tokens.text),
                cursorBrush = SolidColor(tokens.accent),
                singleLine = true,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = { if (ready) onCreate(name.trim()) }),
                modifier = Modifier.fillMaxWidth().focusRequester(focus),
                decorationBox = { inner ->
                    Box(contentAlignment = Alignment.CenterStart) {
                        if (name.isEmpty()) {
                            Text(
                                stringResource(R.string.milestone_name_hint),
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
        PrimaryButton(
            label = stringResource(R.string.milestone_create),
            enabled = ready,
            onClick = { onCreate(name.trim()) },
            modifier = Modifier.fillMaxWidth(),
        )
    }
}
