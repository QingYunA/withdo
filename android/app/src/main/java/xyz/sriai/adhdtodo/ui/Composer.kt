package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.MicGlyph
import xyz.sriai.adhdtodo.ui.theme.PixelIcon

/**
 * The one input (v35), docked above the bottom bar: the page's only floating surface. A 24 dp
 * assistant avatar and the 17 sp text, then a row of 40 dp controls (image, microphone, send).
 * Enter creates a plain task without a model. What it answers (the receipt) sits above the box in
 * 13 sp slate 11, with Undo when there is something to take back.
 */
@Composable
fun Composer(
    value: String,
    placeholder: String,
    notice: String?,
    sending: Boolean,
    onValueChange: (String) -> Unit,
    onSend: () -> Unit,
    onUndo: (() -> Unit)? = null,
    onImage: () -> Unit = {},
    onVoice: () -> Unit = {},
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val shape = RoundedCornerShape(20.dp)
    val hasText = value.isNotBlank()
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 10.dp),
    ) {
        if (notice != null) {
            Row(
                modifier = Modifier.padding(start = 12.dp, bottom = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Text(notice, style = type.meta, color = tokens.textSecondary)
                if (onUndo != null) {
                    Text(
                        stringResource(R.string.menu_undo),
                        style = type.meta,
                        color = tokens.accentInk,
                        modifier = Modifier.clip(RoundedCornerShape(6.dp)).clickable(onClick = onUndo).padding(4.dp),
                    )
                }
            }
        }
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .shadow(
                    elevation = 12.dp,
                    shape = shape,
                    ambientColor = Color.Black.copy(alpha = 0.10f),
                    spotColor = Color.Black.copy(alpha = 0.14f),
                )
                .background(tokens.bg, shape)
                .border(Dims.hairline, tokens.borderControl, shape),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth().padding(start = 14.dp, end = 14.dp, top = 14.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Avatar("assistant", size = 24.dp)
                BasicTextField(
                    value = value,
                    onValueChange = onValueChange,
                    textStyle = type.input.copy(color = tokens.text),
                    cursorBrush = SolidColor(tokens.accent),
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Send),
                    keyboardActions = KeyboardActions(
                        onSend = { if (hasText && !sending) onSend() },
                        onDone = { if (hasText && !sending) onSend() },
                    ),
                    modifier = Modifier.weight(1f),
                    decorationBox = { inner ->
                        Box(Modifier.heightIn(min = 25.dp), contentAlignment = Alignment.CenterStart) {
                            if (value.isEmpty()) {
                                Text(
                                    placeholder,
                                    style = type.input,
                                    color = tokens.textSecondary,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            }
                            inner()
                        }
                    },
                )
            }
            Row(
                modifier = Modifier.fillMaxWidth().padding(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                val image = stringResource(R.string.composer_image)
                val voice = stringResource(R.string.composer_voice)
                val send = stringResource(R.string.action_send)
                ControlButton(label = image, onClick = onImage) {
                    PixelIcon("image", size = 24.dp, tint = tokens.textSecondary)
                }
                ControlButton(label = voice, onClick = onVoice) {
                    MicGlyph(size = 24.dp, tint = tokens.textSecondary)
                }
                Spacer(Modifier.weight(1f))
                ControlButton(
                    label = send,
                    fill = if (hasText) tokens.accent else tokens.bgSubtle,
                    onClick = { if (hasText && !sending) onSend() },
                ) {
                    PixelIcon("arrow-up", size = 24.dp, tint = if (hasText) tokens.onAccent else tokens.textTertiary)
                }
            }
        }
    }
}

/** A 40 dp, radius 12 control of the composer's row. */
@Composable
private fun ControlButton(
    label: String,
    onClick: () -> Unit,
    fill: Color = Color.Transparent,
    content: @Composable () -> Unit,
) {
    Box(
        modifier = Modifier
            .size(40.dp)
            .clip(RoundedCornerShape(12.dp))
            .background(fill)
            .clickable(onClick = onClick)
            .semantics { contentDescription = label },
        contentAlignment = Alignment.Center,
    ) { content() }
}
