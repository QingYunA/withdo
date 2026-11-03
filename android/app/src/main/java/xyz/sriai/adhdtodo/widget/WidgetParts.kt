package xyz.sriai.adhdtodo.widget

import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.ColorFilter
import androidx.glance.GlanceModifier
import androidx.glance.Image
import androidx.glance.ImageProvider
import androidx.glance.LocalContext
import androidx.glance.background
import androidx.glance.layout.Alignment
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.layout.padding
import androidx.glance.layout.size
import androidx.glance.layout.width
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import androidx.glance.unit.ColorProvider
import xyz.sriai.adhdtodo.R

/** The card behind every widget: the app's bg token, rounded (drawable-night has the dark one). */
fun GlanceModifier.card(): GlanceModifier = background(ImageProvider(R.drawable.widget_card))

/** A pixel picture at its exact size; [tint] recolors it (the pictures are drawn in white). */
@Composable
fun PixelImage(picture: PixelPicture, modifier: GlanceModifier = GlanceModifier, tint: ColorProvider? = null) {
    Image(
        provider = ImageProvider(picture.bitmap),
        contentDescription = null,
        modifier = modifier.size(picture.size),
        colorFilter = tint?.let { ColorFilter.tint(it) },
    )
}

/** An 18 dp sprite from the avatars: you, bot, assistant. */
@Composable
fun AvatarImage(name: String, sizeDp: Float = WidgetDims.avatar) {
    val picture = WidgetPixels.avatar(LocalContext.current, name, sizeDp) ?: return
    PixelImage(picture)
}

/**
 * The header row of Today and Agents: avatar, the label in 15 sp semibold, the count in secondary
 * (left out at zero), then whatever [trailing] draws at the right.
 */
@Composable
fun WidgetHeader(
    avatar: String,
    label: String,
    count: Int,
    trailing: @Composable () -> Unit = {},
) {
    Row(
        modifier = GlanceModifier.fillMaxWidth().height(WidgetDims.header.dp).padding(horizontal = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        AvatarImage(avatar)
        Spacer(GlanceModifier.width(8.dp))
        Text(label, style = TextStyle(color = WidgetColors.text, fontSize = 15.sp, fontWeight = FontWeight.Medium))
        if (count > 0) {
            Spacer(GlanceModifier.width(6.dp))
            Text(count.toString(), style = TextStyle(color = WidgetColors.textSecondary, fontSize = 15.sp))
        }
        Spacer(GlanceModifier.defaultWeight())
        trailing()
    }
}

/** The one quiet line of an empty widget. */
@Composable
fun QuietLine(text: String, modifier: GlanceModifier = GlanceModifier) {
    if (text.isEmpty()) return
    Text(
        text,
        modifier = modifier.padding(horizontal = 4.dp),
        style = TextStyle(color = WidgetColors.textSecondary, fontSize = 13.sp),
    )
}
