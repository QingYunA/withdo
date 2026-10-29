package xyz.sriai.adhdtodo.ui

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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentType
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Account
import xyz.sriai.adhdtodo.data.Device
import xyz.sriai.adhdtodo.data.SignInState
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.data.UpdateUi
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.GoogleMark
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import java.net.URI

/** The states the Sync screen draws, derived from the core or overridden in a debug build. */
enum class SyncPhase {
    SIGNED_OUT, SIGNING_IN, CHOOSE, OTHER_ACCOUNT, WRONG_ACCOUNT, SYNCED, UPLOADING, FAILED, OFFLINE,
}

/** Everything the Sync screen draws in one frame. */
data class SyncModel(
    val phase: SyncPhase,
    val status: SyncStatus,
    val account: Account = Account(),
    val devices: List<Device> = emptyList(),
    /** The counts of SignIn's choose answer, for the both-have-tasks sheet. */
    val choose: SignInState = SignInState(),
    /** The emailed-code sign-in's step. */
    val flow: SignInFlow = SignInFlow(),
    /** The six digits already typed, for the screenshots. */
    val typedCode: String = "",
    /** The password already typed, for the screenshots. */
    val typedPassword: String = "",
    /** The server lists a Google provider: the email step shows Continue with Google. */
    val googleAvailable: Boolean = false,
    /** The other_account answer, for the sheet that asks which account this phone belongs to. */
    val otherAccount: SignInState = SignInState(),
    /** The Google row of the signed-in card. */
    val googleLink: GoogleLinkRow = GoogleLinkRow(),
)

