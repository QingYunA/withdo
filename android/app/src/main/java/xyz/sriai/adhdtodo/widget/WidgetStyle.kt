package xyz.sriai.adhdtodo.widget

import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.glance.color.ColorProvider
import xyz.sriai.adhdtodo.MainActivity
import xyz.sriai.adhdtodo.platform.Reminders
import xyz.sriai.adhdtodo.ui.theme.DarkTokens
import xyz.sriai.adhdtodo.ui.theme.LightTokens
import xyz.sriai.adhdtodo.ui.theme.Tokens

/** Sizes in dp, shared by the three widgets and by the count of rows a height has room for. */
object WidgetDims {
    /** The card's top and bottom padding together. */
    const val cardChrome = 20f
    const val header = 32f
    const val row = 36f
    const val avatar = 18f
    const val well = 28f
}

/**
 * The app's tokens as Glance colors: the system's light/dark picks one (on Android 12 and later the
 * home screen switches without waiting for an update).
 */
object WidgetColors {
    val text = pick { it.text }
    val textSecondary = pick { it.textSecondary }
    val accentInk = pick { it.accentInk }

    fun status(status: String) = ColorProvider(
        day = LightTokens.status(status),
        night = DarkTokens.status(status),
    )

    private fun pick(of: (Tokens) -> androidx.compose.ui.graphics.Color) =
        ColorProvider(day = of(LightTokens), night = of(DarkTokens))
}

/** The Intents the widgets fire; MainActivity's routeFrom reads them. */
object WidgetIntents {
    /** Today with the composer focused: the Today widget's + and Note something. */
    fun compose(context: Context): Intent = launch(context)
        .putExtra(MainActivity.EXTRA_COMPOSE, true)
        .setData(Uri.parse("withdo-widget://compose"))

    /** The task's page. */
    fun task(context: Context, id: Long): Intent = launch(context)
        .putExtra(Reminders.EXTRA_TASK_ID, id)
        .setData(Uri.parse("withdo-widget://task/$id"))

    /** The app as it was. */
    fun open(context: Context): Intent = launch(context).setData(Uri.parse("withdo-widget://open"))

    // MainActivity is singleTop: CLEAR_TOP hands the intent to the running one (onNewIntent). The
    // data URI is only there so each widget action gets its own PendingIntent; extras do not count.
    private fun launch(context: Context): Intent = Intent(context, MainActivity::class.java)
        .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP)
}
