package xyz.sriai.adhdtodo.testing

import androidx.compose.runtime.Composable
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.test.SemanticsNodeInteraction
import androidx.compose.ui.test.down
import androidx.compose.ui.test.junit4.ComposeContentTestRule
import androidx.compose.ui.test.moveBy
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.up
import org.robolectric.RuntimeEnvironment
import xyz.sriai.adhdtodo.data.PageJson
import xyz.sriai.adhdtodo.ui.theme.AdhdTheme
import java.io.File
import java.time.Instant
import java.time.ZoneId

/**
 * The moment the screenshots and tests are taken at, near the golden files' times, and their zone:
 * ages and dates then read the same on every machine and every day.
 */
object FixedTime {
    val now: Instant = Instant.parse("2026-10-20T09:00:00Z")
    val zone: ZoneId = ZoneId.of("Asia/Shanghai")
    val clock: () -> Instant = { now }
}

/**
 * The real page JSON the Go core writes, decoded the way the app decodes it. The tests run in
 * android/app, so the path is found by walking up to the repository root.
 */
object Golden {
    val root: File by lazy {
        val start = File("").absoluteFile
        generateSequence(start) { it.parentFile }
            .firstOrNull { File(it, "mobile/core/testdata").isDirectory }
            ?: error("could not find mobile/core/testdata above ${start.absolutePath}")
    }

    fun file(path: String): File = File(root, path).also {
        check(it.exists()) { "missing $path under ${root.absolutePath}" }
    }

    inline fun <reified T> load(name: String): T =
        PageJson.decodeFromString(file("mobile/core/testdata/$name").readText())
}

/** Where a recorded screenshot lands before scripts/android-screens.sh copies it into docs/. */
fun screenshotFile(name: String): File =
    File(Golden.root, "android/app/build/screenshots/$name.png").also { it.parentFile?.mkdirs() }

/**
 * Robolectric's screen for the screenshots: 411 dp wide, a little taller than a Pixel so a whole
 * page fits above the composer, at xxhdpi. The night and zh qualifiers go through the same
 * configuration that AdhdTheme's isSystemInDarkTheme reads, so the dark pictures are the app's
 * own dark tokens, not a test-only palette.
 */
fun screenQualifiers(night: Boolean = false, zh: Boolean = false): String = buildString {
    if (zh) append("zh-rCN-")
    append("w411dp-h1000dp")
    if (night) append("-night")
    append("-xxhdpi")
}

/** Applies the qualifiers, then composes under the app's own theme. */
fun ComposeContentTestRule.setThemedContent(
    night: Boolean = false,
    zh: Boolean = false,
    content: @Composable () -> Unit,
) {
    RuntimeEnvironment.setQualifiers(screenQualifiers(night, zh))
    setContent { AdhdTheme(content = content) }
}

/**
 * Drags [node] vertically by [dy] physical pixels with a real down / move / up gesture, for the
 * draggable-based reorder tests to come.
 */
fun dragBy(node: SemanticsNodeInteraction, dy: Float) {
    node.performTouchInput {
        down(center)
        moveBy(Offset(0f, dy))
        up()
    }
}
