package xyz.sriai.adhdtodo.ui

import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.provider.Settings
import android.text.format.DateFormat
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TimePicker
import androidx.compose.material3.rememberTimePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.platform.EndOfDay
import xyz.sriai.adhdtodo.platform.Reminders
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import java.time.LocalTime
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

/**
 * Settings > General > Notifications: a switch for the reminders that come due, and the end-of-day
 * reminder (weekdays, the week's words on Friday) with its time. Off, nothing notifies.
 */
@Composable
fun NotificationSettingsScreen(
    reminders: Boolean,
    endOfDay: Boolean,
    minutes: Int,
    systemAllows: Boolean,
    onReminders: (Boolean) -> Unit,
    onEndOfDay: (Boolean) -> Unit,
    onMinutes: (Int) -> Unit,
    onOpenSystemSettings: () -> Unit,
) {
    val tokens = LocalTokens.current
    var picking by remember { mutableStateOf(false) }
    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = Dims.pagePadding)
            .padding(bottom = 24.dp),
    ) {
        PageTitle(stringResource(R.string.settings_notifications))
        GroupCard {
            SettingsRow(
                title = stringResource(R.string.settings_reminders),
                subtitle = stringResource(if (systemAllows) R.string.settings_reminders_line else R.string.settings_notify_denied),
                subtitleColor = if (systemAllows) androidx.compose.ui.graphics.Color.Unspecified else tokens.stale,
                trailing = { QuietSwitch(reminders, onReminders) },
            )
            if (!systemAllows) {
                RowRule()
                Row(Modifier.fillMaxWidth().padding(horizontal = 12.dp), horizontalArrangement = Arrangement.Start) {
                    TextLink(stringResource(R.string.settings_open_system), onOpenSystemSettings)
                }
            }
        }
        GroupLabel(stringResource(R.string.settings_end_of_day))
        GroupCard {
            SettingsRow(
                title = stringResource(R.string.settings_end_of_day_switch),
                subtitle = stringResource(R.string.settings_end_of_day_line),
                trailing = { QuietSwitch(endOfDay && reminders, { onEndOfDay(it) }, enabled = reminders) },
            )
            if (endOfDay && reminders) {
                RowRule()
                SettingsRow(
                    title = stringResource(R.string.settings_end_of_day_time),
                    value = clockLabel(minutes),
                    chevron = true,
                    onClick = { picking = true },
                )
            }
        }
    }
    if (picking) {
        TimeDialog(minutes, onPick = { onMinutes(it); picking = false }, onDismiss = { picking = false })
    }
}

/** The screen as App draws it: the saved choices, and the schedule follows each change. */
@Composable
fun NotificationSettingsRoute(onChanged: () -> Unit) {
    val context = LocalContext.current
    var reminders by remember { mutableStateOf(Reminders.enabled(context)) }
    var endOfDay by remember { mutableStateOf(EndOfDay.switchOn(context)) }
    var minutes by remember { mutableIntStateOf(EndOfDay.minutes(context)) }
    NotificationSettingsScreen(
        reminders = reminders,
        endOfDay = endOfDay,
        minutes = minutes,
        systemAllows = systemAllows(context),
        onReminders = {
            Reminders.setEnabled(context, it)
            reminders = it
            EndOfDay.schedule(context)
            onChanged()
        },
        onEndOfDay = { EndOfDay.setEnabled(context, it); endOfDay = it },
        onMinutes = { EndOfDay.setMinutes(context, it); minutes = it },
        onOpenSystemSettings = {
            runCatching {
                context.startActivity(
                    Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS)
                        .putExtra(Settings.EXTRA_APP_PACKAGE, context.packageName)
                        .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
                )
            }
        },
    )
}

private fun systemAllows(context: Context): Boolean =
    context.getSystemService(NotificationManager::class.java)?.areNotificationsEnabled() ?: true

/** 18:30 as the phone writes a time: `6:30 PM` or `18:30`. */
@Composable
private fun clockLabel(minutes: Int): String {
    val context = LocalContext.current
    val locale = LocalConfiguration.current.locales[0]
    val time = LocalTime.of(minutes / 60, minutes % 60)
    val pattern = DateFormat.getBestDateTimePattern(locale, if (DateFormat.is24HourFormat(context)) "Hm" else "hm")
    return DateTimeFormatter.ofPattern(pattern, locale).format(time).takeIf { it.isNotEmpty() }
        ?: DateTimeFormatter.ofLocalizedTime(FormatStyle.SHORT).format(time)
}

/** The accent-on, quiet-off switch of the app's tokens. */
@Composable
fun QuietSwitch(checked: Boolean, onChange: (Boolean) -> Unit, enabled: Boolean = true) {
    val tokens = LocalTokens.current
    Switch(
        checked = checked,
        onCheckedChange = onChange,
        enabled = enabled,
        colors = SwitchDefaults.colors(
            checkedThumbColor = tokens.onAccent,
            checkedTrackColor = tokens.accent,
            checkedBorderColor = tokens.accent,
            uncheckedThumbColor = tokens.bg,
            uncheckedTrackColor = tokens.borderControl,
            uncheckedBorderColor = tokens.borderControl,
        ),
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun TimeDialog(minutes: Int, onPick: (Int) -> Unit, onDismiss: () -> Unit) {
    val tokens = LocalTokens.current
    val state = rememberTimePickerState(
        initialHour = minutes / 60,
        initialMinute = minutes % 60,
        is24Hour = DateFormat.is24HourFormat(LocalContext.current),
    )
    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = tokens.bg,
        shape = RoundedCornerShape(28.dp),
        title = { Text(stringResource(R.string.settings_end_of_day_time), color = tokens.text) },
        text = { TimePicker(state = state) },
        confirmButton = {
            TextButton(onClick = { onPick(state.hour * 60 + state.minute) }) {
                Text(stringResource(R.string.action_done), color = tokens.accentInk)
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text(stringResource(R.string.action_cancel), color = tokens.textSecondary) }
        },
    )
}
