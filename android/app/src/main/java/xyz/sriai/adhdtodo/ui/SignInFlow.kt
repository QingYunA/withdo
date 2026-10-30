package xyz.sriai.adhdtodo.ui

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateValue
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.animation.core.VectorConverter
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.contentType
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.LinkAnnotation
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextLinkStyles
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.withLink
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.delay
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.GoogleMark
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens

/** What the signed-out card can ask the app to do, in the steps of v42. */
data class SignInActions(
    /** Sign in: the email and the password. */
    val onSignInWithPassword: (String, String) -> Unit = { _, _ -> },
    /** Create account: the email and the password; the app adds the language of the code mails. */
    val onSignUp: (String, String) -> Unit = { _, _ -> },
    /** The sixth digit of the Verify step. */
    val onSubmitCode: (String) -> Unit = {},
    val onResend: () -> Unit = {},
    /** Send code of Forgot password. */
    val onSendResetCode: (String) -> Unit = {},
    /** Set and sign in: the code and the new password. */
    val onResetPassword: (String, String) -> Unit = { _, _ -> },
    /** Goes to a step with the email kept; the app clears the failure line. */
    val onStep: (SignInStep) -> Unit = {},
    /** "Continue with Google": the app opens the browser and waits for the answer. */
    val onGoogle: () -> Unit = {},
    /** Cancel on the card that waits for the browser. */
    val onCancelGoogle: () -> Unit = {},
    /** The terms and the privacy policy open in the browser. */
    val onOpenUrl: (String) -> Unit = {},
)

/** The width the card's lines may take, so a line never runs the length of a wide phone. */
private val LineWidth = 340.dp

/**
 * The signed-out card (v42): Sign in | Sign up, the Verify step, and Forgot password. [stage] is set
 * while a request is on its way. [initialCode] and [initialPassword] are for the screenshots.
 */
@Composable
fun SignInFlowCard(
    flow: SignInFlow,
    stage: SignInStage?,
    actions: SignInActions,
    initialCode: String = "",
    initialPassword: String = "",
    googleAvailable: Boolean = false,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    var email by remember { mutableStateOf(flow.email) }
    var password by remember { mutableStateOf(initialPassword) }
    var showPassword by remember { mutableStateOf(false) }
    // The email is kept across the steps; the password is not.
    LaunchedEffect(flow.email) { if (flow.email.isNotBlank()) email = flow.email }
    LaunchedEffect(flow.step) {
        password = ""
        showPassword = false
    }
    // A refused password or an unacceptable new one empties the field and takes the keyboard back.
    LaunchedEffect(flow.attempt) {
        if (flow.failure == CodeFailure.WRONG_PASSWORD || flow.failure == CodeFailure.WEAK_PASSWORD) password = ""
    }
    val form = FormState(
        email = email, onEmail = { email = it },
        password = password, onPassword = { password = it },
        showPassword = showPassword, onShowPassword = { showPassword = it },
    )
    Column(Modifier.fillMaxWidth()) {
        GroupCard {
            Column(
                modifier = Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 20.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                if (stage == SignInStage.GOOGLE) {
                    GoogleStep(actions)
                    return@Column
                }
                when (flow.step) {
                    SignInStep.SIGN_IN, SignInStep.SIGN_UP -> TabsStep(flow, stage, actions, form, googleAvailable)
                    SignInStep.VERIFY -> VerifyStep(flow, stage, actions, initialCode)
                    SignInStep.RESET_EMAIL -> ResetEmailStep(flow, stage, actions, form)
                    SignInStep.RESET_CODE -> ResetCodeStep(flow, stage, actions, form, initialCode)
                }
            }
        }
        if (stage != SignInStage.GOOGLE) {
            when {
                flow.step == SignInStep.VERIFY -> NoteCard(stringResource(R.string.sync_verify_note))
                flow.step.isTab() -> Text(
                    stringResource(R.string.sync_signin_note),
                    style = type.meta.copy(fontSize = 12.sp),
                    color = tokens.textTertiary,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                )
            }
        }
    }
}

/** The fields the steps share, kept by the card so the email survives a tab or a step. */
private class FormState(
    val email: String,
    val onEmail: (String) -> Unit,
    val password: String,
    val onPassword: (String) -> Unit,
    val showPassword: Boolean,
    val onShowPassword: (Boolean) -> Unit,
)

