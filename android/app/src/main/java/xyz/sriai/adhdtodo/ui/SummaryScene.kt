package xyz.sriai.adhdtodo.ui

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Matrix
import android.provider.Settings
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.FilterQuality
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.withTransform
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.delay
import xyz.sriai.adhdtodo.ui.theme.LocalAppType

/** The pictures of the scene (app/theme/sprites in the app's assets), decoded once and kept. */
class SceneSprites(private val context: Context) {
    private val bitmaps = HashMap<String, Bitmap?>()
    private val images = HashMap<String, ImageBitmap?>()

    private fun bitmap(path: String): Bitmap? = bitmaps.getOrPut(path) {
        runCatching {
            context.assets.open("$path.png").use {
                BitmapFactory.decodeStream(it, null, BitmapFactory.Options().apply { inScaled = false })
            }
        }.getOrNull()
    }

    private fun image(key: String, build: () -> Bitmap?): ImageBitmap? =
        images.getOrPut(key) { build()?.asImageBitmap() }

    fun file(path: String): ImageBitmap? = image(path) { bitmap(path) }

    /** An actor in a pose; a pose its folder has no file for is its idle. Robots are tinted. */
    fun actor(actor: SceneActor, pose: String): ImageBitmap? {
        val use = if (actor.hasPose(pose)) pose else "idle"
        val path = "actors/${actor.folder}/$use"
        val tint = actor.tint
        if (actor.folder != SceneActor.BOT || tint == null) return file(path)
        return image("$path#$tint") { bitmap(path)?.let { tinted(it, tint) } }
    }

    /** A good turned [turns] quarter turns clockwise as it flies: its 16 x 16 body, without its shadow. */
    fun spun(kind: String, turns: Int): ImageBitmap? = image("goods/$kind#$turns") {
        val body = bitmap("goods/$kind")?.let { Bitmap.createBitmap(it, 0, 0, GOOD_GRID, GOOD_GRID) } ?: return@image null
        if (turns % 4 == 0) body else Bitmap.createBitmap(body, 0, 0, GOOD_GRID, GOOD_GRID, Matrix().apply { postRotate(90f * (turns % 4)) }, false)
    }

    /** The avatar of a robot, tinted like its actor, for the ledger. */
    fun avatar(name: String, tint: Int?): ImageBitmap? {
        if (tint == null) return null
        return image("avatars/$name#$tint") {
            val src = runCatching { context.assets.open("$name.png").use { BitmapFactory.decodeStream(it, null, BitmapFactory.Options().apply { inScaled = false }) } }.getOrNull()
            src?.let { tinted(it, tint) }
        }
    }

    companion object {
        private val GRAYS = intArrayOf(0xFF8E8E99.toInt(), 0xFF6F6F77.toInt(), 0xFFC1C1C7.toInt())

        /** The robot's three grays (body, shade, light) become the same three shades of [tint]. */
        fun tinted(src: Bitmap, tint: Int): Bitmap {
            val to = intArrayOf(tint, mix(tint, 0x000000, 0.22), mix(tint, 0xFFFFFF, 0.45))
            val out = src.copy(Bitmap.Config.ARGB_8888, true)
            val px = IntArray(out.width * out.height)
            out.getPixels(px, 0, out.width, 0, 0, out.width, out.height)
            for (i in px.indices) {
                val k = GRAYS.indexOf(px[i])
                if (k >= 0) px[i] = to[k]
            }
            out.setPixels(px, 0, out.width, 0, 0, out.width, out.height)
            return out
        }

        private fun mix(base: Int, other: Int, t: Double): Int {
            fun ch(shift: Int): Int {
                val a = (base shr shift) and 0xff
                val b = (other shr shift) and 0xff
                return Math.round(a + (b - a) * t).toInt()
            }
            return (0xFF shl 24) or (ch(16) shl 16) or (ch(8) shl 8) or ch(0)
        }
    }
}

/** The time of the scene in ms; a state, so only what reads it is redrawn. */
@Stable
class SceneClock {
    var ms by mutableIntStateOf(0)
}

/**
 * How the scene is timed. [frozenMs] is the screenshots' fixed frame; [calm] is "remove animations":
 * the last frame at once and nothing sways.
 */
class SceneTime(
    val plan: ScenePlan,
    private val clock: SceneClock,
    private val frozenMs: Int?,
    val calm: Boolean,
) {
    /** The time to draw at; reads the clock, so call it in a draw or a derived state. */
    fun now(): Int = frozenMs ?: if (calm) plan.calm else clock.ms

    /** The time the ledger is lit by: with the scene calm or still, every row is lit and counted. */
    fun ledgerNow(): Int = if (frozenMs != null || calm) plan.end + 3_600_000 else now()
}

/** Runs the clock: every frame while something travels, every 100 ms at rest (the sway, the fireflies). */
@Composable
fun rememberSceneTime(plan: ScenePlan, restart: Any, calm: Boolean, frozenMs: Int?): SceneTime {
    val clock = remember(restart) { SceneClock() }
    val current by rememberUpdatedState(plan)
    LaunchedEffect(restart, calm, frozenMs) {
        if (calm || frozenMs != null) return@LaunchedEffect
        val t0 = androidx.compose.runtime.withFrameMillis { it }
        while (true) {
            val now = (androidx.compose.runtime.withFrameMillis { it } - t0).toInt()
            clock.ms = now
            if (!current.moving(now)) delay(100)
        }
    }
    return remember(plan, clock, calm, frozenMs) { SceneTime(plan, clock, frozenMs, calm) }
}

