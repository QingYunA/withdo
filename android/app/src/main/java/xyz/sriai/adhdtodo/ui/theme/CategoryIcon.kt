package xyz.sriai.adhdtodo.ui.theme

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.unit.Dp
import xyz.sriai.adhdtodo.data.Category

/**
 * A category's 12 x 12 picture: the Go '#' pixels in the category's own color and the palette
 * letters a to l in their fixed colors, on the same whole-pixel grid as the other pixel art.
 */
@Composable
fun CategoryIcon(category: Category, modifier: Modifier = Modifier, size: Dp = Dims.icon) {
    val tokens = LocalTokens.current
    val ink = tokens.categoryColor(category.color)
    val palette = tokens.categoryPalette
    val rows = category.icon
    PixelGrid(size = size, modifier = modifier) { cell ->
        for (y in rows.indices) {
            val row = rows[y]
            for (x in row.indices) {
                val pixel = row[x]
                if (pixel == 0) continue
                val color = when {
                    pixel == 35 -> ink // pixmap.Own
                    pixel in 97..108 -> palette[pixel - 97]
                    else -> ink
                }
                drawRect(color = color, topLeft = Offset(x * cell, y * cell), size = Size(cell, cell))
            }
        }
    }
}
