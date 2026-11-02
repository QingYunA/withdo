package xyz.sriai.adhdtodo.ui

import xyz.sriai.adhdtodo.data.SummaryPage
import xyz.sriai.adhdtodo.data.SummaryPerson
import kotlin.math.abs
import kotlin.math.floor
import kotlin.math.max
import kotlin.math.min
import kotlin.math.sin

/*
 * The choreography of the Summary scene, a port of app/theme/scene*.go (ui-spec, Summary). Every
 * frame is a pure function of the plan and the time in milliseconds: [DayPlan.frame] and
 * [WeekPlan.frame] list what is drawn, in cells from the scene's top left, and SummaryScene draws
 * the list. Nothing here touches Android, so the timings and the geometry are unit tested.
 */

const val SCENE_W = 264
const val SCENE_H = 80
const val ACTOR_GRID = 24
const val GOOD_GRID = 16

private const val FEET = 71
private const val ACTOR_TOP = FEET - 22
private const val BIN_X = 112
private const val BIN_RIM = FEET - 16

enum class Sky { Day, Dusk, Night }

/** Day before 17:00, dusk from 17:00, night from 20:00, by the local hour the page opens. */
fun skyAt(hour: Int): Sky = when {
    hour < 17 -> Sky.Day
    hour < 20 -> Sky.Dusk
    else -> Sky.Night
}

/** A person of the scene: the folder of its pictures under actors/, and a robot's tint (ARGB). */
data class SceneActor(val folder: String, val tint: Int? = null) {
    /** A mark with a set direction is not turned round when it stands right of the bin. */
    val neverTurned: Boolean get() = folder in setOf("codex", "cursor", "gemini", "opencode")

    fun hasPose(pose: String): Boolean = pose in (POSES[folder] ?: emptyList())

    companion object {
        val You = SceneActor("you")
        const val BOT = "bot"
        private val FULL = listOf("idle", "idle2", "lift", "throw", "hop")
        private val POSES = mapOf(
            "you" to listOf("idle", "lift", "throw", "hop"),
            "claude" to FULL, "codex" to FULL, "gemini" to FULL, "cursor" to FULL,
            "copilot" to FULL, "opencode" to FULL,
            "bot" to listOf("idle", "lift", "throw", "hop"),
        )
    }
}

/** What the scene screen needs of a plan, the day's or the week's. [end] and [calm] are in ms. */
interface ScenePlan {
    val end: Int
    val calm: Int
    fun frame(now: Int, calm: Boolean): SceneFrame
    fun moving(now: Int): Boolean
}

enum class OpKind { Actor, Good, GoodSpin, Prop, Rect }

/** One thing of a frame, in cells from the scene's top left. */
data class SceneOp(
    val kind: OpKind,
    val name: String = "",
    val actor: SceneActor? = null,
    val spin: Int = 0,
    val flip: Boolean = false,
    val x: Int,
    val y: Int,
    val w: Int = 0,
    val h: Int = 0,
    val argb: Int = 0,
)

data class SceneFrame(val sky: Sky, val ops: List<SceneOp>)

private fun rect(x: Int, y: Int, w: Int, h: Int, argb: Int) = SceneOp(OpKind.Rect, x = x, y = y, w = w, h = h, argb = argb)

private fun argb(a: Int, r: Int, g: Int, b: Int) = (a shl 24) or (r shl 16) or (g shl 8) or b

private fun round(f: Double): Int = floor(f + 0.5).toInt()

/** How many cells the scene's grid is shifted right in a column [cells] wide: negative crops both sides alike. */
fun sceneOffset(cells: Int): Int {
    val d = cells - SCENE_W
    return if (d < 0) -((-d + 1) / 2) else d / 2
}

/**
 * Where the scene is drawn: [cellPx] whole device pixels a cell, [offset] cells shifted right,
 * [heightPx] the box's height, and [top] the pixels left above the 80 cell grid (the grid sits on the
 * bottom edge); a negative [top] crops the sky.
 */
data class SceneLayout(val cellPx: Int, val offset: Int, val heightPx: Int, val top: Int)

/**
 * The day is drawn at the integer pixel size nearest to 160 dp for 80 cells and cropped equally at
 * both sides. A week needs all seven stacks, so it takes the largest whole cell that fits 264 cells
 * across, and the box is only as tall as that makes it.
 */
