package xyz.sriai.adhdtodo.ui

import android.content.res.Resources
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.BulkPatch
import xyz.sriai.adhdtodo.data.CoreApi
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import xyz.sriai.adhdtodo.ui.theme.StatusGlyph

/*
 * Select mode's bulk bar and the five things it does (v43). Every action is one change with one
 * Undo: the core's UpdateTasks, DoTodayMany and DeleteTasks return what Undo needs.
 */

/** What the bulk bar can do to the ticked tasks. */
sealed interface BulkAction {
    data object Done : BulkAction
    data object DoToday : BulkAction

    /** [projectId] 0 is the Inbox; [name] is how the snackbar names it. */
    data class MoveTo(val projectId: Long, val name: String) : BulkAction
    data class SetPriority(val priority: String) : BulkAction
    data object Delete : BulkAction
}

/** What happened, in words for the snackbar, and how to take it back (null: nothing to undo). */
class BulkOutcome(val message: String, val undo: (() -> Unit)?)

/** Blocks on the core: call it from Dispatchers.IO. */
fun runBulkAction(api: CoreApi, action: BulkAction, ids: List<Long>, res: Resources, words: Words): BulkOutcome {
    val n = ids.size
    return when (action) {
        BulkAction.Done -> {
            val before = api.updateTasks(ids, BulkPatch(status = "done"))
            BulkOutcome(res.getQuantityString(R.plurals.bulk_marked_done, n, n)) { api.restoreTaskStates(before) }
        }
        BulkAction.DoToday -> {
            val moved = api.doTodayMany(ids)
            if (moved.isEmpty()) {
                BulkOutcome(res.getString(R.string.bulk_nothing_to_move), null)
            } else {
                BulkOutcome(res.getQuantityString(R.plurals.bulk_moved_today, moved.size, moved.size)) {
                    api.restoreDues(moved)
                }
            }
        }
        is BulkAction.MoveTo -> {
            val before = api.updateTasks(ids, BulkPatch(projectId = action.projectId))
            BulkOutcome(res.getQuantityString(R.plurals.bulk_moved_to, n, n, action.name)) {
                api.restoreTaskStates(before)
            }
        }
        is BulkAction.SetPriority -> {
            val before = api.updateTasks(ids, BulkPatch(priority = action.priority))
            val name = "${action.priority} ${words.priorityName(action.priority)}".trim()
            BulkOutcome(res.getQuantityString(R.plurals.bulk_priority_set, n, n, name)) {
                api.restoreTaskStates(before)
            }
        }
        BulkAction.Delete -> {
            val snapshots = api.deleteTasks(ids)
            BulkOutcome(res.getQuantityString(R.plurals.bulk_deleted, n, n)) { api.restoreTasks(snapshots) }
        }
    }
}

/**
 * The bar that takes the bottom bar's place: Done, Do today, Move to…, Priority and Delete (red),
 * each a 20 dp icon over a 12 sp label, on bg.sidebar under a hairline.
 */
@Composable
fun BulkBar(
    enabled: Boolean,
    onDone: () -> Unit,
    onDoToday: () -> Unit,
    onMove: () -> Unit,
    onPriority: () -> Unit,
    onDelete: () -> Unit,
) {
    val tokens = LocalTokens.current
    Column(Modifier.fillMaxWidth().background(tokens.bgSidebar).navigationBarsPadding()) {
        Box(Modifier.fillMaxWidth().height(Dims.hairline).background(tokens.border))
        Row(
            Modifier.fillMaxWidth().height(Dims.navHeight).padding(horizontal = 8.dp),
            horizontalArrangement = Arrangement.SpaceAround,
        ) {
            BulkItem(stringResource(R.string.bulk_done), enabled, onDone) {
                StatusGlyph("done", size = 20.dp)
            }
            BulkItem(stringResource(R.string.bulk_do_today), enabled, onDoToday) {
                PixelIcon("calendar", size = 20.dp, tint = tokens.textSecondary)
            }
            BulkItem(stringResource(R.string.bulk_move), enabled, onMove) {
                PixelIcon("folder", size = 20.dp, tint = tokens.textSecondary)
            }
            BulkItem(stringResource(R.string.bulk_priority), enabled, onPriority) {
                PixelIcon("flag", size = 20.dp, tint = tokens.textSecondary)
            }
            BulkItem(stringResource(R.string.bulk_delete), enabled, onDelete, danger = true) {
                PixelIcon("trash", size = 20.dp, tint = tokens.statusBlocked)
            }
        }
    }
}

