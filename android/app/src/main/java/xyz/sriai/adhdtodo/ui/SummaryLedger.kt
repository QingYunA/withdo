package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.SummaryPage
import xyz.sriai.adhdtodo.data.SummaryPerson
import xyz.sriai.adhdtodo.data.SummaryTask
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelBitmap
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import java.time.ZoneId
import java.time.format.DateTimeFormatter

private const val LEDGER_LIGHT_MS = 420 // between two rows lighting up
private const val LEDGER_COUNT_MS = 60 // a step of a count running up

/** How far a row has come: 0 while it waits, else the count it shows. [at] is the scene's time, [end] its end. */
fun ledgerShown(at: Int, end: Int, row: Int, count: Int): Int {
    val age = at - end - row * LEDGER_LIGHT_MS
    return if (age < 0) -1 else minOf(count, age / LEDGER_COUNT_MS)
}

/**
 * The ledger: a row per person (avatar, name, a dotted leader, the pull requests merged, the count and a
 * chevron), the rows lighting up one by one once the scene has played; a tap opens the person's tasks
 * inline. The last row is the total.
 */
@Composable
fun SummaryLedger(
    page: SummaryPage,
    words: Words,
    time: SceneTime,
    zone: ZoneId,
    onOpenTask: (Long) -> Unit,
    modifier: Modifier = Modifier,
) {
    val open = remember(page.from, page.week) { mutableStateMapOf<String, Boolean>() }
    Column(modifier.fillMaxWidth()) {
        page.ledger.forEachIndexed { k, person ->
            val key = person.agent.ifEmpty { "you" }
            LedgerRow(person, k, words, time, open[key] == true, { open[key] = open[key] != true }, page.week, zone, onOpenTask)
        }
        TotalRow(page, page.ledger.size, time)
    }
}

@Composable
private fun LedgerRow(
    person: SummaryPerson,
    index: Int,
    words: Words,
    time: SceneTime,
    open: Boolean,
    onToggle: () -> Unit,
    week: Boolean,
    zone: ZoneId,
    onOpenTask: (Long) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val shown by remember(time, index, person.tasks.size) {
        derivedStateOf { ledgerShown(time.ledgerNow(), time.plan.end, index, person.tasks.size) }
    }
    val name = if (person.you) stringResource(R.string.done_by_you) else words.agent(person.agent)
    val describe = pluralStringResource(R.plurals.summary_row_a11y, person.tasks.size, name, person.tasks.size)
    Column(Modifier.alpha(if (shown < 0) 0f else 1f)) {
        Row(
            Modifier
                .fillMaxWidth()
                .height(48.dp)
                .clickable(onClick = onToggle)
                .semantics { contentDescription = describe; role = Role.Button }
                .padding(horizontal = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            PersonAvatar(person, words)
            Text(name, style = type.body, color = tokens.text, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Leader(Modifier.weight(1f))
            if (person.merged > 0) {
                Text(pluralStringResource(R.plurals.summary_merged, person.merged, person.merged), style = type.meta, color = tokens.textSecondary, maxLines = 1)
            }
            CountAndUnit(shown.coerceAtLeast(0), person.tasks.size)
            PixelIcon(if (open) "chevron-down" else "chevron-right", size = 16.dp, tint = tokens.textTertiary)
        }
        Hairline()
        if (open) {
            for (task in person.tasks) TaskLine(task, words, week, zone, onOpenTask)
        }
    }
}

@Composable
private fun TotalRow(page: SummaryPage, index: Int, time: SceneTime) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val shown by remember(time, index, page.total) {
        derivedStateOf { ledgerShown(time.ledgerNow(), time.plan.end, index, page.total) }
    }
    val label = stringResource(
        when {
            page.week && page.over -> R.string.summary_shipped_past_week
            page.week -> R.string.summary_shipped_week
            page.over -> R.string.summary_shipped_day
            else -> R.string.summary_shipped_today
        },
    )
    Row(
        Modifier.fillMaxWidth().height(48.dp).padding(horizontal = 4.dp).alpha(if (shown < 0) 0f else 1f),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(label, style = type.bodyStrong.copy(fontWeight = FontWeight.SemiBold), color = tokens.text, maxLines = 1)
        Leader(Modifier.weight(1f))
        CountAndUnit(shown.coerceAtLeast(0), page.total)
        Box(Modifier.size(16.dp))
    }
}

/** The count in a bold 17 sp, as wide as its final number so the row does not shift while it runs up, and its unit. */
@Composable
private fun CountAndUnit(shown: Int, total: Int) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
        Text(
            shown.toString(),
            style = type.body.copy(fontSize = 17.sp, fontWeight = FontWeight.Bold),
            color = tokens.text,
            textAlign = TextAlign.End,
            maxLines = 1,
            modifier = Modifier.width((10 * total.toString().length + 4).dp),
        )
        Text(pluralStringResource(R.plurals.summary_unit, total), style = type.meta, color = tokens.textSecondary, maxLines = 1)
    }
}

