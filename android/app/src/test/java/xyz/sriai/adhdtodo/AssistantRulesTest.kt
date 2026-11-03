package xyz.sriai.adhdtodo

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import xyz.sriai.adhdtodo.data.AssistantBackend
import xyz.sriai.adhdtodo.data.AssistantChange
import xyz.sriai.adhdtodo.data.AssistantDetail
import xyz.sriai.adhdtodo.data.AssistantPlace
import xyz.sriai.adhdtodo.data.AssistantQuestion
import xyz.sriai.adhdtodo.data.AssistantReceipt
import xyz.sriai.adhdtodo.data.AssistantReply
import xyz.sriai.adhdtodo.data.ModelPhase
import xyz.sriai.adhdtodo.data.ModelState
import xyz.sriai.adhdtodo.data.PageJson
import xyz.sriai.adhdtodo.ui.AnswerCard
import xyz.sriai.adhdtodo.ui.AssistantState
import xyz.sriai.adhdtodo.ui.CardUndo
import xyz.sriai.adhdtodo.ui.QuietNote
import xyz.sriai.adhdtodo.ui.Tab
import xyz.sriai.adhdtodo.ui.TurnOutcome
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.answerCard
import xyz.sriai.adhdtodo.ui.assistantPlace
import xyz.sriai.adhdtodo.ui.outcomeOf
import xyz.sriai.adhdtodo.ui.quietForModel
import xyz.sriai.adhdtodo.ui.runTurn
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

