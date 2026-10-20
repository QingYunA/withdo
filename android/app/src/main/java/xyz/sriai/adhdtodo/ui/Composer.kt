package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens

/**
 * The one input, pinned above the bottom bar. Enter creates a plain task without a model; the
 * quiet line above the box says it went to the Inbox and goes away after a moment.
 */
@Composable
fun Composer(
    value: String,
    placeholder: String,
    notice: String?,
    sending: Boolean,
    onValueChange: (String) -> Unit,
    onSend: () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 10.dp),
    ) {
        if (notice != null) {
            Text(
                notice,
                style = type.meta,
                color = tokens.textSecondary,
                modifier = Modifier.padding(start = 4.dp, bottom = 6.dp),
            )
        }
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .height(Dims.composerHeight)
                .background(tokens.bgSubtle, RoundedCornerShape(12.dp))
                .padding(horizontal = 14.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Avatar("assistant", size = Dims.avatar)
            BasicTextField(
                value = value,
                onValueChange = onValueChange,
                textStyle = type.input.copy(color = tokens.text),
                cursorBrush = SolidColor(tokens.accent),
                singleLine = true,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Send),
                keyboardActions = KeyboardActions(
                    onSend = { if (value.isNotBlank() && !sending) onSend() },
                    onDone = { if (value.isNotBlank() && !sending) onSend() },
                ),
                modifier = Modifier.weight(1f),
                decorationBox = { inner ->
                    Box(contentAlignment = Alignment.CenterStart) {
                        if (value.isEmpty()) {
                            Text(
                                placeholder,
                                style = type.input,
                                color = tokens.textTertiary,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        inner()
                    }
                },
            )
        }
    }
}
