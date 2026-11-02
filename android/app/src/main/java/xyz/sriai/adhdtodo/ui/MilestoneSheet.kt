package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Milestone
import xyz.sriai.adhdtodo.data.MilestonePatch
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId

private sealed interface MilestoneStep {
    data object Menu : MilestoneStep
    data object New : MilestoneStep
    data class Edit(val milestone: Milestone) : MilestoneStep
    data class Rename(val milestone: Milestone) : MilestoneStep
    data class Due(val milestone: Milestone) : MilestoneStep
}

/** The day a milestone is due, in the phone's zone, or null. */
internal fun Milestone.dueDay(zone: ZoneId = ZoneId.systemDefault()): LocalDate? =
    dueAt?.takeIf { it.isNotBlank() }
        ?.let { runCatching { Instant.parse(it).atZone(zone).toLocalDate() }.getOrNull() }

/** Open milestones first (soonest due first), closed ones after. */
internal fun sortedMilestones(milestones: List<Milestone>): List<Milestone> =
    milestones.sortedWith(
        compareBy<Milestone>({ !it.open }, { it.dueDay() == null }, { it.dueDay() }, { it.id }),
    )

/** A milestone as the filter lists it: `v0.2 · due Nov 14`, `v0.1 · closed`, or its title alone. */
@Composable
internal fun milestoneLabel(m: Milestone): String {
    val day = m.dueDay()
    return when {
        !m.open -> stringResource(R.string.milestone_closed, m.title)
        day != null -> stringResource(R.string.milestone_due, m.title, dueLabel(day))
        else -> m.title
    }
}

/**
 * The milestone filter's sheet: All milestones, each milestone (checked when it filters the page),
 * New milestone…, and at the right of each a `⋯` to its Rename… / Due date… / Close or Reopen.
 */
@Composable
fun MilestoneFilterSheet(
    projectId: Long,
    milestones: List<Milestone>,
    selected: Long?,
    actions: ProjectActions,
    onSelect: (Long?) -> Unit,
    onDismiss: () -> Unit,
) {
    val tokens = LocalTokens.current
    val scope = rememberCoroutineScope()
    var step by remember { mutableStateOf<MilestoneStep>(MilestoneStep.Menu) }

    fun run(action: suspend () -> Unit) {
        scope.launch {
            try {
                action()
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                // The page reads again; a refused change leaves the milestone as it was.
            }
            onDismiss()
        }
    }

    when (val current = step) {
        MilestoneStep.Menu -> SheetFrame(onDismiss) {
            SheetRow(
                label = stringResource(R.string.milestone_all),
                checked = selected == null,
                onClick = { onSelect(null); onDismiss() },
            )
            for (m in sortedMilestones(milestones)) {
                SheetRow(
                    label = milestoneLabel(m),
                    checked = selected == m.id,
                    selected = selected == m.id,
                    onClick = { onSelect(m.id); onDismiss() },
                    trailing = {
                        Box(
                            Modifier
                                .size(40.dp)
                                .clickable { step = MilestoneStep.Edit(m) }
                                .semantics { contentDescription = m.title },
                            contentAlignment = Alignment.Center,
                        ) {
                            xyz.sriai.adhdtodo.ui.MoreGlyph(tint = tokens.textTertiary)
                        }
                    },
                )
            }
            SheetRule()
            SheetRow(stringResource(R.string.milestone_new), icon = "plus", onClick = { step = MilestoneStep.New })
        }
        MilestoneStep.New -> PromptSheet(
            title = stringResource(R.string.milestone_new_title),
            placeholder = stringResource(R.string.milestone_placeholder),
            confirm = stringResource(R.string.action_done),
            withDue = true,
            onDismiss = onDismiss,
            onConfirm = { title, due -> actions.createMilestone(projectId, title, due?.toString().orEmpty()) },
        )
        is MilestoneStep.Edit -> SheetFrame(onDismiss) {
            val m = current.milestone
            SheetBack(stringResource(R.string.milestone_edit, m.title)) { step = MilestoneStep.Menu }
            SheetRow(stringResource(R.string.manage_rename_ellipsis), icon = "bulletlist", onClick = { step = MilestoneStep.Rename(m) })
            SheetRow(
                label = stringResource(R.string.milestone_due_ellipsis),
                icon = "calendar",
                hint = m.dueDay()?.let { dueLabel(it) },
                onClick = { step = MilestoneStep.Due(m) },
            )
            SheetRow(
                label = stringResource(if (m.open) R.string.milestone_close else R.string.milestone_reopen),
                icon = "check",
                onClick = { run { actions.updateMilestone(m.id, MilestonePatch(closed = m.open)) } },
            )
        }
        is MilestoneStep.Rename -> PromptSheet(
            title = stringResource(R.string.milestone_rename_title),
            placeholder = stringResource(R.string.milestone_placeholder),
            confirm = stringResource(R.string.manage_save),
            initial = current.milestone.title,
            onDismiss = onDismiss,
            onConfirm = { title, _ -> actions.updateMilestone(current.milestone.id, MilestonePatch(title = title)) },
        )
        is MilestoneStep.Due -> DueDateDialog(
            initial = current.milestone.dueDay(),
            onPick = { day -> run { actions.updateMilestone(current.milestone.id, MilestonePatch(dueAt = day.toString())) } },
            onClear = { run { actions.updateMilestone(current.milestone.id, MilestonePatch(clearDueAt = true)) } },
            onDismiss = onDismiss,
        )
    }
}

/** A sheet page's back bar: the chevron and the page's name. */
@Composable
fun SheetBack(title: String, onBack: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        Modifier.fillMaxWidth().height(52.dp).padding(horizontal = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Box(Modifier.size(40.dp).clickable(onClick = onBack), contentAlignment = Alignment.Center) {
            PixelIcon("chevron-left", size = 18.dp, tint = tokens.textSecondary)
        }
        Text(
            title,
            style = type.bodyStrong.copy(fontSize = 17.sp, lineHeight = 24.sp),
            color = tokens.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}
