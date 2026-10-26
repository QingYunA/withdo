package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.Glyphs
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelGrid
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import kotlin.math.roundToInt

/** A subpage's bar: the back chevron, the page it returns to, and what the page wants at right. */
@Composable
fun BackBar(
    title: String,
    onBack: () -> Unit,
    right: @Composable RowScope.() -> Unit = {},
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(48.dp)
            .padding(horizontal = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        PixelIcon(
            "chevron-left",
            size = 18.dp,
            tint = tokens.textSecondary,
            modifier = Modifier.clickable(onClick = onBack),
        )
        Text(
            title,
            style = type.bodyStrong,
            color = tokens.textSecondary,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        Spacer(Modifier.weight(1f))
        right()
    }
}

/** The accent button of the Sync card and the sheets. */
@Composable
fun PrimaryButton(
    label: String,
    enabled: Boolean = true,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Box(
        modifier = modifier
            .heightIn(min = 44.dp)
            .alpha(if (enabled) 1f else 0.45f)
            .clip(RoundedCornerShape(10.dp))
            .background(tokens.accent)
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 11.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(label, style = type.bodyStrong, color = tokens.onAccent)
    }
}

/** The outlined quiet button of a group row: Sync now, Retry. */
@Composable
fun QuietButton(label: String, enabled: Boolean = true, onClick: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Box(
        modifier = Modifier
            .heightIn(min = 40.dp)
            .clip(RoundedCornerShape(6.dp))
            .border(Dims.hairline, tokens.border, RoundedCornerShape(6.dp))
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 9.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            label,
            style = type.meta,
            color = if (enabled) tokens.text else tokens.textTertiary,
        )
    }
}

@Composable
fun TextLink(label: String, onClick: () -> Unit, color: Color? = null) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Text(
        label,
        style = type.meta,
        color = color ?: tokens.accentInk,
        maxLines = 1,
        modifier = Modifier
            .clip(RoundedCornerShape(6.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 4.dp, vertical = 10.dp),
    )
}

/** A white rounded group with a hairline, as Settings draws them. */
@Composable
fun GroupCard(content: @Composable ColumnScope.() -> Unit) {
    val tokens = LocalTokens.current
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .background(tokens.bg)
            .border(Dims.hairline, tokens.border, RoundedCornerShape(12.dp)),
        content = content,
    )
}

@Composable
fun RowRule() {
    Box(
        Modifier
            .fillMaxWidth()
            .height(Dims.hairline)
            .padding(horizontal = 14.dp)
            .background(LocalTokens.current.border),
    )
}

@Composable
fun GroupLabel(text: String) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Text(
        text,
        style = type.section,
        color = tokens.textSecondary,
        modifier = Modifier.padding(start = 4.dp, top = 20.dp, bottom = 8.dp),
    )
}

/** The outlined "This phone" badge of a device row, or "Recommended" on a choice. */
@Composable
fun OutlineTag(text: String, color: Color? = null) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Box(
        modifier = Modifier
            .border(Dims.hairline, color ?: tokens.border, RoundedCornerShape(5.dp))
            .padding(horizontal = 5.dp),
    ) {
        Text(text, style = type.badge, color = color ?: tokens.textSecondary)
    }
}

/** The filled "Recommended" mark of the merge choice. */
@Composable
fun AccentTag(text: String) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Box(
        modifier = Modifier
            .background(tokens.accentSoft, RoundedCornerShape(5.dp))
            .padding(horizontal = 5.dp),
    ) {
        Text(text, style = type.badge, color = tokens.accentInk)
    }
}

/**
 * A pixel radio on the status ring's grid, as the desktop's theme.RadioGlyph: the ring, and when
 * picked the centered dot of the review glyph inside it, in the accent.
 */
@Composable
fun RadioDot(selected: Boolean) {
    val tokens = LocalTokens.current
    val rows = Glyphs.rows(if (selected) "review" else "todo")
    val ink = if (selected) tokens.accent else tokens.textTertiary
    PixelGrid(size = 16.dp) { cell ->
        for (y in rows.indices) {
            for (x in rows[y].indices) {
                if (rows[y][x] == '.') continue
                drawRect(color = ink, topLeft = Offset(x * cell, y * cell), size = Size(cell, cell))
            }
        }
    }
}

/**
 * The first upload's 40 cell bar: accent for the done share and 8 % ink for the rest. The cells
 * and their 2 dp gaps are laid out in whole physical pixels and centered in the width the row
 * gives them, so the bar stays sharp on a phone's fractional density.
 */
@Composable
fun PixelProgressBar(done: Long, total: Long, modifier: Modifier = Modifier) {
    val tokens = LocalTokens.current
    val filled = syncBarFilled(done, total)
    androidx.compose.foundation.Canvas(modifier.fillMaxWidth().height(8.dp)) {
        val gap = 2.dp.toPx().roundToInt().coerceAtLeast(1)
        val cell = ((size.width - gap * (SYNC_BAR_CELLS - 1)) / SYNC_BAR_CELLS).toInt().coerceAtLeast(1)
        val barWidth = cell * SYNC_BAR_CELLS + gap * (SYNC_BAR_CELLS - 1)
        val start = ((size.width - barWidth) / 2f).roundToInt().toFloat()
        val height = size.height.roundToInt().toFloat()
        for (i in 0 until SYNC_BAR_CELLS) {
            drawRect(
                color = if (i < filled) tokens.accent else tokens.text.copy(alpha = 0.08f),
                topLeft = androidx.compose.ui.geometry.Offset(start + i * (cell + gap), 0f),
                size = androidx.compose.ui.geometry.Size(cell.toFloat(), height),
            )
        }
    }
}