/** The system's "remove animations": the scene shows its last frame at once. */
fun reduceMotion(context: Context): Boolean =
    Settings.Global.getFloat(context.contentResolver, Settings.Global.ANIMATOR_DURATION_SCALE, 1f) == 0f

/**
 * The scene, edge to edge: [layout] says the whole device pixels a cell takes and how it is cropped;
 * a tap plays it again. [labels] are the week's day names under the stacks.
 */
@Composable
fun SummaryScene(
    time: SceneTime,
    weekLabels: List<String>?,
    description: String,
    onReplay: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val density = LocalDensity.current
    val sprites = remember(context) { SceneSprites(context) }
    var widthPx by remember { mutableIntStateOf(0) }
    val week = weekLabels != null
    val layout = if (widthPx > 0) layoutScene(widthPx, density.density, week) else null
    val heightDp = with(density) { (layout?.heightPx ?: (160 * density.density).toInt()).toDp() }
    Box(
        modifier
            .fillMaxWidth()
            .height(heightDp)
            .clipToBounds()
            .onSizeChanged { widthPx = it.width }
            .semantics { contentDescription = description }
            .clickable(interactionSource = remember { MutableInteractionSource() }, indication = null, onClick = onReplay),
    ) {
        if (layout == null) return@Box
        Canvas(Modifier.fillMaxWidth().height(heightDp)) {
            drawFrame(time.plan.frame(time.now(), time.calm), layout, sprites)
        }
        if (weekLabels != null) WeekLabels(weekLabels, layout)
    }
}

@Composable
private fun WeekLabels(labels: List<String>, layout: SceneLayout) {
    val density = LocalDensity.current
    val type = LocalAppType.current
    val cell = layout.cellPx
    Box(Modifier.fillMaxWidth().height(with(density) { layout.heightPx.toDp() })) {
        for (d in 0 until 7) {
            val left = (layout.offset + dayX(d)) * cell
            Box(
                Modifier
                    .offset { IntOffset(left, layout.heightPx - 13.dp.roundToPx()) }
                    .width(with(density) { (DAY_LABEL_W * cell).toDp() }),
                contentAlignment = Alignment.Center,
            ) {
                Text(labels[d], style = type.meta.copy(fontSize = 10.sp, lineHeight = 11.sp), color = Color.White.copy(alpha = 0.8f), maxLines = 1)
            }
        }
    }
}

/** Draws one frame at whole device pixels, nearest neighbour: the backdrop, then every op. */
private fun DrawScope.drawFrame(frame: SceneFrame, layout: SceneLayout, sprites: SceneSprites) {
    val cell = layout.cellPx
    val ox = layout.offset * cell
    val oy = layout.top
    val width = size.width.toInt()
    sprites.file(frame.sky.file())?.let { sky ->
        fun strip(srcX: Int, srcW: Int, srcY: Int, srcH: Int, dstX: Int, dstY: Int, dstW: Int, dstH: Int) {
            if (dstW <= 0 || dstH <= 0) return
            drawImage(sky, IntOffset(srcX, srcY), IntSize(srcW, srcH), IntOffset(dstX, dstY), IntSize(dstW, dstH), filterQuality = FilterQuality.None)
        }
        val gridW = SCENE_W * cell
        strip(0, SCENE_W, 0, SCENE_H, ox, oy, gridW, SCENE_H * cell)
        // A wider column repeats the first and last column of the backdrop outward.
        strip(0, 1, 0, SCENE_H, 0, oy, ox, SCENE_H * cell)
        strip(SCENE_W - 1, 1, 0, SCENE_H, ox + gridW, oy, width - (ox + gridW), SCENE_H * cell)
        // The day's box can be a few pixels taller than 80 cells: its top row goes up.
        if (oy > 0) {
            strip(0, SCENE_W, 0, 1, ox, 0, gridW, oy)
            strip(0, 1, 0, 1, 0, 0, ox, oy)
            strip(SCENE_W - 1, 1, 0, 1, ox + gridW, 0, width - (ox + gridW), oy)
        }
    }
    for (op in frame.ops) {
        val x = ox + op.x * cell
        val y = oy + op.y * cell
        if (op.kind == OpKind.Rect) {
            drawRect(Color(op.argb), Offset(x.toFloat(), y.toFloat()), Size((op.w * cell).toFloat(), (op.h * cell).toFloat()))
            continue
        }
        val image = when (op.kind) {
            OpKind.Actor -> sprites.actor(op.actor ?: SceneActor.You, op.name)
            OpKind.Good -> sprites.file("goods/${op.name}")
            OpKind.GoodSpin -> sprites.spun(op.name, op.spin)
            else -> sprites.file(op.name)
        } ?: continue
        val w = image.width * cell
        val h = image.height * cell
        val draw = { drawImage(image, IntOffset.Zero, IntSize(image.width, image.height), IntOffset(x, y), IntSize(w, h), filterQuality = FilterQuality.None) }
        if (op.flip) {
            withTransform({ scale(-1f, 1f, Offset(x + w / 2f, 0f)) }) { draw() }
        } else {
            draw()
        }
    }
}

private fun Sky.file(): String = when (this) {
    Sky.Day -> "scenes/day"
    Sky.Dusk -> "scenes/dusk"
    Sky.Night -> "scenes/night"
}
