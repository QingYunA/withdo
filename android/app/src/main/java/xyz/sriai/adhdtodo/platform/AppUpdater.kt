package xyz.sriai.adhdtodo.platform

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.provider.Settings
import androidx.compose.runtime.mutableStateOf
import androidx.core.content.FileProvider
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import xyz.sriai.adhdtodo.data.AvailableUpdate
import xyz.sriai.adhdtodo.data.LATEST_RELEASE_URL
import xyz.sriai.adhdtodo.data.UpdatePhase
import xyz.sriai.adhdtodo.data.UpdateUi
import xyz.sriai.adhdtodo.data.parseSemver
import xyz.sriai.adhdtodo.data.pickUpdate
import xyz.sriai.adhdtodo.data.sha256For
import xyz.sriai.adhdtodo.data.updateCheckDue
import java.io.File
import java.net.HttpURLConnection
import java.net.URL
import java.security.MessageDigest

/**
 * The phone's own updater: once a day it asks the public releases repository for the latest release;
 * a newer one with an APK becomes [ui]. Update downloads the APK into the app's cache, checks its
 * SHA-256 against the release's SHA256SUMS.txt and hands it to the system installer, which asks the
 * user as it does for any app. The pure decisions are in data/UpdateRules.kt.
 *
 * The offer is remembered in preferences so a start inside the day still shows it. Closing the
 * banner remembers the version it was closed for.
 */
