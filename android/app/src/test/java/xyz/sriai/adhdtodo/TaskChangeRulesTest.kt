package xyz.sriai.adhdtodo

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.unit.dp
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.QuickDraft
import xyz.sriai.adhdtodo.data.QuickParse
import xyz.sriai.adhdtodo.ui.DefaultDueTime
import xyz.sriai.adhdtodo.ui.DueKind
import xyz.sriai.adhdtodo.ui.HighlightTransformation
import xyz.sriai.adhdtodo.ui.SelectionUi
import xyz.sriai.adhdtodo.ui.dueDay
import xyz.sriai.adhdtodo.ui.dueOptions
import xyz.sriai.adhdtodo.ui.dueRowDate
import xyz.sriai.adhdtodo.ui.dueStamp
import xyz.sriai.adhdtodo.ui.mineOrder
import xyz.sriai.adhdtodo.ui.quickHighlights
import xyz.sriai.adhdtodo.ui.readAny
import xyz.sriai.adhdtodo.ui.sectionIds
import xyz.sriai.adhdtodo.ui.toggled
import java.time.LocalDate
import java.time.ZoneId
import java.util.Locale

/** The plain rules behind the task page's Due page, the composer's reading and select mode. */
class TaskChangeRulesTest {
    private val zone = ZoneId.of("Asia/Shanghai")

    @Test
    fun `due options on a Monday are today, tomorrow, this Friday and next Monday`() {
        val options = dueOptions(LocalDate.of(2026, 11, 2)) // a Monday
        assertEquals(listOf(DueKind.Today, DueKind.Tomorrow, DueKind.Friday, DueKind.Monday), options.map { it.kind })
        assertEquals(LocalDate.of(2026, 11, 6), options[2].date)
        assertEquals(LocalDate.of(2026, 11, 9), options[3].date)
    }

    @Test
    fun `a friday or monday that is today or tomorrow is not listed twice`() {
        // Thursday: Friday is tomorrow.
        assertEquals(
            listOf(DueKind.Today, DueKind.Tomorrow, DueKind.Monday),
            dueOptions(LocalDate.of(2026, 11, 5)).map { it.kind },
        )
        // Friday: This Friday is today.
        assertEquals(
            listOf(DueKind.Today, DueKind.Tomorrow, DueKind.Monday),
            dueOptions(LocalDate.of(2026, 11, 6)).map { it.kind },
        )
        // Sunday: next Monday is tomorrow.
        assertEquals(
            listOf(DueKind.Today, DueKind.Tomorrow, DueKind.Friday),
            dueOptions(LocalDate.of(2026, 11, 8)).map { it.kind },
        )
    }

    @Test
    fun `a due day is stored at 18 00 with seconds, or keeps the time the task had`() {
        assertEquals(
            "2026-11-06T18:00:00+08:00",
            dueStamp(LocalDate.of(2026, 11, 6), null, zone),
        )
        assertEquals(
            "2026-11-06T15:30:00+08:00",
            dueStamp(LocalDate.of(2026, 11, 6), "2026-11-02T07:30:00Z", zone),
        )
        assertEquals(DefaultDueTime, java.time.LocalTime.of(18, 0))
    }

    @Test
    fun `a due time reads as the local day`() {
        assertEquals(LocalDate.of(2026, 11, 7), dueDay("2026-11-06T20:00:00Z", zone))
        assertEquals(null, dueDay(null, zone))
        assertEquals(null, dueDay("", zone))
    }

    @Test
    fun `due rows say the date in the language`() {
        val day = LocalDate.of(2026, 11, 6)
        assertEquals("Fri, Nov 6", dueRowDate(day, Locale.US))
        assertEquals("11月6日 周五", dueRowDate(day, Locale.SIMPLIFIED_CHINESE))
    }

    @Test
    fun `read ranges become highlights clamped to the text`() {
        val parse = QuickParse(listOf(QuickDraft(title = "x", remindAt = "r", ranges = listOf(listOf(0, 5), listOf(8, 99), listOf(4, 4), listOf(-1, 2)))))
        assertEquals(listOf(0 until 5, 8 until 10), quickHighlights(parse, 10))
        assertEquals(emptyList<IntRange>(), quickHighlights(null, 10))
        assertTrue(parse.readAny())
        assertFalse(QuickParse(listOf(QuickDraft(title = "plain"))).readAny())
        assertFalse(QuickParse().readAny())
    }

    @Test
    fun `the highlight keeps the text and styles only the read words`() {
        val style = SpanStyle(background = androidx.compose.ui.graphics.Color.Red)
        val out = HighlightTransformation(listOf(0 until 3), style).filter(AnnotatedString("abcdef"))
        assertEquals("abcdef", out.text.text)
        assertEquals(1, out.text.spanStyles.size)
        assertEquals(0, out.text.spanStyles[0].start)
        assertEquals(3, out.text.spanStyles[0].end)
        assertEquals(4, out.offsetMapping.originalToTransformed(4))
    }

    @Test
    fun `ticking toggles a task in and out of the set`() {
        assertEquals(setOf(1L, 2L), setOf(1L).toggled(2))
        assertEquals(emptySet<Long>(), setOf(1L).toggled(1))
    }

    @Test
    fun `sections join with a zero between and empty ones leave no gap`() {
        assertEquals(listOf(1L, 2L, 0L, 3L), sectionIds(listOf(1, 2), emptyList(), listOf(3)))
        assertEquals(emptyList<Long>(), sectionIds())
    }

    @Test
    fun `today's order is its sections in order and the closed fold counts for nothing`() {
        val page = MinePage(
            today = listOf(Item(id = 1), Item(id = 2)),
            next = listOf(Item(id = 3)),
            noDate = listOf(Item(id = 4)),
            done = listOf(Item(id = 5)),
        )
        assertEquals(listOf(1L, 2L, 0L, 3L, 0L, 5L), mineOrder(page, noDateOpen = false))
        assertEquals(listOf(1L, 2L, 0L, 3L, 0L, 4L, 0L, 5L), mineOrder(page, noDateOpen = true))
    }

    @Test
    fun `selected rows join their selected neighbours and not across a section`() {
        val selection = SelectionUi(setOf(1, 2, 3), listOf(1, 2, 0, 3), toggle = {}, report = {})
        assertTrue(selection.isSelected(1))
        assertFalse(selection.isSelected(0))
        assertFalse(selection.isSelected(9))
        // 1 is the top of a block, 2 its bottom (the separator breaks it), 3 a block of one.
        assertEquals(RoundedCornerShape(10.dp, 10.dp, 0.dp, 0.dp), selection.shape(1))
        assertEquals(RoundedCornerShape(0.dp, 0.dp, 10.dp, 10.dp), selection.shape(2))
        assertEquals(RoundedCornerShape(10.dp), selection.shape(3))
    }
}
