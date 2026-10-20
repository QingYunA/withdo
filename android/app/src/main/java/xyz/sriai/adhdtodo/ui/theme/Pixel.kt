package xyz.sriai.adhdtodo.ui.theme

import android.content.Context
import android.graphics.BitmapFactory
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.Image
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.FilterQuality
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.scale
import androidx.compose.ui.graphics.vector.PathParser
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import kotlin.math.max
import kotlin.math.roundToInt

/**
 * A picture on the 12 x 12 pixel grid, drawn at the nearest whole number of physical pixels per
 * cell and centered in its slot. The slot is the dp size the layout asks for; the drawing is
 * 12 * round(slot * density / 12) physical pixels, so no pixel is ever split or blurred.
 */
@Composable
fun PixelGrid(
    size: Dp,
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    draw: DrawScope.(Float) -> Unit,
) {
    val density = LocalDensity.current
    val cellPx = remember(size, density) {
        max(1, (size.value * density.density / 12f).roundToInt())
    }
    val drawn = with(density) { (cellPx * 12).toDp() }
    Box(
        modifier = modifier
            .size(size)
            .then(if (onClick != null) Modifier.clickable { onClick() } else Modifier),
        contentAlignment = Alignment.Center,
    ) {
        Canvas(Modifier.size(drawn)) { draw(cellPx.toFloat()) }
    }
}

/** One status glyph, 18 dp by default: the rectangles of Glyphs.rows. */
@Composable
fun StatusGlyph(
    status: String,
    modifier: Modifier = Modifier,
    size: Dp = Dims.glyph,
    onClick: (() -> Unit)? = null,
) {
    val tokens = LocalTokens.current
    val rows = remember(status) { Glyphs.rows(status) }
    val ink = tokens.status(status)
    PixelGrid(size = size, modifier = modifier, onClick = onClick) { cell ->
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

/** A 12 x 12 sprite from app/theme/sprites/avatars, at nearest-neighbour, sharp pixels. */
@Composable
fun Avatar(name: String, modifier: Modifier = Modifier, size: Dp = Dims.avatar) {
    val context = LocalContext.current
    val dark = LocalTokens.current.dark
    val file = remember(name, dark) {
        val darkFile = "$name.dark.png"
        if (dark && assetExists(context, darkFile)) darkFile else "$name.png"
    }
    val image = remember(file) { loadBitmap(context, file) }
    if (image == null) {
        Box(modifier.size(size))
        return
    }
    PixelBitmap(image = image, size = size, modifier = modifier)
}

/** A decoded sprite drawn at whole physical-pixel cells and centered in its slot. */
@Composable
fun PixelBitmap(image: ImageBitmap, size: Dp, modifier: Modifier = Modifier) {
    val density = LocalDensity.current
    val cellPx = remember(size, density) {
        max(1, (size.value * density.density / 12f).roundToInt())
    }
    val drawn = with(density) { (cellPx * 12).toDp() }
    Box(modifier.size(size), contentAlignment = Alignment.Center) {
        Image(
            bitmap = image,
            contentDescription = null,
            modifier = Modifier.size(drawn),
            filterQuality = FilterQuality.None,
        )
    }
}

/**
 * One pixelarticon by name, parsed from app/theme/pixel-icons with Compose's PathParser and filled
 * in the caller's color. The 24-unit viewBox is 12 cells, so it uses the same whole-pixel rule as
 * the sprites.
 */
@Composable
fun PixelIcon(name: String, modifier: Modifier = Modifier, size: Dp = Dims.icon, tint: Color) {
    val context = LocalContext.current
    val paths = remember(name) { loadIcon(context, name) }
    PixelGrid(size = size, modifier = modifier) { cell ->
        val factor = cell / 2f
        scale(factor, factor, pivot = Offset.Zero) {
            for (path in paths) drawPath(path, tint)
        }
    }
}

private fun assetExists(context: Context, path: String): Boolean =
    runCatching { context.assets.open(path).close() }.isSuccess

private fun loadBitmap(context: Context, path: String): ImageBitmap? = runCatching {
    context.assets.open(path).use { BitmapFactory.decodeStream(it).asImageBitmap() }
}.getOrNull()

private val pathData = Regex("""d\s*=\s*"([^"]+)"""")

private fun loadIcon(context: Context, name: String): List<Path> = runCatching {
    val svg = context.assets.open("$name.svg").bufferedReader().use { it.readText() }
    pathData.findAll(svg).mapNotNull { match ->
        runCatching { PathParser().parsePathString(match.groupValues[1]).toPath() }.getOrNull()
    }.toList()
}.getOrDefault(emptyList())
