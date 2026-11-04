package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Billing
import xyz.sriai.adhdtodo.data.ModelPhase
import xyz.sriai.adhdtodo.data.ModelState
import xyz.sriai.adhdtodo.data.PlanKey
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import java.time.Instant
import java.time.ZoneId

/*
 * What Withdo Pro draws (v45): the plan card of Settings > Assistant and the upgrade sheet.
 * BillingState.kt holds the state and the wording.
 */

/** The Assistant group's plan card shows: Pro is sold here, the plan is known, and an account is signed in. */
fun showsPlanCard(billing: Billing, model: ModelState): Boolean =
    billing.shown && (model.phase == ModelPhase.READY || model.phase == ModelPhase.USED_UP)

/**
 * The plan card (radius 16, a hairline, 16 dp inside): the assistant, `Free` or the `Pro` pill, the
 * quota line and bar; then Free's accent button, or Pro's renewal line and the quiet `Manage subscription`.
 */
@Composable
fun PlanCard(
    model: ModelState,
    billing: Billing,
    onUpgrade: () -> Unit,
    onManage: () -> Unit,
    now: Instant = Instant.now(),
    zone: ZoneId = ZoneId.systemDefault(),
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val res = LocalContext.current.resources
    val shape = RoundedCornerShape(16.dp)
    Column(
        Modifier
            .fillMaxWidth()
            .testTag("plan-card")
            .clip(shape)
            .background(tokens.bg)
            .border(Dims.hairline, tokens.border, shape)
            .padding(16.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Avatar("assistant", size = 28.dp)
            Text(
                stringResource(R.string.assistant_model_name),
                style = type.body.copy(fontSize = 16.sp, fontWeight = FontWeight.SemiBold),
                color = tokens.text,
                modifier = Modifier.weight(1f),
            )
            if (billing.pro) {
                ProPill()
            } else {
                Text(stringResource(R.string.billing_plan_free), style = type.meta, color = tokens.textSecondary, modifier = Modifier.testTag("plan-free"))
            }
        }
        Text(quotaLine(model), style = type.meta, color = tokens.textSecondary, modifier = Modifier.padding(top = 4.dp).testTag("plan-quota"))
        if (model.left >= 0 && model.limit > 0) QuotaBar(model.left, model.limit)
        if (billing.pro) {
            val line = renewalLine(res, billing, now, zone)
            if (line.isNotEmpty()) {
                Text(line, style = type.meta, color = tokens.textSecondary, modifier = Modifier.padding(top = 12.dp).testTag("plan-renewal"))
            }
            PlanButton(stringResource(R.string.billing_manage), accent = false, tag = "plan-manage", onClick = onManage)
        } else {
            val label = if (billing.trialUsed) R.string.billing_upgrade else R.string.billing_upgrade_trial
            PlanButton(stringResource(label), accent = true, tag = "plan-upgrade", onClick = onUpgrade)
        }
    }
}

/** `● Available · 14 / 20 left today` with the dot and the word in the done green, or why there is no model. */
@Composable
private fun quotaLine(model: ModelState): AnnotatedString {
    val tokens = LocalTokens.current
    if (model.phase != ModelPhase.READY) return AnnotatedString(modelLine(model))
    val available = "● " + stringResource(R.string.assistant_state_available)
    val left = if (model.left >= 0 && model.limit > 0) stringResource(R.string.billing_quota_left, model.left, model.limit) else ""
    return buildAnnotatedString {
        withStyle(SpanStyle(color = tokens.categoryInk("green"))) { append(available) }
        if (left.isNotEmpty()) append(" · $left")
    }
}

@Composable
private fun QuotaBar(left: Int, limit: Int) {
    val tokens = LocalTokens.current
    val fraction = (left.toFloat() / limit).coerceIn(0f, 1f)
    Box(Modifier.padding(top = 12.dp).fillMaxWidth().height(4.dp).background(tokens.bgSelected).testTag("plan-bar")) {
        Box(Modifier.fillMaxWidth(fraction).height(4.dp).background(tokens.statusDone))
    }
}

