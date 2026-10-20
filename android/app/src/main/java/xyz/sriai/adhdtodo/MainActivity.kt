package xyz.sriai.adhdtodo

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import xyz.sriai.adhdtodo.core.Core
import xyz.sriai.adhdtodo.core.Core_
import xyz.sriai.adhdtodo.core.Listener
import xyz.sriai.adhdtodo.platform.KeystoreSecrets
import xyz.sriai.adhdtodo.platform.ReminderReceiver
import xyz.sriai.adhdtodo.platform.Reminders
import xyz.sriai.adhdtodo.ui.App
import xyz.sriai.adhdtodo.ui.theme.AdhdTheme
import xyz.sriai.adhdtodo.ui.theme.AppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens

/**
 * Opens the Go core in filesDir and owns its listener. A debug build started with the intent extra
 * demo=true seeds app/data's demo workspace first, through mobile/core's SeedDemo. Debug extras
 * route and sync_state let the reviewer open a screen and fake a Sync state for screenshots.
 *
 * The background sync runs while the Activity is started; reminders get one inexact alarm at the
 * next remind_at, set here and by ReminderReceiver after each claim.
 */
class MainActivity : ComponentActivity(), Listener {
    private var core: Core_? = null
    private val revision = mutableIntStateOf(0)
    private val status = mutableStateOf("")
    private val initialRoute = mutableStateOf<String?>(null)
    private val demoSyncState = mutableStateOf<String?>(null)

    private val notificationPermission =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        Reminders.ensureChannel(this)
        initialRoute.value = routeFrom(intent)
        demoSyncState.value = if (BuildConfig.DEBUG) intent.getStringExtra("sync_state") else null
        val demo = BuildConfig.DEBUG &&
            (intent.getBooleanExtra("demo", false) || intent.getStringExtra("demo") == "true")
        lifecycleScope.launch {
            val opened = withContext(Dispatchers.IO) {
                try {
                    if (demo) Core.seedDemo(filesDir.absolutePath)
                    Core.open(filesDir.absolutePath, KeystoreSecrets(this@MainActivity))
                } catch (_: Exception) {
                    null
                }
            }
            if (opened == null) {
                setContent { AdhdTheme { CoreError() } }
                return@launch
            }
            core = opened
            opened.setListener(this@MainActivity)
            status.value = runCatching { opened.syncStatus() }.getOrDefault("")
            setContent {
                AdhdTheme {
                    App(
                        core = opened,
                        revision = revision.intValue,
                        syncStatusJson = status.value,
                        initialRoute = initialRoute.value,
                        demoSyncState = demoSyncState.value,
                        onInitialRouteConsumed = { initialRoute.value = null },
                    )
                }
            }
            withContext(Dispatchers.IO) { runCatching { opened.startSync() } }
            scheduleReminders()
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        routeFrom(intent)?.let { initialRoute.value = it }
        if (BuildConfig.DEBUG) demoSyncState.value = intent.getStringExtra("sync_state")
    }

    /** The route a launch asked for: a task notification's task_id, else a debug route extra. */
    private fun routeFrom(intent: Intent): String? {
        val taskID = intent.getLongExtra(Reminders.EXTRA_TASK_ID, 0)
        if (taskID > 0) return "task:$taskID:0"
        if (!BuildConfig.DEBUG) return null
        return intent.getStringExtra("route")
    }

    override fun onStart() {
        super.onStart()
        val opened = core ?: return
        lifecycleScope.launch(Dispatchers.IO) { runCatching { opened.startSync() } }
    }

    override fun onStop() {
        val opened = core
        if (opened != null) {
            lifecycleScope.launch(Dispatchers.IO) { runCatching { opened.stopSync() } }
        }
        super.onStop()
    }

    override fun onResume() {
        super.onResume()
        scheduleReminders()
    }

    /** Called on a Go goroutine: move to the UI thread before touching Compose state. */
    override fun onChange() {
        runOnUiThread { revision.intValue++ }
        scheduleReminders()
    }

    override fun onStatus(statusJSON: String?) {
        runOnUiThread { status.value = statusJSON.orEmpty() }
    }

    /** Sets the one alarm and asks for notification permission the first time an alarm exists. */
    private fun scheduleReminders() {
        val opened = core ?: return
        lifecycleScope.launch {
            val at = withContext(Dispatchers.IO) {
                runCatching {
                    Reminders.schedule(this@MainActivity, opened)
                    opened.nextReminderAt()
                }.getOrDefault(0)
            }
            if (at > 0) askNotifications()
        }
    }

    private fun askNotifications() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return
        if (checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED) return
        val prefs = getSharedPreferences("reminders", MODE_PRIVATE)
        if (prefs.getBoolean("asked_notifications", false)) return
        prefs.edit().putBoolean("asked_notifications", true).apply()
        notificationPermission.launch(Manifest.permission.POST_NOTIFICATIONS)
    }

    override fun onDestroy() {
        core?.setListener(null)
        core?.close()
        core = null
        super.onDestroy()
    }
}

@androidx.compose.runtime.Composable
private fun CoreError() {
    val tokens = LocalTokens.current
    Box(
        modifier = Modifier.fillMaxSize().padding(24.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            stringResource(R.string.error_core),
            style = AppType.body,
            color = tokens.textSecondary,
        )
    }
}