/** The account's display name, else its email, else the quiet "Account". */
@Composable
private fun accountName(account: Account): String {
    val name = account.display
    return name.ifBlank { stringResource(R.string.sync_account) }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SyncScreen(
    model: SyncModel,
    words: Words,
    serverEditing: Boolean,
    serverDraft: String,
    serverError: Boolean,
    serverSaving: Boolean,
    onSignIn: (String, String) -> Unit,
    signInActions: SignInActions = SignInActions(),
    onSignOut: () -> Unit,
    onSyncNow: () -> Unit,
    onRetry: () -> Unit,
    onMerge: () -> Unit,
    onReplace: () -> Unit,
    onCancelChoose: () -> Unit,
    onStartServerEdit: () -> Unit,
    onServerDraft: (String) -> Unit,
    onSaveServer: () -> Unit,
    onCancelServerEdit: () -> Unit,
    /** Choice 1 of the other-account sheet: sign out, then the email card with this address. */
    onUseOldAccount: (String) -> Unit = {},
    /** Choice 2 of the other-account sheet: link this phone to the account that signed in. */
    onMoveToAccount: () -> Unit = {},
    onGoogleLink: () -> Unit = {},
    onCancelGoogleLink: () -> Unit = {},
    update: UpdateUi = UpdateUi(),
    onCheckUpdate: () -> Unit = {},
    onUpdate: () -> Unit = {},
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    var sheetOpen by remember(model.phase) {
        mutableStateOf(model.phase == SyncPhase.CHOOSE || model.phase == SyncPhase.OTHER_ACCOUNT)
    }
    val signInStage = signInStageOf(model.phase == SyncPhase.SIGNING_IN, model.status.signInStage)

    Column(Modifier.fillMaxWidth()) {
        when (model.phase) {
            SyncPhase.SIGNED_OUT, SyncPhase.SIGNING_IN -> SignInFlowCard(
                flow = model.flow,
                stage = when {
                    model.flow.creating -> SignInStage.CREATING
                    model.flow.sending -> SignInStage.SENDING
                    else -> signInStage
                },
                actions = signInActions.copy(onSignInWithPassword = onSignIn),
                initialCode = model.typedCode,
                initialPassword = model.typedPassword,
                googleAvailable = model.googleAvailable,
            )
            SyncPhase.CHOOSE, SyncPhase.OTHER_ACCOUNT -> {
                GroupCard {
                    SyncAccountRow(model, words, onSignOut)
                }
            }
            SyncPhase.WRONG_ACCOUNT -> {
                GroupCard {
                    SyncAccountRow(model, words, onSignOut, showSignOut = false)
                    RowRule()
                    WrongAccountRow(onSignOut)
                }
            }
            SyncPhase.SYNCED, SyncPhase.UPLOADING, SyncPhase.FAILED, SyncPhase.OFFLINE -> {
                GroupCard {
                    SyncAccountRow(model, words, onSignOut)
                    if (model.googleLink.shown) {
                        RowRule()
                        GoogleLinkRowView(model.googleLink, onGoogleLink, onCancelGoogleLink)
                    }
                    RowRule()
                    ThisPhoneRow(model, words, onSyncNow, onRetry)
                }
                if (model.phase == SyncPhase.UPLOADING) {
                    Text(
                        stringResource(R.string.sync_upload_note),
                        style = type.meta,
                        color = tokens.textTertiary,
                        modifier = Modifier.padding(start = 4.dp, top = 10.dp),
                    )
                }
                if (model.devices.isNotEmpty()) {
                    GroupLabel(stringResource(R.string.sync_devices))
                    GroupCard {
                        val sorted = model.devices.sortedByDescending { it.isThis }
                        sorted.forEachIndexed { index, device ->
                            if (index > 0) RowRule()
                            DeviceRow(device, words)
                        }
                    }
                }
            }
        }
        Spacer(Modifier.height(16.dp))
        GroupCard { UpdateRow(update, onCheckUpdate, onUpdate) }
        ServerFooter(
            server = model.status.server.ifBlank { DEFAULT_SYNC_SERVER },
            editing = serverEditing,
            draft = serverDraft,
            error = serverError,
            saving = serverSaving,
            onStartEdit = onStartServerEdit,
            onDraft = onServerDraft,
            onSave = onSaveServer,
            onCancel = onCancelServerEdit,
        )
    }

    if (sheetOpen && model.phase == SyncPhase.OTHER_ACCOUNT) {
        ModalBottomSheet(
            onDismissRequest = {
                sheetOpen = false
                onCancelChoose()
            },
            sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
            containerColor = tokens.bg,
            contentColor = tokens.text,
        ) {
            OtherAccountSheet(
                answer = model.otherAccount,
                onCancel = {
                    sheetOpen = false
                    onCancelChoose()
                },
                onContinue = { useOld ->
                    sheetOpen = false
                    if (useOld) onUseOldAccount(model.otherAccount.previousEmail) else onMoveToAccount()
                },
            )
        }
    }

    if (sheetOpen && model.phase == SyncPhase.CHOOSE) {
        ModalBottomSheet(
            onDismissRequest = {
                sheetOpen = false
                onCancelChoose()
            },
            sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
            containerColor = tokens.bg,
            contentColor = tokens.text,
        ) {
            ChooseSheet(
                counts = model.choose,
                onCancel = {
                    sheetOpen = false
                    onCancelChoose()
                },
                onContinue = { merge ->
                    sheetOpen = false
                    if (merge) onMerge() else onReplace()
                },
            )
        }
    }
}

/** One 48 dp field of the sign-in card, drawn as the Server footer's editor. */
@Composable
internal fun SignInField(
    value: String,
    onValueChange: (String) -> Unit,
    placeholder: String,
    enabled: Boolean,
    focused: Boolean,
    onFocused: (Boolean) -> Unit,
    keyboardType: KeyboardType,
    imeAction: ImeAction,
    contentType: ContentType,
    focus: FocusRequester,
    secret: Boolean = false,
    onNext: () -> Unit = {},
    onDone: () -> Unit = {},
    modifier: Modifier = Modifier,
    /** The ground of the field; a field on a white card takes the tinted one. */
    ground: Color? = null,
    /** Outlines the field in status.blocked. */
    error: Boolean = false,
    /** Something at the right end, such as a password's Show. */
    trailing: (@Composable () -> Unit)? = null,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val edge = when {
        error -> tokens.statusBlocked
        focused -> tokens.accent
        else -> tokens.border
    }
    BasicTextField(
        value = value,
        onValueChange = onValueChange,
        enabled = enabled,
        textStyle = type.body.copy(color = tokens.text),
        cursorBrush = SolidColor(tokens.accent),
        singleLine = true,
        visualTransformation = if (secret) PasswordVisualTransformation() else androidx.compose.ui.text.input.VisualTransformation.None,
        keyboardOptions = KeyboardOptions(keyboardType = keyboardType, imeAction = imeAction),
        keyboardActions = KeyboardActions(onNext = { onNext() }, onDone = { onDone() }),
        modifier = modifier
            .fillMaxWidth()
            .height(48.dp)
            .focusRequester(focus)
            .onFocusChanged { onFocused(it.isFocused) }
            .semantics { this.contentType = contentType }
            .background(ground ?: tokens.bg, RoundedCornerShape(10.dp))
            .border(
                if (focused || error) 1.dp else Dims.hairline,
                edge,
                RoundedCornerShape(10.dp),
            )
            .padding(start = 14.dp, end = if (trailing != null) 8.dp else 14.dp, top = 14.dp, bottom = 14.dp),
        decorationBox = { inner ->
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.weight(1f), contentAlignment = Alignment.CenterStart) {
                    if (value.isEmpty()) {
                        Text(placeholder, style = type.body, color = tokens.textTertiary, maxLines = 1)
                    }
                    inner()
                }
                trailing?.invoke()
            }
        },
    )
}