/** The quiet card under the Verify step. */
@Composable
private fun NoteCard(text: String) {
    val tokens = LocalTokens.current
    Box(Modifier.padding(top = 16.dp)) {
        GroupCard {
            Text(
                text,
                style = LocalAppType.current.meta.copy(fontSize = 12.sp),
                color = tokens.textSecondary,
                modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 12.dp),
            )
        }
    }
}

@Composable
private fun StepTitle(title: String) {
    Text(
        title,
        style = LocalAppType.current.input.copy(fontWeight = FontWeight.SemiBold),
        color = LocalTokens.current.text,
        textAlign = TextAlign.Center,
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
        modifier = modifier.widthIn(max = LineWidth),
    )
}

/** The label over a field. */
@Composable
private fun FieldLabel(text: String, modifier: Modifier = Modifier) {
    Text(
        text,
        style = LocalAppType.current.meta.copy(fontWeight = FontWeight.Medium),
        color = LocalTokens.current.textSecondary,
        modifier = modifier.padding(start = 2.dp),
    )
}

/** A failure or a hint line left-aligned under a field, in status.blocked unless [ink] says otherwise. */
@Composable
private fun FieldLine(text: String, ink: Color = LocalTokens.current.statusBlocked, modifier: Modifier = Modifier) {
    Text(
        text,
        style = LocalAppType.current.meta.copy(fontSize = 12.sp),
        color = ink,
        modifier = modifier.fillMaxWidth().padding(start = 2.dp, top = 5.dp),
    )
}

/** The words of a failure: its localized line, else the core's own sentence, else the generic one. */
@Composable
private fun failureWords(flow: SignInFlow): String? {
    val failure = flow.failure ?: return null
    return codeFailureText(failure)?.let { stringResource(it) }
        ?: flow.failureText
        ?: stringResource(R.string.sync_sign_in_failed)
}

/** The centered one-line failure under a button or the cells. */
@Composable
private fun FailureLine(flow: SignInFlow, busy: Boolean, skip: Set<CodeFailure> = emptySet()) {
    if (busy || flow.failure == null || flow.failure in skip) return
    Text(
        failureWords(flow) ?: return,
        style = LocalAppType.current.meta,
        color = LocalTokens.current.statusBlocked,
        textAlign = TextAlign.Center,
        modifier = Modifier.padding(top = 10.dp).widthIn(max = LineWidth),
    )
}

/** The pixel busy indicator and what is being waited for. */
@Composable
private fun BusyLine(stage: SignInStage, step: SignInStep) {
    val tokens = LocalTokens.current
    val label = when (stage) {
        SignInStage.SENDING -> stringResource(R.string.sync_sending_code)
        SignInStage.CREATING -> stringResource(R.string.sync_creating_account)
        SignInStage.DOWNLOADING -> stringResource(R.string.sync_signing_in_downloading)
        SignInStage.GOOGLE -> stringResource(R.string.sync_google_title)
        SignInStage.CHECKING -> if (step == SignInStep.VERIFY || step == SignInStep.RESET_CODE) {
            stringResource(R.string.sync_checking_code)
        } else {
            stringResource(R.string.sync_signing_in_checking)
        }
    }
    Row(
        modifier = Modifier.padding(top = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        PixelSpinner(size = 14.dp, color = tokens.textSecondary)
        Text(label, style = LocalAppType.current.meta, color = tokens.textSecondary)
    }
}

/** The Sign in | Sign up switch: two equal segments in a quiet well, the chosen one raised. */
@Composable
private fun AuthTabs(signUp: Boolean, enabled: Boolean, onSelect: (SignInStep) -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val label = stringResource(R.string.sync_tabs)
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .background(tokens.bgSubtle)
            .padding(3.dp)
            .selectableGroup()
            .semantics { contentDescription = label },
    ) {
        for ((step, name) in listOf(
            SignInStep.SIGN_IN to stringResource(R.string.sync_sign_in_short),
            SignInStep.SIGN_UP to stringResource(R.string.sync_sign_up_short),
        )) {
            val selected = (step == SignInStep.SIGN_UP) == signUp
            Box(
                modifier = Modifier
                    .weight(1f)
                    .height(34.dp)
                    .clip(RoundedCornerShape(6.dp))
                    .then(
                        if (selected) {
                            Modifier.background(tokens.bg).border(Dims.hairline, tokens.border, RoundedCornerShape(6.dp))
                        } else {
                            Modifier
                        },
                    )
                    .selectable(selected = selected, enabled = enabled, role = Role.RadioButton) { onSelect(step) },
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    name,
                    style = if (selected) type.section else type.meta,
                    color = if (selected) tokens.text else tokens.textSecondary,
                )
            }
        }
    }
}

