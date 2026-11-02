package xyz.sriai.adhdtodo.widget

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.ui.theme.Glyphs
import kotlin.math.max
import kotlin.math.roundToInt

/**
 * A pixel picture for a RemoteViews image: a bitmap of 12 x 12 cells at a whole number of physical
 * pixels a cell, and the dp size that shows exactly those pixels, so nothing is split or blurred
 * (the rule PixelGrid follows in the app).
 */
class PixelPicture(val bitmap: Bitmap, val size: Dp)

/** A status glyph in two layers: [ink] is tinted with the status color, [mark] stays white. */
class GlyphPicture(val ink: PixelPicture, val mark: PixelPicture?)

object WidgetPixels {
    /** The plus of docs/design/mockups/v43.html, two cells thick. */
    private val plus = listOf(
        "............", ".....##.....", ".....##.....", ".....##.....", ".....##.....", "############",
        "############", ".....##.....", ".....##.....", ".....##.....", ".....##.....", "............",
    )

    /** A sprite from the avatars folder of the assets, scaled with nearest-neighbour. */
    fun avatar(context: Context, name: String, sizeDp: Float): PixelPicture? {
        val density = context.resources.displayMetrics.density
        val cell = cellFor(sizeDp, density)
        val sprite = runCatching {
            context.assets.open("$name.png").use { BitmapFactory.decodeStream(it) }
        }.getOrNull() ?: return null
        val scaled = Bitmap.createScaledBitmap(sprite, 12 * cell, 12 * cell, false)
        return PixelPicture(scaled, dpOf(12 * cell, density))
    }

    /**
     * Glyphs.rows drawn in white so the widget can tint them with a day/night color; the white
     * cells of the blocked bang and the done check go to a second layer that is not tinted.
     */
    fun glyph(context: Context, status: String, sizeDp: Float): GlyphPicture {
        val rows = Glyphs.rows(status)
        val density = context.resources.displayMetrics.density
        val cell = cellFor(sizeDp, density)
        val ink = draw(rows, cell) { true }
        val hasMark = rows.any { 'w' in it }
        val mark = if (hasMark) draw(rows, cell) { it == 'w' } else null
        val size = dpOf(12 * cell, density)
        return GlyphPicture(PixelPicture(ink, size), mark?.let { PixelPicture(it, size) })
    }

    fun plus(context: Context, sizeDp: Float): PixelPicture {
        val density = context.resources.displayMetrics.density
        val cell = cellFor(sizeDp, density)
        return PixelPicture(draw(plus, cell) { true }, dpOf(12 * cell, density))
    }

    private fun draw(rows: List<String>, cell: Int, lit: (Char) -> Boolean): Bitmap {
        val bitmap = Bitmap.createBitmap(12 * cell, 12 * cell, Bitmap.Config.ARGB_8888)
        val canvas = Canvas(bitmap)
        val paint = Paint().apply { color = Color.WHITE }
        for (y in rows.indices) {
            for (x in rows[y].indices) {
                val c = rows[y][x]
                if (c == '.' || !lit(c)) continue
                canvas.drawRect(
                    (x * cell).toFloat(), (y * cell).toFloat(),
                    ((x + 1) * cell).toFloat(), ((y + 1) * cell).toFloat(), paint,
                )
            }
        }
        return bitmap
    }

    /** Physical pixels a cell takes so that 12 cells come closest to [sizeDp], never fewer than one. */
    internal fun cellFor(sizeDp: Float, density: Float): Int =
        max(1, (sizeDp * density / 12f).roundToInt())

    private fun dpOf(pixels: Int, density: Float): Dp = (pixels / density).dp
}
