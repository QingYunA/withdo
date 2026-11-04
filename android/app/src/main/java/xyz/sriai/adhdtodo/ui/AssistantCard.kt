package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.input.pointer.PointerEventPass
import androidx.compose.ui.input.pointer.PointerEventType
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.LinkAnnotation
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextLinkStyles
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withLink
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.ModelPhase
import xyz.sriai.adhdtodo.data.ModelState
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import xyz.sriai.adhdtodo.ui.theme.StatusGlyph

/*
 * What the assistant draws around the box (v44): the card above it, the quiet line under the card,
 * `Thinking…`, and under the box the model chip with its sheet. AssistantState.kt holds the data.
 */

/** The card and its quiet line, as they stand above the box. */
@Composable
fun AssistantAbove(
    card: AnswerCard,
    onUndo: () -> Unit,
    onPick: (String) -> Unit,
    onSignIn: () -> Unit,
    onUpgrade: (() -> Unit)? = null,
) {
    Column(Modifier.padding(bottom = 8.dp)) {
        AnswerCardView(card, onUndo, onPick)
        card.quiet?.let { QuietLine(it, onSignIn, onUpgrade) }
    }
}

/** The card: `bg.sidebar`, radius 16, a hairline; the avatar and the summary with Undo, then the rows. */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun AnswerCardView(card: AnswerCard, onUndo: () -> Unit, onPick: (String) -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val shape = RoundedCornerShape(16.dp)
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .testTag("assistant-card")
            .clip(shape)
            .background(tokens.bgSidebar)
            .border(Dims.hairline, tokens.border, shape)
            .padding(horizontal = 14.dp, vertical = 10.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Avatar("assistant", size = 16.dp)
            Text(
                card.summary.ifEmpty { stringResource(R.string.assistant_name) },
                style = type.meta,
                color = tokens.textSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            if (card.undo != null) {
                Text(
                    stringResource(R.string.menu_undo),
                    style = type.meta,
                    color = tokens.accentInk,
                    modifier = Modifier
                        .testTag("assistant-undo")
                        .clip(RoundedCornerShape(6.dp))
                        .clickable(onClick = onUndo)
                        .padding(horizontal = 4.dp, vertical = 2.dp),
                )
            }
        }
        for (row in card.rows) AnswerRowView(row)
        if (card.text.isNotBlank()) {
            Text(card.text, style = type.body, color = tokens.text, modifier = Modifier.padding(top = 6.dp).testTag("assistant-text"))
        }
        card.question?.let { question ->
            Text(question.text, style = type.body, color = tokens.text, modifier = Modifier.padding(top = 6.dp).testTag("assistant-question"))
            FlowRow(
                modifier = Modifier.padding(top = 10.dp, bottom = 2.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                question.options.forEachIndexed { index, option -> OptionPill(option, tinted = index == 0) { onPick(option) } }
            }
        }
    }
}

/** One change of the card: the status glyph, the title and the short chips of what it read. */
@Composable
private fun AnswerRowView(row: AnswerRow) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier.fillMaxWidth().heightIn(min = 36.dp).testTag("assistant-row"),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        StatusGlyph(row.status)
        Text(
            row.title,
            style = type.body,
            color = tokens.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f, fill = false),
        )
        for (chip in row.chips) {
            Text(
                chip,
                style = type.meta.copy(fontSize = 12.sp),
                color = tokens.textSecondary,
                maxLines = 1,
                modifier = Modifier
                    .background(tokens.bgSelected, RoundedCornerShape(5.dp))
                    .padding(horizontal = 6.dp, vertical = 1.dp),
            )
        }
    }
}

/** An option of a question, a 36 dp pill: the first is tinted (the assistant's surest answer). */
@Composable
private fun OptionPill(label: String, tinted: Boolean, onClick: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val shape = RoundedCornerShape(18.dp)
    Box(
        modifier = Modifier
            .testTag("assistant-option")
            .heightIn(min = 36.dp)
            .clip(shape)
            .background(if (tinted) tokens.accentSoft else tokens.bg)
            .then(if (tinted) Modifier else Modifier.border(Dims.hairline, tokens.borderControl, shape))
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(label, style = type.body, color = if (tinted) tokens.accentInk else tokens.text, maxLines = 1)
    }
}

/**
 * The one quiet line under the card: what the local rules did not do. `Sign in` is the way to
 * Sync; with [onUpgrade] (a Free account on a server that sells Pro) the used-up line ends with
 * `Upgrade to Pro for 100 a day.`, which opens the upgrade sheet.
 */
