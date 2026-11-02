package xyz.sriai.adhdtodo.widget

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.Image
import androidx.glance.ImageProvider
import androidx.glance.LocalContext
import androidx.glance.LocalSize
import androidx.glance.action.ActionParameters
import androidx.glance.action.actionParametersOf
import androidx.glance.action.clickable
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.GlanceAppWidgetReceiver
import androidx.glance.appwidget.SizeMode
import androidx.glance.appwidget.action.ActionCallback
import androidx.glance.appwidget.action.actionRunCallback
import androidx.glance.appwidget.action.actionStartActivity
import androidx.glance.appwidget.provideContent
import androidx.glance.background
import androidx.glance.layout.Alignment
import androidx.glance.layout.Box
import androidx.glance.layout.Column
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.layout.padding
import androidx.glance.layout.size
import androidx.glance.layout.width
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import androidx.glance.ColorFilter
import xyz.sriai.adhdtodo.R

/** Today, 4 x 2: you, the day's count, a round + for the composer, and the first rows. */
class TodayWidget : GlanceAppWidget() {
    // The rows that fit are drawn, so the widget needs the size it was given.
    override val sizeMode = SizeMode.Exact

    override suspend fun provideGlance(context: Context, id: GlanceId) {
        val data = WidgetSource.today(context)
        provideContent { TodayContent(data) }
    }
}

class TodayWidgetReceiver : GlanceAppWidgetReceiver() {
    override val glanceAppWidget: GlanceAppWidget = TodayWidget()
}

@Composable
fun TodayContent(data: TodayData) {
    val context = LocalContext.current
    val rows = data.rows.take(rowsThatFit(LocalSize.current.height.value))
    Column(
        modifier = GlanceModifier.fillMaxSize().card().padding(start = 10.dp, end = 10.dp, top = 12.dp, bottom = 8.dp)
            .clickable(actionStartActivity(WidgetIntents.open(context))),
    ) {
        WidgetHeader(avatar = "you", label = data.label, count = data.count) {
            AddButton(data.addLabel)
        }
        if (rows.isEmpty()) {
            Box(GlanceModifier.fillMaxSize(), contentAlignment = Alignment.CenterStart) { QuietLine(data.empty) }
        }
        for (row in rows) TodayRowView(row, data.completeLabel(row.title))
    }
}

/** The round + on accent-soft: opens the composer. */
@Composable
private fun AddButton(description: String) {
    val context = LocalContext.current
    Box(
        modifier = GlanceModifier.size(WidgetDims.well.dp).background(ImageProvider(R.drawable.widget_well_round))
            .clickable(actionStartActivity(WidgetIntents.compose(context))),
        contentAlignment = Alignment.Center,
    ) {
        val plus = WidgetPixels.plus(context, 14f)
        Image(
            provider = ImageProvider(plus.bitmap),
            contentDescription = description,
            modifier = GlanceModifier.size(plus.size),
            colorFilter = ColorFilter.tint(WidgetColors.accentInk),
        )
    }
}

/** A row: the status glyph completes the task, the title and the due label open it. */
@Composable
private fun TodayRowView(row: TodayRow, completeLabel: String) {
    val context = LocalContext.current
    Row(
        modifier = GlanceModifier.fillMaxWidth().height(WidgetDims.row.dp)
            .clickable(actionStartActivity(WidgetIntents.task(context, row.id))),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        val glyph = WidgetPixels.glyph(context, row.status, WidgetDims.avatar)
        Box(
            // 36 dp square around the 18 dp glyph: a thumb-sized target for completing.
            modifier = GlanceModifier.size(WidgetDims.row.dp)
                .clickable(actionRunCallback<CompleteAction>(actionParametersOf(CompleteAction.TaskId to row.id))),
            contentAlignment = Alignment.Center,
        ) {
            Image(
                provider = ImageProvider(glyph.ink.bitmap),
                contentDescription = completeLabel,
                modifier = GlanceModifier.size(glyph.ink.size),
                colorFilter = ColorFilter.tint(WidgetColors.status(row.status)),
            )
            glyph.mark?.let { PixelImage(it) }
        }
        Text(
            row.title,
            modifier = GlanceModifier.defaultWeight(),
            maxLines = 1,
            style = TextStyle(color = WidgetColors.text, fontSize = 14.sp),
        )
        if (row.due.isNotEmpty()) {
            Spacer(GlanceModifier.width(8.dp))
            Text(
                row.due,
                maxLines = 1,
                style = TextStyle(color = WidgetColors.textSecondary, fontSize = 12.sp),
            )
            Spacer(GlanceModifier.width(4.dp))
        }
    }
}

/** The glyph's tap: finish the task in a short-lived core, then redraw the widgets. */
class CompleteAction : ActionCallback {
    override suspend fun onAction(context: Context, glanceId: GlanceId, parameters: ActionParameters) {
        val id = parameters[TaskId] ?: return
        WidgetSource.complete(context, id)
        Widgets.refreshNow(context)
    }

    companion object {
        val TaskId = ActionParameters.Key<Long>("task_id")
    }
}
