package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.data.SummaryGood
import xyz.sriai.adhdtodo.data.SummaryPage
import xyz.sriai.adhdtodo.data.SummaryPerson
import xyz.sriai.adhdtodo.data.SummaryTask
import xyz.sriai.adhdtodo.ui.DayPlan
import xyz.sriai.adhdtodo.ui.DayThrow
import xyz.sriai.adhdtodo.ui.OpKind
import xyz.sriai.adhdtodo.ui.SceneActor
import xyz.sriai.adhdtodo.ui.Sky
import xyz.sriai.adhdtodo.ui.WeekPlan
import xyz.sriai.adhdtodo.ui.botTintOf
import xyz.sriai.adhdtodo.ui.dayX
import xyz.sriai.adhdtodo.ui.digits
import xyz.sriai.adhdtodo.ui.hopLift
import xyz.sriai.adhdtodo.ui.layoutScene
import xyz.sriai.adhdtodo.ui.ledgerShown
import xyz.sriai.adhdtodo.ui.scenePlanOf
import xyz.sriai.adhdtodo.ui.sceneOffset
import xyz.sriai.adhdtodo.ui.skyAt
import xyz.sriai.adhdtodo.ui.skyOf
import xyz.sriai.adhdtodo.ui.summaryPeriodTitle
import java.util.Locale

/** The scene's geometry and timings, as app/theme/scene*.go has them. */
class SummaryPlanTest {
    private val you = SceneActor.You
    private val claude = SceneActor("claude")

    @Test
    fun `a narrow column crops both sides alike and a wide one extends the sky`() {
        assertEquals(0, sceneOffset(264))
        assertEquals(-30, sceneOffset(205)) // 59 cells too many: 30 off the left, 29 off the right
        assertEquals(-30, sceneOffset(204))
        assertEquals(18, sceneOffset(300))
    }

    @Test
    fun `the day takes whole pixels a cell near 160 dp and the week fits all seven stacks`() {
        // 411 dp at xxhdpi is 1233 px: 2 dp is 6 px a cell, 80 cells are exactly 160 dp.
        val day = layoutScene(1233, 3f, week = false)
        assertEquals(6, day.cellPx)
        assertEquals(480, day.heightPx)
        assertEquals(0, day.top)
        assertEquals(sceneOffset(1233 / 6), day.offset)
        val week = layoutScene(1233, 3f, week = true)
        assertEquals(4, week.cellPx) // 4 px x 264 cells <= 1233
        assertTrue(week.offset >= 0)
        assertEquals(320, week.heightPx)
        // 2.625 density: 160 dp is 420 px, 5 px a cell, the 20 px left over go to the sky above the grid.
        val mid = layoutScene(1079, 2.625f, week = false)
        assertEquals(5, mid.cellPx)
        assertEquals(420, mid.heightPx)
        assertEquals(20, mid.top)
    }

    @Test
    fun `the sky follows the clock for today, dusk for a past day, night for the week`() {
        assertEquals(Sky.Day, skyAt(16))
        assertEquals(Sky.Dusk, skyAt(17))
        assertEquals(Sky.Dusk, skyAt(19))
        assertEquals(Sky.Night, skyAt(20))
        assertEquals(Sky.Dusk, skyOf(week = false, back = 1, hour = 9))
        assertEquals(Sky.Night, skyOf(week = true, back = 0, hour = 9))
        assertEquals(Sky.Day, skyOf(week = false, back = 0, hour = 9))
    }

    @Test
    fun `goods are thrown 480 ms apart in finishing order and the lid opens first`() {
        val plan = DayPlan(Sky.Day, listOf(you, claude), listOf(DayThrow(0, "mail", false), DayThrow(1, "flask", false), DayThrow(0, "gem", false)))
        fun props(now: Int) = plan.frame(now, false).ops.filter { it.kind == OpKind.Prop }.map { it.name }
        assertTrue("props/lid-shut" in props(0))
        assertTrue("props/lid-half" in props(300))
        assertTrue("props/lid-open" in props(400))
        // The first throw leaves at 700 ms, the next 480 ms later, then 480 ms after that.
        fun pose(actor: Int, now: Int) = plan.frame(now, false).ops.filter { it.kind == OpKind.Actor }[actor].name
        assertEquals("idle", pose(0, 400))
        assertEquals("lift", pose(0, 500))
        assertEquals("throw", pose(0, 750))
        assertEquals("lift", pose(1, 1000))
        assertEquals("throw", pose(1, 1200))
        assertEquals(0, plan.landed(700))
        assertTrue(plan.landed(plan.end) == 3)
        assertTrue(plan.end > 1660)
    }

