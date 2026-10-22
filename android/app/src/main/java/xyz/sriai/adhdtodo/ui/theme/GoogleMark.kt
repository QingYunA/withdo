package xyz.sriai.adhdtodo.ui.theme

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.size
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.drawscope.scale
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.graphics.vector.PathParser
import androidx.compose.ui.unit.Dp

/**
 * Google's four-color "G", the one logo in the app that is not pixel art: Google's branding rules
 * for "Continue with Google" forbid redrawing or recoloring it. The same paths as the desktop's
 * app/theme/google-g.svg, in a square viewBox of -3 0 262 262.
 */
private val googlePaths: List<Pair<Color, String>> = listOf(
    Color(0xFF4285F4) to "M255.9 133.5c0-10.8-.9-18.6-2.8-26.7H130.6v48.4h71.9a64 64 0 0 1-26.7 42.4l-.2 1.6 38.7 30 2.7.3c24.7-22.8 38.9-56.3 38.9-96",
    Color(0xFF34A853) to "M130.6 261.1c35.2 0 64.8-11.6 86.4-31.6l-41.2-32a76 76 0 0 1-45.2 13.1 79 79 0 0 1-74.3-54.2l-1.5.1-40.3 31.2-.6 1.5A131 131 0 0 0 130.6 261",
    Color(0xFFFBBC05) to "M56.3 156.4a80 80 0 0 1-.2-51.7V103L15.3 71.3l-1.4.6a131 131 0 0 0 0 117.3z",
    Color(0xFFEA4335) to "M130.6 50.5c24.5 0 41 10.6 50.4 19.4L218 34c-22.8-21-52.2-34-87.4-34C79.5 0 35.4 29.3 13.9 72l42.2 32.7a79 79 0 0 1 74.5-54.2",
)

@Composable
fun GoogleMark(size: Dp, modifier: Modifier = Modifier) {
    val paths = remember { googlePaths.map { (color, d) -> color to PathParser().parsePathString(d).toPath() } }
    Canvas(modifier.size(size)) {
        val factor = this.size.minDimension / 262f
        scale(factor, factor, pivot = Offset.Zero) {
            translate(left = 3f) {
                for ((color, path) in paths) drawPath(path, color)
            }
        }
    }
}
