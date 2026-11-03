package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import xyz.sriai.adhdtodo.data.CalendarDay
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.Tokens
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.format.TextStyle
import java.util.Locale

/** The height of a cell (v44). */
val CalendarCellHeight = 52.dp

/** The tag of a cell in tests: `calendar-cell-2026-11-03`. */
fun calendarCellTag(date: String): String = "calendar-cell-$date"

/** Slate 10: an open task's dot. */
private val Tokens.dotOpen: Color get() = if (dark) Color(0xFF777B84) else Color(0xFF80838D)

private fun Tokens.dotColor(dot: Dot): Color = when (dot) {
    Dot.Open -> dotOpen
    Dot.Overdue -> statusBlocked
    Dot.Finished -> borderControl // slate 7
    Dot.Milestone -> accent
}

/**
 * The weekday row and the month grid: a 0.5 dp hairline card of 52 dp cells, weekends on bg.sidebar,
 * other months' days in text.tertiary. A horizontal swipe over it pages the month ([onShift] -1 or 1).
 */
@Composable
fun CalendarGrid(
    days: List<CalendarDay>,
    weekStart: DayOfWeek,
    selected: LocalDate,
    onSelect: (LocalDate) -> Unit,
    onShift: (Int) -> Unit,
    locale: Locale,
    modifier: Modifier = Modifier,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val threshold = with(LocalDensity.current) { 48.dp.toPx() }
    Column(
        modifier.pointerInput(onShift) {
            var total = 0f
            detectHorizontalDragGestures(
                onDragStart = { total = 0f },
                onDragEnd = { swipeMonths(total, threshold).takeIf { it != 0 }?.let(onShift) },
                onDragCancel = { total = 0f },
                onHorizontalDrag = { _, dx -> total += dx },
            )
        },
    ) {
        Row(Modifier.fillMaxWidth().padding(top = 14.dp, bottom = 4.dp)) {
            for (day in weekdaysFrom(weekStart)) {
                Text(
                    day.getDisplayName(TextStyle.NARROW, locale),
                    style = type.meta.copy(fontSize = 12.sp),
                    color = tokens.textTertiary,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.weight(1f),
                )
            }
        }
        val shape = RoundedCornerShape(12.dp)
        Column(Modifier.fillMaxWidth().clip(shape).border(Dims.hairline, tokens.border, shape)) {
            val weeks = days.chunked(7)
            weeks.forEachIndexed { row, week ->
                Row(Modifier.fillMaxWidth()) {
                    week.forEachIndexed { col, day ->
                        CalendarCell(
                            day = day,
                            selected = day.date == selected.toString(),
                            label = calendarDayHeading(LocalDate.parse(day.date), locale),
                            lastColumn = col == week.lastIndex,
                            lastRow = row == weeks.lastIndex,
                            onClick = { onSelect(LocalDate.parse(day.date)) },
                            modifier = Modifier.weight(1f),
                        )
                    }
                }
            }
        }
    }
}

/**
 * One cell: the day number (today in an iris 9 pill with white text), at most three 5 dp square
 * dots under it, and, when it is the selected day, a 2 dp ring in the accent at 60 %.
 */
@Composable
private fun CalendarCell(
    day: CalendarDay,
    selected: Boolean,
    label: String,
    lastColumn: Boolean,
    lastRow: Boolean,
    onClick: () -> Unit,
    modifier: Modifier,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val number = LocalDate.parse(day.date).dayOfMonth
    val line = tokens.border
    val hairline = Dims.hairline
    Column(
        modifier = modifier
            .height(CalendarCellHeight)
            .background(if (day.weekend) tokens.bgSidebar else Color.Transparent)
            .drawBehind {
                val w = hairline.toPx()
                if (!lastColumn) drawLine(line, Offset(size.width - w / 2, 0f), Offset(size.width - w / 2, size.height), w)
                if (!lastRow) drawLine(line, Offset(0f, size.height - w / 2), Offset(size.width, size.height - w / 2), w)
            }
            .then(if (selected) Modifier.border(2.dp, tokens.accent.copy(alpha = 0.6f)) else Modifier)
            .clickable(onClick = onClick)
            .semantics { this.selected = selected; contentDescription = label }
            .testTag(calendarCellTag(day.date))
            .padding(top = 6.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(5.dp),
    ) {
        Box(
            Modifier
                .size(width = 26.dp, height = 22.dp)
                .clip(RoundedCornerShape(11.dp))
                .background(if (day.today) tokens.accent else Color.Transparent),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                number.toString(),
                style = type.body.copy(fontSize = 14.sp, lineHeight = 18.sp),
                fontWeight = if (day.today) FontWeight.SemiBold else null,
                color = when {
                    day.today -> tokens.onAccent
                    day.inMonth -> tokens.text
                    else -> tokens.statusBacklog // slate 8
                },
            )
        }
        Row(Modifier.height(5.dp), horizontalArrangement = Arrangement.spacedBy(3.dp)) {
            for (dot in calendarDots(day.items)) {
                Box(Modifier.size(5.dp).background(tokens.dotColor(dot)))
            }
        }
    }
}
