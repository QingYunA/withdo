package xyz.sriai.adhdtodo

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
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
import xyz.sriai.adhdtodo.ui.App
import xyz.sriai.adhdtodo.ui.theme.AdhdTheme
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.AppType

/**
 * Opens the Go core in filesDir and owns its listener. A debug build started with the intent extra
 * demo=true seeds app/data's demo workspace first, through mobile/core's SeedDemo.
 */
class MainActivity : ComponentActivity(), Listener {
    private var core: Core_? = null
    private val revision = mutableIntStateOf(0)
    private val status = mutableStateOf("")

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        val demo = BuildConfig.DEBUG &&
            (intent.getBooleanExtra("demo", false) || intent.getStringExtra("demo") == "true")
        lifecycleScope.launch {
            val opened = withContext(Dispatchers.IO) {
                try {
                    if (demo) Core.seedDemo(filesDir.absolutePath)
                    Core.open(filesDir.absolutePath, null)
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
                    App(core = opened, revision = revision.intValue, syncStatus = status.value)
                }
            }
        }
    }

    /** Called on a Go goroutine: move to the UI thread before touching Compose state. */
    override fun onChange() {
        runOnUiThread { revision.intValue++ }
    }

    override fun onStatus(statusJSON: String?) {
        runOnUiThread { status.value = statusJSON.orEmpty() }
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
