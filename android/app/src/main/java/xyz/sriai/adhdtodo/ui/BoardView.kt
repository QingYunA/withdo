package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.PageSize
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Milestone
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import xyz.sriai.adhdtodo.ui.theme.StatusGlyph

/** The desktop's board lanes, left to right. Backlog and Cancelled are not on the board. */
val boardStatuses = listOf("todo", "doing", "review", "done")

/** Done shows its newest five; `Show all 12` opens the lane in place. */
const val BOARD_DONE_LIMIT = 5

/** One lane of the board: its status and its cards, top first. */
data class BoardLane(val status: String, val tasks: List<Task>)

/**
 * The board's four lanes from a project's top-level [tasks] (the desktop's rules): a blocked task sits
 * in Doing, first; Done is newest first; Backlog and Cancelled stay off the board.
 */
fun boardLanes(tasks: List<Task>): List<BoardLane> = boardStatuses.map { lane ->
    val cards = when (lane) {
        "doing" -> tasks.filter { it.status == "blocked" } + tasks.filter { it.status == "doing" }
        "done" -> tasks.filter { it.status == "done" }.sortedByDescending { it.completedAt ?: it.updatedAt }
        else -> tasks.filter { it.status == lane }
    }
    BoardLane(lane, cards)
}

/** The `List | Board` control: two 40 x 32 dp cells in a `bg.hover` well, the chosen one raised. */
@Composable
fun ProjectViewToggle(view: ProjectView, onChange: (ProjectView) -> Unit) {
    val tokens = LocalTokens.current
    val well = RoundedCornerShape(10.dp)
    Row(
        Modifier
            .background(tokens.bgHover, well)
            .padding(2.dp),
    ) {
        for ((value, icon, label) in listOf(
            Triple(ProjectView.List, "bulletlist", stringResource(R.string.project_view_list)),
            Triple(ProjectView.Board, "grid-3x2", stringResource(R.string.project_view_board)),
        )) {
            val on = value == view
            val cell = RoundedCornerShape(8.dp)
            Box(
                Modifier
                    .size(width = 40.dp, height = 32.dp)
                    .then(
                        if (on) Modifier.background(tokens.bg, cell).border(Dims.hairline, tokens.border, cell) else Modifier,
                    )
                    .clip(cell)
                    .clickable { onChange(value) }
                    .semantics { contentDescription = label },
                contentAlignment = Alignment.Center,
            ) {
                PixelIcon(icon, size = 16.dp, tint = if (on) tokens.text else tokens.textTertiary)
            }
        }
    }
}

/** Quiet `meta` text with a chevron under the title: `All milestones ⌄`, or the one filtered by. */
@Composable
fun MilestoneFilterRow(
    label: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    trailing: @Composable () -> Unit = {},
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier.fillMaxWidth().height(36.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Row(
            Modifier.clip(RoundedCornerShape(6.dp)).clickable(onClick = onClick).height(36.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            Text(label, style = type.meta, color = tokens.textSecondary, maxLines = 1, overflow = TextOverflow.Ellipsis)
            PixelIcon("chevron-down", size = 12.dp, tint = tokens.textTertiary)
        }
        Box(Modifier.weight(1f))
        trailing()
    }
}

/** One 6 dp square per lane; the visible lane's is `text.secondary`. */
@Composable
private fun LanePosition(count: Int, current: Int) {
    val tokens = LocalTokens.current
    Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
        for (i in 0 until count) {
            Box(Modifier.size(6.dp).background(if (i == current) tokens.textSecondary else tokens.border))
        }
    }
}

/**
 * The project's board (v43): lanes 300 dp wide, 12 dp apart, in a snapping pager with the next lane
 * peeking at the right. [filter] is the milestone text under the title (null when the project has no
 * milestone); the lane squares sit at its right.
 */
@Composable
fun BoardScreen(
    tasks: List<Task>,
    agents: Map<String, String>,
    milestones: List<Milestone>,
    words: Words,
    filter: String?,
    onFilter: () -> Unit,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
) {
    val lanes = remember(tasks) { boardLanes(tasks) }
    val pager = rememberPagerState { lanes.size }
    Column(Modifier.fillMaxSize()) {
        val squares: @Composable () -> Unit = { LanePosition(lanes.size, pager.currentPage) }
        if (filter != null) {
            MilestoneFilterRow(filter, onFilter, Modifier.padding(horizontal = Dims.pagePadding), trailing = squares)
        } else {
            Row(Modifier.fillMaxWidth().height(36.dp).padding(horizontal = Dims.pagePadding), verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.weight(1f))
                squares()
            }
        }
        BoxWithConstraints(Modifier.weight(1f)) {
            // Room for the last lane to reach the left edge, so the squares can say "4 of 4".
            val end = (maxWidth - Dims.pagePadding - LANE_WIDTH).coerceAtLeast(Dims.pagePadding)
            HorizontalPager(
                state = pager,
                pageSize = PageSize.Fixed(LANE_WIDTH),
                pageSpacing = 12.dp,
                contentPadding = PaddingValues(start = Dims.pagePadding, end = end),
                verticalAlignment = Alignment.Top,
                modifier = Modifier.fillMaxSize(),
            ) { index ->
                LaneColumn(lanes[index], agents, milestones, words, onOpenTask, onLongPress)
            }
        }
    }
}