/** The dotted line between a name and its count: 2 dp dots, 4 dp apart. */
@Composable
private fun Leader(modifier: Modifier) {
    val dot = LocalTokens.current.borderControl
    Box(
        modifier
            .padding(top = 6.dp)
            .height(2.dp)
            .drawBehind {
                val step = 4.dp.toPx()
                val side = 2.dp.toPx()
                var x = 0f
                while (x + side <= size.width) {
                    drawRect(dot, Offset(x, 0f), Size(side, size.height))
                    x += step
                }
            },
    )
}

@Composable
private fun Hairline() {
    val tokens = LocalTokens.current
    Box(Modifier.fillMaxWidth().height(0.5.dp).background(tokens.border))
}

/** You, an agent with its avatar, or a robot in the color its actor stands in. */
@Composable
private fun PersonAvatar(person: SummaryPerson, words: Words) {
    val context = LocalContext.current
    val file = if (person.you) "you" else words.avatar(person.agent)
    if (file == SceneActor.BOT) {
        val robot = remember(person.agent) { SceneSprites(context).avatar("bot", botTintOf(person.agent)) }
        if (robot != null) {
            PixelBitmap(robot, 18.dp)
            return
        }
    }
    Avatar(file, size = 18.dp)
}

/** A task under its person: the title, its pull request and last report, and when it was finished. */
@Composable
private fun TaskLine(task: SummaryTask, words: Words, week: Boolean, zone: ZoneId, onOpenTask: (Long) -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val locale = LocalConfiguration.current.locales[0]
    val quiet = listOfNotNull(
        task.project.ifEmpty { null },
        when {
            task.prNumber > 0 && task.merged -> stringResource(R.string.summary_row_merged, task.prNumber.toString())
            task.prNumber > 0 -> "PR #${task.prNumber}"
            else -> null
        },
        task.said.ifEmpty { null },
    ).joinToString(" · ")
    val clock = words.timeOfDay(task.at)
    val whenText = if (week) {
        val day = words.instant(task.at)?.atZone(zone)?.let { DateTimeFormatter.ofPattern("EEE", locale).format(it) }.orEmpty()
        "$day $clock".trim()
    } else {
        clock
    }
    Row(
        Modifier.fillMaxWidth().clickable { onOpenTask(task.id) }.padding(start = 30.dp, end = 4.dp, top = 8.dp, bottom = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Column(Modifier.weight(1f)) {
            Text(task.title, style = type.body, color = tokens.text, maxLines = 2, overflow = TextOverflow.Ellipsis)
            if (quiet.isNotEmpty()) Text(quiet, style = type.meta, color = tokens.textSecondary, maxLines = 2, overflow = TextOverflow.Ellipsis)
        }
        Text(whenText, style = type.meta, color = tokens.textSecondary, maxLines = 1)
    }
}
