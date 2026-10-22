package xyz.sriai.adhdtodo.ui

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateValue
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.animation.core.VectorConverter
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.autofill.ContentType
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.contentType
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.foundation.Canvas
import kotlinx.coroutines.delay
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.ui.theme.GoogleMark
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon

/** What the sign-in card can ask the app to do, in the steps of M13 P2. */
data class SignInActions(
    val onRequestCode: (String) -> Unit = {},
    val onSubmitCode: (String) -> Unit = {},
    val onResend: () -> Unit = {},
    val onForgotPassword: (String) -> Unit = {},
    val onSignInWithPassword: (String, String) -> Unit = { _, _ -> },
    /** Goes to a step with the email kept; the app clears the failure line. */
    val onStep: (SignInStep) -> Unit = {},
    /** "Continue with Google": the app opens the browser and waits for the answer. */
    val onGoogle: () -> Unit = {},
    /** Cancel on the card that waits for the browser. */
    val onCancelGoogle: () -> Unit = {},
)

/**
 * The signed-out card of a server that signs in with an emailed code: the email, the six digits,
 * or the password. [stage] is set while a request is on its way.
 */
@Composable
fun SignInFlowCard(
    flow: SignInFlow,
    stage: SignInStage?,
    actions: SignInActions,
    initialCode: String = "",
    googleAvailable: Boolean = false,
) {
    val busy = stage != null
    GroupCard {
        Column(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 28.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            if (stage == SignInStage.GOOGLE) {
                GoogleStep(actions)
                return@Column
            }
            when (flow.step) {
                SignInStep.EMAIL -> EmailStep(flow, stage, actions, googleAvailable)
                SignInStep.CODE -> CodeStep(flow, stage, actions, initialCode)
                SignInStep.PASSWORD -> PasswordStep(flow, stage, actions)
                SignInStep.RESET_SENT -> ResetSentStep(flow, busy, actions)
                SignInStep.LEGACY -> Unit
            }
        }
    }
}

@Composable
private fun StepTitle(icon: String, title: String) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    PixelIcon(icon, size = 36.dp, tint = tokens.accent)
    Text(
        title,
        style = type.input.copy(fontWeight = FontWeight.SemiBold),
        color = tokens.text,
        textAlign = TextAlign.Center,
        modifier = Modifier.padding(top = 4.dp),
    )
}

@Composable
private fun MetaLine(text: String, modifier: Modifier = Modifier) = MetaLine(AnnotatedString(text), modifier)

@Composable
private fun MetaLine(text: AnnotatedString, modifier: Modifier = Modifier) {
    Text(
        text,
        style = LocalAppType.current.meta,
        color = LocalTokens.current.textSecondary,
        textAlign = TextAlign.Center,
        modifier = modifier.widthIn(max = 340.dp),
    )
}

/** The one-line failure of a step, in status.blocked. */
@Composable
private fun FailureLine(flow: SignInFlow, busy: Boolean) {
    val failure = flow.failure
    if (busy || failure == null) return
    val text = codeFailureText(failure)?.let { stringResource(it) }
        ?: flow.failureText
        ?: stringResource(R.string.sync_sign_in_failed)
    Text(
        text,
        style = LocalAppType.current.meta,
        color = LocalTokens.current.statusBlocked,
        textAlign = TextAlign.Center,
        modifier = Modifier.padding(top = 2.dp).widthIn(max = 340.dp),
    )
}