/** The `Pro` pill: the accent wash with accent text. */
@Composable
private fun ProPill() {
    val tokens = LocalTokens.current
    Text(
        stringResource(R.string.billing_plan_pro),
        style = LocalAppType.current.badge.copy(fontSize = 11.sp),
        color = tokens.accentInk,
        modifier = Modifier
            .testTag("plan-pro")
            .background(tokens.accentSoft, RoundedCornerShape(5.dp))
            .padding(horizontal = 6.dp, vertical = 1.dp),
    )
}

/** A 44 dp button of the card and the sheet: accent for the way forward, the quiet wash for the rest. */
@Composable
private fun PlanButton(label: String, accent: Boolean, tag: String, enabled: Boolean = true, onClick: () -> Unit) {
    val tokens = LocalTokens.current
    val shape = RoundedCornerShape(12.dp)
    Box(
        Modifier
            .padding(top = 14.dp)
            .fillMaxWidth()
            .height(44.dp)
            .testTag(tag)
            .clip(shape)
            .background(if (accent) tokens.accent else tokens.bgSubtle)
            .then(if (enabled) Modifier.clickable(onClick = onClick) else Modifier),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            label,
            style = LocalAppType.current.body.copy(fontWeight = FontWeight.SemiBold),
            color = if (accent) tokens.onAccent else tokens.text,
            maxLines = 1,
        )
    }
}

/**
 * The upgrade sheet over the page: the plans to choose from, then, once the browser has the
 * checkout page, a wait with a way to open it again. It closes by itself when the account is Pro.
 */
@Composable
fun UpgradeSheet(
    holder: BillingHolder,
    onStart: () -> Unit,
    onReopen: () -> Unit,
    onDismiss: () -> Unit,
) {
    SheetFrame(onDismiss) {
        Column(Modifier.fillMaxWidth().testTag("billing-sheet")) {
            if (holder.stage == SheetStage.WAITING) {
                WaitingBody(onReopen)
            } else {
                ChoosingBody(holder, onStart)
            }
        }
    }
}

@Composable
private fun ChoosingBody(holder: BillingHolder, onStart: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val res = LocalContext.current.resources
    val billing = holder.billing
    Column(Modifier.padding(horizontal = 16.dp)) {
        Text(
            stringResource(R.string.billing_sheet_title),
            style = type.title.copy(fontSize = 20.sp, lineHeight = 26.sp, fontWeight = FontWeight.Bold),
            color = tokens.text,
            modifier = Modifier.padding(top = 4.dp),
        )
        Text(stringResource(R.string.billing_sheet_line), style = type.body.copy(fontSize = 14.sp), color = tokens.textSecondary, modifier = Modifier.padding(top = 4.dp))
        Segment(holder.plan, onPick = { holder.plan = it })
        Row(Modifier.padding(top = 16.dp), verticalAlignment = Alignment.Bottom, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(
                priceOf(billing, holder.plan),
                style = type.title.copy(fontSize = 28.sp, lineHeight = 34.sp, fontWeight = FontWeight.Bold),
                color = tokens.text,
                modifier = Modifier.testTag("billing-price"),
            )
            Text(
                priceNote(res, billing, holder.plan),
                style = type.body.copy(fontSize = 14.sp),
                color = tokens.textSecondary,
                modifier = Modifier.padding(bottom = 4.dp).testTag("billing-price-note"),
            )
        }
        Column(Modifier.padding(top = 12.dp), verticalArrangement = Arrangement.spacedBy(9.dp)) {
            for (tick in listOf(R.string.billing_tick_model, R.string.billing_tick_devices, R.string.billing_tick_sync, R.string.billing_tick_cancel)) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    PixelIcon("check", size = 14.dp, tint = tokens.statusDone)
                    Text(stringResource(tick), style = type.body.copy(fontSize = 14.sp, lineHeight = 20.sp), color = tokens.text)
                }
            }
        }
        holder.failure?.let {
            Text(
                failureLine(res, it),
                style = type.meta,
                color = tokens.statusBlocked,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth().padding(top = 12.dp).testTag("billing-error"),
            )
        }
        val starting = holder.stage == SheetStage.STARTING
        val cta = when {
            starting -> R.string.billing_cta_starting
            billing.trialUsed -> R.string.billing_cta_subscribe
            else -> R.string.billing_cta_trial
        }
        PlanButton(stringResource(cta), accent = true, tag = "billing-cta", enabled = !starting, onClick = onStart)
        Text(
            stringResource(if (billing.trialUsed) R.string.billing_fine_subscribe else R.string.billing_fine_trial),
            style = type.meta.copy(fontSize = 12.sp, lineHeight = 17.sp),
            color = tokens.textSecondary,
            textAlign = TextAlign.Center,
            modifier = Modifier.fillMaxWidth().padding(top = 10.dp, bottom = 6.dp).testTag("billing-fine"),
        )
    }
}

