package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.RectangleShape
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.AgentsPage
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelGrid
import xyz.sriai.adhdtodo.ui.theme.StatusGlyph

/*
 * Select mode (v43). The App holds who is ticked and gives the list rows a [SelectionUi] through
 * [LocalSelection]; a row that knows its task id (LongPressRow's taskId) then ticks on a tap, draws
 * a pixel checkbox where its glyph or avatar was, and joins its selected neighbours into one
 * bg.selected block. The bulk bar is in BulkBar.kt.
 */

/** Who is ticked on the page, and the rows the page draws in order (0 separates sections). */
@Immutable
class SelectionUi(
    val ids: Set<Long>,
    val order: List<Long>,
    val toggle: (Long) -> Unit,
    val report: (List<Long>) -> Unit,
) {
    fun isSelected(id: Long): Boolean = id != 0L && id in ids

    /** A selected row is square where it joins a selected neighbour and round at the block's ends. */
    fun shape(id: Long): Shape {
        val at = order.indexOf(id)
        val joinsAbove = at > 0 && isSelected(order[at - 1])
        val joinsBelow = at >= 0 && at < order.lastIndex && isSelected(order[at + 1])
        return RoundedCornerShape(
            topStart = if (joinsAbove) 0.dp else 10.dp,
            topEnd = if (joinsAbove) 0.dp else 10.dp,
            bottomStart = if (joinsBelow) 0.dp else 10.dp,
            bottomEnd = if (joinsBelow) 0.dp else 10.dp,
        )
    }
}

/** Null outside select mode. */
val LocalSelection = staticCompositionLocalOf<SelectionUi?> { null }

/** A row's own rounded clip is dropped in select mode, so selected neighbours can join. */
fun SelectionUi?.rowClip(): Shape = if (this == null) RoundedCornerShape(8.dp) else RectangleShape

/** [ids] with one id added or taken out. */
fun Set<Long>.toggled(id: Long): Set<Long> = if (id in this) this - id else this + id

/** The rows Today draws in order, a 0 between sections; the No date fold counts when it is open. */
fun mineOrder(page: MinePage, noDateOpen: Boolean): List<Long> = sectionIds(
    page.today.map { it.id },
    page.next.map { it.id },
    if (noDateOpen) page.noDate.map { it.id } else emptyList(),
    page.done.map { it.id },
)

/** The rows of Agents in order: Working, then Done today. */
fun agentsOrder(page: AgentsPage): List<Long> =
    sectionIds(page.working.map { it.id }, page.done.map { it.id })

/** Joins lists of ids into one, a 0 between non-empty lists. */
fun sectionIds(vararg lists: List<Long>): List<Long> =
    lists.filter { it.isNotEmpty() }.flatMapIndexed { index, list ->
        (if (index > 0) listOf(0L) else emptyList()) + list
    }

/** A page tells the App which rows it draws, while in select mode, so Select all knows them. */
@Composable
fun ReportSelectable(order: List<Long>) {
    val selection = LocalSelection.current
    LaunchedEffect(order, selection != null) { selection?.report?.invoke(order) }
}

/**
 * The pixel checkbox: an empty box in slate 8, or the ticked one in the accent with a white tick.
 * The 12 x 12 grids are the mockup's.
 */
@Composable
fun SelectBox(checked: Boolean, size: Dp = Dims.glyph) {
    val tokens = LocalTokens.current
    val rows = if (checked) CheckedRows else EmptyRows
    val ink = if (checked) tokens.accent else tokens.textTertiary
    val label = stringResource(if (checked) R.string.bulk_checkbox_on else R.string.bulk_checkbox_off)
    PixelGrid(size = size, modifier = Modifier.semantics { contentDescription = label }) { cell ->
        for (y in rows.indices) {
            for (x in rows[y].indices) {
                val c = rows[y][x]
                if (c == '.') continue
                drawRect(
                    color = if (c == 'w') Color.White else ink,
                    topLeft = Offset(x * cell, y * cell),
                    size = Size(cell, cell),
                )
            }
        }
    }
}

private val EmptyRows = listOf(
    "............",
    ".##########.",
    ".#........#.",
    ".#........#.",
    ".#........#.",
    ".#........#.",
    ".#........#.",
    ".#........#.",
    ".#........#.",
    ".#........#.",
    ".##########.",
    "............",
)

private val CheckedRows = listOf(
    "............",
    ".##########.",
    ".##########.",
    ".########w#.",
    ".#######ww#.",
    ".#w####ww##.",
    ".#ww##ww###.",
    ".##wwww####.",
    ".###ww#####.",
    ".##########.",
    ".##########.",
    "............",
)

/** The status glyph of a row, or in select mode its checkbox. */
@Composable
fun RowGlyph(
    taskId: Long,
    status: String,
    modifier: Modifier = Modifier,
    size: Dp = Dims.glyph,
    onClick: (() -> Unit)? = null,
) {
    val selection = LocalSelection.current
    if (selection != null) {
        SelectBox(selection.isSelected(taskId), size)
    } else {
        StatusGlyph(status, modifier, size, onClick)
    }
}

/** The author's avatar of a row, or in select mode its checkbox. */
@Composable
fun RowAvatar(taskId: Long, name: String, size: Dp = Dims.avatar) {
    val selection = LocalSelection.current
    if (selection != null) SelectBox(selection.isSelected(taskId), size) else Avatar(name, size = size)
}

/** `已选 2 件`, Select all and Done, in the place of the page's title. */
@Composable
fun SelectHeader(count: Int, onSelectAll: () -> Unit, onDone: () -> Unit, start: Dp = Dims.pagePadding + 8.dp) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(56.dp)
            .padding(start = start, end = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            pluralStringResource(R.plurals.select_count, count, count),
            style = type.bodyStrong.copy(fontSize = 17.sp, lineHeight = 24.sp),
            color = tokens.text,
        )
        Spacer(Modifier.weight(1f))
        HeaderAction(stringResource(R.string.select_all), onSelectAll)
        HeaderAction(stringResource(R.string.bulk_done), onDone)
    }
}

@Composable
private fun HeaderAction(label: String, onClick: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Text(
        label,
        style = type.bodyStrong,
        color = tokens.accentInk,
        maxLines = 1,
        modifier = Modifier
            .clip(RoundedCornerShape(8.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 10.dp),
    )
}
