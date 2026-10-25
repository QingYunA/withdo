package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.UpdatePhase
import xyz.sriai.adhdtodo.data.UpdateUi
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens

/** The quiet offer at the top of the main screen: Version 0.9.0 is available, Update, and Later. */
@Composable
fun UpdateBanner(update: UpdateUi, onUpdate: () -> Unit, onDismiss: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Box(Modifier.padding(horizontal = Dims.pagePadding, vertical = 6.dp)) {
        GroupCard {
            Row(
                modifier = Modifier.fillMaxWidth().heightIn(min = 56.dp).padding(horizontal = 14.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Column(Modifier.weight(1f)) {
                    Text(
                        when (update.phase) {
                            UpdatePhase.DOWNLOADING -> stringResource(R.string.update_downloading, update.percent)
                            UpdatePhase.NEEDS_PERMISSION -> stringResource(R.string.update_allow_installs)
                            UpdatePhase.FAILED -> stringResource(R.string.update_failed)
                            else -> stringResource(R.string.update_available, update.version)
                        },
                        style = type.body,
                        color = if (update.phase == UpdatePhase.FAILED) tokens.statusBlocked else tokens.text,
                    )
                    if (update.phase == UpdatePhase.DOWNLOADING) {
                        PixelProgressBar(done = update.percent.toLong(), total = 100, modifier = Modifier.padding(top = 6.dp))
                    }
                }
                if (update.phase == UpdatePhase.AVAILABLE || update.phase == UpdatePhase.FAILED) {
                    TextLink(stringResource(R.string.update_later), onDismiss, color = tokens.textSecondary)
                    QuietButton(stringResource(R.string.update_action), onClick = onUpdate)
                }
            }
        }
    }
}

/** The Sync screen's row: this version, what the last check found, and Check now or Update. */
@Composable
fun UpdateRow(update: UpdateUi, onCheck: () -> Unit, onUpdate: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier.fillMaxWidth().heightIn(min = 56.dp).padding(horizontal = 14.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Column(Modifier.weight(1f)) {
            Text(stringResource(R.string.update_row_title), style = type.body, color = tokens.text)
            val failed = update.phase == UpdatePhase.FAILED || update.phase == UpdatePhase.CHECK_FAILED
            Text(
                when (update.phase) {
                    UpdatePhase.CHECKING -> stringResource(R.string.update_row_checking)
                    UpdatePhase.UP_TO_DATE -> stringResource(R.string.update_row_up_to_date, update.current)
                    UpdatePhase.CHECK_FAILED -> stringResource(R.string.update_row_check_failed)
                    UpdatePhase.AVAILABLE -> stringResource(R.string.update_available, update.version)
                    UpdatePhase.DOWNLOADING -> stringResource(R.string.update_downloading, update.percent)
                    UpdatePhase.NEEDS_PERMISSION -> stringResource(R.string.update_allow_installs)
                    UpdatePhase.FAILED -> stringResource(R.string.update_failed)
                    UpdatePhase.IDLE -> stringResource(R.string.update_row_version, update.current)
                },
                style = type.meta,
                color = if (failed) tokens.statusBlocked else tokens.textSecondary,
            )
            if (update.phase == UpdatePhase.DOWNLOADING) {
                PixelProgressBar(done = update.percent.toLong(), total = 100, modifier = Modifier.padding(top = 6.dp))
            }
        }
        when (update.phase) {
            UpdatePhase.AVAILABLE, UpdatePhase.FAILED -> QuietButton(stringResource(R.string.update_action), onClick = onUpdate)
            UpdatePhase.IDLE, UpdatePhase.UP_TO_DATE, UpdatePhase.CHECK_FAILED ->
                QuietButton(stringResource(R.string.update_row_check), onClick = onCheck)
            else -> Unit
        }
    }
}