private val LANE_WIDTH = 300.dp

@Composable
private fun LaneColumn(
    lane: BoardLane,
    agents: Map<String, String>,
    milestones: List<Milestone>,
    words: Words,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    var all by remember { mutableStateOf(false) }
    val shown = if (lane.status == "done" && !all) lane.tasks.take(BOARD_DONE_LIMIT) else lane.tasks
    Column(Modifier.width(LANE_WIDTH).fillMaxSize().verticalScroll(rememberScrollState()).padding(bottom = 16.dp)) {
        Row(
            Modifier.height(36.dp).padding(horizontal = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            StatusGlyph(lane.status, size = Dims.glyph)
            Text(words.menuStatus(lane.status), style = type.bodyStrong.copy(fontWeight = FontWeight.SemiBold), color = tokens.text)
            Text(lane.tasks.size.toString(), style = type.body, color = tokens.textSecondary)
        }
        if (lane.tasks.isEmpty()) {
            Text(
                stringResource(if (lane.status == "review") R.string.board_empty_review else R.string.board_empty),
                style = type.meta,
                color = tokens.textTertiary,
                modifier = Modifier.padding(horizontal = 4.dp, vertical = 6.dp),
            )
        }
        for (task in shown) {
            BoardCard(
                task = task,
                agent = agents[task.id.toString()].orEmpty(),
                milestone = milestones.firstOrNull { it.id == task.milestoneId }?.title,
                words = words,
                onOpen = { onOpenTask(task.id) },
                onLongPress = { onLongPress(task.id) },
            )
        }
        if (lane.status == "done" && !all && lane.tasks.size > BOARD_DONE_LIMIT) {
            Text(
                stringResource(R.string.board_show_all, lane.tasks.size),
                style = type.meta,
                color = tokens.accentInk,
                modifier = Modifier.clickable { all = true }.padding(horizontal = 4.dp, vertical = 8.dp),
            )
        }
    }
}

/**
 * A card: the title (three lines at most), then one meta line: the author's avatar, the blocked mark,
 * the P0 / P1 pill, the branch in mono and at the right the due label, the milestone or the age.
 * Done cards sit on `bg.hover` with the title in `text.secondary`.
 */
@Composable
fun BoardCard(
    task: Task,
    agent: String,
    milestone: String?,
    words: Words,
    onOpen: () -> Unit,
    onLongPress: () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val done = task.status == "done"
    val shape = RoundedCornerShape(12.dp)
    Box(Modifier.padding(bottom = 10.dp)) {
        LongPressRow(
            onClick = onOpen,
            onLongPress = onLongPress,
            modifier = Modifier
                .clip(shape)
                .background(if (done) tokens.bgHover else tokens.bg)
                .then(if (done) Modifier else Modifier.border(Dims.hairline, tokens.border, shape)),
        ) { row ->
            Column(row.padding(horizontal = 14.dp, vertical = 12.dp)) {
                Text(
                    task.title,
                    style = if (done) type.body else type.bodyStrong,
                    color = if (done) tokens.textSecondary else tokens.text,
                    maxLines = 3,
                    overflow = TextOverflow.Ellipsis,
                )
                Row(
                    Modifier.fillMaxWidth().padding(top = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Avatar(rowAvatar(task, agent, words), size = 14.dp)
                    if (task.status == "blocked") StatusGlyph("blocked", size = 14.dp)
                    PriorityPill(task.priority)
                    if (task.branch.isNotEmpty()) BranchPill(task.branch, Modifier.weight(1f, fill = false))
                    Box(Modifier.weight(1f))
                    val right = cardRight(task, milestone, words)
                    if (right.isNotEmpty()) {
                        Text(
                            right,
                            style = type.meta,
                            color = if (task.status == "blocked") tokens.stale else tokens.textSecondary,
                            maxLines = 1,
                        )
                    }
                }
            }
        }
    }
}

/** What a card says at the right: when it is due, else its milestone, else (an agent's) how old it is. */
internal fun cardRight(task: Task, milestone: String?, words: Words): String = when {
    task.status == "done" -> words.ago(task.completedAt ?: task.updatedAt)
    words.due(task).isNotEmpty() -> words.due(task)
    !milestone.isNullOrEmpty() -> milestone
    task.source == "agent" -> words.ago(task.updatedAt)
    else -> ""
}

@Composable
private fun PriorityPill(priority: String) {
    if (priority != "P0" && priority != "P1") return
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val urgent = priority == "P0"
    Box(
        Modifier
            .height(20.dp)
            .background(if (urgent) tokens.statusBlocked.copy(alpha = 0.12f) else tokens.accentSoft, RoundedCornerShape(6.dp))
            .padding(horizontal = 7.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(priority, style = type.badge, color = if (urgent) tokens.statusBlocked else tokens.accentInk)
    }
}

@Composable
private fun BranchPill(branch: String, modifier: Modifier = Modifier) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Box(
        modifier
            .height(20.dp)
            .background(tokens.bgSubtle, RoundedCornerShape(6.dp))
            .padding(horizontal = 7.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            branch,
            style = type.mono.copy(fontSize = 12.sp),
            color = tokens.textSecondary,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}