class AppUpdater(
    private val context: Context,
    private val scope: CoroutineScope,
    private val current: String,
) {
    val ui = mutableStateOf(UpdateUi(current = current))

    private val prefs = context.getSharedPreferences("updates", Context.MODE_PRIVATE)
    private var offer: AvailableUpdate? = null
    private var downloaded: File? = null
    private var busy = false

    init {
        val saved = savedOffer()
        if (saved != null) showOffer(saved)
    }

    /** On start and on return to the app: a check if a day has passed since the last one. */
    fun checkIfDue() {
        if (busy || !updateCheckDue(prefs.getLong(LAST_CHECK, 0), System.currentTimeMillis())) return
        check(manual = false)
    }

    /** The Sync screen's Check now. */
    fun checkNow() {
        if (busy) return
        check(manual = true)
    }

    /** Hides the banner until a newer version than the one on offer. */
    fun dismiss() {
        val version = offer?.version ?: return
        prefs.edit().putString(DISMISSED, version).apply()
        ui.value = ui.value.copy(dismissed = true)
    }

    /** The Update button: download, check, install. */
    fun install() {
        val update = offer ?: return
        if (busy) return
        busy = true
        ui.value = ui.value.copy(phase = UpdatePhase.DOWNLOADING, percent = 0)
        scope.launch(Dispatchers.IO) {
            val file = runCatching { download(update) }.getOrNull()
            withContext(Dispatchers.Main) {
                busy = false
                if (file == null) {
                    ui.value = ui.value.copy(phase = UpdatePhase.FAILED)
                } else {
                    downloaded = file
                    launchInstaller(file)
                }
            }
        }
    }

    /** Back from the "allow installs from this source" screen: carry on if it was allowed. */
    fun resumeAfterSettings() {
        val file = downloaded ?: return
        if (ui.value.phase == UpdatePhase.NEEDS_PERMISSION && context.packageManager.canRequestPackageInstalls()) {
            launchInstaller(file)
        }
    }

    private fun check(manual: Boolean) {
        busy = true
        if (manual) ui.value = ui.value.copy(phase = UpdatePhase.CHECKING)
        scope.launch(Dispatchers.IO) {
            // A failed answer (offline, GitHub's rate limit) counts as a check: no retry until tomorrow.
            val body = runCatching { get(LATEST_RELEASE_URL) }.getOrNull()
            val found = body?.let { pickUpdate(it, current) }
            withContext(Dispatchers.Main) {
                busy = false
                prefs.edit().putLong(LAST_CHECK, System.currentTimeMillis()).apply()
                when {
                    found != null -> {
                        saveOffer(found)
                        showOffer(found)
                    }
                    body != null -> {
                        clearOffer()
                        ui.value = UpdateUi(phase = if (manual) UpdatePhase.UP_TO_DATE else UpdatePhase.IDLE, current = current)
                    }
                    manual -> ui.value = ui.value.copy(phase = UpdatePhase.CHECK_FAILED)
                    else -> Unit
                }
            }
        }
    }

    private fun showOffer(update: AvailableUpdate) {
        offer = update
        ui.value = UpdateUi(
            phase = UpdatePhase.AVAILABLE,
            current = current,
            version = update.version,
            dismissed = prefs.getString(DISMISSED, null) == update.version,
        )
    }

    private fun savedOffer(): AvailableUpdate? {
        val version = prefs.getString(OFFER_VERSION, null) ?: return null
        val apk = prefs.getString(OFFER_APK, null) ?: return null
        val sums = prefs.getString(OFFER_SUMS, null) ?: return null
        // The app may have been updated since: an offer that is not newer is gone.
        val newer = parseSemver(version)?.let { v -> parseSemver(current)?.let { v > it } } ?: false
        return if (newer) AvailableUpdate(version, apk, sums) else null.also { clearOffer() }
    }

    private fun saveOffer(update: AvailableUpdate) {
        prefs.edit()
            .putString(OFFER_VERSION, update.version)
            .putString(OFFER_APK, update.apkUrl)
            .putString(OFFER_SUMS, update.sumsUrl)
            .apply()
    }

    private fun clearOffer() {
        offer = null
        prefs.edit().remove(OFFER_VERSION).remove(OFFER_APK).remove(OFFER_SUMS).apply()
    }

    /** The APK in the cache, after its size and SHA-256 matched the release's SHA256SUMS.txt; throws otherwise. */
    private fun download(update: AvailableUpdate): File {
        val name = "withdo-${update.version}.apk"
        val expected = sha256For(get(update.sumsUrl), name) ?: error("no checksum for $name")
        val dir = File(context.cacheDir, "updates").also { it.mkdirs() }
        dir.listFiles()?.forEach { it.delete() }
        val file = File(dir, name)
        val digest = MessageDigest.getInstance("SHA-256")
        val connection = open(update.apkUrl, readTimeoutMillis = 30_000)
        try {
            val total = connection.contentLengthLong
            var done = 0L
            connection.inputStream.use { input ->
                file.outputStream().use { output ->
                    val buffer = ByteArray(64 * 1024)
                    while (true) {
                        val n = input.read(buffer)
                        if (n < 0) break
                        output.write(buffer, 0, n)
                        digest.update(buffer, 0, n)
                        done += n
                        if (total > 0) {
                            val percent = (done * 100 / total).toInt().coerceIn(0, 100)
                            if (percent != ui.value.percent) ui.value = ui.value.copy(percent = percent)
                        }
                    }
                }
            }
        } finally {
            connection.disconnect()
        }
        val actual = digest.digest().joinToString("") { "%02x".format(it) }
        if (actual != expected) {
            file.delete()
            error("checksum mismatch for $name")
        }
        return file
    }

    /** The system installer, or first the settings screen that allows this app to install. */
    private fun launchInstaller(file: File) {
        if (!context.packageManager.canRequestPackageInstalls()) {
            ui.value = ui.value.copy(phase = UpdatePhase.NEEDS_PERMISSION)
            context.startActivity(
                Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES, Uri.parse("package:${context.packageName}"))
                    .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
            )
            return
        }
        val uri = FileProvider.getUriForFile(context, "${context.packageName}.updates", file)
        val intent = Intent(Intent.ACTION_VIEW)
            .setDataAndType(uri, "application/vnd.android.package-archive")
            .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION or Intent.FLAG_ACTIVITY_NEW_TASK)
        val started = runCatching { context.startActivity(intent) }.isSuccess
        ui.value = ui.value.copy(phase = if (started) UpdatePhase.AVAILABLE else UpdatePhase.FAILED, percent = 0)
    }

    private fun get(url: String): String {
        val connection = open(url, readTimeoutMillis = 15_000)
        try {
            return connection.inputStream.bufferedReader().use { it.readText() }
        } finally {
            connection.disconnect()
        }
    }

    /** GET without credentials; a non-200 answer (GitHub's 403 rate limit, 404) throws. */
    private fun open(url: String, readTimeoutMillis: Int): HttpURLConnection {
        val connection = URL(url).openConnection() as HttpURLConnection
        connection.connectTimeout = 15_000
        connection.readTimeout = readTimeoutMillis
        connection.setRequestProperty("User-Agent", "adhd-todo-android/$current")
        if (url.startsWith("https://api.github.com/")) connection.setRequestProperty("Accept", "application/vnd.github+json")
        if (connection.responseCode != HttpURLConnection.HTTP_OK) {
            connection.disconnect()
            error("GET $url answered ${connection.responseCode}")
        }
        return connection
    }

    private companion object {
        const val LAST_CHECK = "last_check"
        const val DISMISSED = "dismissed_version"
        const val OFFER_VERSION = "offer_version"
        const val OFFER_APK = "offer_apk"
        const val OFFER_SUMS = "offer_sums"
    }
}
