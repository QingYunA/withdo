package xyz.sriai.adhdtodo.ui

import android.content.Context
import android.content.Intent
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.CoreApi
import xyz.sriai.adhdtodo.data.SummaryPage
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.ZoneId
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.util.Locale

/** What the Summary route shows and asks for: the period, the page read for it, and the replay count. */
@Stable
class SummaryHolder {
    var week by mutableStateOf(false)
    var back by mutableLongStateOf(0L)
    var page by mutableStateOf<SummaryPage?>(null)
    var replay by mutableIntStateOf(0)

    /** The page opens on today and plays the scene again. */
    fun open() {
        week = false
        back = 0
        page = null
        replay++
    }

    fun select(week: Boolean) {
        if (this.week == week && back == 0L) return
        this.week = week
        back = 0
    }

    fun earlier() { back++ }

    fun later() { if (back > 0) back-- }
}

/** Reads the page of the period whenever it changes or the data does; failed reads leave the last page. */
@Composable
fun SummaryLoader(holder: SummaryHolder, api: CoreApi, active: Boolean, revision: Int, tick: Int) {
    LaunchedEffect(active, holder.week, holder.back, revision, tick) {
        if (!active) return@LaunchedEffect
        val loaded = runCatching { withContext(Dispatchers.IO) { api.summary(holder.week, holder.back) } }.getOrNull()
        if (loaded != null) holder.page = loaded
    }
}

/** Sends the Markdown of the period through the system share sheet, in the app's language. */
suspend fun shareSummary(context: Context, api: CoreApi, week: Boolean, back: Long) {
    val zh = context.resources.configuration.locales[0].language == "zh"
    val text = runCatching {
        withContext(Dispatchers.IO) { api.summaryMarkdown(week, back, if (zh) "zh-Hans" else "en") }
    }.getOrNull() ?: return
    val send = Intent(Intent.ACTION_SEND).setType("text/plain").putExtra(Intent.EXTRA_TEXT, text)
    context.startActivity(Intent.createChooser(send, null).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
}

/** `Mon, Oct 19` for a day (`10月19日 周一`), `Oct 19 – 25` for a week, `Oct 30 – Nov 5` across a month. */
fun summaryPeriodTitle(from: String, to: String, week: Boolean, locale: Locale): String {
    val zh = locale.language == "zh"
    val start = LocalDate.parse(from)
    if (!week) return DateTimeFormatter.ofPattern(if (zh) "M月d日 EEE" else "EEE, MMM d", locale).format(start)
    val end = LocalDate.parse(to)
    val month = DateTimeFormatter.ofPattern(if (zh) "M月d日" else "MMM d", locale)
    val tail = when {
        start.month != end.month -> month.format(end)
        zh -> "${end.dayOfMonth}日"
        else -> "${end.dayOfMonth}"
    }
    return "${month.format(start)} – $tail"
}

/**
 * The Summary page (v43): the title with `share`, the `Today | This week` segment and `‹ date ›`, the
 * scene edge to edge, the assistant's bubble when a paragraph synced, and the ledger. It scrolls as one.
 * [frozenMs] fixes the scene's time, for the screenshots; [calm] defaults to the system's "remove animations".
 */
@Composable
fun SummaryScreen(
    holder: SummaryHolder,
    words: Words,
    onOpenTask: (Long) -> Unit,
    onShare: () -> Unit,
    zone: ZoneId = ZoneId.systemDefault(),
    now: ZonedDateTime = ZonedDateTime.now(zone),
    frozenMs: Int? = null,
    calm: Boolean? = null,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val context = LocalContext.current
    val locale = LocalConfiguration.current.locales[0]
    val page = holder.page?.takeIf { it.week == holder.week && it.back == holder.back }
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        Row(
            Modifier.fillMaxWidth().padding(start = 16.dp, end = 4.dp, top = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(stringResource(R.string.summary_title), style = type.title, color = tokens.text, modifier = Modifier.weight(1f))
            val share = stringResource(R.string.summary_share)
            Box(
                Modifier.size(48.dp).clickable(onClick = onShare).semantics { contentDescription = share; role = Role.Button },
                contentAlignment = Alignment.Center,
            ) { PixelIcon("share", size = 24.dp, tint = tokens.textSecondary) }
        }
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            PeriodSegment(week = holder.week, onSelect = holder::select)
            Spacer(Modifier.weight(1f))
            Paging(
                title = summaryPeriodTitle(page?.from ?: pageFrom(holder, now), page?.to ?: pageTo(holder, now), holder.week, locale),
                canGoLater = holder.back > 0,
                onEarlier = holder::earlier,
                onLater = holder::later,
            )
        }
        if (page == null) {
            Spacer(Modifier.height(160.dp))
            return@Column
        }
        val sky = remember(holder.week, holder.back, holder.replay) { skyOf(holder.week, holder.back, now.hour) }
        val plan = remember(page, sky) { scenePlanOf(page, sky) { words.avatar(it) } }
        val motionOff = calm ?: remember { reduceMotion(context) }
        val time = rememberSceneTime(plan, Triple(page.week, page.back, holder.replay), motionOff, frozenMs)
        Spacer(Modifier.height(6.dp))
        SummaryScene(
            time = time,
            weekLabels = if (page.week) weekdayLabels(page.from, locale) else null,
            description = stringResource(R.string.summary_scene_a11y),
            onReplay = { holder.replay++ },
        )
        if (page.paragraph.isNotEmpty()) {
            AssistantBubble(
                period = bubblePeriod(page, locale),
                text = page.paragraph,
                modifier = Modifier.padding(start = 16.dp, end = 19.dp, top = 16.dp),
            )
        } else if (page.total == 0) {
            Text(
                emptyLine(page),
                style = type.body,
                color = tokens.textSecondary,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 20.dp),
            )
        }
        if (page.total > 0) {
            SummaryLedger(page, words, time, zone, onOpenTask, Modifier.padding(horizontal = 16.dp, vertical = 12.dp))
        }
        Spacer(Modifier.height(24.dp))
    }
}