@Composable
private fun SyncAccountRow(model: SyncModel, words: Words, onSignOut: () -> Unit, showSignOut: Boolean = true) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier.fillMaxWidth().heightIn(min = 56.dp).padding(horizontal = 14.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Column(Modifier.weight(1f)) {
            Text(
                accountName(model.account),
                style = type.body,
                color = tokens.text,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                when (model.devices.size) {
                    0 -> stringResource(R.string.sync_signed_in)
                    1 -> stringResource(R.string.sync_signed_in_one_device)
                    else -> stringResource(R.string.sync_signed_in_devices, model.devices.size)
                },
                style = type.meta,
                color = tokens.textSecondary,
            )
        }
        if (showSignOut) TextLink(stringResource(R.string.sync_sign_out), onSignOut)
    }
}

/** The phone's row in the wrong account: no sync line, no retry; the way out is Sign out. */
@Composable
private fun WrongAccountRow(onSignOut: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier.fillMaxWidth().heightIn(min = 56.dp).padding(horizontal = 14.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Column(Modifier.weight(1f)) {
            Text(stringResource(R.string.sync_this_phone), style = type.body, color = tokens.text)
            Text(stringResource(R.string.sync_wrong_account), style = type.meta, color = tokens.statusBlocked)
        }
        QuietButton(stringResource(R.string.sync_sign_out), onClick = onSignOut)
    }
}

/** The signed-in card's Google row: Link, a wait with Cancel, or Linked. */
@Composable
private fun GoogleLinkRowView(row: GoogleLinkRow, onLink: () -> Unit, onCancel: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier.fillMaxWidth().heightIn(min = 56.dp).padding(horizontal = 14.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        GoogleMark(20.dp)
        Column(Modifier.weight(1f)) {
            Text(stringResource(R.string.sync_google), style = type.body, color = tokens.text)
            row.message?.let {
                Text(stringResource(it), style = type.meta, color = tokens.statusBlocked)
            }
        }
        when {
            row.linked -> Text(stringResource(R.string.sync_google_linked), style = type.meta, color = tokens.statusDone)
            row.busy -> {
                PixelSpinner(size = 14.dp, color = tokens.textSecondary)
                TextLink(stringResource(R.string.sync_google_cancel), onCancel, color = tokens.textSecondary)
            }
            else -> QuietButton(stringResource(R.string.sync_google_link), onClick = onLink)
        }
    }
}

@Composable
private fun ThisPhoneRow(model: SyncModel, words: Words, onSyncNow: () -> Unit, onRetry: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier.fillMaxWidth().heightIn(min = 56.dp).padding(horizontal = 14.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Column(Modifier.weight(1f)) {
            Text(stringResource(R.string.sync_this_phone), style = type.body, color = tokens.text)
            when (model.phase) {
                SyncPhase.UPLOADING -> {
                    val total = model.status.seedTotal
                    val done = model.status.seedDone
                    Text(
                        stringResource(
                            R.string.sync_upload_progress,
                            groupThousands(done),
                            groupThousands(total),
                        ),
                        style = type.meta,
                        color = tokens.textSecondary,
                    )
                    PixelProgressBar(done = done, total = total, modifier = Modifier.padding(top = 6.dp))
                }
                SyncPhase.FAILED -> {
                    val reason = stringResource(syncFailureText(syncFailureOf(model.status.lastError)))
                    Text(
                        buildAnnotatedString {
                            withStyle(SpanStyle(color = tokens.statusBlocked)) {
                                append(stringResource(R.string.sync_failed, reason))
                            }
                            withStyle(SpanStyle(color = tokens.textSecondary)) {
                                append(" · " + words.retryLine(model.status.retryAt))
                            }
                        },
                        style = type.meta,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                    Text(
                        stringResource(R.string.sync_waiting, model.status.pending),
                        style = type.meta,
                        color = tokens.textSecondary,
                    )
                }
                SyncPhase.OFFLINE -> Text(
                    stringResource(R.string.sync_offline, model.status.pending),
                    style = type.meta,
                    color = tokens.textSecondary,
                )
                else -> Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(stringResource(R.string.sync_synced), style = type.meta, color = tokens.statusDone)
                    val age = model.status.lastSynced?.let { words.syncAge(it) }
                        ?: stringResource(R.string.time_just_now)
                    Text(
                        " · $age",
                        style = type.meta,
                        color = tokens.textSecondary,
                    )
                }
            }
        }
        when (model.phase) {
            SyncPhase.FAILED -> QuietButton(stringResource(R.string.sync_retry), onClick = onRetry)
            SyncPhase.SYNCED -> QuietButton(stringResource(R.string.sync_now), onClick = onSyncNow)
            else -> Unit
        }
    }
}