    @Test
    fun `the bin dips and a plus one rises as a good lands, a starred one trails gold`() {
        val plan = DayPlan(Sky.Day, listOf(you), listOf(DayThrow(0, "gem", true)))
        // Find the landing by walking the frames: the good in the air is a spinning one.
        val flying = (700..1400 step 20).first { t -> plan.frame(t, false).ops.any { it.kind == OpKind.GoodSpin } }
        val dust = plan.frame(flying + 200, false).ops.filter { it.kind == OpKind.Rect && it.w == 1 && it.h == 1 }
        assertTrue("a starred good in flight has gold dust behind it", dust.isNotEmpty())
        var landing = flying
        while (plan.landed(landing) == 0) landing += 10
        val ops = plan.frame(landing, false).ops
        val bin = ops.first { it.name == "props/bin" }
        assertEquals(55 + 1, bin.y) // one cell lower than at rest, the rim being 55
        assertTrue("the +1 is drawn in cells", ops.count { it.kind == OpKind.Rect } > 20)
        assertTrue("a star beside it", ops.any { it.name == "props/star" })
        val later = plan.frame(landing + 300, false).ops
        assertEquals(55, later.first { it.name == "props/bin" }.y)
    }

    @Test
    fun `the throws are squeezed into 5760 ms when a day has many`() {
        val many = List(24) { DayThrow(0, "gem", false) }
        val plan = DayPlan(Sky.Day, listOf(you), many)
        // 24 throws: gap = max(120, min(480, 5760 / 24)) = 240, the last leaves at 700 + 23 * 240.
        assertEquals(24, plan.landed(plan.end))
        assertTrue(plan.end < 700 + 24 * 480)
        // The heap draws seven of them.
        val goods = plan.frame(plan.end, false).ops.filter { it.kind == OpKind.Good && it.y > 20 && it.x in 100..160 }
        assertEquals(7, goods.size)
    }

    @Test
    fun `everyone hops once after the last landing and then sways, unless calm`() {
        val plan = DayPlan(Sky.Day, listOf(you, claude), listOf(DayThrow(0, "gem", false), DayThrow(1, "gem", false)))
        val lastLanding = (0..plan.end).first { plan.landed(it) == 2 }
        val hop = plan.frame(lastLanding + 350 + 200, false).ops.filter { it.kind == OpKind.Actor }
        assertEquals("hop", hop[0].name)
        assertTrue(hop[0].y < 49)
        val sway = (plan.end + 700..plan.end + 1700 step 50).map { plan.frame(it, false).ops.filter { o -> o.kind == OpKind.Actor }[1].name }
        assertTrue("claude has an idle2 to sway with", "idle2" in sway)
        val still = (plan.end + 700..plan.end + 1700 step 50).map { plan.frame(it, true).ops.filter { o -> o.kind == OpKind.Actor }[1].name }
        assertFalse("calm never sways", "idle2" in still)
        assertEquals(0, hopLift(0.0))
        assertEquals(-10, hopLift(0.5))
    }

    @Test
    fun `a seventh actor throws from off the right edge and is not drawn`() {
        val actors = List(7) { if (it == 0) you else SceneActor("claude") }
        val plan = DayPlan(Sky.Day, actors, listOf(DayThrow(6, "gem", false)))
        assertEquals(264 + 8, plan.actorX(6))
        assertEquals(6, plan.frame(plan.end, false).ops.count { it.kind == OpKind.Actor })
    }

    @Test
    fun `an empty day keeps the lid shut and has no end`() {
        val plan = DayPlan(Sky.Dusk, emptyList(), emptyList())
        val ops = plan.frame(5000, false).ops
        assertTrue(ops.any { it.name == "props/lid-shut" })
        assertFalse(plan.moving(0))
        assertEquals(0, plan.end)
    }