/** The pixel busy indicator and what is being waited for. */
@Composable
private fun BusyLine(stage: SignInStage, step: SignInStep) {
    val tokens = LocalTokens.current
    val label = when (stage) {
        SignInStage.SENDING -> stringResource(R.string.sync_sending_code)
        SignInStage.DOWNLOADING -> stringResource(R.string.sync_signing_in_downloading)
        SignInStage.GOOGLE -> stringResource(R.string.sync_google_title)
        SignInStage.CHECKING -> if (step == SignInStep.CODE) {
            stringResource(R.string.sync_checking_code)
        } else {
            stringResource(R.string.sync_signing_in_checking)
        }
    }
    Row(
        modifier = Modifier.padding(top = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        PixelSpinner(size = 14.dp, color = tokens.textSecondary)
        Text(label, style = LocalAppType.current.meta, color = tokens.textSecondary)
    }
}

@Composable
private fun EmailStep(flow: SignInFlow, stage: SignInStage?, actions: SignInActions, googleAvailable: Boolean) {
    val tokens = LocalTokens.current
    val busy = stage != null
    var email by remember { mutableStateOf(flow.email) }
    var focused by remember { mutableStateOf(false) }
    val focus = remember { FocusRequester() }
    val ready = looksLikeEmail(email) && !busy
    LaunchedEffect(Unit) { focus.requestFocus() }

    StepTitle("user", stringResource(R.string.sync_signin_title))
    MetaLine(stringResource(R.string.sync_signin_meta))
    if (googleAvailable) {
        OutlineButton(
            label = stringResource(R.string.sync_signin_google),
            enabled = !busy,
            onClick = actions.onGoogle,
            modifier = Modifier.padding(top = 10.dp).fillMaxWidth(),
            mark = { GoogleMark(16.dp) },
        )
        OrRule(Modifier.padding(top = 6.dp))
    }
    SignInField(
        value = email,
        onValueChange = { email = it },
        placeholder = stringResource(R.string.sync_signin_email_hint),
        enabled = !busy,
        focused = focused,
        onFocused = { focused = it },
        keyboardType = KeyboardType.Email,
        imeAction = ImeAction.Go,
        contentType = ContentType.EmailAddress,
        focus = focus,
        onDone = { if (ready) actions.onRequestCode(email.trim()) },
        modifier = Modifier.padding(top = if (googleAvailable) 0.dp else 10.dp).alpha(if (busy) 0.5f else 1f),
    )
    PrimaryButton(
        label = stringResource(R.string.sync_signin_continue),
        enabled = ready,
        onClick = { actions.onRequestCode(email.trim()) },
        modifier = Modifier.padding(top = 6.dp).fillMaxWidth(),
    )
    if (stage != null) BusyLine(stage, SignInStep.EMAIL)
    FailureLine(flow, busy)
    Text(
        stringResource(R.string.sync_signin_note),
        style = LocalAppType.current.meta,
        color = tokens.textTertiary,
        textAlign = TextAlign.Center,
        modifier = Modifier.padding(top = 6.dp).widthIn(max = 340.dp),
    )
}

/** An outlined full-width button, as Continue with Google and Cancel are on the card. */
@Composable
private fun OutlineButton(
    label: String,
    enabled: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    mark: (@Composable () -> Unit)? = null,
) {
    val tokens = LocalTokens.current
    val shape = RoundedCornerShape(10.dp)
    Row(
        modifier = modifier
            .heightIn(min = 44.dp)
            .alpha(if (enabled) 1f else 0.45f)
            .clip(shape)
            .background(tokens.bg, shape)
            .border(1.dp, tokens.border, shape)
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 11.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp, Alignment.CenterHorizontally),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        mark?.invoke()
        Text(label, style = LocalAppType.current.bodyStrong, color = tokens.text)
    }
}

