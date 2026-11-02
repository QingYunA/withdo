package xyz.sriai.adhdtodo.widget

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.ImageProvider
import androidx.glance.LocalContext
import androidx.glance.action.clickable
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.GlanceAppWidgetReceiver
import androidx.glance.appwidget.action.actionStartActivity
import androidx.glance.appwidget.provideContent
import androidx.glance.background
import androidx.glance.layout.Alignment
import androidx.glance.layout.Box
import androidx.glance.layout.Column
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.height
import androidx.glance.layout.padding
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import xyz.sriai.adhdtodo.R

/** Agents waiting, 2 x 2: the first card that waits for you, its age, and Reply. */
class AgentsWidget : GlanceAppWidget() {
    override suspend fun provideGlance(context: Context, id: GlanceId) {
        val data = WidgetSource.agents(context)
        provideContent { AgentsContent(data) }
    }
}

class AgentsWidgetReceiver : GlanceAppWidgetReceiver() {
    override val glanceAppWidget: GlanceAppWidget = AgentsWidget()
}

@Composable
fun AgentsContent(data: AgentsData) {
    val context = LocalContext.current
    val card = data.card
    // The whole card opens the task (or the app when nothing waits); Reply is the visible target.
    val open = actionStartActivity(if (card != null) WidgetIntents.task(context, card.id) else WidgetIntents.open(context))
    Column(
        modifier = GlanceModifier.fillMaxSize().card().padding(start = 10.dp, end = 10.dp, top = 12.dp, bottom = 12.dp)
            .clickable(open),
    ) {
        WidgetHeader(avatar = "bot", label = data.label, count = data.count)
        if (card == null) {
            Box(GlanceModifier.fillMaxSize(), contentAlignment = Alignment.CenterStart) { QuietLine(data.empty) }
            return@Column
        }
        Text(
            card.title,
            modifier = GlanceModifier.padding(start = 4.dp, end = 4.dp, top = 4.dp),
            maxLines = 2,
            style = TextStyle(color = WidgetColors.text, fontSize = 14.sp, fontWeight = FontWeight.Medium),
        )
        Text(
            card.meta,
            modifier = GlanceModifier.padding(start = 4.dp, end = 4.dp, top = 2.dp),
            maxLines = 2,
            style = TextStyle(color = WidgetColors.textSecondary, fontSize = 12.sp),
        )
        Spacer(GlanceModifier.defaultWeight())
        ReplyButton(data.replyLabel)
    }
}

/** The tinted button: accent-soft with the accent ink. */
@Composable
private fun ReplyButton(label: String) {
    Box(
        modifier = GlanceModifier.padding(horizontal = 4.dp),
    ) {
        Box(
            modifier = GlanceModifier.height(32.dp).padding(horizontal = 12.dp)
                .background(ImageProvider(R.drawable.widget_well_button)),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                label,
                maxLines = 1,
                style = TextStyle(color = WidgetColors.accentInk, fontSize = 13.sp, fontWeight = FontWeight.Medium),
            )
        }
    }
}
