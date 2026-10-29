package xyz.sriai.adhdtodo.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * The pure half of the phone's own updater (mygo does not cover Android): which release is newer,
 * which of its files is the APK and its checksum, and when to ask. The network, the download and the
 * system installer are platform/AppUpdater.
 */

/** Where the latest release is asked, from the public repository that holds the installers. */
const val LATEST_RELEASE_URL = "https://api.github.com/repos/QingYunA/withdo/releases/latest"

/** At most one check a day on start or return to the app. */
const val UPDATE_CHECK_INTERVAL_MILLIS = 24L * 60 * 60 * 1000

/** x.y.z, compared number by number. */
data class Semver(val major: Int, val minor: Int, val patch: Int) : Comparable<Semver> {
    override fun compareTo(other: Semver): Int =
        compareValuesBy(this, other, Semver::major, Semver::minor, Semver::patch)

    override fun toString() = "$major.$minor.$patch"
}

/** `v0.9.0` or `0.9.0`; anything else (a pre-release such as `0.9.0-rc1`, text) is not a version. */
fun parseSemver(raw: String): Semver? {
    val parts = raw.trim().removePrefix("v").split(".")
    if (parts.size != 3) return null
    val numbers = parts.map { p -> if (p.isNotEmpty() && p.all { it in '0'..'9' }) p.toIntOrNull() else null }
    if (numbers.any { it == null }) return null
    return Semver(numbers[0]!!, numbers[1]!!, numbers[2]!!)
}

/** A newer release with an APK: where to get the file and the line that checks it. */
data class AvailableUpdate(val version: String, val apkUrl: String, val sumsUrl: String)

@Serializable
private data class ReleaseAsset(val name: String = "", @SerialName("browser_download_url") val url: String = "")

@Serializable
private data class Release(
    @SerialName("tag_name") val tag: String = "",
    val draft: Boolean = false,
    val prerelease: Boolean = false,
    val assets: List<ReleaseAsset> = emptyList(),
)

private val releaseJson = Json { ignoreUnknownKeys = true; isLenient = true }

/**
 * The update the GitHub "latest release" answer offers over [current], or null: the tag is not newer,
 * the release has no `withdo-<version>.apk` or no `SHA256SUMS.txt` (the desktop files may be
 * uploaded first), or the answer is not a release.
 */
fun pickUpdate(releaseBody: String, current: String): AvailableUpdate? {
    val release = runCatching { releaseJson.decodeFromString<Release>(releaseBody) }.getOrNull() ?: return null
    if (release.draft || release.prerelease) return null
    val latest = parseSemver(release.tag) ?: return null
    val now = parseSemver(current) ?: return null
    if (latest <= now) return null
    val apk = release.assets.firstOrNull { it.name == "withdo-$latest.apk" } ?: return null
    val sums = release.assets.firstOrNull { it.name == "SHA256SUMS.txt" } ?: return null
    if (!apk.url.startsWith("https://") || !sums.url.startsWith("https://")) return null
    return AvailableUpdate(latest.toString(), apk.url, sums.url)
}

/** The hex SHA-256 that a `shasum -a 256` file lists for [name] (`<hex>  name`, or `<hex> *name`), lower case. */
fun sha256For(sums: String, name: String): String? {
    for (line in sums.lineSequence()) {
        val trimmed = line.trim()
        val space = trimmed.indexOf(' ')
        if (space != 64) continue
        val file = trimmed.substring(space).trim().removePrefix("*")
        val hex = trimmed.substring(0, space)
        if (file == name && hex.all { it in '0'..'9' || it in 'a'..'f' || it in 'A'..'F' }) return hex.lowercase()
    }
    return null
}

/** True when the last check was a day or more ago, or the clock went back past it. */
fun updateCheckDue(lastCheckMillis: Long, nowMillis: Long): Boolean =
    lastCheckMillis <= 0 || nowMillis < lastCheckMillis || nowMillis - lastCheckMillis >= UPDATE_CHECK_INTERVAL_MILLIS

enum class UpdatePhase {
    /** Not checked yet in this run. */
    IDLE,
    CHECKING,
    UP_TO_DATE,
    /** A manual check could not reach GitHub; an automatic one says nothing. */
    CHECK_FAILED,
    AVAILABLE,
    DOWNLOADING,
    /** Downloaded and checked; the phone must first allow installs from this app. */
    NEEDS_PERMISSION,
    /** The download or its checksum failed; [UpdateUi.version] is still the offer. */
    FAILED,
}

/** What the banner and the Sync row draw. */
data class UpdateUi(
    val phase: UpdatePhase = UpdatePhase.IDLE,
    /** This app's version. */
    val current: String = "",
    /** The newer version on offer, in every phase from AVAILABLE on. */
    val version: String = "",
    /** 0 to 100 while DOWNLOADING. */
    val percent: Int = 0,
    /** The user closed the banner for [version]; the Sync row still offers it. */
    val dismissed: Boolean = false,
) {
    val offered: Boolean
        get() = phase == UpdatePhase.AVAILABLE || phase == UpdatePhase.DOWNLOADING ||
            phase == UpdatePhase.NEEDS_PERMISSION || phase == UpdatePhase.FAILED

    /** The quiet banner at the top of the main screen: an offer the user has not put away, or work in progress. */
    val bannerShown: Boolean
        get() = offered && (!dismissed || phase == UpdatePhase.DOWNLOADING || phase == UpdatePhase.NEEDS_PERMISSION)
}
