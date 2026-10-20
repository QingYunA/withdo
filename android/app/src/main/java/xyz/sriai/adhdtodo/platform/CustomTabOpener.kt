package xyz.sriai.adhdtodo.platform

import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.browser.customtabs.CustomTabsIntent
import xyz.sriai.adhdtodo.core.BrowserOpener

/**
 * Opens the OIDC sign-in page in a Custom Tab, so the page stays in the browser's process and the
 * app does not handle the account password. Without a browser that supports Custom Tabs it falls
 * back to a plain ACTION_VIEW, still outside the app.
 */
class CustomTabOpener(private val context: Context) : BrowserOpener {
    override fun open(url: String) {
        val tabs = CustomTabsIntent.Builder().build()
        tabs.intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        try {
            tabs.launchUrl(context, Uri.parse(url))
        } catch (_: ActivityNotFoundException) {
            val view = Intent(Intent.ACTION_VIEW, Uri.parse(url))
            view.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            context.startActivity(view)
        }
    }
}