fun layoutScene(widthPx: Int, density: Float, week: Boolean): SceneLayout {
    val boxPx = round(160.0 * density)
    val dayCell = max(1, round(boxPx / SCENE_H.toDouble()))
    val cell = if (week) max(1, min(dayCell, widthPx / SCENE_W)) else dayCell
    val height = if (week) SCENE_H * cell else boxPx
    return SceneLayout(cell, sceneOffset(widthPx / cell), height, height - SCENE_H * cell)
}

private val HEAP_SLOTS = listOf(
    intArrayOf(115, 49), intArrayOf(133, 49), intArrayOf(124, 44), intArrayOf(116, 39),
    intArrayOf(132, 39), intArrayOf(124, 34), intArrayOf(124, 29),
)
private const val HEAP_CAP = 7
private const val MAX_ACTORS = 6
private const val OFF_EDGE = 8
private const val THROW_START = 700
private const val THROW_GAP = 480
private const val THROW_BUDGET = 5760
private const val MIN_GAP = 120
private const val LIFT_LEAD = 260
private const val THROW_HOLD = 180
private const val SINK_FOR = 110
private const val PLUS_FOR = 700
private const val HOP_GAP = 90
private const val HOP_FOR = 420
private const val JUMP_AFTER = 350
private const val JUMP_TOTAL = 900
private const val SWAY_AFTER = 600
private const val SWAY_BEAT = 450
private const val SWAY_STAGGER = 170
private const val SPIN_BEAT = 100
private const val LID_SHUT_TO = 250
private const val LID_HALF_TO = 350
private const val FLIES = 7

private val SLOTS_4 = intArrayOf(26, 66, 198, 160)
private val SLOTS_6 = intArrayOf(24, 52, 216, 188, 80, 160)

private val GROUND_SHADOW = argb(56, 30, 20, 40)
private val DUST = intArrayOf(argb(255, 0xff, 0xf6, 0xc0), argb(255, 0xfe, 0xe7, 0x61), argb(255, 0xe8, 0xa0, 0x3a))
private val PLUS_BRIGHT = argb(255, 255, 255, 255)
private val PLUS_DIM = argb(255, 0xc8, 0xc4, 0xe8)
private val FIREFLY_CORE = argb(255, 0xff, 0xf6, 0xb0)
private val FIREFLY_GLOW = argb(115, 254, 231, 97)
private val DIGIT_OUTLINE = argb(255, 0x3e, 0x27, 0x31)

/** One finished task as the day throws it: who, which good, and whether its pull request merged. */
data class DayThrow(val actor: Int, val kind: String, val starred: Boolean)

private class Flight(
    val start: Int, val dur: Int, val fromX: Double, val fromY: Double, val toX: Double, val toY: Double,
    val arc: Double, val dir: Int,
) {
    val land: Int get() = start + dur

    fun at(t: Int): Pair<Double, Double> {
        val u = ((t - start).toDouble() / dur).coerceIn(0.0, 1.0)
        return (fromX + (toX - fromX) * u) to (fromY + (toY - fromY) * u - arc * 4 * u * (1 - u))
    }
}

/** A day: everyone throws what they finished into the bin, in the order it was finished, then all hop. */
class DayPlan(private val sky: Sky, private val actors: List<SceneActor>, private val throws: List<DayThrow>) : ScenePlan {
    private val xs = IntArray(actors.size)
    private val flips = BooleanArray(actors.size)
    private val throwsOf = List(actors.size) { mutableListOf<Int>() }
    private val flights = mutableListOf<Flight>()
    private var last = 0
    private var jump = 0

    /** When the scene has played and the ledger starts to light up. */
    override var end = 0
        private set

    init {
        val slots = if (actors.size <= SLOTS_4.size) SLOTS_4 else SLOTS_6
        actors.forEachIndexed { i, a ->
            xs[i] = if (i < MAX_ACTORS) slots[i] else SCENE_W + OFF_EDGE
            flips[i] = xs[i] > BIN_X && !a.neverTurned
        }
        val n = throws.size
        val gap = if (n > 0) max(MIN_GAP, min(THROW_GAP, THROW_BUDGET / n)) else THROW_GAP
        throws.forEachIndexed { i, th ->
            throwsOf[th.actor].add(i)
            flights.add(plan(th, THROW_START + i * gap, i))
            last = max(last, flights[i].land)
        }
        if (n > 0) {
            jump = last + JUMP_AFTER
            end = jump + JUMP_TOTAL
        }
    }

    override val calm: Int get() = end + 2000