@Composable
private fun BulkItem(
    label: String,
    enabled: Boolean,
    onClick: () -> Unit,
    danger: Boolean = false,
    icon: @Composable () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Column(
        modifier = Modifier
            .size(width = 68.dp, height = Dims.navHeight)
            .clickable(enabled = enabled, onClick = onClick)
            .padding(top = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(Modifier.size(width = 48.dp, height = 28.dp), contentAlignment = Alignment.Center) { icon() }
        Text(
            label,
            style = type.nav,
            color = if (danger) tokens.statusBlocked else tokens.text,
            maxLines = 1,
            modifier = Modifier.padding(top = 4.dp),
        )
    }
}

/** The modal sheet of the task menu, for the two pickers the bar opens. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun BulkSheet(onDismiss: () -> Unit, content: @Composable () -> Unit) {
    val tokens = LocalTokens.current
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        containerColor = tokens.bg,
        scrimColor = tokens.scrim,
        shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp),
        dragHandle = {
            Box(Modifier.fillMaxWidth().height(22.dp), contentAlignment = Alignment.Center) {
                Box(
                    Modifier
                        .size(width = 32.dp, height = 4.dp)
                        .background(tokens.textTertiary.copy(alpha = 0.7f), RoundedCornerShape(2.dp)),
                )
            }
        },
    ) {
        Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(bottom = 18.dp)) { content() }
    }
}

/** Move to…: every project, then the Inbox. [onPick] gets the project's id (0 for the Inbox) and its name. */
@Composable
fun BulkMoveSheet(projects: List<Project>, onPick: (Long, String) -> Unit, onDismiss: () -> Unit) {
    val tokens = LocalTokens.current
    val inbox = stringResource(R.string.title_inbox)
    BulkSheet(onDismiss) {
        SheetTitle(stringResource(R.string.bulk_move))
        for (project in projects) {
            MenuRow(
                label = project.name,
                leading = { PixelIcon("folder", size = 20.dp, tint = tokens.projectColor(project.name)) },
                onClick = { onPick(project.id, project.name) },
            )
        }
        MenuRow(
            label = inbox,
            leading = { PixelIcon("inbox", size = 20.dp, tint = tokens.textSecondary) },
            onClick = { onPick(0, inbox) },
        )
    }
}

/** Priority: P0 Urgent to P3 Low. */
@Composable
fun BulkPrioritySheet(words: Words, onPick: (String) -> Unit, onDismiss: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    BulkSheet(onDismiss) {
        SheetTitle(stringResource(R.string.bulk_priority))
        for (priority in listOf("P0", "P1", "P2", "P3")) {
            MenuRow(
                label = words.priorityName(priority),
                leading = {
                    Text(
                        priority,
                        style = type.bodyStrong.copy(fontSize = 15.sp, lineHeight = 22.sp),
                        color = if (priority == "P0") tokens.statusBlocked else tokens.text,
                    )
                },
                onClick = { onPick(priority) },
            )
        }
    }
}

@Composable
private fun SheetTitle(title: String) {
    Text(
        title,
        style = LocalAppType.current.bodyStrong.copy(fontSize = 17.sp, lineHeight = 24.sp),
        color = LocalTokens.current.text,
        modifier = Modifier.padding(start = 20.dp, end = 20.dp, top = 4.dp, bottom = 10.dp),
    )
}
