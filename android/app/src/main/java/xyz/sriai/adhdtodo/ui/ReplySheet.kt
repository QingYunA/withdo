package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens

/**
 * Answer files a reply through Core.Reply. A phone cannot open the agent's app, so the reply goes
 * into the task's activity, where the agent reads it with its next report.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReplySheet(agent: String, onReply: (String) -> Unit, onDismiss: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    var text by remember { mutableStateOf("") }
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = tokens.bg,
        contentColor = tokens.text,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = 20.dp, end = 20.dp, bottom = 24.dp)
                .imePadding(),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(stringResource(R.string.reply_title, agent), style = type.title, color = tokens.text)
            Text(stringResource(R.string.reply_hint), style = type.meta, color = tokens.textSecondary)
            BasicTextField(
                value = text,
                onValueChange = { text = it },
                textStyle = type.body.copy(color = tokens.text),
                cursorBrush = SolidColor(tokens.accent),
                modifier = Modifier
                    .fillMaxWidth()
                    .height(120.dp)
                    .background(tokens.bgSubtle, RoundedCornerShape(12.dp))
                    .padding(12.dp),
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.End,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                TextButton(onClick = onDismiss) {
                    Text(stringResource(R.string.action_cancel), color = tokens.textSecondary)
                }
                TextButton(
                    onClick = { if (text.isNotBlank()) onReply(text.trim()) },
                    enabled = text.isNotBlank(),
                ) {
                    Text(
                        stringResource(R.string.action_send),
                        color = if (text.isNotBlank()) tokens.accent else tokens.textTertiary,
                    )
                }
            }
        }
    }
}