    private fun handsOf(a: SceneActor): Pair<Int, Int> = when (a.folder) {
        "codex" -> 2 to 15
        "you" -> 20 to 4
        SceneActor.BOT -> 12 to -4
        "cursor" -> 12 to -2
        else -> 12 to 2
    }

    private fun plan(th: DayThrow, start: Int, i: Int): Flight {
        val a = actors[th.actor]
        var x0 = xs[th.actor]
        var (fx, fy) = handsOf(a)
        when {
            th.actor >= MAX_ACTORS -> { fx = 0; x0 = SCENE_W + OFF_EDGE }
            flips[th.actor] -> fx = ACTOR_GRID - 1 - fx
        }
        val slot = HEAP_SLOTS[min(i, HEAP_CAP - 1)]
        val toX = (slot[0] + GOOD_GRID / 2).toDouble()
        val fromX = (x0 + fx).toDouble()
        val dx = abs(toX - fromX)
        return Flight(
            start = start, dur = (560 + dx * 2.2).toInt(), fromX = fromX, fromY = (ACTOR_TOP + fy).toDouble(),
            toX = toX, toY = (slot[1] + GOOD_GRID / 2).toDouble(), arc = 16 + dx / 5, dir = if (toX < fromX) -1 else 1,
        )
    }

    /** The cell the left edge of an actor stands on; past the sixth it is off the right edge. */
    fun actorX(i: Int): Int = xs[i]

    fun landed(now: Int): Int = flights.count { now >= it.land }

    /** Whether something travels (a good in the air, a hop), so the scene wants a frame every refresh. */
    override fun moving(now: Int): Boolean =
        throws.isNotEmpty() && now < jump + (actors.size - 1) * HOP_GAP + HOP_FOR

    private enum class Lid { Shut, Half, Open }

    private fun lidAt(now: Int) = when {
        throws.isEmpty() || now < LID_SHUT_TO -> Lid.Shut
        now < LID_HALF_TO -> Lid.Half
        else -> Lid.Open
    }

    private fun sinkAt(now: Int) = if (flights.any { now >= it.land && now < it.land + SINK_FOR }) 1 else 0

    override fun frame(now: Int, calm: Boolean): SceneFrame {
        val ops = mutableListOf<SceneOp>()
        val sink = sinkAt(now)
        val lid = lidAt(now)
        actors.forEachIndexed { i, _ -> if (i < MAX_ACTORS) ops.add(rect(xs[i] + 5, FEET + 1, 15, 2, GROUND_SHADOW)) }
        ops.add(rect(BIN_X - 1, FEET + 1, 42, 2, GROUND_SHADOW))
        if (lid == Lid.Open) ops.add(SceneOp(OpKind.Prop, "props/lid-open", x = BIN_X, y = BIN_RIM - 6 + sink))
        throws.forEachIndexed { i, th ->
            if (i < HEAP_CAP && now >= flights[i].land) {
                ops.add(SceneOp(OpKind.Good, th.kind, x = HEAP_SLOTS[i][0], y = HEAP_SLOTS[i][1] + sink))
            }
        }
        ops.addAll(flightOps(now))
        ops.add(SceneOp(OpKind.Prop, "props/bin", x = BIN_X, y = BIN_RIM + sink))
        when (lid) {
            Lid.Half -> ops.add(SceneOp(OpKind.Prop, "props/lid-half", x = BIN_X, y = BIN_RIM - 4))
            Lid.Shut -> ops.add(SceneOp(OpKind.Prop, "props/lid-shut", x = BIN_X - 1, y = BIN_RIM - 3))
            Lid.Open -> Unit
        }
        ops.addAll(actorOps(now, calm))
        ops.addAll(plusOps(now))
        if (throws.isNotEmpty() && now > jump) ops.addAll(fireflies(now))
        return SceneFrame(sky, ops)
    }

    private fun flightOps(now: Int): List<SceneOp> {
        val ops = mutableListOf<SceneOp>()
        throws.forEachIndexed { i, th ->
            val fl = flights[i]
            if (now < fl.start || now >= fl.land) return@forEachIndexed
            if (th.starred) ops.addAll(dust(fl, now))
            val (x, y) = fl.at(now)
            val turns = (((now - fl.start) / SPIN_BEAT * fl.dir) % 4 + 4) % 4
            ops.add(SceneOp(OpKind.GoodSpin, th.kind, spin = turns, x = round(x) - GOOD_GRID / 2, y = round(y) - GOOD_GRID / 2))
        }
        return ops
    }

