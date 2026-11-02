package xyz.sriai.adhdtodo.ui

import android.content.res.Resources
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.Orientation
import androidx.compose.foundation.gestures.draggable
import androidx.compose.foundation.gestures.rememberDraggableState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.wrapContentWidth
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.RectangleShape
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.BulkPatch
import xyz.sriai.adhdtodo.data.CoreApi
import xyz.sriai.adhdtodo.data.TaskPatch
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import java.time.LocalDate
import java.time.ZoneId
import kotlin.math.abs
import kotlin.math.roundToInt

/*
 * Swipe a row (v43 P2): right completes, left is Do today, or Do tomorrow for a task already due
 * today. Past half the row's width the action runs on release; short of it the row springs back.
 */

/** What a swipe does. */
enum class SwipeAction { Done, Today, Tomorrow }

/** The two directions of one row; null where that side does not swipe. */
data class RowSwipes(val right: SwipeAction?, val left: SwipeAction?)

/**
 * The swipes of a task: none for a finished one; right (Done) for any open task; left only for a
 * task of mine, Do tomorrow when it is due [today] already, else Do today. An agent's task only
 * swipes right.
 */
fun rowSwipes(source: String, status: String, dueAt: String?, today: LocalDate, zone: ZoneId): RowSwipes? {
    if (status == "done" || status == "cancelled") return null
    val left = when {
        source != "user" -> null
        dueDay(dueAt, zone) == today -> SwipeAction.Tomorrow
        else -> SwipeAction.Today
    }
    return RowSwipes(right = SwipeAction.Done, left = left)
}

/** What a swipe did: the snackbar's words and its Undo, which writes to the core. */
class SwipeOutcome(val message: String, val undo: () -> Unit)

/**
 * Runs [action] on a task (off the UI thread): Done through the bulk edit so Undo puts the status
 * back; Do today and Do tomorrow keep the task's time of day, and Undo restores its old due date.
 */
fun runSwipe(api: CoreApi, res: Resources, task: SwipedTask, action: SwipeAction, today: LocalDate, zone: ZoneId): SwipeOutcome =
    when (action) {
        SwipeAction.Done -> {
            val before = api.updateTasks(listOf(task.id), BulkPatch(status = "done"))
            SwipeOutcome(res.getString(R.string.toast_completed, task.title)) { api.restoreTaskStates(before) }
        }
        SwipeAction.Today -> {
            val before = api.doToday(task.id)
            SwipeOutcome(res.getString(R.string.toast_moved_to_today)) { api.restoreDue(task.id, before) }
        }
        SwipeAction.Tomorrow -> {
            api.updateTask(task.id, TaskPatch(dueAt = dueStamp(today.plusDays(1), task.dueAt, zone)))
            SwipeOutcome(res.getString(R.string.toast_moved_to_tomorrow)) { api.restoreDue(task.id, task.dueAt.orEmpty()) }
        }
    }

/** The parts of a row's task a swipe needs. */
data class SwipedTask(val id: Long, val title: String, val dueAt: String?)

/** The action a drag of [offset] px over a row [width] px wide runs on release, or null. */
fun swipeOnRelease(swipes: RowSwipes, offset: Float, width: Int): SwipeAction? {
    if (width <= 0 || abs(offset) < width / 2f) return null
    return if (offset > 0) swipes.right else swipes.left
}

/**
 * A row that swipes. The row slides over a colored layer that names the action: `status.done`
 * with a white check and Done on the left, iris 9 with Do today (or Do tomorrow) and the calendar
 * on the right. Rows in select mode, and rows with [swipes] null, do not swipe. [preview] places
 * the row at that fraction of its width, for the design screenshots.
 */
@Composable
fun SwipeRow(
    swipes: RowSwipes?,
    onSwipe: (SwipeAction) -> Unit,
    modifier: Modifier = Modifier,
    shape: Shape = RectangleShape,
    preview: Float = 0f,
    content: @Composable () -> Unit,
) {
    val tokens = LocalTokens.current
    if (swipes == null || LocalSelection.current != null) {
        Box(modifier) { content() }
        return
    }
    val haptics = LocalHapticFeedback.current
    val scope = rememberCoroutineScope()
    var width by remember { mutableIntStateOf(0) }
    val offset = remember { Animatable(0f) }
    val shown = if (preview != 0f && offset.value == 0f) preview * width else offset.value
    val drag = rememberDraggableState { delta ->
        val max = if (swipes.right != null) width.toFloat() else 0f
        val min = if (swipes.left != null) -width.toFloat() else 0f
        scope.launch { offset.snapTo((offset.value + delta).coerceIn(min, max)) }
    }
    Box(
        modifier
            .fillMaxWidth()
            .clip(shape)
            .onSizeChanged { width = it.width }
            .draggable(
                state = drag,
                orientation = Orientation.Horizontal,
                onDragStopped = {
                    val action = swipeOnRelease(swipes, offset.value, width)
                    if (action != null) {
                        haptics.performHapticFeedback(HapticFeedbackType.SegmentTick)
                        onSwipe(action)
                    }
                    offset.animateTo(0f, spring(stiffness = Spring.StiffnessMediumLow))
                },
            ),
    ) {
        // The layer covers only what the row uncovers, so no color shows at the clip's far corners.
        val uncovered = with(LocalDensity.current) { abs(shown).toDp() }
        Box(Modifier.matchParentSize()) {
            if (shown > 0f) {
                SwipeLayer(swipes.right, tokens.statusDone, Alignment.CenterStart, Modifier.align(Alignment.CenterStart).fillMaxHeight().width(uncovered))
            } else if (shown < 0f) {
                SwipeLayer(swipes.left, tokens.accent, Alignment.CenterEnd, Modifier.align(Alignment.CenterEnd).fillMaxHeight().width(uncovered))
            }
        }
        Box(
            Modifier
                .offset { IntOffset(shown.roundToInt(), 0) }
                .background(tokens.bg),
        ) { content() }
    }
}

/** The colored layer under a sliding row: the action's words and icon in white at [alignment]. */
@Composable
private fun SwipeLayer(action: SwipeAction?, color: Color, alignment: Alignment, modifier: Modifier) {
    if (action == null) return
    val type = LocalAppType.current
    val white = Color.White
    val label = stringResource(
        when (action) {
            SwipeAction.Done -> R.string.swipe_done
            SwipeAction.Today -> R.string.swipe_today
            SwipeAction.Tomorrow -> R.string.swipe_tomorrow
        },
    )
    Box(modifier.clipToBounds().background(color).padding(horizontal = 16.dp), contentAlignment = alignment) {
        // The words keep their width and are cut by the layer's edge while it is narrow.
        Row(
            modifier = Modifier.wrapContentWidth(align = if (alignment == Alignment.CenterStart) Alignment.Start else Alignment.End, unbounded = true),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            val style = type.bodyStrong.copy(fontWeight = FontWeight.SemiBold)
            if (action == SwipeAction.Done) {
                PixelIcon("check", size = 18.dp, tint = white)
                Text(label, style = style, color = white, maxLines = 1)
            } else {
                Text(label, style = style, color = white, maxLines = 1)
                PixelIcon("calendar", size = 20.dp, tint = white)
            }
        }
    }
}
