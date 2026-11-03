package xyz.sriai.adhdtodo.ui

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.CalendarItem
import xyz.sriai.adhdtodo.data.CalendarPage
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.YearMonth
import java.time.ZoneId
import java.util.Locale

/** The inset of the day's text from the screen edge, as Today's rows have it (16 + 12 dp). */
private val DayInset = 28.dp

/**
 * The Calendar tab's header (v44): the month in the title, then the `‹ Today ›` segment (40 x 32 dp
 * cells in a bg.hover well, Today raised) and the sync cloud.
 */
@Composable
fun CalendarHeader(
    title: String,
    syncBlocked: Boolean,
    onSync: () -> Unit,
    onPrevious: () -> Unit,
    onToday: () -> Unit,
    onNext: () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier.fillMaxWidth().padding(start = DayInset, end = Dims.pagePadding - 4.dp, top = 10.dp, bottom = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(title, style = type.title, color = tokens.text, modifier = Modifier.weight(1f))
        Row(
            Modifier.background(tokens.bgHover, RoundedCornerShape(10.dp)).padding(2.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            SegmentCell(stringResource(R.string.calendar_previous), onPrevious) {
                PixelIcon("chevron-left", size = 12.dp, tint = tokens.textSecondary)
            }
            val raised = RoundedCornerShape(8.dp)
            SegmentCell(
                label = stringResource(R.string.calendar_today),
                onClick = onToday,
                modifier = Modifier.background(tokens.bg, raised).border(Dims.hairline, tokens.border, raised),
            ) {
                Text(
                    stringResource(R.string.calendar_today),
                    style = type.meta.copy(fontWeight = FontWeight.SemiBold),
                    color = tokens.text,
                    maxLines = 1,
                    modifier = Modifier.padding(horizontal = 6.dp),
                )
            }
            SegmentCell(stringResource(R.string.calendar_next), onNext) {
                PixelIcon("chevron-right", size = 12.dp, tint = tokens.textSecondary)
            }
        }
        SyncCloud(blocked = syncBlocked, onSync = onSync)
    }
}

@Composable
private fun SegmentCell(
    label: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    content: @Composable () -> Unit,
) {
    Box(
        modifier = modifier
            .widthIn(min = 40.dp)
            .height(32.dp)
            .clip(RoundedCornerShape(8.dp))
            .clickable(onClick = onClick)
            .semantics { contentDescription = label; role = Role.Button },
        contentAlignment = Alignment.Center,
    ) { content() }
}

/**
 * The Calendar tab (v44): the month grid, then the selected day: its heading and count, its
 * milestones as accent chips, and its rows, which behave like Today's (a tap opens the task, a long
 * press the task menu, the glyph completes a task of mine). The page is the core's [page] when it is
 * the one asked for, else an empty grid.
 */
@Composable
fun CalendarScreen(
    page: CalendarPage?,
    month: YearMonth,
    selected: LocalDate,
    today: LocalDate,
    weekStart: DayOfWeek,
    words: Words,
    categories: List<Category>,
    onSelect: (LocalDate) -> Unit,
    onShift: (Int) -> Unit,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
    onComplete: (Long) -> Unit,
    highlightId: Long = 0,
    locale: Locale = LocalConfiguration.current.locales[0],
    zone: ZoneId = ZoneId.systemDefault(),
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val days = calendarDays(page, month, weekStart, today)
    val items = days.firstOrNull { it.date == selected.toString() }?.items.orEmpty()
    val milestones = items.filter { it.kind == "milestone" }
    val rows = items.filter { it.kind != "milestone" }
    ReportSelectable(rows.mapNotNull { it.task?.id })
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        CalendarGrid(
            days = days,
            weekStart = weekStart,
            selected = selected,
            onSelect = onSelect,
            onShift = onShift,
            locale = locale,
            modifier = Modifier.padding(horizontal = 16.dp),
        )
        Row(
            Modifier.fillMaxWidth().padding(start = DayInset, end = DayInset, top = 18.dp, bottom = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(
                calendarDayHeading(selected, locale),
                style = type.body.copy(fontWeight = FontWeight.SemiBold),
                color = tokens.text,
            )
            if (rows.isNotEmpty()) Text(rows.size.toString(), style = type.body, color = tokens.textSecondary)
        }
        for (item in milestones) MilestoneChip(item)
        for (item in rows) {
            CalendarRow(item, categories, words, locale, zone, highlightId, onOpenTask, onLongPress, onComplete)
        }
        if (items.isEmpty()) {
            Text(
                stringResource(R.string.calendar_day_empty),
                style = type.meta,
                color = tokens.textSecondary,
                modifier = Modifier.padding(start = DayInset, top = 8.dp),
            )
        }
        Box(Modifier.height(8.dp))
    }
}

/** A milestone due that day: the flag and `v0.13 due` on iris 3, radius 8, 28 dp tall. */
@Composable
private fun MilestoneChip(item: CalendarItem) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val title = item.milestone?.title ?: return
    Row(
        Modifier
            .padding(start = DayInset, end = DayInset, top = 6.dp, bottom = 2.dp)
            .height(28.dp)
            .background(tokens.accentSoft, RoundedCornerShape(8.dp))
            .padding(horizontal = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        PixelIcon("flag", size = 14.dp, tint = tokens.accent)
        Text(
            stringResource(R.string.calendar_milestone_due, title),
            style = type.meta.copy(fontWeight = FontWeight.Medium),
            color = tokens.accentInk,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

/**
 * A row of the day, 48 dp, Today's way: the status glyph (which ticks a task of mine off), the
 * title, the category word and the project, and at the right the time: a reminder with the bell, an
 * overdue one in the blocked red. A finished task is struck through and has no time.
 */
@Composable
private fun CalendarRow(
    item: CalendarItem,
    categories: List<Category>,
    words: Words,
    locale: Locale,
    zone: ZoneId,
    highlightId: Long,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
    onComplete: (Long) -> Unit,
) {
    val task = item.task ?: return
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val done = item.kind == "done"
    val category = if (task.source == "user" && task.categoryId != 0) {
        categories.firstOrNull { it.id.toInt() == task.categoryId }
    } else {
        null
    }
    val fill = animateColorAsState(
        if (task.id == highlightId) tokens.accentSoft else Color.Transparent,
        animationSpec = tween(250),
        label = "calendar-new-row",
    )
    val shape = LocalSelection.current.rowClip()
    Box(Modifier.padding(horizontal = 16.dp).fillMaxWidth().clip(shape).background(fill.value).testTag("calendar-row-${task.id}")) {
        LongPressRow(
            onClick = { onOpenTask(task.id) },
            onLongPress = { onLongPress(task.id) },
            taskId = task.id,
        ) { row ->
            Row(
                modifier = row.height(Dims.rowHeight).padding(horizontal = 12.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                RowGlyph(
                    taskId = task.id,
                    status = task.status,
                    onClick = {
                        if (task.source == "user" && task.status != "done" && task.status != "cancelled") {
                            onComplete(task.id)
                        } else {
                            onOpenTask(task.id)
                        }
                    },
                )
                Row(
                    Modifier.weight(1f),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Text(
                        task.title,
                        style = type.bodyStrong.copy(textDecoration = if (done) TextDecoration.LineThrough else null),
                        color = if (done) tokens.textSecondary else tokens.text,
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
                if (!done) {
                    Row(horizontalArrangement = Arrangement.spacedBy(4.dp), verticalAlignment = Alignment.CenterVertically) {
                        val ink = if (item.overdue) tokens.statusBlocked else tokens.textSecondary
                        if (item.kind == "reminder") PixelIcon("bell", size = 14.dp, tint = ink)
                        Text(calendarClock(item.at, locale, zone), style = type.meta, color = ink, maxLines = 1)
                    }
                }
            }
        }
    }
}