private fun pageFrom(holder: SummaryHolder, now: ZonedDateTime): String = periodStart(holder, now).toString()

private fun pageTo(holder: SummaryHolder, now: ZonedDateTime): String =
    periodStart(holder, now).let { if (holder.week) it.plusDays(6) else it }.toString()

/** The first day of the period asked for, for the title while its page is on its way (weeks start on Monday). */
private fun periodStart(holder: SummaryHolder, now: ZonedDateTime): LocalDate {
    val today = now.toLocalDate()
    return if (holder.week) today.with(DayOfWeek.MONDAY).minusWeeks(holder.back) else today.minusDays(holder.back)
}

private fun weekdayLabels(from: String, locale: Locale): List<String> {
    val first = LocalDate.parse(from)
    val format = DateTimeFormatter.ofPattern("EEE", locale)
    return List(7) { format.format(first.plusDays(it.toLong())) }
}

@Composable
private fun emptyLine(page: SummaryPage): String = stringResource(
    when {
        page.week && page.over -> R.string.summary_empty_past_week
        page.week -> R.string.summary_empty_week
        page.over -> R.string.summary_empty_day
        else -> R.string.summary_empty_today
    },
)

@Composable
private fun bubblePeriod(page: SummaryPage, locale: Locale): String = when {
    !page.week && page.back == 0L -> stringResource(R.string.summary_segment_day)
    page.week && page.back == 0L -> stringResource(R.string.summary_segment_week)
    else -> summaryPeriodTitle(page.from, page.to, page.week, locale)
}

/** `Today | This week`: two cells in a bg.subtle well, the chosen one raised. */
@Composable
private fun PeriodSegment(week: Boolean, onSelect: (Boolean) -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        Modifier.background(tokens.bgSubtle, RoundedCornerShape(10.dp)).padding(2.dp),
        horizontalArrangement = Arrangement.spacedBy(0.dp),
    ) {
        for ((label, isWeek) in listOf(stringResource(R.string.summary_segment_day) to false, stringResource(R.string.summary_segment_week) to true)) {
            val on = week == isWeek
            Box(
                Modifier
                    .height(30.dp)
                    .then(if (on) Modifier.background(tokens.bg, RoundedCornerShape(8.dp)).border(0.5.dp, tokens.border, RoundedCornerShape(8.dp)) else Modifier)
                    .clickable(interactionSource = remember { MutableInteractionSource() }, indication = null) { onSelect(isWeek) }
                    .semantics { role = Role.Tab }
                    .padding(horizontal = 14.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    label,
                    style = type.meta,
                    color = if (on) tokens.text else tokens.textSecondary,
                    fontWeight = if (on) FontWeight.SemiBold else FontWeight.Normal,
                    maxLines = 1,
                )
            }
        }
    }
}

/** `‹ date ›`: back a day or a week; the forward arrow stops at today. */
@Composable
private fun Paging(title: String, canGoLater: Boolean, onEarlier: () -> Unit, onLater: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(verticalAlignment = Alignment.CenterVertically) {
        val previous = stringResource(R.string.summary_previous)
        val next = stringResource(R.string.summary_next)
        Box(
            Modifier.size(36.dp).clickable(onClick = onEarlier).semantics { contentDescription = previous; role = Role.Button },
            contentAlignment = Alignment.Center,
        ) { PixelIcon("chevron-left", size = 16.dp, tint = tokens.textSecondary) }
        Text(title, style = type.meta, color = tokens.textSecondary, maxLines = 1)
        Box(
            Modifier
                .size(36.dp)
                .alpha(if (canGoLater) 1f else 0.35f)
                .clickable(enabled = canGoLater, onClick = onLater)
                .semantics { contentDescription = next; role = Role.Button },
            contentAlignment = Alignment.Center,
        ) { PixelIcon("chevron-right", size = 16.dp, tint = tokens.textSecondary) }
    }
}

/**
 * The assistant's avatar and its pixel speech bubble: a 2 dp square border and a hard 3 dp shadow, a
 * quiet header and the paragraph. The phone cannot rewrite it, so there is no Rewrite.
 */
@Composable
fun AssistantBubble(period: String, text: String, modifier: Modifier = Modifier) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        Avatar("assistant", size = 36.dp)
        Column(
            Modifier
                .weight(1f)
                .drawBehind { drawRect(tokens.text, Offset(3.dp.toPx(), 3.dp.toPx()), Size(size.width, size.height)) }
                .background(tokens.bg)
                .border(2.dp, tokens.text)
                .padding(horizontal = 12.dp, vertical = 10.dp),
            verticalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            Text(stringResource(R.string.summary_written, period), style = type.meta.copy(fontSize = 12.sp), color = tokens.textSecondary, maxLines = 1)
            Text(text, style = type.body.copy(fontSize = 14.sp, lineHeight = 21.sp), color = tokens.text)
        }
    }
}