/** The Sign in and the Sign up tabs: the switch, Google, the email, and the form of the tab. */
@Composable
private fun TabsStep(flow: SignInFlow, stage: SignInStage?, actions: SignInActions, form: FormState, googleAvailable: Boolean) {
    val busy = stage != null
    val signUp = flow.step == SignInStep.SIGN_UP
    val focusManager = LocalFocusManager.current
    val emailFocus = remember { FocusRequester() }
    val passwordFocus = remember { FocusRequester() }
    var emailFocused by remember { mutableStateOf(false) }
    var passwordFocused by remember { mutableStateOf(false) }
    // The keyboard goes where the work starts: the email, or the password when the email is there.
    LaunchedEffect(Unit) { if (form.email.isBlank()) emailFocus.requestFocus() }
    LaunchedEffect(flow.attempt) {
        if (flow.failure == CodeFailure.WRONG_PASSWORD) passwordFocus.requestFocus()
    }

    AuthTabs(signUp, enabled = !busy, onSelect = actions.onStep)
    if (googleAvailable) {
        OutlineButton(
            label = stringResource(R.string.sync_signin_google),
            enabled = !busy,
            onClick = actions.onGoogle,
            modifier = Modifier.padding(top = 14.dp).fillMaxWidth(),
            mark = { GoogleMark(16.dp) },
        )
        OrRule(Modifier.padding(top = 4.dp))
    }

    val taken = flow.failure == CodeFailure.EMAIL_TAKEN && signUp
    val invalid = flow.failure == CodeFailure.INVALID_EMAIL && signUp
    val wrong = flow.failure == CodeFailure.WRONG_PASSWORD && !signUp
    val ready = if (signUp) {
        looksLikeEmail(form.email) && passwordLongEnough(form.password)
    } else {
        form.email.isNotBlank() && form.password.isNotEmpty()
    } && !busy
    val submit = {
        if (ready) {
            if (signUp) actions.onSignUp(form.email.trim(), form.password)
            else actions.onSignInWithPassword(form.email.trim(), form.password)
        }
    }

    FieldLabel(stringResource(R.string.sync_email), Modifier.fillMaxWidth().padding(top = if (googleAvailable) 6.dp else 14.dp, bottom = 4.dp))
    SignInField(
        value = form.email,
        onValueChange = form.onEmail,
        placeholder = stringResource(R.string.sync_signin_email_hint),
        enabled = !busy,
        focused = emailFocused,
        onFocused = { emailFocused = it },
        keyboardType = KeyboardType.Email,
        imeAction = ImeAction.Next,
        contentType = ContentType.EmailAddress,
        focus = emailFocus,
        onNext = { passwordFocus.requestFocus() },
        ground = LocalTokens.current.bgSubtle,
        error = taken || invalid,
        modifier = Modifier.alpha(if (busy) 0.5f else 1f),
    )
    if (taken) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(start = 2.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            Text(
                stringResource(R.string.sync_email_taken),
                style = LocalAppType.current.meta.copy(fontSize = 12.sp),
                color = LocalTokens.current.statusBlocked,
            )
            TextLink(stringResource(R.string.sync_go_sign_in), { actions.onStep(SignInStep.SIGN_IN) })
        }
    } else if (invalid) {
        FieldLine(stringResource(R.string.sync_invalid_email))
    }

    Row(
        modifier = Modifier.fillMaxWidth().padding(top = 10.dp, bottom = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        FieldLabel(stringResource(R.string.sync_password))
        Spacer(Modifier.weight(1f))
        if (!signUp && !busy) {
            TextLink(stringResource(R.string.sync_forgot_password), { actions.onStep(SignInStep.RESET_EMAIL) })
        }
    }
    PasswordField(
        form = form,
        enabled = !busy,
        focus = passwordFocus,
        focused = passwordFocused,
        onFocused = { passwordFocused = it },
        error = wrong || flow.failure == CodeFailure.WEAK_PASSWORD,
        imeAction = ImeAction.Go,
        onDone = { submit() },
    )
    if (signUp) {
        StrengthBar(form.password)
        FieldLine(
            stringResource(R.string.sync_password_min),
            if (flow.failure == CodeFailure.WEAK_PASSWORD) LocalTokens.current.statusBlocked else LocalTokens.current.textSecondary,
        )
    } else if (wrong) {
        FieldLine(stringResource(R.string.sync_wrong_password))
    }
    PrimaryButton(
        label = stringResource(if (signUp) R.string.sync_create_account else R.string.sync_sign_in_short),
        enabled = ready,
        onClick = { submit() },
        modifier = Modifier.padding(top = 14.dp).fillMaxWidth(),
    )
    if (stage != null) BusyLine(stage, flow.step)
    FailureLine(
        flow, busy,
        skip = setOf(CodeFailure.EMAIL_TAKEN, CodeFailure.INVALID_EMAIL, CodeFailure.WEAK_PASSWORD, CodeFailure.WRONG_PASSWORD),
    )
    if (!signUp) {
        Row(
            modifier = Modifier.padding(top = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            Text(stringResource(R.string.sync_no_account), style = LocalAppType.current.meta, color = LocalTokens.current.textSecondary)
            if (!busy) TextLink(stringResource(R.string.sync_sign_up_short), { actions.onStep(SignInStep.SIGN_UP) })
        }
    } else {
        TermsLine(actions.onOpenUrl)
    }
}

/** The password field with a quiet Show at its right end. */
@Composable
private fun PasswordField(
    form: FormState,
    enabled: Boolean,
    focus: FocusRequester,
    focused: Boolean,
    onFocused: (Boolean) -> Unit,
    error: Boolean,
    imeAction: ImeAction,
    onDone: () -> Unit,
    placeholder: String = "",
) {
    val tokens = LocalTokens.current
    SignInField(
        value = form.password,
        onValueChange = form.onPassword,
        placeholder = placeholder,
        enabled = enabled,
        focused = focused,
        onFocused = onFocused,
        keyboardType = KeyboardType.Password,
        imeAction = imeAction,
        contentType = ContentType.Password,
        focus = focus,
        secret = !form.showPassword,
        onDone = onDone,
        ground = tokens.bgSubtle,
        error = error,
        modifier = Modifier.alpha(if (enabled) 1f else 0.5f),
        trailing = {
            Text(
                stringResource(if (form.showPassword) R.string.sync_hide_password else R.string.sync_show_password),
                style = LocalAppType.current.meta.copy(fontSize = 12.sp),
                color = tokens.textTertiary,
                modifier = Modifier
                    .clip(RoundedCornerShape(6.dp))
                    .clickable(enabled = enabled) { form.onShowPassword(!form.showPassword) }
                    .padding(horizontal = 6.dp),
            )
        },
    )
}

/** Four steps in the done green under a new password: the length and the kinds of characters. */
@Composable
private fun StrengthBar(password: String) {
    val tokens = LocalTokens.current
    val score = passwordStrength(password)
    Row(
        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(3.dp),
    ) {
        for (i in 0 until STRENGTH_STEPS) {
            Box(
                Modifier
                    .weight(1f)
                    .height(3.dp)
                    .background(if (i < score) tokens.statusDone else tokens.border),
            )
        }
    }
}

/** "By creating an account you agree to the Terms of Service and the Privacy Policy", links inline. */
@Composable
private fun TermsLine(onOpenUrl: (String) -> Unit) {
    val tokens = LocalTokens.current
    val tos = stringResource(R.string.sync_terms_tos)
    val privacy = stringResource(R.string.sync_privacy_policy)
    val full = stringResource(R.string.sync_terms, tos, privacy)
    val links = TextLinkStyles(SpanStyle(color = tokens.accentInk))
    val text = buildAnnotatedString {
        val a = full.indexOf(tos)
        val b = full.indexOf(privacy)
        if (a < 0 || b < 0 || a > b) {
            append(full)
            return@buildAnnotatedString
        }
        append(full.substring(0, a))
        withLink(LinkAnnotation.Clickable("terms", links) { onOpenUrl(TERMS_URL) }) { append(tos) }
        append(full.substring(a + tos.length, b))
        withLink(LinkAnnotation.Clickable("privacy", links) { onOpenUrl(PRIVACY_URL) }) { append(privacy) }
        append(full.substring(b + privacy.length))
    }
    Text(
        text,
        style = LocalAppType.current.meta.copy(fontSize = 11.5.sp),
        color = tokens.textTertiary,
        textAlign = TextAlign.Center,
        modifier = Modifier.padding(top = 12.dp).widthIn(max = LineWidth),
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
            .border(Dims.hairline, tokens.border, shape)
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 11.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp, Alignment.CenterHorizontally),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        mark?.invoke()
        Text(label, style = LocalAppType.current.bodyStrong, color = tokens.text)
    }
}