@Composable
private fun DeviceRow(device: Device, words: Words) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier.fillMaxWidth().heightIn(min = 44.dp).padding(horizontal = 14.dp, vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        PixelIcon(
            if (deviceUsesLaptop(device.name)) "laptop" else "smartphone",
            size = Dims.icon,
            tint = tokens.textTertiary,
        )
        Row(
            modifier = Modifier.weight(1f),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(
                device.name,
                style = type.body,
                color = tokens.text,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f, fill = false),
            )
            if (device.isThis) OutlineTag(stringResource(R.string.sync_this_phone_badge))
        }
        Text(
            words.deviceAge(device.lastSeenAt),
            style = type.meta,
            color = tokens.textSecondary,
            maxLines = 1,
        )
    }
}

@Composable
private fun ServerFooter(
    server: String,
    editing: Boolean,
    draft: String,
    error: Boolean,
    saving: Boolean,
    onStartEdit: () -> Unit,
    onDraft: (String) -> Unit,
    onSave: () -> Unit,
    onCancel: () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Column(Modifier.fillMaxWidth().padding(start = 4.dp, end = 4.dp, top = 14.dp)) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            val official = isOfficialSyncServer(server)
            if (editing || !official) {
                Text(stringResource(R.string.sync_server), style = type.meta, color = tokens.textTertiary)
            }
            if (editing) {
                BasicTextField(
                    value = draft,
                    onValueChange = onDraft,
                    textStyle = type.body.copy(color = tokens.text),
                    cursorBrush = SolidColor(tokens.accent),
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                    keyboardActions = KeyboardActions(onDone = { if (!saving) onSave() }),
                    modifier = Modifier
                        .weight(1f)
                        .height(36.dp)
                        .background(tokens.bgSubtle, RoundedCornerShape(6.dp))
                        .padding(horizontal = 8.dp, vertical = 8.dp),
                    decorationBox = { field ->
                        if (draft.isEmpty()) {
                            Text(
                                stringResource(R.string.sync_server_hint),
                                style = type.body,
                                color = tokens.textTertiary,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        field()
                    },
                )
                if (saving) {
                    Text(stringResource(R.string.sync_server_saving), style = type.meta, color = tokens.textTertiary)
                } else {
                    TextLink(stringResource(R.string.sync_change_save), onSave)
                    TextLink(stringResource(R.string.action_cancel), onCancel, color = tokens.textSecondary)
                }
            } else if (official) {
                TextLink(stringResource(R.string.sync_own_server), onStartEdit)
            } else {
                Text(
                    serverHost(server),
                    style = type.mono,
                    color = tokens.textTertiary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
                Text("·", style = type.meta, color = tokens.textTertiary)
                TextLink(stringResource(R.string.sync_change), onStartEdit)
            }
        }
        if (editing && error) {
            Text(
                stringResource(R.string.sync_server_invalid),
                style = type.meta,
                color = tokens.statusBlocked,
                modifier = Modifier.padding(top = 2.dp),
            )
        }
    }
}

@Composable
private fun ChooseSheet(
    counts: SignInState,
    onCancel: () -> Unit,
    onContinue: (Boolean) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    var merge by remember { mutableStateOf(true) }
    Column(
        modifier = Modifier.fillMaxWidth().padding(start = 20.dp, end = 20.dp, bottom = 24.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(
            stringResource(R.string.sync_both_title),
            style = type.input.copy(fontWeight = FontWeight.SemiBold),
            color = tokens.text,
        )
        Text(
            stringResource(
                R.string.sync_both_body,
                counts.accountDevices,
                counts.localTasks,
                counts.localProjects,
            ),
            style = type.meta,
            color = tokens.textSecondary,
        )
        ChoiceCard(
            selected = merge,
            title = stringResource(R.string.sync_merge),
            body = stringResource(R.string.sync_merge_body),
            recommended = true,
            onClick = { merge = true },
        )
        ChoiceCard(
            selected = !merge,
            title = stringResource(R.string.sync_use_account),
            body = stringResource(R.string.sync_use_account_body),
            recommended = false,
            onClick = { merge = false },
        )
        Row(
            modifier = Modifier.fillMaxWidth().padding(top = 10.dp),
            horizontalArrangement = Arrangement.End,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            QuietButton(stringResource(R.string.action_cancel), onClick = onCancel)
            Spacer(Modifier.width(8.dp))
            PrimaryButton(stringResource(R.string.sync_continue), onClick = { onContinue(merge) })
        }
    }
}

/** The sheet after a sign-in found this phone linked to another account. */
@Composable
private fun OtherAccountSheet(
    answer: SignInState,
    onCancel: () -> Unit,
    onContinue: (Boolean) -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    var useOld by remember { mutableStateOf(true) }
    val previous = answer.previousEmail
    val current = answer.accountEmail
    Column(
        modifier = Modifier.fillMaxWidth().padding(start = 20.dp, end = 20.dp, bottom = 24.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(
            stringResource(R.string.sync_other_title),
            style = type.input.copy(fontWeight = FontWeight.SemiBold),
            color = tokens.text,
        )
        Text(
            if (previous.isNotEmpty()) {
                stringResource(R.string.sync_other_body_previous, previous, current)
            } else {
                stringResource(R.string.sync_other_body, current)
            },
            style = type.meta,
            color = tokens.textSecondary,
        )
        ChoiceCard(
            selected = useOld,
            title = if (previous.isNotEmpty()) {
                stringResource(R.string.sync_other_back, previous)
            } else {
                stringResource(R.string.sync_other_back_unknown)
            },
            body = stringResource(R.string.sync_other_back_body, current),
            recommended = true,
            onClick = { useOld = true },
        )
        ChoiceCard(
            selected = !useOld,
            title = stringResource(R.string.sync_other_move, current),
            body = if (previous.isNotEmpty()) {
                stringResource(R.string.sync_other_move_body_previous, current, previous)
            } else {
                stringResource(R.string.sync_other_move_body, current)
            },
            recommended = false,
            onClick = { useOld = false },
        )
        Row(
            modifier = Modifier.fillMaxWidth().padding(top = 10.dp),
            horizontalArrangement = Arrangement.End,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            QuietButton(stringResource(R.string.action_cancel), onClick = onCancel)
            Spacer(Modifier.width(8.dp))
            PrimaryButton(stringResource(R.string.sync_continue), onClick = { onContinue(useOld) })
        }
    }
}

@Composable
private fun ChoiceCard(
    selected: Boolean,
    title: String,
    body: String,
    recommended: Boolean,
    onClick: () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 8.dp)
            .border(
                if (selected) 1.dp else Dims.hairline,
                if (selected) tokens.accent else tokens.border,
                RoundedCornerShape(10.dp),
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Box(Modifier.padding(top = 3.dp)) { RadioDot(selected) }
        Column(Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(title, style = type.body, color = tokens.text, modifier = Modifier.weight(1f, fill = false))
                if (recommended) AccentTag(stringResource(R.string.sync_recommended))
            }
            Text(body, style = type.meta, color = tokens.textSecondary)
        }
    }
}

/** The footer's server line shows only the host, as the desktop's pane does. */
private fun serverHost(server: String): String {
    if (server.isBlank()) return DEFAULT_SYNC_SERVER.removePrefix("https://")
    return runCatching {
        val uri = URI(server)
        if (uri.host.isNullOrBlank()) server else uri.host
    }.getOrDefault(server.removePrefix("https://").removePrefix("http://"))
}

/** Derives the screen's phase from the core's answer. */
fun syncPhaseOf(
    status: SyncStatus,
    signingIn: Boolean,
    choosing: Boolean,
    otherAccount: Boolean = false,
): SyncPhase = when {
    otherAccount -> SyncPhase.OTHER_ACCOUNT
    choosing -> SyncPhase.CHOOSE
    signingIn -> SyncPhase.SIGNING_IN
    !status.linked -> SyncPhase.SIGNED_OUT
    !status.signedIn -> SyncPhase.SIGNED_OUT
    status.state == SYNC_WRONG_ACCOUNT -> SyncPhase.WRONG_ACCOUNT
    status.state == "seeding" -> SyncPhase.UPLOADING
    status.state == "error" -> SyncPhase.FAILED
    status.state == "offline" -> SyncPhase.OFFLINE
    else -> SyncPhase.SYNCED
}