    private fun dust(fl: Flight, now: Int): List<SceneOp> {
        val ops = mutableListOf<SceneOp>()
        for (k in 1..6) {
            val at = now - k * 45
            if (at < fl.start) break
            val (x, y) = fl.at(at)
            val c = when {
                k >= 5 -> DUST[2]
                k >= 3 -> DUST[1]
                else -> DUST[0]
            }
            ops.add(rect(round(x) + (k * 7) % 5 - 2, round(y) + (k * 3) % 5 - 2, 1, 1, c))
        }
        return ops
    }

    private fun actorOps(now: Int, calm: Boolean): List<SceneOp> {
        val ops = mutableListOf<SceneOp>()
        actors.forEachIndexed { i, a ->
            if (i >= MAX_ACTORS) return@forEachIndexed
            val (pose, throwIdx) = poseAt(i, now, calm)
            val y = ACTOR_TOP + hopAt(i, now) + bobAt(i, now, calm)
            ops.add(SceneOp(OpKind.Actor, pose, actor = a, flip = flips[i], x = xs[i], y = y))
            if (throwIdx >= 0 && pose == "lift" && a.folder != "codex") {
                val fl = flights[throwIdx]
                ops.add(
                    SceneOp(
                        OpKind.Good, throws[throwIdx].kind,
                        x = round(fl.fromX) - GOOD_GRID / 2, y = round(fl.fromY) - GOOD_GRID / 2 + (y - ACTOR_TOP),
                    ),
                )
            }
        }
        return ops
    }

    private fun poseAt(i: Int, now: Int, calm: Boolean): Pair<String, Int> {
        for (n in throwsOf[i]) {
            val start = flights[n].start
            if (now >= start - LIFT_LEAD && now < start) return "lift" to n
            if (now >= start && now < start + THROW_HOLD) return "throw" to n
        }
        if (throws.isNotEmpty()) {
            val at = now - jump - i * HOP_GAP
            if (at in 0 until HOP_FOR) return "hop" to -1
            if (!calm && now > jump + SWAY_AFTER && ((now + i * SWAY_STAGGER) / SWAY_BEAT) % 2 == 1 && actors[i].hasPose("idle2")) {
                return "idle2" to -1
            }
        }
        return "idle" to -1
    }

    private fun hopAt(i: Int, now: Int): Int =
        if (throws.isEmpty()) 0 else hopLift((now - jump - i * HOP_GAP).toDouble() / HOP_FOR)

    private fun bobAt(i: Int, now: Int, calm: Boolean): Int =
        if (calm || throws.isEmpty() || now <= jump + SWAY_AFTER) 0 else ((now + i * SWAY_STAGGER) / SWAY_BEAT) % 2

    private fun plusOps(now: Int): List<SceneOp> {
        val ops = mutableListOf<SceneOp>()
        flights.forEachIndexed { i, fl ->
            val age = now - fl.land
            if (age < 0 || age > PLUS_FOR) return@forEachIndexed
            val rise = (min(age / 600.0, 1.0) * 10).toInt()
            val ink = if (age >= 450) PLUS_DIM else PLUS_BRIGHT
            ops.addAll(digits("+1", BIN_X + 15, BIN_RIM - 14 - rise, ink))
            if (throws[i].starred) ops.add(SceneOp(OpKind.Prop, "props/star", x = BIN_X + 26, y = BIN_RIM - 16 - rise))
        }
        return ops
    }

    private fun fireflies(now: Int): List<SceneOp> {
        val ops = mutableListOf<SceneOp>()
        for (i in 0 until FLIES) {
            if ((now / 300 + i) % 3 == 0) continue
            val t = now.toDouble()
            val x = round(196 + 26 * sin(t / 1100 + i * 1.7) + i * 6)
            val y = round(30 + 12 * sin(t / 800 + i * 2.3))
            ops.add(rect(x - 1, y, 1, 1, FIREFLY_GLOW))
            ops.add(rect(x + 1, y, 1, 1, FIREFLY_GLOW))
            ops.add(rect(x, y, 1, 1, FIREFLY_CORE))
        }
        return ops
    }
}

/** The height, in cells, of a hop [u] (0 to 1) of the way through it, as a negative offset. */
fun hopLift(u: Double): Int = if (u <= 0 || u >= 1) 0 else -round(10 * 4 * u * (1 - u))

