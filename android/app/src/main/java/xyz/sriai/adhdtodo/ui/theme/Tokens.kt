package xyz.sriai.adhdtodo.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * The tokens of docs/design/ui-spec.md, light and dark. Screens take every color and type style
 * from here; no screen holds a literal color or size.
 */
@Immutable
data class Tokens(
    val dark: Boolean,
    val bg: Color,
    /** The bottom bar (slate 2). */
    val bgSidebar: Color,
    /** Wells: the idle send button, the segmented control (slate 3). */
    val bgSubtle: Color,
    /** The Agents tray (slate 2). */
    val bgTray: Color,
    /** The neutral selection: the bar's pill, a pressed row, a highlighted sheet row (slate 4). */
    val bgSelected: Color,
    /** The hover tint a chosen row of the task menu's pages takes (slate 3). */
    val bgHover: Color,
    /** The 32 % black behind a modal sheet. */
    val scrim: Color,
    /** Hairlines, always 0.5 dp (slate 6). */
    val border: Color,
    /** The composer's edge, controls (slate 7). */
    val borderControl: Color,
    val text: Color,
    val textSecondary: Color,
    /** Icons and glyphs only, never words. */
    val textTertiary: Color,
    /** iris 9: the send button, the caret, a ring. */
    val accent: Color,
    val onAccent: Color,
    /** iris 3: tinted buttons, a just-added row. */
    val accentSoft: Color,
    /** iris 11: links, tinted button text, Undo. */
    val accentInk: Color,
    val statusBacklog: Color,
    val statusTodo: Color,
    val statusDoing: Color,
    val statusBlocked: Color,
    val statusReview: Color,
    val statusDone: Color,
    val stale: Color,
    /** The eight folder colors of app/theme/icons.go, in the same order (step 9). */
    val projectColors: List<Color>,
    /** Step 11 of the same eight scales, for a category's name as a word. */
    val projectInks: List<Color>,
) {
    /**
     * A category's color comes from the eight project colors, in domain.CategoryColors' order, as
     * the desktop's CategoryColor does; a key from a newer build falls back to the first.
     */
    fun categoryColor(key: String): Color {
        val index = CategoryColorKeys.indexOf(key)
        return projectColors[if (index >= 0) index else 0]
    }

    /** A category's name as a word: step 11 of its scale (app/theme CategoryInk). */
    fun categoryInk(key: String): Color {
        val index = CategoryColorKeys.indexOf(key)
        return projectInks[if (index >= 0) index else 0]
    }

    /** The fixed colors of a category icon's palette letters a to l, after the project colors. */
    val categoryPalette: List<Color>
        get() = projectColors + listOf(
            Color(0xFFFFFFFF), Color(0xFF8E8E93), Color(0xFF1C1C1E), Color(0xFF8A5A3B),
        )

    fun status(status: String): Color = when (status) {
        "backlog" -> statusBacklog
        "todo" -> statusTodo
        "doing" -> statusDoing
        "blocked" -> statusBlocked
        "review" -> statusReview
        "done" -> statusDone
        "cancelled" -> statusBacklog
        else -> statusTodo
    }

    /** A status tint for a pill: the spec's 12 % of the status color over the page. */
    fun statusTint(status: String): Color = status(status).copy(alpha = 0.12f)

    /** The color of the priority badge's ring: blocked at 35 % for P0, the hairline for P1. */
    fun priorityRing(priority: String): Color =
        if (priority == "P0") statusBlocked.copy(alpha = 0.35f) else border

    fun priorityInk(priority: String): Color = if (priority == "P0") statusBlocked else textSecondary

    /**
     * The folder color of a project: app/theme/icons.go's FNV-1a hash of the name into the eight
     * colors. It follows the name, as on the desktop; the user does not choose it.
     */
    fun projectColor(name: String): Color {
        var hash = -2128831035 // 2166136261 as a signed 32-bit int
        for (byte in name.toByteArray(Charsets.UTF_8)) {
            hash = (hash xor (byte.toInt() and 0xff)) * 16777619
        }
        val index = ((hash.toLong() and 0xffffffffL) % projectColors.size).toInt()
        return projectColors[index]
    }
}

/** internal/domain.CategoryColors, the keys of the eight project colors in the same order. */
private val CategoryColorKeys = listOf(
    "indigo", "yellow", "green", "red", "blue", "pink", "teal", "orange",
)

/** app/theme/icons.go's ProjectColors, the same eight in both appearances. */
private val ProjectPalette = listOf(
    Color(0xFF5B5BD6), Color(0xFFFFC53D), Color(0xFF30A46C), Color(0xFFE5484D),
    Color(0xFF0090FF), Color(0xFFD6409F), Color(0xFF12A594), Color(0xFFF76B15),
)

private val InkLight = listOf(
    Color(0xFF5753C6), Color(0xFFAB6400), Color(0xFF218358), Color(0xFFCE2C31),
    Color(0xFF0D74CE), Color(0xFFC2298A), Color(0xFF008573), Color(0xFFCC4E00),
)
private val InkDark = listOf(
    Color(0xFFB1A9FF), Color(0xFFFFCA16), Color(0xFF3DD68C), Color(0xFFFF9592),
    Color(0xFF70B8FF), Color(0xFFFF8DCC), Color(0xFF0BD8B6), Color(0xFFFFA057),
)