/** The 12 sp "or" between two hairlines, under the Google button. */
@Composable
private fun OrRule(modifier: Modifier = Modifier) {
    val tokens = LocalTokens.current
    Row(
        modifier = modifier.fillMaxWidth().padding(vertical = 6.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(Modifier.weight(1f).height(1.dp).background(tokens.border))
        Text(
            stringResource(R.string.sync_signin_or),
            style = LocalAppType.current.meta.copy(fontSize = 12.sp),
            color = tokens.textTertiary,
        )
        Box(Modifier.weight(1f).height(1.dp).background(tokens.border))
    }
}

/** The card while the browser shows Google's consent page: the G, what to do there, and Cancel. */
@Composable
private fun GoogleStep(actions: SignInActions) {
    val tokens = LocalTokens.current
    GoogleMark(36.dp)
    Text(
        stringResource(R.string.sync_google_title),
        style = LocalAppType.current.input.copy(fontWeight = FontWeight.SemiBold),
        color = tokens.text,
        textAlign = TextAlign.Center,
        modifier = Modifier.padding(top = 4.dp),
    )
    MetaLine(stringResource(R.string.sync_google_body))
    OutlineButton(
        label = stringResource(R.string.sync_google_cancel),
        enabled = true,
        onClick = actions.onCancelGoogle,
        modifier = Modifier.padding(top = 10.dp).fillMaxWidth(),
    )
}

@Composable
private fun CodeStep(flow: SignInFlow, stage: SignInStage?, actions: SignInActions, initialCode: String) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val busy = stage != null
    StepTitle("mail", stringResource(R.string.sync_code_title))
    val sent = stringResource(R.string.sync_code_sent, flow.email)
    MetaLine(
        buildAnnotatedString {
            val at = sent.indexOf(flow.email)
            if (flow.email.isEmpty() || at < 0) {
                append(sent)
            } else {
                append(sent.substring(0, at))
                withStyle(SpanStyle(color = tokens.text, fontWeight = FontWeight.SemiBold)) { append(flow.email) }
                append(sent.substring(at + flow.email.length))
            }
        },
    )
    TextLink(stringResource(R.string.sync_code_other_email), { actions.onStep(SignInStep.EMAIL) })
    CodeEntry(
        attempt = flow.attempt,
        enabled = !busy,
        initial = initialCode,
        onComplete = actions.onSubmitCode,
        modifier = Modifier.padding(top = 8.dp),
    )
    if (stage != null) BusyLine(stage, SignInStep.CODE)
    FailureLine(flow, busy)
    if (!busy) {
        var left by remember(flow.resendIn, flow.attempt) { mutableIntStateOf(flow.resendIn) }
        LaunchedEffect(flow.resendIn, flow.attempt) {
            while (left > 0) {
                delay(1000)
                left -= 1
            }
        }
        if (left > 0) {
            Text(
                stringResource(R.string.sync_code_resend_in, resendClock(left)),
                style = type.meta,
                color = tokens.textTertiary,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(top = 4.dp),
            )
        } else {
            TextLink(stringResource(R.string.sync_code_resend), actions.onResend)
        }
        TextLink(stringResource(R.string.sync_code_use_password), { actions.onStep(SignInStep.PASSWORD) })
    }
}

/**
 * Six cells drawn from one real text field: paste and the system's one-time-code autofill fill
 * it as they would any field, and the sixth digit sends the code. [attempt] empties the cells
 * and puts the focus on the first one after a failure.
 */
@Composable
private fun CodeEntry(
    attempt: Int,
    enabled: Boolean,
    initial: String,
    onComplete: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    var text by remember(attempt) { mutableStateOf(codeDigits(initial)) }
    var focused by remember { mutableStateOf(false) }
    val focus = remember { FocusRequester() }
    LaunchedEffect(attempt, enabled) { if (enabled) focus.requestFocus() }
    val label = stringResource(R.string.sync_code_field)

    BasicTextField(
        value = text,
        onValueChange = { raw ->
            val digits = codeDigits(raw)
            text = digits
            if (digits.length == CODE_LENGTH) onComplete(digits)
        },
        enabled = enabled,
        singleLine = true,
        textStyle = type.mono.copy(color = Color.Transparent),
        cursorBrush = SolidColor(Color.Transparent),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number, imeAction = ImeAction.Done),
        keyboardActions = KeyboardActions(onDone = { if (text.length == CODE_LENGTH) onComplete(text) }),
        modifier = modifier
            .focusRequester(focus)
            .onFocusChanged { focused = it.isFocused }
            .semantics {
                contentType = ContentType.SmsOtpCode
                contentDescription = label
            },
        decorationBox = { inner ->
            Box {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    val active = minOf(text.length, CODE_LENGTH - 1)
                    for (i in 0 until CODE_LENGTH) {
                        val ring = focused && enabled && i == active
                        Box(
                            modifier = Modifier
                                .size(width = 44.dp, height = 54.dp)
                                .alpha(if (enabled) 1f else 0.5f)
                                .background(tokens.bg, RoundedCornerShape(10.dp))
                                .border(
                                    if (ring) 2.dp else 1.dp,
                                    if (ring) tokens.accent else tokens.border,
                                    RoundedCornerShape(10.dp),
                                ),
                            contentAlignment = Alignment.Center,
                        ) {
                            Text(
                                text.getOrNull(i)?.toString() ?: "",
                                style = type.mono.copy(fontSize = 24.sp, fontWeight = FontWeight.SemiBold),
                                color = tokens.text,
                            )
                        }
                    }
                }
                Box(Modifier.matchParentSize().alpha(0f)) { inner() }
            }
        },
    )
}

