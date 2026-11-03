package xyz.sriai.adhdtodo.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/*
 * The assistant's JSON (mobile/core/assistant.go): what ModelState says, the place a request is
 * typed in, and the reply to it. The core sends data, the app words it.
 */

/** The phases of ModelState: ready, signed_out, used_up or hidden. */
object ModelPhase {
    const val READY = "ready"
    const val SIGNED_OUT = "signed_out"
    const val USED_UP = "used_up"
    const val HIDDEN = "hidden"
}

/**
 * ModelState(): which model would answer the composer. [source] is `official` when Ask may be
 * called, else `none` (the local rules read the text). [left] is today's turns left, -1 until the
 * server has said; [limit] the day's turns.
 */
@Serializable
data class ModelState(
    @SerialName("source") val source: String = "none",
    @SerialName("phase") val phase: String = ModelPhase.HIDDEN,
    @SerialName("left") val left: Int = -1,
    @SerialName("limit") val limit: Int = 0,
) {
    /** Ask may be called: the official model is signed in, offered and has turns left. */
    val official: Boolean get() = source == "official" && phase == ModelPhase.READY

    /** The model is there to sign in for or to use up: the quiet line under the card applies. */
    val offered: Boolean get() = phase == ModelPhase.SIGNED_OUT || phase == ModelPhase.USED_UP
}

/**
 * Where a request was typed: [kind] is today, inbox, project, task or calendar. [zone] and
 * [offset] (seconds) are the phone's zone, which the core's Go cannot always read itself.
 */
@Serializable
data class AssistantPlace(
    @SerialName("kind") val kind: String = "today",
    @SerialName("project_id") val projectId: Long = 0,
    @SerialName("task_id") val taskId: Long = 0,
    @SerialName("day") val day: String = "",
    @SerialName("category_id") val categoryId: Long = 0,
    @SerialName("zone") val zone: String = "",
    @SerialName("offset") val offset: Int = 0,
)

/** One field an update changed: [kind] is reminder, due, status, priority, project, category, renamed or notes. */
@Serializable
data class AssistantChange(
    @SerialName("kind") val kind: String = "",
    @SerialName("at") val at: String = "",
    @SerialName("value") val value: String = "",
)

/** What a receipt says: [kind] is added, reminder, due, updated, done, remembered or forgot. */
@Serializable
data class AssistantDetail(
    @SerialName("kind") val kind: String = "added",
    @SerialName("at") val at: String = "",
    @SerialName("due") val due: String = "",
    @SerialName("project") val project: String = "",
    @SerialName("priority") val priority: String = "",
    @SerialName("category") val category: String = "",
    @SerialName("auto_filed") val autoFiled: Boolean = false,
    @SerialName("changes") val changes: List<AssistantChange> = emptyList(),
)

/** One change the assistant made; [id] is what Undo takes. */
@Serializable
data class AssistantReceipt(
    @SerialName("id") val id: Long = 0,
    @SerialName("task_id") val taskId: Long = 0,
    @SerialName("title") val title: String = "",
    @SerialName("status") val status: String = "todo",
    @SerialName("bell") val bell: Boolean = false,
    @SerialName("detail") val detail: AssistantDetail = AssistantDetail(),
)

@Serializable
data class AssistantQuestion(
    @SerialName("text") val text: String = "",
    @SerialName("options") val options: List<String> = emptyList(),
)

/**
 * Ask()'s answer: a plain answer in [text], the changes as [receipts], or a [question]. [problem]
 * says why the model did not answer (quota, signed_out, off, cancelled, busy, timeout, unreachable,
 * rate_limited, other); receipts made before it stopped are still listed.
 */
@Serializable
data class AssistantReply(
    @SerialName("text") val text: String = "",
    @SerialName("receipts") val receipts: List<AssistantReceipt> = emptyList(),
    @SerialName("question") val question: AssistantQuestion? = null,
    @SerialName("problem") val problem: String = "",
    @SerialName("error") val error: String = "",
)

/** The core's assistant calls the screens use; [CoreApi] implements them, the tests fake them. Each blocks. */
interface AssistantBackend {
    fun modelState(): ModelState

    /** Gives the official model [text] typed in [place]. */
    fun ask(text: String, place: AssistantPlace): AssistantReply

    /** Reverts the change a receipt of [ask] describes. */
    fun undo(receiptId: Long)

    /** Cancels the [ask] in flight. */
    fun stop()
}
