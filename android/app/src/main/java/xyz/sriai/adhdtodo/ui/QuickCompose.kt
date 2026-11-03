package xyz.sriai.adhdtodo.ui

import android.content.res.Resources
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.input.OffsetMapping
import androidx.compose.ui.text.input.TransformedText
import androidx.compose.ui.text.input.VisualTransformation
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.CoreApi
import xyz.sriai.adhdtodo.data.QuickDraft
import xyz.sriai.adhdtodo.data.QuickParse
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.ui.theme.Tokens
import java.time.OffsetDateTime

/*
 * The composer reads dates without a model (v43). The core's ParseQuick is the desktop's rule
 * parser; while the person types, the words it read take an iris 4 wash and a line above the box
 * says what Send will store. Tapping that line stores the text as typed.
 */

/** What the line above the box says: the accent [phrase] and, after a dot, [where] it goes. */
data class QuickPreview(val phrase: String, val where: String)

/** Iris 4: the wash behind the words the parser read. */
val Tokens.quickWash: Color get() = if (dark) Color(0xFF262A65) else Color(0xFFE6E7FF)

/** A drafts' words are shown only when the parser read something out of the text. */
fun QuickParse.readAny(): Boolean = drafts.any { it.read }

/**
 * A time as the line says it: `Tomorrow 9:00 AM` / `明天 上午 9:00`; a time today carries Today in
 * front of the clock. A due time at the end of the working day (18:00) reads as the day alone.
 */
fun quickWhen(words: Words, res: Resources, value: String, dayOnly: Boolean): String {
    if (dayOnly && isDefaultDueTime(value)) return words.due(Task(dueAt = value))
    val today = words.due(Task(remindAt = value)).isNotEmpty()
    val at = words.reminderValue(value)
    return if (today) "${res.getString(R.string.due_today)} $at" else at
}

private fun isDefaultDueTime(value: String): Boolean =
    runCatching { OffsetDateTime.parse(value).toLocalTime() == DefaultDueTime }.getOrDefault(false)

/** The accent half of the line for one draft: `Tomorrow 9:00 AM reminder, Work, important`. */
fun quickPhrase(res: Resources, words: Words, draft: QuickDraft): String {
    val parts = buildList {
        if (draft.remindAt.isNotBlank()) {
            add(res.getString(R.string.quick_reminder, quickWhen(words, res, draft.remindAt, dayOnly = false)))
        }
        if (draft.dueAt.isNotBlank()) {
            add(res.getString(R.string.quick_due, quickWhen(words, res, draft.dueAt, dayOnly = true)))
        }
        if (draft.category.isNotBlank()) add(draft.category)
        if (draft.important) add(res.getString(R.string.quick_important))
    }
    return parts.joinToString(res.getString(R.string.quick_join))
}

/** The whole accent half: one draft in words, several as a count. */
fun quickSummary(res: Resources, words: Words, parse: QuickParse): String {
    val drafts = parse.drafts
    return if (drafts.size > 1) {
        res.getQuantityString(R.plurals.quick_many, drafts.size, drafts.size)
    } else {
        drafts.firstOrNull()?.let { quickPhrase(res, words, it) }.orEmpty()
    }
}

/** The line above the box, or null when nothing was read. [where] is the project or the Inbox. */
fun quickPreview(res: Resources, words: Words, parse: QuickParse?, where: String): QuickPreview? {
    if (parse == null || !parse.readAny()) return null
    return QuickPreview(quickSummary(res, words, parse), where)
}

/** The receipt after Send: `Added · Tomorrow 9:00 AM reminder`, or the count for several tasks. */
fun quickReceipt(res: Resources, words: Words, parse: QuickParse): String =
    if (parse.drafts.size > 1) {
        res.getQuantityString(R.plurals.quick_added_many, parse.drafts.size, parse.drafts.size)
    } else {
        res.getString(R.string.quick_added, quickSummary(res, words, parse))
    }

/** The read words as ranges of the text, in order, clamped to [length] and with empty ones left out. */
fun quickHighlights(parse: QuickParse?, length: Int): List<IntRange> =
    parse?.drafts.orEmpty()
        .flatMap { it.ranges }
        .mapNotNull { r ->
            val start = r.getOrNull(0) ?: return@mapNotNull null
            val end = (r.getOrNull(1) ?: return@mapNotNull null).coerceAtMost(length)
            if (start in 0 until end) start until end else null
        }
        .sortedBy { it.first }

/** Draws the [ranges] of the text with [style] and changes nothing else. */
class HighlightTransformation(
    private val ranges: List<IntRange>,
    private val style: SpanStyle,
) : VisualTransformation {
    override fun filter(text: AnnotatedString): TransformedText {
        val built = buildAnnotatedString {
            append(text)
            for (r in ranges) if (r.last < text.length) addStyle(style, r.first, r.last + 1)
        }
        return TransformedText(built, OffsetMapping.Identity)
    }
}

/**
 * The parse of [text], read about 120 ms after the last key. The answer is kept only while it is
 * for the text on screen, so a late answer never marks the wrong words.
 */
@Composable
fun rememberQuickParse(api: CoreApi, text: String, enabled: Boolean): QuickParse? {
    var result by remember { mutableStateOf<Pair<String, QuickParse>?>(null) }
    LaunchedEffect(text, enabled) {
        if (!enabled || text.isBlank()) {
            result = null
            return@LaunchedEffect
        }
        delay(120)
        val parse = try {
            withContext(Dispatchers.IO) { api.parseQuick(text) }
        } catch (e: CancellationException) {
            throw e
        } catch (_: Exception) {
            null
        }
        result = parse?.let { text to it }
    }
    return result?.takeIf { it.first == text }?.second
}

/**
 * Stores every draft of [parse] as a task of the user's, in [projectId] or the Inbox, and returns their ids.
 * A draft with no category read from its text gets [categoryId] (the chip narrowing Today), if any.
 */
fun createQuickDrafts(api: CoreApi, parse: QuickParse, projectId: Long?, categoryId: Long? = null): List<Long> =
    parse.drafts.map { draft ->
        api.createTask(
            title = draft.title,
            projectId = projectId,
            remindAt = draft.remindAt.ifBlank { null },
            dueAt = draft.dueAt.ifBlank { null },
            categoryId = draft.categoryId.takeIf { it > 0 } ?: categoryId,
            priority = if (draft.important) "P1" else null,
        )
    }
