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
    val bgSubtle: Color,
    val bgSelected: Color,
    /** The hover tint a chosen row of the task menu's pages takes. */
    val bgHover: Color,
    /** The 32 % black behind a modal sheet. */
    val scrim: Color,
    val border: Color,
    val text: Color,
    val textSecondary: Color,
    val textTertiary: Color,
    val accent: Color,
    val onAccent: Color,
    val statusBacklog: Color,
    val statusTodo: Color,
    val statusDoing: Color,
    val statusBlocked: Color,
    val statusReview: Color,
    val statusDone: Color,
    val stale: Color,
    /** The eight folder colors of app/theme/icons.go, in the same order. */
    val projectColors: List<Color>,
) {
    /**
     * A category's color comes from the eight project colors, in domain.CategoryColors' order, as
     * the desktop's CategoryColor does; a key from a newer build falls back to the first.
     */
    fun categoryColor(key: String): Color {
        val index = CategoryColorKeys.indexOf(key)
        return projectColors[if (index >= 0) index else 0]
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
        "cancelled" -> textTertiary
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
    Color(0xFF5B5BD6), Color(0xFFE0A800), Color(0xFF30A46C), Color(0xFFE5484D),
    Color(0xFF3B82F6), Color(0xFFD9488B), Color(0xFF0EA5A4), Color(0xFFE08A1E),
)

val LightTokens = Tokens(
    dark = false,
    bg = Color(0xFFFFFFFF),
    bgSubtle = Color(0xFFF5F5F7),
    bgSelected = Color(0xFF5B5BD6).copy(alpha = 0.12f),
    bgHover = Color(0xFF000000).copy(alpha = 0.05f),
    scrim = Color(0xFF000000).copy(alpha = 0.32f),
    border = Color(0xFF000000).copy(alpha = 0.10f),
    text = Color(0xFF1D1D1F),
    textSecondary = Color(0xFF6E6E73),
    textTertiary = Color(0xFFAEAEB2),
    accent = Color(0xFF5B5BD6),
    onAccent = Color(0xFFFFFFFF),
    statusBacklog = Color(0xFFAEAEB2),
    statusTodo = Color(0xFF8E8E93),
    statusDoing = Color(0xFF3B82F6),
    statusBlocked = Color(0xFFE5484D),
    statusReview = Color(0xFFE0A800),
    statusDone = Color(0xFF30A46C),
    stale = Color(0xFFD97706),
    projectColors = ProjectPalette,
)

val DarkTokens = Tokens(
    dark = true,
    bg = Color(0xFF1C1C1E),
    bgSubtle = Color(0xFF232326),
    bgSelected = Color(0xFF7C7CF0).copy(alpha = 0.22f),
    bgHover = Color(0xFFFFFFFF).copy(alpha = 0.06f),
    scrim = Color(0xFF000000).copy(alpha = 0.32f),
    border = Color(0xFFFFFFFF).copy(alpha = 0.10f),
    text = Color(0xFFF5F5F7),
    textSecondary = Color(0xFFA1A1A6),
    textTertiary = Color(0xFF636366),
    accent = Color(0xFF7C7CF0),
    onAccent = Color(0xFFFFFFFF),
    statusBacklog = Color(0xFF636366),
    statusTodo = Color(0xFF8E8E93),
    statusDoing = Color(0xFF60A5FA),
    statusBlocked = Color(0xFFFF6369),
    statusReview = Color(0xFFF5C542),
    statusDone = Color(0xFF4CC38A),
    stale = Color(0xFFF59E0B),
    projectColors = ProjectPalette,
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
    title = TextStyle(fontFamily = system, fontSize = 26.sp, lineHeight = 32.sp, fontWeight = FontWeight.Bold),
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