private const val STACK_WIDE = 2
private const val STACK_ROWS = 4
private const val STACK_STEP = 15
private const val WEEK_START = 400
private const val WEEK_GAP = 110
private const val WEEK_FALL = 520
private const val WEEK_BOUNCE = 0.35
private const val WEEK_HOP_EVERY = 1400
private const val WEEK_HOP_FOR = 420
private const val WEEK_AFTER = 900
private val STACK_SHADOW = argb(77, 10, 10, 30)

/** The left cell of the stack of a day of the week. */
fun dayX(d: Int): Int = 6 + d * 37

/** How wide a day's label is under the scene, in cells. */
const val DAY_LABEL_W = 33

private class WeekGood(val day: Int, val kind: String, val x: Int, val y: Int, var start: Int = 0) {
    /** Where the good is [now] ms into the scene, or null before it begins to fall. */
    fun fallY(now: Int): Int? {
        val u = (now - start).toDouble() / WEEK_FALL
        return when {
            u < 0 -> null
            u < 1 -> round(-20 + (y + 20) * u * u)
            u < 1 + WEEK_BOUNCE -> {
                val v = (u - 1) / WEEK_BOUNCE
                y - round(3.0 * 4 * v * (1 - v))
            }
            else -> y
        }
    }
}

/** A week at night: a stack of goods a day, falling one by one; whoever finished most hops beside the busiest. */
class WeekPlan(private val days: List<List<String>>, private val lead: SceneActor?) : ScenePlan {
    private val goods = mutableListOf<WeekGood>()
    private val more = IntArray(7)
    private var busiest = 0
    private val beside: Int

    /** When the goods have all landed and the lead starts to hop. */
    override var end = 0
        private set

    init {
        var best = -1
        days.forEachIndexed { d, kinds ->
            if (kinds.size > best) { best = kinds.size; busiest = d }
            val shown = min(kinds.size, STACK_ROWS * STACK_WIDE)
            more[d] = kinds.size - shown
            for (j in 0 until shown) {
                val row = j / STACK_WIDE
                val single = j == shown - 1 && j % STACK_WIDE == 0
                val x = if (single) dayX(d) + 8 else dayX(d) + (j % STACK_WIDE) * 17
                goods.add(WeekGood(d, kinds[j], x, FEET - GOOD_GRID - row * STACK_STEP))
            }
        }
        goods.forEachIndexed { n, g -> g.start = WEEK_START + n * WEEK_GAP }
        if (goods.isNotEmpty()) end = goods.last().start + WEEK_AFTER
        beside = listOf(busiest + 1, busiest - 1).firstOrNull { it in 0..6 && days[it].isEmpty() } ?: busiest
    }

    override val calm: Int get() = end + 2000

    fun more(day: Int): Int = more[day]

    override fun moving(now: Int): Boolean {
        if (goods.isEmpty()) return false
        if (now <= goods.last().start + (WEEK_FALL * (1 + WEEK_BOUNCE)).toInt()) return true
        return now > end && (now - end) % WEEK_HOP_EVERY < WEEK_HOP_FOR
    }

    private fun landedBy(day: Int): Int = goods.lastOrNull { it.day == day }?.let { it.start + WEEK_FALL } ?: 0

    override fun frame(now: Int, calm: Boolean): SceneFrame {
        val ops = mutableListOf<SceneOp>()
        days.forEachIndexed { d, kinds ->
            if (kinds.isNotEmpty()) ops.add(rect(dayX(d) - 1, FEET + 1, 35, 2, STACK_SHADOW))
        }
        for (g in goods) g.fallY(now)?.let { ops.add(SceneOp(OpKind.Good, g.kind, x = g.x, y = it)) }
        if (lead != null) {
            var pose = "idle"
            var y = ACTOR_TOP
            if (!calm && goods.isNotEmpty() && now > end) {
                val u = ((now - end) % WEEK_HOP_EVERY).toDouble() / WEEK_HOP_FOR
                if (u < 1) { pose = "hop"; y = ACTOR_TOP + hopLift(u) }
            }
            ops.add(SceneOp(OpKind.Actor, pose, actor = lead, x = dayX(beside) + 2, y = y))
        }
        for (d in 0 until 7) {
            val n = more[d]
            if (n > 0 && now >= landedBy(d)) {
                val label = "+$n"
                ops.addAll(digits(label, dayX(d) + DAY_LABEL_W / 2 - label.length * 2, 2, PLUS_BRIGHT))
            }
        }
        return SceneFrame(Sky.Night, ops)
    }
}

