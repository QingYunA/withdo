package xyz.sriai.adhdtodo

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.material3.Text

// A stand-in until A2's screens: it proves the Go core loads and answers.
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val mine = runCatching {
            val core = xyz.sriai.adhdtodo.core.Core.open(filesDir.absolutePath, null)
            core.mine()
        }.getOrElse { "core failed: ${it.message}" }
        setContent { Text(mine.take(200)) }
    }
}
