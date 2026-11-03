package xyz.sriai.adhdtodo.ui

import android.content.res.Resources
import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.AssistantBackend
import xyz.sriai.adhdtodo.data.AssistantDetail
import xyz.sriai.adhdtodo.data.AssistantPlace
import xyz.sriai.adhdtodo.data.AssistantQuestion
import xyz.sriai.adhdtodo.data.AssistantReceipt
import xyz.sriai.adhdtodo.data.AssistantReply
import xyz.sriai.adhdtodo.data.ModelState
import xyz.sriai.adhdtodo.data.QuickDraft
import xyz.sriai.adhdtodo.data.QuickParse
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId

/*
 * The composer's assistant (v44). When the official model is there the box asks it instead of
 * reading the text with local rules: the text stays while it thinks, and the answer is a card above
 * the box. This file is the state and the rules, without drawing (AssistantCard.kt draws them).
 */

/** What is left unsaid when the local rules answered instead of the model: the quiet line under the card. */
enum class QuietNote { SIGNED_OUT, USED_UP, UNREACHABLE }

/** How a card takes its changes back: the receipts of the turn, or the tasks the local rules made. */
sealed interface CardUndo {
    data class Receipts(val ids: List<Long>) : CardUndo
    data class Tasks(val ids: List<Long>) : CardUndo
}

/** One change of the card: the task, and what the assistant read or did in short chips (`Tomorrow 9:00 AM`). */
data class AnswerRow(val taskId: Long, val title: String, val chips: List<String>, val status: String)

/**
 * The card above the box. [summary] counts the changes (empty for a plain answer or a question);
 * [text] is a plain answer; [question] shows its options as pills; [undo] is null when there is
 * nothing to take back.
 */
data class AnswerCard(
    val summary: String = "",
    val rows: List<AnswerRow> = emptyList(),
    val text: String = "",
    val question: AssistantQuestion? = null,
    val undo: CardUndo? = null,
    val quiet: QuietNote? = null,
    /** The row's task to wash in iris 3 for 2 s: the first one the turn added, 0 for none. */
    val newTaskId: Long = 0,
)

/** What the composer shows of the assistant: the model, the turn in flight and the card. */
@Stable
class AssistantState {
    var model by mutableStateOf(ModelState())

    /** A turn is waiting for the model: the box keeps its text and Send is Stop. */
    var thinking by mutableStateOf(false)
    var card by mutableStateOf<AnswerCard?>(null)

    /** The model chip's sheet is open. */
    var sheet by mutableStateOf(false)

    /** The card and its quiet line close on the next send or a touch outside. */
    fun dismiss() {
        card = null
    }

    /** The box asks the model: official, ready, and not on a task page (there the box replies). */
    fun asks(onTaskPage: Boolean): Boolean = model.official && !onTaskPage
}

/** What came of one Ask. */
sealed interface TurnOutcome {
    /** The model answered (or stopped after changing something: its receipts can still be undone). */
    data class Answer(val reply: AssistantReply) : TurnOutcome

    /** The model could not answer: the local rules take the text, and [note] says what was left out. */
    data class Fallback(val note: QuietNote?) : TurnOutcome

    /** The person pressed Stop before anything was changed. */
    data object Stopped : TurnOutcome

    /** A turn is already running. */
    data object Busy : TurnOutcome
}

/** The rule of the outcome: receipts are never dropped, so a failure after a change still shows its Undo. */
fun outcomeOf(reply: AssistantReply): TurnOutcome = when {
    reply.problem.isEmpty() -> TurnOutcome.Answer(reply)
    reply.problem == "busy" -> TurnOutcome.Busy
    reply.receipts.isNotEmpty() -> TurnOutcome.Answer(reply)
    reply.problem == "cancelled" -> TurnOutcome.Stopped
    else -> TurnOutcome.Fallback(quietFor(reply.problem))
}

/** The quiet line a problem leaves; the server having no model needs none (the chip and Settings say so). */
fun quietFor(problem: String): QuietNote? = when (problem) {
    "quota" -> QuietNote.USED_UP
    "signed_out" -> QuietNote.SIGNED_OUT
    "off" -> null
    else -> QuietNote.UNREACHABLE
}