@Composable
private fun PasswordStep(flow: SignInFlow, stage: SignInStage?, actions: SignInActions) {
    val tokens = LocalTokens.current
    val busy = stage != null
    var password by remember(flow.attempt) { mutableStateOf("") }
    var focused by remember { mutableStateOf(false) }
    val focus = remember { FocusRequester() }
    val ready = password.isNotEmpty() && !busy
    LaunchedEffect(flow.attempt) { focus.requestFocus() }

    StepTitle("user", stringResource(R.string.sync_password_title))
    MetaLine(flow.email)
    TextLink(stringResource(R.string.sync_code_other_email), { actions.onStep(SignInStep.EMAIL) })
    SignInField(
        value = password,
        onValueChange = { password = it },
        placeholder = stringResource(R.string.sync_password),
        enabled = !busy,
        focused = focused,
        onFocused = { focused = it },
        keyboardType = KeyboardType.Password,
        imeAction = ImeAction.Done,
        contentType = ContentType.Password,
        focus = focus,
        secret = true,
        onDone = { if (ready) actions.onSignInWithPassword(flow.email, password) },
        modifier = Modifier.padding(top = 4.dp).alpha(if (busy) 0.5f else 1f),
    )
    PrimaryButton(
        label = stringResource(R.string.sync_sign_in_short),
        enabled = ready,
        onClick = { actions.onSignInWithPassword(flow.email, password) },
        modifier = Modifier.padding(top = 6.dp).fillMaxWidth(),
    )
    if (stage != null) BusyLine(stage, SignInStep.PASSWORD)
    FailureLine(flow, busy)
    if (!busy) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            TextLink(stringResource(R.string.sync_forgot_password), { actions.onForgotPassword(flow.email) })
            Text("·", style = LocalAppType.current.meta, color = tokens.textTertiary)
            TextLink(stringResource(R.string.sync_code_use_code), { actions.onRequestCode(flow.email) })
        }
    }
}

@Composable
private fun ResetSentStep(flow: SignInFlow, busy: Boolean, actions: SignInActions) {
    StepTitle("mail", stringResource(R.string.sync_reset_title))
    MetaLine(stringResource(R.string.sync_reset_body, flow.email))
    PrimaryButton(
        label = stringResource(R.string.sync_reset_back),
        enabled = !busy,
        onClick = { actions.onStep(SignInStep.PASSWORD) },
        modifier = Modifier.padding(top = 10.dp).fillMaxWidth(),
    )
}

/**
 * The busy indicator of the pixel language (ui-spec, `theme.PixelSpinner`): eight 2 x 2 blocks
 * round a 12 x 12 grid; one is lit and steps clockwise every 100 ms, the two behind it at 60 %
 * and 35 %, the rest at 25 %.
 */
@Composable
fun PixelSpinner(size: androidx.compose.ui.unit.Dp, color: Color, modifier: Modifier = Modifier) {
    val transition = rememberInfiniteTransition(label = "pixel-spinner")
    val step by transition.animateValue(
        initialValue = 0,
        targetValue = SPINNER_BLOCKS,
        typeConverter = Int.VectorConverter,
        animationSpec = infiniteRepeatable(tween(SPINNER_BLOCKS * 100, easing = LinearEasing), RepeatMode.Restart),
        label = "pixel-spinner-step",
    )
    Canvas(modifier.size(size)) { drawSpinner(color, step % SPINNER_BLOCKS) }
}

private const val SPINNER_BLOCKS = 8
private val SPINNER_CELLS = listOf(5 to 1, 8 to 2, 9 to 5, 8 to 8, 5 to 9, 2 to 8, 1 to 5, 2 to 2)

private fun DrawScope.drawSpinner(color: Color, lit: Int) {
    val cell = this.size.minDimension / 12f
    for ((index, xy) in SPINNER_CELLS.withIndex()) {
        val behind = (lit - index + SPINNER_BLOCKS) % SPINNER_BLOCKS
        val alpha = when (behind) {
            0 -> 1f
            1 -> 0.6f
            2 -> 0.35f
            else -> 0.25f
        }
        drawRect(
            color = color.copy(alpha = alpha),
            topLeft = Offset(xy.first * cell, xy.second * cell),
            size = Size(2 * cell, 2 * cell),
        )
    }
}