private val DIGIT_GLYPHS = mapOf(
    '+' to listOf("...", ".x.", "xxx", ".x.", "..."),
    '0' to listOf("xxx", "x.x", "x.x", "x.x", "xxx"),
    '1' to listOf(".x.", "xx.", ".x.", ".x.", "xxx"),
    '2' to listOf("xxx", "..x", "xxx", "x..", "xxx"),
    '3' to listOf("xxx", "..x", "xxx", "..x", "xxx"),
    '4' to listOf("x.x", "x.x", "xxx", "..x", "..x"),
    '5' to listOf("xxx", "x..", "xxx", "..x", "xxx"),
    '6' to listOf("xxx", "x..", "xxx", "x.x", "xxx"),
    '7' to listOf("xxx", "..x", ".x.", ".x.", ".x."),
    '8' to listOf("xxx", "x.x", "xxx", "x.x", "xxx"),
    '9' to listOf("xxx", "x.x", "xxx", "..x", "xxx"),
)

/** A number in 3 x 5 pixel digits at (x, y): a one cell dark outline, then the digits in [fill]. */
fun digits(s: String, x: Int, y: Int, fill: Int): List<SceneOp> {
    val cells = mutableListOf<Pair<Int, Int>>()
    s.forEachIndexed { i, ch ->
        DIGIT_GLYPHS[ch]?.forEachIndexed { gy, row ->
            row.forEachIndexed { gx, c -> if (c == 'x') cells.add((x + i * 4 + gx) to (y + gy)) }
        }
    }
    val ops = mutableListOf<SceneOp>()
    for ((cx, cy) in cells) {
        for ((dx, dy) in listOf(-1 to 0, 1 to 0, 0 to -1, 0 to 1)) ops.add(rect(cx + dx, cy + dy, 1, 1, DIGIT_OUTLINE))
    }
    for ((cx, cy) in cells) ops.add(rect(cx, cy, 1, 1, fill))
    return ops
}

/** The tint of a robot, by the agent's name as the desktop picks it: a brand color, else by the name's hash. */
fun botTintOf(name: String): Int {
    val n = name.lowercase()
    val brand = listOf("qwen" to 0x615CED, "kimi" to 0x1F6FEB, "windsurf" to 0x0EA5A4, "zed" to 0x3F9E4D, "cline" to 0xD9488B)
    brand.firstOrNull { n.contains(it.first) }?.let { return 0xFF000000.toInt() or it.second }
    val tints = intArrayOf(0x8E8E99, 0x0EA5A4, 0xD9488B, 0xE08A1E, 0x3F9E4D, 0x6B7CD9)
    if (n.isEmpty() || n == "agent") return 0xFF000000.toInt() or tints[0]
    var h = -2128831035 // FNV-1a, 32 bits
    for (b in n.toByteArray(Charsets.UTF_8)) h = (h xor (b.toInt() and 0xff)) * 16777619
    val idx = 1 + ((h.toLong() and 0xffffffffL) % (tints.size - 1)).toInt()
    return 0xFF000000.toInt() or tints[idx]
}

/** How a person of the ledger is drawn: you, an agent with a figure of its own, or a tinted robot. */
fun sceneActorOf(person: SummaryPerson, avatarOf: (String) -> String): SceneActor {
    if (person.you) return SceneActor.You
    val folder = avatarOf(person.agent)
    return if (folder == SceneActor.BOT) SceneActor(SceneActor.BOT, botTintOf(person.agent)) else SceneActor(folder)
}

/** The scene of a page: the day throws its goods in order, the week stacks them by day. */
fun scenePlanOf(page: SummaryPage, sky: Sky, avatarOf: (String) -> String): ScenePlan {
    val actors = page.ledger.map { sceneActorOf(it, avatarOf) }
    if (!page.week) {
        return DayPlan(sky, actors, page.goods.map { DayThrow(it.actor.coerceIn(0, max(0, actors.size - 1)), it.kind, it.starred) })
    }
    val days = List(7) { d -> page.goods.filter { it.day == d }.map { it.kind } }
    var lead = -1
    page.ledger.forEachIndexed { i, p -> if (lead < 0 || p.tasks.size > page.ledger[lead].tasks.size) lead = i }
    return WeekPlan(days, if (lead >= 0 && page.goods.isNotEmpty()) actors[lead] else null)
}

/** Day before 17:00, dusk from 17:00, night from 20:00 for today; a past day is dusk and the week is night. */
fun skyOf(week: Boolean, back: Long, hour: Int): Sky = when {
    week -> Sky.Night
    back > 0 -> Sky.Dusk
    else -> skyAt(hour)
}
