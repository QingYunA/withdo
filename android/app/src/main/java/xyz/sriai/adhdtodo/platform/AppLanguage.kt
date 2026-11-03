package xyz.sriai.adhdtodo.platform

import android.app.Activity
import android.content.Context
import android.content.res.Configuration
import android.os.LocaleList
import java.util.Locale

/**
 * The app's own language (Settings > General > Language): the system's, English or Simplified
 * Chinese. The choice is kept in SharedPreferences and applied by MainActivity.attachBaseContext, so
 * every resource lookup, Words and the code mails' language follow it; changing it recreates the
 * Activity. It is one mechanism on every API level (the Activity is not an AppCompatActivity).
 */
object AppLanguage {
    const val SYSTEM = ""
    const val ENGLISH = "en"
    const val CHINESE = "zh-Hans"

    private const val PREFS = "settings"
    private const val KEY = "language"

    /** The saved choice: [SYSTEM], [ENGLISH] or [CHINESE]. */
    fun saved(context: Context): String =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getString(KEY, SYSTEM).orEmpty()

    /** [base] with the saved language applied, or [base] itself when the system's is chosen. */
    fun wrap(base: Context): Context {
        val locale = localeOf(saved(base)) ?: return base
        Locale.setDefault(locale)
        val config = Configuration(base.resources.configuration)
        config.setLocales(LocaleList(locale))
        return base.createConfigurationContext(config)
    }

    /** Keeps [tag] and rebuilds the Activity in it. */
    fun set(activity: Activity, tag: String) {
        activity.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit().putString(KEY, tag).apply()
        activity.recreate()
    }

    fun localeOf(tag: String): Locale? = when (tag) {
        ENGLISH -> Locale.ENGLISH
        CHINESE -> Locale.SIMPLIFIED_CHINESE
        else -> null
    }
}