/** The quiet line the local rules leave when no model answers at all: signed out or used up. */
fun quietForModel(model: ModelState): QuietNote? = when (model.phase) {
    "signed_out" -> QuietNote.SIGNED_OUT
    "used_up" -> QuietNote.USED_UP
    else -> null
}

/**
 * Asks [backend] off the main thread. A failure of the call itself (not of the model) falls back to
 * the local rules without a line, so the person's text is never lost.
 */
suspend fun runAsk(backend: AssistantBackend, text: String, place: AssistantPlace): TurnOutcome =
    try {
        outcomeOf(withContext(Dispatchers.IO) { backend.ask(text, place) })
    } catch (e: CancellationException) {
        throw e
    } catch (_: Exception) {
        TurnOutcome.Fallback(null)
    }

/** One turn: the box keeps its text and thinks, then the answer is a card; the outcome says what to do next. */
suspend fun AssistantState.runTurn(
    backend: AssistantBackend,
    res: Resources,
    words: Words,
    text: String,
    place: AssistantPlace,
): TurnOutcome {
    card = null
    thinking = true
    val outcome = try {
        runAsk(backend, text, place)
    } finally {
        thinking = false
    }
    if (outcome is TurnOutcome.Answer) card = answerCard(res, words, outcome.reply)
    return outcome
}

/** Where a request typed on the page [route] of [tab] is: the Today page, a project, a task or a calendar day. */
fun assistantPlace(
    route: String,
    tab: Tab,
    calendarDay: LocalDate,
    categoryId: Long,
    zone: ZoneId,
    now: Instant = Instant.now(),
): AssistantPlace {
    val id = route.substringAfter(':').substringBefore(':').toLongOrNull() ?: 0
    val base = AssistantPlace(zone = zone.id, offset = zone.rules.getOffset(now).totalSeconds)
    return when {
        route.startsWith("project:") -> base.copy(kind = "project", projectId = id)
        route.startsWith("task:") -> base.copy(kind = "task", taskId = id)
        tab == Tab.CALENDAR -> base.copy(kind = "calendar", day = calendarDay.toString())
        tab == Tab.MINE -> base.copy(kind = "today", categoryId = categoryId)
        else -> base.copy(kind = "inbox")
    }
}

// ---- Wording ---------------------------------------------------------------------------------

/** The card of an answer: the summary line, a row per change, or the plain text or the question. */
fun answerCard(res: Resources, words: Words, reply: AssistantReply): AnswerCard {
    val shown = reply.receipts.filter { it.detail.kind !in NoteKinds }
    return AnswerCard(
        summary = receiptSummary(res, reply.receipts),
        rows = shown.map { AnswerRow(it.taskId, it.title, receiptChips(res, words, it.detail), it.status) },
        text = reply.text,
        question = reply.question?.takeIf { it.text.isNotBlank() },
        undo = reply.receipts.takeIf { it.isNotEmpty() }?.let { CardUndo.Receipts(it.map { r -> r.id }.reversed()) },
        newTaskId = reply.receipts.firstOrNull { it.detail.kind in AddedKinds }?.taskId ?: 0,
    )
}

/** The card of what the local rules added, as the model's would show it. */
fun localCard(res: Resources, words: Words, parse: QuickParse?, titles: List<String>, ids: List<Long>, note: QuietNote?): AnswerCard {
    val rows = ids.mapIndexed { i, id ->
        val draft = parse?.drafts?.getOrNull(i)
        AnswerRow(
            taskId = id,
            title = titles.getOrElse(i) { "" },
            chips = draft?.let { quickChips(res, words, it) }.orEmpty(),
            status = "todo",
        )
    }
    return AnswerCard(
        summary = res.getQuantityString(R.plurals.assistant_added, ids.size, ids.size),
        rows = rows,
        undo = CardUndo.Tasks(ids),
        quiet = note,
        newTaskId = ids.firstOrNull() ?: 0,
    )
}