/** The assistant's rules (v44): which outcome a reply is, where a request is typed, and what a card says. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class AssistantRulesTest {
    private val zone = ZoneId.of("Asia/Shanghai")
    private val tuesday = Instant.parse("2026-11-03T01:41:00Z")
    private val resources get() = RuntimeEnvironment.getApplication().resources
    private val words get() = Words(resources, { tuesday }, zone)

    private fun receipt(id: Long, task: Long, kind: String, title: String = "t$id", detail: AssistantDetail = AssistantDetail(kind = kind)) =
        AssistantReceipt(id = id, taskId = task, title = title, detail = detail)

    @Test
    fun `a reply with no problem is an answer`() {
        val reply = AssistantReply(text = "ok")
        assertEquals(TurnOutcome.Answer(reply), outcomeOf(reply))
    }

    @Test
    fun `the quota, a signed out session and a dead network fall back to the local rules with their line`() {
        assertEquals(TurnOutcome.Fallback(QuietNote.USED_UP), outcomeOf(AssistantReply(problem = "quota")))
        assertEquals(TurnOutcome.Fallback(QuietNote.SIGNED_OUT), outcomeOf(AssistantReply(problem = "signed_out")))
        assertEquals(TurnOutcome.Fallback(QuietNote.UNREACHABLE), outcomeOf(AssistantReply(problem = "unreachable")))
        assertEquals(TurnOutcome.Fallback(QuietNote.UNREACHABLE), outcomeOf(AssistantReply(problem = "timeout")))
        // A server with no model needs no line: the chip and Settings already say so.
        assertEquals(TurnOutcome.Fallback(null), outcomeOf(AssistantReply(problem = "off")))
    }

    @Test
    fun `a failure after a change keeps its receipts so they can be undone`() {
        val made = AssistantReply(receipts = listOf(receipt(1, 5, "added")), problem = "quota")
        assertEquals(TurnOutcome.Answer(made), outcomeOf(made))
        assertEquals(TurnOutcome.Stopped, outcomeOf(AssistantReply(problem = "cancelled")))
        assertEquals(TurnOutcome.Answer(made.copy(problem = "cancelled")), outcomeOf(made.copy(problem = "cancelled")))
        assertEquals(TurnOutcome.Busy, outcomeOf(AssistantReply(problem = "busy")))
    }

    @Test
    fun `the quiet line follows the model state when nothing was asked`() {
        assertEquals(QuietNote.SIGNED_OUT, quietForModel(ModelState(phase = ModelPhase.SIGNED_OUT)))
        assertEquals(QuietNote.USED_UP, quietForModel(ModelState(phase = ModelPhase.USED_UP)))
        assertNull(quietForModel(ModelState(phase = ModelPhase.HIDDEN)))
        assertNull(quietForModel(ModelState("official", ModelPhase.READY, 86, 100)))
    }

    @Test
    fun `the model asks only when it is official and ready, and never on a task page`() {
        val state = AssistantState()
        assertFalse(state.asks(onTaskPage = false))
        state.model = ModelState("official", ModelPhase.READY, 86, 100)
        assertTrue(state.asks(onTaskPage = false))
        assertFalse(state.asks(onTaskPage = true))
        state.model = ModelState("none", ModelPhase.USED_UP, 0, 100)
        assertFalse(state.asks(onTaskPage = false))
    }

    @Test
    fun `a request is typed in the page it came from, in the phone's zone`() {
        val day = LocalDate.of(2026, 11, 5)
        val today = assistantPlace("home", Tab.MINE, day, categoryId = 3, zone = zone, now = tuesday)
        assertEquals(AssistantPlace(kind = "today", categoryId = 3, zone = "Asia/Shanghai", offset = 28800), today)
        assertEquals("project", assistantPlace("project:7", Tab.PROJECTS, day, 0, zone, tuesday).kind)
        assertEquals(7L, assistantPlace("project:7", Tab.PROJECTS, day, 0, zone, tuesday).projectId)
        val task = assistantPlace("task:9:1", Tab.MINE, day, 0, zone, tuesday)
        assertEquals("task" to 9L, task.kind to task.taskId)
        val calendar = assistantPlace("home", Tab.CALENDAR, day, 0, zone, tuesday)
        assertEquals("calendar" to "2026-11-05", calendar.kind to calendar.day)
        assertEquals("inbox", assistantPlace("home", Tab.PROJECTS, day, 0, zone, tuesday).kind)
        // The place is the JSON the core reads.
        val json = PageJson.encodeToString(AssistantPlace.serializer(), calendar)
        for (part in listOf("\"kind\":\"calendar\"", "\"day\":\"2026-11-05\"", "\"zone\":\"Asia/Shanghai\"", "\"offset\":28800")) {
            assertTrue("$part in $json", json.contains(part))
        }
    }

    @Test
    fun `a card counts what the turn changed and names what each row read`() {
        val reply = AssistantReply(
            receipts = listOf(
                receipt(1, 2, "reminder", "Call the landlord", AssistantDetail(kind = "reminder", at = "2026-11-04T09:00:00+08:00")),
                receipt(2, 7, "added", "Fix the login bug", AssistantDetail(kind = "added", project = "withdo", priority = "P1")),
                receipt(3, 4, "updated", "Renew the passport", AssistantDetail(kind = "updated", changes = listOf(AssistantChange("due", at = "2026-11-06T18:00:00+08:00")))),
                receipt(4, 5, "done", "Pay rent"),
            ),
        )
        val card = answerCard(resources, words, reply)
        assertEquals("Added 2, Updated 1, Finished 1", card.summary)
        assertEquals(listOf("Tomorrow 9:00 AM"), card.rows[0].chips)
        assertEquals(listOf("P1", "withdo"), card.rows[1].chips)
        assertEquals(listOf("due Fri"), card.rows[2].chips)
        assertEquals(listOf("Done"), card.rows[3].chips)
        // Undo takes the changes back last to first; the first added row is the one that glows.
        assertEquals(CardUndo.Receipts(listOf(4, 3, 2, 1)), card.undo)
        assertEquals(2L, card.newTaskId)
    }

    @Test
    fun `a question card has its options and no undo, a plain answer is its paragraph alone`() {
        val question = answerCard(resources, words, AssistantReply(question = AssistantQuestion("Which project?", listOf("withdo", "picpocket"))))
        assertEquals(listOf("withdo", "picpocket"), question.question?.options)
        assertNull(question.undo)
        assertEquals("", question.summary)
        val plain = answerCard(resources, words, AssistantReply(text = "Five things."))
        assertEquals(AnswerCard(text = "Five things."), plain)
    }

    @Test
    fun `a note the assistant keeps is counted but has no row`() {
        val card = answerCard(resources, words, AssistantReply(receipts = listOf(receipt(1, 0, "remembered", "Prefers 9:00"))))
        assertEquals("Noted", card.summary)
        assertTrue(card.rows.isEmpty())
    }

    private class Fake(private val reply: AssistantReply) : AssistantBackend {
        val entered = CountDownLatch(1)
        val release = CountDownLatch(1)
        var text = ""
        var place: AssistantPlace? = null
        override fun modelState() = ModelState()
        override fun ask(text: String, place: AssistantPlace): AssistantReply {
            this.text = text
            this.place = place
            entered.countDown()
            release.await(5, TimeUnit.SECONDS)
            return reply
        }
        override fun undo(receiptId: Long) = Unit
        override fun stop() = release.countDown()
    }

    @Test
    fun `the box thinks while the model works, then the answer is a card`() = runBlocking {
        val state = AssistantState()
        val backend = Fake(AssistantReply(text = "Done."))
        val place = AssistantPlace(kind = "today")
        val turn = launch(Dispatchers.Default) { state.runTurn(backend, resources, words, "hello", place) }
        assertTrue(backend.entered.await(5, TimeUnit.SECONDS))
        assertTrue(state.thinking)
        assertNull(state.card)
        backend.release.countDown()
        turn.join()
        assertFalse(state.thinking)
        assertEquals("Done.", state.card?.text)
        assertEquals("hello" to place, backend.text to backend.place)
    }

    @Test
    fun `stopping a turn leaves no card and puts the text back in the box`() = runBlocking {
        val state = AssistantState()
        val backend = Fake(AssistantReply(problem = "cancelled"))
        var outcome: TurnOutcome? = null
        val turn = launch(Dispatchers.Default) { outcome = state.runTurn(backend, resources, words, "hello", AssistantPlace()) }
        assertTrue(backend.entered.await(5, TimeUnit.SECONDS))
        backend.stop()
        turn.join()
        assertEquals(TurnOutcome.Stopped, outcome)
        assertFalse(state.thinking)
        assertNull(state.card)
    }
}
