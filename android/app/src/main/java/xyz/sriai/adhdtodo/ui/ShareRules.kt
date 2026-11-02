package xyz.sriai.adhdtodo.ui

import java.net.URI

/*
 * Share into Withdo (v43 P2): what another app shared becomes a draft task. The rules are kept
 * apart from the sheet so they are tested without Android.
 */

/** What the sheet starts from: the title for the field and the link for the chip (null without one). */
data class ShareDraft(val title: String, val url: String?)

// A link ends at a space or at Chinese punctuation, which shares often run straight into.
private val WebLink = Regex("""https?://[^\s<>"，。、；：！？）】]+""", RegexOption.IGNORE_CASE)

/**
 * The draft of a share: the first web link in [text] is the link; the title is the page's title
 * ([subject], as browsers send it), else the text without the link, else the link itself. Null when
 * nothing usable was shared.
 */
fun shareDraft(text: String?, subject: String?): ShareDraft? {
    val body = text.orEmpty().trim()
    val found = WebLink.find(body)?.value
    val url = found?.trimEnd('.', ',', ')', ']', '。', '，')
    // Punctuation the link was followed by goes with it.
    val rest = (if (found != null) body.replace(found, " ") else body)
        .replace(Regex("\\s+[，。、；：,;:]+"), " ").replace(Regex("\\s+"), " ").trim()
        .takeIf { it.any(Char::isLetterOrDigit) }.orEmpty()
    val title = subject?.trim().orEmpty().ifEmpty { rest }.ifEmpty { url.orEmpty() }
    if (title.isEmpty()) return null
    return ShareDraft(title = title, url = url)
}

/**
 * The chip's words for a link: the host without `www.`, then the last two parts of its path
 * (`github.com › issues/219`); the host alone when the path is empty.
 */
fun linkLabel(url: String): String {
    val uri = runCatching { URI(url) }.getOrNull() ?: return url
    val host = uri.host?.removePrefix("www.") ?: return url
    val parts = uri.path.orEmpty().split('/').filter { it.isNotEmpty() }
    return if (parts.isEmpty()) host else "$host › ${parts.takeLast(2).joinToString("/")}"
}

/** When the shared task is due: what the parser read from the title, no date, or a picked day. */
sealed interface ShareWhen {
    data object Read : ShareWhen
    data object NoDate : ShareWhen
    data class Day(val option: DueOption) : ShareWhen
}

/**
 * The When in force: the person's pick ([picked], null until they pick), else what the parser read
 * when it read something ([parserRead]), else no date. A pick of the reading falls back to no date
 * once the title no longer reads as a date.
 */
fun shareWhen(picked: ShareWhen?, parserRead: Boolean): ShareWhen = when {
    picked == ShareWhen.Read && !parserRead -> ShareWhen.NoDate
    picked != null -> picked
    parserRead -> ShareWhen.Read
    else -> ShareWhen.NoDate
}