/** The values of docs/design/ui-spec.md's Color table: Radix slate and iris, light. */
val LightTokens = Tokens(
    dark = false,
    bg = Color(0xFFFFFFFF),
    bgSidebar = Color(0xFFF9F9FB),
    bgSubtle = Color(0xFFF0F0F3),
    bgTray = Color(0xFFF9F9FB),
    bgSelected = Color(0xFFE8E8EC),
    bgHover = Color(0xFFF0F0F3),
    scrim = Color(0xFF000000).copy(alpha = 0.32f),
    border = Color(0xFFD9D9E0),
    borderControl = Color(0xFFCDCED6),
    text = Color(0xFF1C2024),
    textSecondary = Color(0xFF60646C),
    textTertiary = Color(0xFF8B8D98),
    accent = Color(0xFF5B5BD6),
    onAccent = Color(0xFFFFFFFF),
    accentSoft = Color(0xFFF0F1FE),
    accentInk = Color(0xFF5753C6),
    statusBacklog = Color(0xFFB9BBC6),
    statusTodo = Color(0xFF8B8D98),
    statusDoing = Color(0xFF0090FF),
    statusBlocked = Color(0xFFE5484D),
    statusReview = Color(0xFFFFC53D),
    statusDone = Color(0xFF30A46C),
    stale = Color(0xFFCC4E00),
    projectColors = ProjectPalette,
    projectInks = InkLight,
)

/** The same table in the dark scales. */
val DarkTokens = Tokens(
    dark = true,
    bg = Color(0xFF111113),
    bgSidebar = Color(0xFF18191B),
    bgSubtle = Color(0xFF212225),
    bgTray = Color(0xFF18191B),
    bgSelected = Color(0xFF272A2D),
    bgHover = Color(0xFF212225),
    scrim = Color(0xFF000000).copy(alpha = 0.32f),
    border = Color(0xFF363A3F),
    borderControl = Color(0xFF43484E),
    text = Color(0xFFEDEEF0),
    textSecondary = Color(0xFFB0B4BA),
    textTertiary = Color(0xFF696E77),
    accent = Color(0xFF5B5BD6),
    onAccent = Color(0xFFFFFFFF),
    accentSoft = Color(0xFF202248),
    accentInk = Color(0xFFB1A9FF),
    statusBacklog = Color(0xFF5A6169),
    statusTodo = Color(0xFF696E77),
    statusDoing = Color(0xFF0090FF),
    statusBlocked = Color(0xFFE5484D),
    statusReview = Color(0xFFFFC53D),
    statusDone = Color(0xFF30A46C),
    stale = Color(0xFFFF801F),
    projectColors = ProjectPalette,
    projectInks = InkDark,
)

@Immutable
data class Type(
    val title: TextStyle,
    val taskTitle: TextStyle,
    val input: TextStyle,
    val body: TextStyle,
    val bodyStrong: TextStyle,
    val meta: TextStyle,
    val section: TextStyle,
    val mono: TextStyle,
    val badge: TextStyle,
    val nav: TextStyle,
    val navSelected: TextStyle,
)

private val system = FontFamily.Default

val AppType = Type(
    title = TextStyle(fontFamily = system, fontSize = 24.sp, lineHeight = 32.sp, fontWeight = FontWeight.SemiBold),
    taskTitle = TextStyle(fontFamily = system, fontSize = 22.sp, lineHeight = 28.sp, fontWeight = FontWeight.Bold),
    input = TextStyle(fontFamily = system, fontSize = 17.sp, lineHeight = 25.sp),
    body = TextStyle(fontFamily = system, fontSize = 15.sp, lineHeight = 22.sp),
    bodyStrong = TextStyle(fontFamily = system, fontSize = 15.sp, lineHeight = 22.sp, fontWeight = FontWeight.Medium),
    meta = TextStyle(fontFamily = system, fontSize = 13.sp, lineHeight = 18.sp),
    section = TextStyle(fontFamily = system, fontSize = 13.sp, lineHeight = 18.sp, fontWeight = FontWeight.SemiBold),
    mono = TextStyle(fontFamily = FontFamily.Monospace, fontSize = 12.5.sp, lineHeight = 18.sp),
    badge = TextStyle(fontFamily = system, fontSize = 11.5.sp, lineHeight = 16.sp, fontWeight = FontWeight.SemiBold),
    nav = TextStyle(fontFamily = system, fontSize = 12.sp, lineHeight = 16.sp),
    navSelected = TextStyle(fontFamily = system, fontSize = 12.sp, lineHeight = 16.sp, fontWeight = FontWeight.SemiBold),
)

object Dims {
    val pagePadding = 20.dp
    val rowHeight = 48.dp

    /** Every border, divider and hairline (Craft rule 6). */
    val hairline = 0.5.dp
    val navHeight = 64.dp
    val composerHeight = 52.dp
    val glyph = 18.dp
    val icon = 18.dp
    val avatar = 18.dp
    val avatarSmall = 12.dp
    val radiusRow = 9.dp
    val radiusPanel = 12.dp
    val radiusControl = 6.dp
}

val LocalTokens = staticCompositionLocalOf { LightTokens }
val LocalAppType = staticCompositionLocalOf { AppType }

@Composable
fun AdhdTheme(content: @Composable () -> Unit) {
    val dark = isSystemInDarkTheme()
    val tokens = if (dark) DarkTokens else LightTokens
    val scheme = if (dark) {
        darkColorScheme(
            primary = tokens.accent,
            background = tokens.bg,
            surface = tokens.bg,
            onBackground = tokens.text,
            onSurface = tokens.text,
            onSurfaceVariant = tokens.textSecondary,
            outline = tokens.border,
            error = tokens.statusBlocked,
        )
    } else {
        lightColorScheme(
            primary = tokens.accent,
            background = tokens.bg,
            surface = tokens.bg,
            onBackground = tokens.text,
            onSurface = tokens.text,
            onSurfaceVariant = tokens.textSecondary,
            outline = tokens.border,
            error = tokens.statusBlocked,
        )
    }
    CompositionLocalProvider(LocalTokens provides tokens, LocalAppType provides AppType) {
        MaterialTheme(colorScheme = scheme, content = content)
    }
}
