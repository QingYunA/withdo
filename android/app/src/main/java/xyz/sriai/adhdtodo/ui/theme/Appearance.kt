package xyz.sriai.adhdtodo.ui.theme

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/** Settings > General > Appearance: follow the system, or always light or dark. */
enum class AppearanceMode(val key: String) {
    System("system"),
    Light("light"),
    Dark("dark"),
    ;

    companion object {
        fun of(key: String?): AppearanceMode = entries.firstOrNull { it.key == key } ?: System
    }
}

/**
 * The saved appearance, read by AdhdTheme. MainActivity calls [init] before the first frame; a
 * change is a state write, so the whole app recomposes in the new tokens at once.
 */
object Appearance {
    private const val PREFS = "settings"
    private const val KEY = "appearance"

    var mode by mutableStateOf(AppearanceMode.System)
        private set

    fun init(context: Context) {
        mode = AppearanceMode.of(context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getString(KEY, null))
    }

    fun set(context: Context, next: AppearanceMode) {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit().putString(KEY, next.key).apply()
        mode = next
    }

    /** Whether the app draws dark, given what the system says. */
    fun dark(system: Boolean): Boolean = when (mode) {
        AppearanceMode.System -> system
        AppearanceMode.Light -> false
        AppearanceMode.Dark -> true
    }
}