private val AddedKinds = setOf("added", "reminder", "due")
private val NoteKinds = setOf("remembered", "forgot")

/** `Added 1, Updated 2`: the count of each kind of change. */
fun receiptSummary(res: Resources, receipts: List<AssistantReceipt>): String {
    val added = receipts.count { it.detail.kind in AddedKinds }
    val updated = receipts.count { it.detail.kind == "updated" }
    val done = receipts.count { it.detail.kind == "done" }
    val parts = buildList {
        if (added > 0) add(res.getQuantityString(R.plurals.assistant_added, added, added))
        if (updated > 0) add(res.getQuantityString(R.plurals.assistant_updated, updated, updated))
        if (done > 0) add(res.getQuantityString(R.plurals.assistant_finished, done, done))
        if (receipts.any { it.detail.kind == "remembered" }) add(res.getString(R.string.assistant_noted))
        if (receipts.any { it.detail.kind == "forgot" }) add(res.getString(R.string.assistant_forgot))
    }
    return parts.joinToString(res.getString(R.string.assistant_join))
}

/** What a row of the card says after the title: the time read, the deadline, the project, what an update changed. */
fun receiptChips(res: Resources, words: Words, d: AssistantDetail): List<String> = buildList {
    when (d.kind) {
        "reminder" -> add(quickWhen(words, res, d.at, dayOnly = false))
        "due" -> add(res.getString(R.string.quick_due, quickWhen(words, res, d.at, dayOnly = true)))
        "done" -> add(res.getString(R.string.assistant_change_done))
        "updated" -> d.changes.forEach { add(changeChip(res, words, it.kind, it.at, it.value)) }
    }
    if (d.due.isNotBlank()) add(res.getString(R.string.quick_due, quickWhen(words, res, d.due, dayOnly = true)))
    if (d.priority.isNotBlank()) add(d.priority)
    if (d.category.isNotBlank()) add(categoryWord(res, d.category))
    if (d.project.isNotBlank() && d.kind in AddedKinds) add(d.project)
}

private fun categoryWord(res: Resources, key: String): String = when (key) {
    "work" -> res.getString(R.string.category_work)
    "life" -> res.getString(R.string.category_life)
    else -> key
}

private fun changeChip(res: Resources, words: Words, kind: String, at: String, value: String): String = when (kind) {
    "reminder" -> if (at.isBlank()) {
        res.getString(R.string.assistant_change_reminder_cleared)
    } else {
        res.getString(R.string.quick_reminder, quickWhen(words, res, at, dayOnly = false))
    }
    "due" -> if (at.isBlank()) {
        res.getString(R.string.assistant_change_due_cleared)
    } else {
        res.getString(R.string.quick_due, quickWhen(words, res, at, dayOnly = true))
    }
    "status" -> if (value == "done") {
        res.getString(R.string.assistant_change_done)
    } else {
        res.getString(R.string.assistant_change_status, words.menuStatus(value))
    }
    "project" -> if (value.isBlank()) {
        res.getString(R.string.assistant_change_moved_inbox)
    } else {
        res.getString(R.string.assistant_change_moved, value)
    }
    "category" -> if (value.isBlank()) {
        res.getString(R.string.assistant_change_category_cleared)
    } else {
        res.getString(R.string.assistant_change_category, categoryWord(res, value))
    }
    "renamed" -> res.getString(R.string.assistant_change_renamed)
    "notes" -> res.getString(R.string.assistant_change_notes)
    else -> value
}

/** The chips of a task the local rules made: the time read, the category, the star. */
fun quickChips(res: Resources, words: Words, draft: QuickDraft): List<String> = buildList {
    if (draft.remindAt.isNotBlank()) add(quickWhen(words, res, draft.remindAt, dayOnly = false))
    if (draft.dueAt.isNotBlank()) add(res.getString(R.string.quick_due, quickWhen(words, res, draft.dueAt, dayOnly = true)))
    if (draft.category.isNotBlank()) add(draft.category)
    if (draft.important) add(res.getString(R.string.quick_important))
}