/** `Monthly | Yearly Save 30%`, 36 dp tall in a pill of the quiet wash; the chosen one is lifted. */
@Composable
private fun Segment(plan: String, onPick: (String) -> Unit) {
    val tokens = LocalTokens.current
    Row(
        Modifier
            .padding(top = 14.dp)
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(tokens.bgSubtle)
            .padding(2.dp),
    ) {
        SegmentCell(stringResource(R.string.billing_monthly), selected = plan == PlanKey.MONTHLY, tag = "billing-seg-monthly") { onPick(PlanKey.MONTHLY) }
        SegmentCell(stringResource(R.string.billing_yearly), selected = plan == PlanKey.YEARLY, tag = "billing-seg-yearly", badge = stringResource(R.string.billing_save)) { onPick(PlanKey.YEARLY) }
    }
}

@Composable
private fun RowScope.SegmentCell(
    label: String,
    selected: Boolean,
    tag: String,
    badge: String? = null,
    onClick: () -> Unit,
) {
    val tokens = LocalTokens.current
    val shape = RoundedCornerShape(8.dp)
    Row(
        Modifier
            .weight(1f)
            .height(36.dp)
            .testTag(tag)
            .then(
                if (selected) {
                    Modifier
                        .shadow(1.dp, shape)
                        .background(tokens.bg, shape)
                        .border(Dims.hairline, tokens.border, shape)
                } else {
                    Modifier
                },
            )
            .clip(shape)
            .clickable(onClick = onClick),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp, Alignment.CenterHorizontally),
    ) {
        Text(
            label,
            style = LocalAppType.current.body.copy(fontSize = 14.sp, fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal),
            color = if (selected) tokens.text else tokens.textSecondary,
        )
        if (badge != null) {
            Text(
                badge,
                style = LocalAppType.current.badge.copy(fontSize = 11.sp),
                color = tokens.categoryInk("green"),
                modifier = Modifier.background(tokens.statusDone.copy(alpha = 0.14f), RoundedCornerShape(5.dp)).padding(horizontal = 5.dp, vertical = 1.dp),
            )
        }
    }
}

/** The sheet while the payment is made in the browser: the assistant, the words, and Reopen. */
@Composable
private fun WaitingBody(onReopen: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Column(Modifier.fillMaxWidth().padding(top = 20.dp, start = 16.dp, end = 16.dp, bottom = 6.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        Avatar("assistant", size = 40.dp)
        Text(
            stringResource(R.string.billing_wait_title),
            style = type.body.copy(fontSize = 17.sp, fontWeight = FontWeight.SemiBold),
            color = tokens.text,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(top = 10.dp).testTag("billing-wait-title"),
        )
        Text(
            stringResource(R.string.billing_wait_line),
            style = type.body.copy(fontSize = 14.sp, lineHeight = 20.sp),
            color = tokens.textSecondary,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(top = 10.dp),
        )
        PlanButton(stringResource(R.string.billing_reopen), accent = false, tag = "billing-reopen", onClick = onReopen)
    }
}