@Composable
fun QuietLine(note: QuietNote, onSignIn: () -> Unit, onUpgrade: (() -> Unit)? = null) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val text = when (note) {
        QuietNote.USED_UP -> if (onUpgrade == null) {
            AnnotatedString(stringResource(R.string.assistant_quiet_used_up))
        } else {
            val before = stringResource(R.string.billing_quiet_used_up_a)
            val link = stringResource(R.string.billing_upgrade)
            val after = stringResource(R.string.billing_quiet_used_up_b)
            val style = TextLinkStyles(SpanStyle(color = tokens.accentInk))
            buildAnnotatedString {
                append(before)
                withLink(LinkAnnotation.Clickable("upgrade", style) { onUpgrade() }) { append(link) }
                append(after)
            }
        }
        QuietNote.UNREACHABLE -> AnnotatedString(stringResource(R.string.assistant_quiet_unreachable))
        QuietNote.SIGNED_OUT -> {
            val before = stringResource(R.string.assistant_quiet_signed_out_a)
            val link = stringResource(R.string.assistant_sign_in)
            val after = stringResource(R.string.assistant_quiet_signed_out_b)
            val style = TextLinkStyles(SpanStyle(color = tokens.accentInk))
            buildAnnotatedString {
                append(before)
                withLink(LinkAnnotation.Clickable("sign-in", style) { onSignIn() }) { append(link) }
                append(after)
            }
        }
    }
    Text(text, style = type.meta, color = tokens.textSecondary, modifier = Modifier.padding(start = 4.dp, end = 4.dp, top = 6.dp).testTag("assistant-quiet"))
}

/** `Thinking…` above the box: the pixel spinner and the words, while the model works. */
@Composable
fun ThinkingLine() {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier.padding(start = 12.dp, bottom = 8.dp).testTag("assistant-thinking"),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        PixelSpinner(size = 14.dp, color = tokens.accent)
        Text(stringResource(R.string.assistant_thinking), style = type.meta, color = tokens.textSecondary)
    }
}

/** The model chip under the box, `Withdo official ⌄` (24 dp, `bg.hover`); a tap opens its sheet. */
@Composable
fun ModelChip(onClick: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier
            .testTag("model-chip")
            .heightIn(min = 24.dp)
            .clip(RoundedCornerShape(8.dp))
            .background(tokens.bgHover)
            .clickable(onClick = onClick)
            .padding(horizontal = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(5.dp),
    ) {
        Avatar("assistant", size = 14.dp)
        Text(stringResource(R.string.assistant_chip), style = type.meta.copy(fontSize = 12.sp), color = tokens.textSecondary, maxLines = 1)
        PixelIcon("chevron-down", size = 10.dp, tint = tokens.textTertiary)
    }
}

/** The chip's sheet: which model answers and today's turns. There is nothing to switch to on the phone. */
@Composable
fun ModelSheet(model: ModelState, onDismiss: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    SheetFrame(onDismiss) {
        SheetTitle(stringResource(R.string.assistant_model_name), modelLine(model))
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Avatar("assistant", size = 28.dp)
            Text(stringResource(R.string.assistant_model_note), style = type.body, color = tokens.text)
        }
    }
}

/** `Available · 86 left today`, or the reason there is no model. */
@Composable
fun modelLine(model: ModelState): String = when (model.phase) {
    ModelPhase.READY -> {
        val left = if (model.left >= 0) stringResource(R.string.assistant_quota_left, model.left) else ""
        listOf(stringResource(R.string.assistant_state_available), left).filter { it.isNotEmpty() }.joinToString(" · ")
    }
    ModelPhase.USED_UP -> stringResource(R.string.assistant_state_used_up)
    else -> stringResource(R.string.assistant_state_signed_out)
}

/**
 * Calls [onTouch] on the first touch of the page while [enabled], without consuming it: the card
 * closes on a tap outside, and the tap still reaches what it was for.
 */
fun Modifier.dismissOnTouch(enabled: Boolean, onTouch: () -> Unit): Modifier =
    if (!enabled) {
        this
    } else {
        pointerInput(onTouch) {
            awaitPointerEventScope {
                while (true) {
                    val event = awaitPointerEvent(PointerEventPass.Initial)
                    if (event.type == PointerEventType.Press) onTouch()
                }
            }
        }
    }
