package xyz.sriai.adhdtodo.widget

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.LocalContext
import androidx.glance.action.clickable
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.GlanceAppWidgetReceiver
import androidx.glance.appwidget.action.actionStartActivity
import androidx.glance.appwidget.provideContent
import androidx.glance.layout.Alignment
import androidx.glance.layout.Column
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.height
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import xyz.sriai.adhdtodo.R

/** Note something, 2 x 2: the assistant avatar; a tap opens the composer, focused. */
class NoteWidget : GlanceAppWidget() {
    override suspend fun provideGlance(context: Context, id: GlanceId) {
        // Nothing here comes from the core; the words follow the app's language setting.
        val label = WidgetSource.resources(context).getString(R.string.widget_note)
        provideContent { NoteContent(label) }
    }
}

class NoteWidgetReceiver : GlanceAppWidgetReceiver() {
    override val glanceAppWidget: GlanceAppWidget = NoteWidget()
}

@Composable
fun NoteContent(label: String) {
    val context = LocalContext.current
    Column(
        modifier = GlanceModifier.fillMaxSize().card().clickable(actionStartActivity(WidgetIntents.compose(context))),
        verticalAlignment = Alignment.CenterVertically,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        AvatarImage("assistant", sizeDp = 40f)
        Spacer(GlanceModifier.height(10.dp))
        Text(
            label,
            maxLines = 1,
            style = TextStyle(color = WidgetColors.text, fontSize = 14.sp, fontWeight = FontWeight.Medium),
        )
    }
}
