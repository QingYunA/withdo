package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.semantics.onClick
import androidx.compose.ui.semantics.onLongClick
import androidx.compose.ui.semantics.semantics
import kotlinx.coroutines.withTimeoutOrNull
import xyz.sriai.adhdtodo.ui.theme.LocalTokens

private enum class PressResult { Tap, Moved, Cancel }

/**
 * A task row that opens its task on a tap and the task menu on a long press. The row takes
 * bg.selected as soon as the long press begins. The sheet opens when the finger lifts without
 * moving; a long press that moves further than the touch slop is left for the coming drag, so it
 * does not open. A tap consumed by a child (the status glyph, a row button) is left alone. The
 * gesture is drawn by hand, so the row also says it can be clicked and long clicked, for TalkBack.
 */
@Composable
fun LongPressRow(
    onClick: () -> Unit,
    onLongPress: () -> Unit,
    modifier: Modifier = Modifier,
    content: @Composable (Modifier) -> Unit,
) {
    val tokens = LocalTokens.current
    var pressed by remember { mutableStateOf(false) }
    val gesture = Modifier.pointerInput(onClick, onLongPress) {
        awaitEachGesture {
            val down = awaitFirstDown(requireUnconsumed = false)
            if (down.isConsumed) return@awaitEachGesture
            val start = down.position
            val slop = viewConfiguration.touchSlop
            var moved = false
            val early = withTimeoutOrNull(viewConfiguration.longPressTimeoutMillis) {
                while (true) {
                    val event = awaitPointerEvent()
                    val change = event.changes.firstOrNull { it.id == down.id }
                        ?: return@withTimeoutOrNull PressResult.Cancel
                    if ((change.position - start).getDistance() > slop) {
                        return@withTimeoutOrNull PressResult.Moved
                    }
                    if (change.isConsumed) return@withTimeoutOrNull PressResult.Cancel
                    if (!change.pressed) {
                        change.consume()
                        return@withTimeoutOrNull PressResult.Tap
                    }
                }
                @Suppress("UNREACHABLE_CODE")
                PressResult.Cancel
            }
            when (early) {
                null -> {
                    // The long press began: hold the row selected until the finger lifts.
                    pressed = true
                    var cancelled = false
                    try {
                        while (true) {
                            val event = awaitPointerEvent()
                            val change = event.changes.firstOrNull { it.id == down.id } ?: break
                            if (change.isConsumed) {
                                cancelled = true
                                break
                            }
                            if ((change.position - start).getDistance() > slop) {
                                moved = true
                                change.consume()
                                break
                            }
                            if (!change.pressed) {
                                change.consume()
                                break
                            }
                            change.consume()
                        }
                    } finally {
                        pressed = false
                    }
                    if (!moved && !cancelled) onLongPress()
                }
                PressResult.Tap -> onClick()
                else -> Unit
            }
        }
    }
    content(
        modifier
            .fillMaxWidth()
            .then(if (pressed) Modifier.background(tokens.bgSelected) else Modifier)
            .then(gesture)
            .semantics(mergeDescendants = false) {
                onClick { onClick(); true }
                onLongClick { onLongPress(); true }
            },
    )
}