/** The 12 sp "or use your email" between two hairlines, under the Google button. */
@Composable
private fun OrRule(modifier: Modifier = Modifier) {
    val tokens = LocalTokens.current
    Row(
        modifier = modifier.fillMaxWidth().padding(vertical = 6.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(Modifier.weight(1f).height(Dims.hairline).background(tokens.border))
        Text(
            stringResource(R.string.sync_signin_or),
            style = LocalAppType.current.meta.copy(fontSize = 12.sp),
            color = tokens.textTertiary,
        )
        Box(Modifier.weight(1f).height(Dims.hairline).background(tokens.border))
    }
}

/** The card while the browser shows Google's consent page: the G, what to do there, and Cancel. */
@Composable
private fun GoogleStep(actions: SignInActions) {
    GoogleMark(36.dp)
    Spacer(Modifier.height(4.dp))
    StepTitle(stringResource(R.string.sync_google_title))
    MetaLine(stringResource(R.string.sync_google_body), Modifier.padding(top = 4.dp))
    OutlineButton(
        label = stringResource(R.string.sync_google_cancel),
        enabled = true,
        onClick = actions.onCancelGoogle,
        modifier = Modifier.padding(top = 14.dp).fillMaxWidth(),
    )
}

/** The "sent to <email>" line, the address in text. */
@Composable
private fun SentLine(resource: Int, email: String) {
    val tokens = LocalTokens.current
    val sent = stringResource(resource, email)
    MetaLine(
        buildAnnotatedString {
            val at = sent.indexOf(email)
            if (email.isEmpty() || at < 0) {
                append(sent)
            } else {
                append(sent.substring(0, at))
                withStyle(SpanStyle(color = tokens.text, fontWeight = FontWeight.SemiBold)) { append(email) }
                append(sent.substring(at + email.length))
            }
        },
        Modifier.padding(top = 4.dp),
    )
}

/** The resend countdown, then the Resend code link, under the cells. */
@Composable
private fun ResendLine(flow: SignInFlow, actions: SignInActions) {
    val tokens = LocalTokens.current
    var left by remember(flow.resendIn, flow.attempt) { mutableIntStateOf(flow.resendIn) }
    LaunchedEffect(flow.resendIn, flow.attempt) {
        while (left > 0) {
            delay(1000)
            left -= 1
        }
    }
    if (left > 0) {
        if (flow.failure == null) {
            Text(
                stringResource(R.string.sync_code_resend_in, resendClock(left)),
                style = LocalAppType.current.meta.copy(fontSize = 12.sp),
                color = tokens.textTertiary,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(top = 10.dp),
            )
        }
    } else {
        TextLink(stringResource(R.string.sync_code_resend), actions.onResend)
    }
}

/** Verify your email: the six cells, and the way back to Sign up with the address kept. */
@Composable
private fun VerifyStep(flow: SignInFlow, stage: SignInStage?, actions: SignInActions, initialCode: String) {
    val busy = stage != null
    StepTitle(stringResource(R.string.sync_verify_title))
    SentLine(R.string.sync_verify_sent, flow.email)
    MetaLine(stringResource(R.string.sync_verify_auto))
    TextLink(stringResource(R.string.sync_code_other_email), { actions.onStep(SignInStep.SIGN_UP) })
    CodeEntry(
        attempt = flow.attempt,
        enabled = !busy,
        initial = initialCode,
        onComplete = actions.onSubmitCode,
        modifier = Modifier.padding(top = 6.dp),
    )
    if (stage != null) BusyLine(stage, SignInStep.VERIFY)
    FailureLine(flow, busy)
    if (!busy) ResendLine(flow, actions)
}

/** Forgot password, first step: the email, kept from Sign in, and Send code. */
@Composable
private fun ResetEmailStep(flow: SignInFlow, stage: SignInStage?, actions: SignInActions, form: FormState) {
    val busy = stage != null
    var focused by remember { mutableStateOf(false) }
    val focus = remember { FocusRequester() }
    val ready = looksLikeEmail(form.email) && !busy
    LaunchedEffect(Unit) { focus.requestFocus() }
    StepTitle(stringResource(R.string.sync_reset_title))
    MetaLine(stringResource(R.string.sync_reset_email_body), Modifier.padding(top = 4.dp))
    FieldLabel(stringResource(R.string.sync_email), Modifier.fillMaxWidth().padding(top = 14.dp, bottom = 4.dp))
    SignInField(
        value = form.email,
        onValueChange = form.onEmail,
        placeholder = stringResource(R.string.sync_signin_email_hint),
        enabled = !busy,
        focused = focused,
        onFocused = { focused = it },
        keyboardType = KeyboardType.Email,
        imeAction = ImeAction.Go,
        contentType = ContentType.EmailAddress,
        focus = focus,
        onDone = { if (ready) actions.onSendResetCode(form.email.trim()) },
        ground = LocalTokens.current.bgSubtle,
        modifier = Modifier.alpha(if (busy) 0.5f else 1f),
    )
    PrimaryButton(
        label = stringResource(R.string.sync_send_code),
        enabled = ready,
        onClick = { actions.onSendResetCode(form.email.trim()) },
        modifier = Modifier.padding(top = 14.dp).fillMaxWidth(),
    )
    if (stage != null) BusyLine(stage, SignInStep.RESET_EMAIL)
    FailureLine(flow, busy)
    BackToSignIn(busy, actions)
}

@Composable
private fun BackToSignIn(busy: Boolean, actions: SignInActions) {
    if (!busy) TextLink(stringResource(R.string.sync_reset_back), { actions.onStep(SignInStep.SIGN_IN) })
}

/** Forgot password, second step: the six cells, the new password with its bar, Set and sign in. */
@Composable
private fun ResetCodeStep(flow: SignInFlow, stage: SignInStage?, actions: SignInActions, form: FormState, initialCode: String) {
    val busy = stage != null
    var code by remember(flow.attempt) { mutableStateOf(codeDigits(initialCode)) }
    var focused by remember { mutableStateOf(false) }
    val passwordFocus = remember { FocusRequester() }
    val ready = code.length == CODE_LENGTH && passwordLongEnough(form.password) && !busy
    val submit = { if (ready) actions.onResetPassword(code, form.password) }
    LaunchedEffect(flow.attempt) {
        if (flow.failure == CodeFailure.WEAK_PASSWORD) passwordFocus.requestFocus()
    }
    StepTitle(stringResource(R.string.sync_reset_title))
    SentLine(R.string.sync_reset_sent, flow.email)
    TextLink(stringResource(R.string.sync_code_other_email), { actions.onStep(SignInStep.RESET_EMAIL) })
    CodeEntry(
        attempt = flow.attempt,
        enabled = !busy,
        initial = initialCode,
        onComplete = {},
        onChange = {
            code = it
            if (it.length == CODE_LENGTH) passwordFocus.requestFocus()
        },
        modifier = Modifier.padding(top = 6.dp),
    )
    FieldLabel(stringResource(R.string.sync_new_password), Modifier.fillMaxWidth().padding(top = 14.dp, bottom = 4.dp))
    PasswordField(
        form = form,
        enabled = !busy,
        focus = passwordFocus,
        focused = focused,
        onFocused = { focused = it },
        error = flow.failure == CodeFailure.WEAK_PASSWORD,
        imeAction = ImeAction.Go,
        onDone = { submit() },
    )
    StrengthBar(form.password)
    FieldLine(
        stringResource(R.string.sync_password_min),
        if (flow.failure == CodeFailure.WEAK_PASSWORD) LocalTokens.current.statusBlocked else LocalTokens.current.textSecondary,
    )
    PrimaryButton(
        label = stringResource(R.string.sync_reset_done),
        enabled = ready,
        onClick = { submit() },
        modifier = Modifier.padding(top = 14.dp).fillMaxWidth(),
    )
    if (stage != null) BusyLine(stage, SignInStep.RESET_CODE)
    FailureLine(flow, busy, skip = setOf(CodeFailure.WEAK_PASSWORD))
    if (!busy) ResendLine(flow, actions)
    BackToSignIn(busy, actions)
}

/**
 * Six cells drawn from one real text field: paste and the system's one-time-code autofill fill
 * it as they would any field, and the sixth digit sends the code. [attempt] empties the cells
 * and puts the focus on the first one after a failure. [onChange] hears every change of the digits.
 */
@Composable
private fun CodeEntry(
    attempt: Int,
    enabled: Boolean,
    initial: String,
    onComplete: (String) -> Unit,
    modifier: Modifier = Modifier,
    onChange: (String) -> Unit = {},
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
            onChange(digits)
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
                                .background(tokens.bgSubtle, RoundedCornerShape(10.dp))
                                .border(
                                    if (ring) 1.dp else Dims.hairline,
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