    @Test
    fun `the week stacks goods by day, two wide and four rows, and counts the rest`() {
        val days = listOf(listOf("gem"), List(10) { "jar" }, emptyList(), emptyList(), emptyList(), emptyList(), emptyList())
        val plan = WeekPlan(days, claude)
        assertEquals(2, plan.more(1))
        assertEquals(0, plan.more(0))
        val settled = plan.frame(plan.end, true)
        assertEquals(Sky.Night, settled.sky)
        val goods = settled.ops.filter { it.kind == OpKind.Good }
        assertEquals(9, goods.size) // 1 + 8 drawn
        val stack = goods.filter { it.x in dayX(1)..dayX(1) + 40 }
        assertEquals(setOf(55, 40, 25, 10), stack.map { it.y }.toSet())
        // The lead stands in the empty day after the busiest one.
        val lead = settled.ops.first { it.kind == OpKind.Actor }
        assertEquals(dayX(2) + 2, lead.x)
        // Goods fall one by one, 110 ms apart: at the start nothing, then the first.
        assertTrue(plan.frame(0, false).ops.none { it.kind == OpKind.Good })
        assertEquals(1, plan.frame(420, false).ops.count { it.kind == OpKind.Good })
        // Last, the lead hops once every 1.4 s; calm stands still.
        val hops = (plan.end..plan.end + 1400 step 30).count { plan.frame(it, false).ops.first { o -> o.kind == OpKind.Actor }.name == "hop" }
        assertTrue(hops > 5)
        assertTrue((plan.end..plan.end + 1400 step 30).none { plan.frame(it, true).ops.first { o -> o.kind == OpKind.Actor }.name == "hop" })
    }

    @Test
    fun `a page becomes a day or a week plan with the ledger's actors`() {
        val ledger = listOf(
            SummaryPerson(you = true, name = "you", count = 1, tasks = listOf(task(1))),
            SummaryPerson(agent = "Claude Code", name = "Claude Code", count = 2, tasks = listOf(task(2), task(3))),
        )
        val goods = listOf(SummaryGood(0, "mail", false, 0, 1), SummaryGood(1, "flask", true, 1, 2), SummaryGood(1, "gem", false, 1, 3))
        val avatarOf = { name: String -> if (name.contains("Claude")) "claude" else "bot" }
        val week = scenePlanOf(SummaryPage(week = true, ledger = ledger, goods = goods), Sky.Night, avatarOf)
        // The lead is whoever finished most: Claude, who stands in the empty day beside the busiest (day 1).
        val actor = week.frame(week.end, true).ops.first { it.kind == OpKind.Actor }
        assertEquals("claude", actor.actor?.folder)
        val day = scenePlanOf(SummaryPage(ledger = ledger, goods = goods), Sky.Day, avatarOf)
        assertEquals(2, day.frame(day.end, true).ops.count { it.kind == OpKind.Actor })
        assertEquals(3, (day as DayPlan).landed(day.end))
    }

    @Test
    fun `the ledger lights up a row every 420 ms after the scene and counts up 60 ms a step`() {
        assertEquals(-1, ledgerShown(at = 5000, end = 5000 + 1, row = 0, count = 3))
        assertEquals(0, ledgerShown(at = 5000, end = 5000, row = 0, count = 3))
        assertEquals(2, ledgerShown(at = 5000 + 120, end = 5000, row = 0, count = 3))
        assertEquals(3, ledgerShown(at = 5000 + 10_000, end = 5000, row = 0, count = 3))
        assertEquals(-1, ledgerShown(at = 5000 + 400, end = 5000, row = 1, count = 3))
        assertEquals(0, ledgerShown(at = 5000 + 420, end = 5000, row = 1, count = 3))
    }

    @Test
    fun `pixel digits are outlined`() {
        val ops = digits("+1", 10, 10, 0xFFFFFFFF.toInt())
        // '+' has 5 cells and '1' has 8; each gets a 4 cell outline before it is filled.
        assertEquals(13 * 5, ops.size)
        assertTrue(ops.takeLast(13).all { it.argb == 0xFFFFFFFF.toInt() })
    }

    @Test
    fun `an agent without a figure gets a robot tint by its name`() {
        assertEquals(0xFF8E8E99.toInt(), botTintOf(""))
        assertEquals(0xFF0EA5A4.toInt(), botTintOf("Windsurf"))
        assertNotEquals(0xFF8E8E99.toInt(), botTintOf("Aider"))
        assertEquals(botTintOf("Aider"), botTintOf("aider"))
    }

    @Test
    fun `the period title is a day, or a week with its month once`() {
        assertEquals("Tue, Oct 20", summaryPeriodTitle("2026-10-20", "2026-10-20", false, Locale.US))
        assertEquals("Oct 19 – 25", summaryPeriodTitle("2026-10-19", "2026-10-25", true, Locale.US))
        assertEquals("Oct 26 – Nov 1", summaryPeriodTitle("2026-10-26", "2026-11-01", true, Locale.US))
        assertEquals("10月20日 周二", summaryPeriodTitle("2026-10-20", "2026-10-20", false, Locale.SIMPLIFIED_CHINESE))
        assertEquals("10月19日 – 25日", summaryPeriodTitle("2026-10-19", "2026-10-25", true, Locale.SIMPLIFIED_CHINESE))
    }

    private fun task(id: Long) = SummaryTask(id = id, title = "t$id")
}
