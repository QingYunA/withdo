package xyz.sriai.adhdtodo.ui.theme

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.Dp

/** The important star of app/theme/star.go (the filled one) on the 12 x 12 grid. */
private val starRows = listOf(
    ".....##.....", ".....##.....", "....####....", "....####....", "############", ".##########.",
    "..########..", "...######...", "...######...", "..###..###..", "..##....##..", ".##......##.",
)

/** The pixel microphone of app/theme/voice.go. */
private val micRows = listOf(
    "....####....",
    "...######...",
    "...######...",
    "...######...",
    "...######...",
    ".#.######.#.",
    ".#.######.#.",
    ".#..####..#.",
    "..##....##..",
    "....####....",
    ".....##.....",
    "...######...",
)

/** One 12 x 12 grid of '#' cells in a single color, on whole physical pixels. */
@Composable
private fun PixelRows(rows: List<String>, size: Dp, tint: Color, modifier: Modifier) {
    PixelGrid(size = size, modifier = modifier) { cell ->
        for (y in rows.indices) {
            for (x in rows[y].indices) {
                if (rows[y][x] != '#') continue
                drawRect(color = tint, topLeft = Offset(x * cell, y * cell), size = Size(cell, cell))
            }
        }
    }
}

/** The filled star a row of mine shows before its date when it is P0 or P1, in status.review. */
@Composable
fun StarGlyph(size: Dp, tint: Color, modifier: Modifier = Modifier) = PixelRows(starRows, size, tint, modifier)

/** The composer's microphone. */
@Composable
fun MicGlyph(size: Dp, tint: Color, modifier: Modifier = Modifier) = PixelRows(micRows, size, tint, modifier)
