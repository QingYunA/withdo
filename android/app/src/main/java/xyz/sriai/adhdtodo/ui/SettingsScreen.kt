package xyz.sriai.adhdtodo.ui

import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Account
import xyz.sriai.adhdtodo.data.Billing
import xyz.sriai.adhdtodo.data.CategoryCount
import xyz.sriai.adhdtodo.data.CoreApi
import xyz.sriai.adhdtodo.data.Device
import xyz.sriai.adhdtodo.data.ModelPhase
import xyz.sriai.adhdtodo.data.ModelState
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.data.UpdatePhase
import xyz.sriai.adhdtodo.data.UpdateUi
import xyz.sriai.adhdtodo.platform.AppLanguage
import xyz.sriai.adhdtodo.ui.theme.Appearance
import xyz.sriai.adhdtodo.ui.theme.AppearanceMode
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import java.time.Instant
import java.time.ZoneId

/** The routes Settings adds to the stack; each is shown with the Projects tab's bottom bar. */
const val ROUTE_SETTINGS = "settings"
const val ROUTE_NOTIFICATIONS = "settings:notifications"
const val ROUTE_CATEGORIES = "categories"

fun isSettingsRoute(route: String): Boolean =
    route == ROUTE_SETTINGS || route == ROUTE_NOTIFICATIONS || route == ROUTE_CATEGORIES

/** What the settings page reads from the core when it opens. */
data class SettingsData(
    val account: Account = Account(),
    val devices: List<Device> = emptyList(),
    val status: SyncStatus? = null,
    val categories: List<CategoryCount> = emptyList(),
    /** Which model answers the composer (v44): the Assistant group's row. */
    val model: ModelState = ModelState(),
    /** Withdo Pro's plan (v45, v47): with it the Assistant group is the plan card and the account card gets the plan row. */
    val billing: Billing = Billing(),
)

/** Reads what the settings pages show; a list that fails to load stays empty. Blocks: call it off the UI thread. */
fun loadSettingsData(api: CoreApi): SettingsData = SettingsData(
    account = runCatching { api.account() }.getOrDefault(Account()),
    devices = runCatching { api.devices() }.getOrDefault(emptyList()),
    status = runCatching { api.syncStatus() }.getOrNull(),
    categories = runCatching { api.categoriesWithCounts() }.getOrDefault(emptyList()),
    model = runCatching { api.modelState() }.getOrDefault(ModelState()),
)

/** The project the app was built from: where About points. */
const val WEBSITE_URL = "https://qingyuna.github.io/withdo"
const val GITHUB_URL = "https://github.com/QingYunA/withdo"

/** The pushed `Settings` page: the account, General, Organize and About, as grouped cards. */
@Composable
fun SettingsScreen(
    data: SettingsData,
    status: SyncStatus,
    language: String,
    appearance: AppearanceMode,
    update: UpdateUi,
    onOpenSync: () -> Unit,
    onLanguage: (String) -> Unit,
    onAppearance: (AppearanceMode) -> Unit,
    onNotifications: () -> Unit,
    onCategories: () -> Unit,
    onCheckUpdate: () -> Unit,
    onUpdate: () -> Unit,
    onOpenUrl: (String) -> Unit,
    onPlan: () -> Unit = {},
    now: Instant = Instant.now(),
    zone: ZoneId = ZoneId.systemDefault(),
) {
    val tokens = LocalTokens.current
    var languageSheet by remember { mutableStateOf(false) }
    var appearanceSheet by remember { mutableStateOf(false) }
    var linksSheet by remember { mutableStateOf(false) }
    val phase = syncPhaseOf(status, signingIn = false, choosing = false)

    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = Dims.pagePadding)
            .padding(bottom = 24.dp),
    ) {
        PageTitle(stringResource(R.string.settings_title))
        GroupCard {
            if (phase == SyncPhase.SIGNED_OUT) {
                SettingsRow(
                    title = stringResource(R.string.settings_sign_in),
                    titleColor = tokens.accentInk,
                    chevron = true,
                    onClick = onOpenSync,
                )
            } else {
                val (line, color) = accountLine(phase, data.devices.size)
                SettingsRow(
                    leading = { Avatar("you", size = 28.dp) },
                    title = data.account.display.ifBlank { stringResource(R.string.settings_account) },
                    subtitle = line,
                    subtitleColor = if (color == AccountTone.Good) tokens.categoryInk("green") else if (color == AccountTone.Bad) tokens.statusBlocked else tokens.textSecondary,
                    chevron = true,
                    onClick = onOpenSync,
                )
                if (data.billing.shown) {
                    RowRule()
                    AccountPlanRow(data.billing, onPlan, now, zone)
                }
            }
        }

        if (data.model.phase != ModelPhase.HIDDEN) {
            GroupLabel(stringResource(R.string.settings_assistant))
            if (showsPlanCard(data.billing, data.model)) {
                PlanCard(data.model, data.billing)
            } else {
                GroupCard { AssistantRow(data.model, onOpenSync) }
            }
        }

        GroupLabel(stringResource(R.string.settings_general))
        GroupCard {
            SettingsRow(
                title = stringResource(R.string.settings_language),
                value = languageName(language),
                chevron = true,
                onClick = { languageSheet = true },
            )
            RowRule()
            SettingsRow(
                title = stringResource(R.string.settings_appearance),
                value = appearanceName(appearance),
                chevron = true,
                onClick = { appearanceSheet = true },
            )
            RowRule()
            SettingsRow(
                title = stringResource(R.string.settings_notifications),
                subtitle = stringResource(R.string.settings_notifications_line),
                chevron = true,
                onClick = onNotifications,
            )
        }

        GroupLabel(stringResource(R.string.settings_organize))
        GroupCard {
            SettingsRow(
                title = stringResource(R.string.settings_categories),
                subtitle = pluralStringResource(R.plurals.settings_category_count, data.categories.size, data.categories.size),
                chevron = true,
                onClick = onCategories,
            )
        }

        GroupLabel(stringResource(R.string.settings_about))
        GroupCard {
            VersionRow(update, onCheckUpdate, onUpdate)
            RowRule()
            SettingsRow(
                title = stringResource(R.string.settings_links),
                chevron = true,
                onClick = { linksSheet = true },
            )
        }
    }

    if (languageSheet) {
        ChoiceSheet(
            options = listOf(
                AppLanguage.SYSTEM to stringResource(R.string.settings_system),
                AppLanguage.ENGLISH to "English",
                AppLanguage.CHINESE to "简体中文",
            ),
            selected = language,
            onPick = onLanguage,
            onDismiss = { languageSheet = false },
        )
    }
    if (appearanceSheet) {
        ChoiceSheet(
            options = AppearanceMode.entries.map { it to appearanceName(it) },
            selected = appearance,
            onPick = onAppearance,
            onDismiss = { appearanceSheet = false },
        )
    }
    if (linksSheet) {
        SheetFrame({ linksSheet = false }) {
            for ((label, url) in listOf(
                stringResource(R.string.settings_website) to WEBSITE_URL,
                stringResource(R.string.settings_privacy) to PRIVACY_URL,
                stringResource(R.string.settings_github) to GITHUB_URL,
            )) {
                SheetRow(label, icon = "external-link", onClick = { onOpenUrl(url); linksSheet = false })
            }
        }
    }
}

/** The settings route as App draws it: real language, appearance and system links. */
@Composable
fun SettingsRoute(
    data: SettingsData,
    status: SyncStatus,
    update: UpdateUi,
    onOpenSync: () -> Unit,
    onNotifications: () -> Unit,
    onCategories: () -> Unit,
    onCheckUpdate: () -> Unit,
    onUpdate: () -> Unit,
    onOpenUrl: (String) -> Unit,
    onPlan: () -> Unit = {},
) {
    val context = LocalContext.current
    SettingsScreen(
        data = data,
        status = status,
        language = remember { AppLanguage.saved(context) },
        appearance = Appearance.mode,
        update = update,
        onOpenSync = onOpenSync,
        onLanguage = { tag -> context.activity()?.let { AppLanguage.set(it, tag) } },
        onAppearance = { Appearance.set(context, it) },
        onNotifications = onNotifications,
        onCategories = onCategories,
        onCheckUpdate = onCheckUpdate,
        onUpdate = onUpdate,
        onOpenUrl = onOpenUrl,
        onPlan = onPlan,
    )
}

private tailrec fun Context.activity(): Activity? = when (this) {
    is Activity -> this
    is ContextWrapper -> baseContext.activity()
    else -> null
}

/** `Withdo official model` and its state: `● Available · 86 left today`, or the reason there is none. */
@Composable
private fun AssistantRow(model: ModelState, onOpenSync: () -> Unit) {
    val tokens = LocalTokens.current
    val ready = model.phase == ModelPhase.READY
    SettingsRow(
        leading = { Avatar("assistant", size = 28.dp) },
        title = stringResource(R.string.assistant_model_name),
        subtitle = if (ready) "● ${modelLine(model)}" else modelLine(model),
        subtitleColor = if (ready) tokens.categoryInk("green") else tokens.textSecondary,
        onClick = if (model.phase == ModelPhase.SIGNED_OUT) onOpenSync else null,
    )
}

private enum class AccountTone { Good, Bad, Quiet }

@Composable
private fun accountLine(phase: SyncPhase, devices: Int): Pair<String, AccountTone> = when (phase) {
    SyncPhase.SYNCED -> stringResource(R.string.settings_synced, pluralStringResource(R.plurals.settings_devices, devices, devices)) to AccountTone.Good
    SyncPhase.UPLOADING, SyncPhase.SIGNING_IN -> stringResource(R.string.settings_syncing) to AccountTone.Quiet
    SyncPhase.OFFLINE -> stringResource(R.string.settings_offline) to AccountTone.Quiet
    else -> stringResource(R.string.settings_sync_failed) to AccountTone.Bad
}

@Composable
private fun languageName(tag: String): String = when (tag) {
    AppLanguage.ENGLISH -> "English"
    AppLanguage.CHINESE -> "简体中文"
    else -> stringResource(R.string.settings_system)
}

@Composable
private fun appearanceName(mode: AppearanceMode): String = stringResource(
    when (mode) {
        AppearanceMode.System -> R.string.settings_system
        AppearanceMode.Light -> R.string.settings_light
        AppearanceMode.Dark -> R.string.settings_dark
    },
)

/** `Withdo 0.13.0`, what the last check found, and Check now (Update when one is on offer). */
@Composable
private fun VersionRow(update: UpdateUi, onCheck: () -> Unit, onUpdate: () -> Unit) {
    val tokens = LocalTokens.current
    val failed = update.phase == UpdatePhase.FAILED || update.phase == UpdatePhase.CHECK_FAILED
    val line = when (update.phase) {
        UpdatePhase.CHECKING -> stringResource(R.string.update_row_checking)
        UpdatePhase.UP_TO_DATE -> stringResource(R.string.settings_up_to_date)
        UpdatePhase.CHECK_FAILED -> stringResource(R.string.update_row_check_failed)
        UpdatePhase.AVAILABLE -> stringResource(R.string.update_available, update.version)
        UpdatePhase.DOWNLOADING -> stringResource(R.string.update_downloading, update.percent)
        UpdatePhase.NEEDS_PERMISSION -> stringResource(R.string.update_allow_installs)
        UpdatePhase.FAILED -> stringResource(R.string.update_failed)
        UpdatePhase.IDLE -> null
    }
    SettingsRow(
        title = stringResource(R.string.settings_about_name, update.current),
        subtitle = line,
        subtitleColor = if (failed) tokens.statusBlocked else tokens.textSecondary,
        trailing = {
            when (update.phase) {
                UpdatePhase.AVAILABLE, UpdatePhase.FAILED -> TextLink(stringResource(R.string.update_action), onUpdate)
                UpdatePhase.CHECKING, UpdatePhase.DOWNLOADING, UpdatePhase.NEEDS_PERMISSION -> Unit
                else -> TextLink(stringResource(R.string.settings_check_now), onCheck)
            }
        },
    )
}

/** The page's title under the back bar, 24 sp / 600, with an optional quiet line. */
@Composable
fun PageTitle(title: String, line: String? = null) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Column(Modifier.padding(top = 4.dp, bottom = 8.dp)) {
        Text(title, style = type.title, color = tokens.text)
        if (line != null) Text(line, style = type.meta, color = tokens.textSecondary, modifier = Modifier.padding(top = 2.dp))
    }
}

/**
 * One row of a group card, 56 dp at least: an optional [leading], the title and a quiet line, and at
 * the right a [value], a [trailing] control and the chevron.
 */
@Composable
fun SettingsRow(
    title: String,
    subtitle: String? = null,
    value: String? = null,
    chevron: Boolean = false,
    titleColor: Color = Color.Unspecified,
    subtitleColor: Color = Color.Unspecified,
    leading: (@Composable () -> Unit)? = null,
    trailing: (@Composable () -> Unit)? = null,
    onClick: (() -> Unit)? = null,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        Modifier
            .fillMaxWidth()
            .heightIn(min = 56.dp)
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier)
            .padding(horizontal = 16.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        leading?.invoke()
        Column(Modifier.weight(1f)) {
            Text(
                title,
                style = type.body.copy(fontSize = 16.sp),
                color = if (titleColor == Color.Unspecified) tokens.text else titleColor,
            )
            if (!subtitle.isNullOrEmpty()) {
                Text(
                    subtitle,
                    style = type.meta,
                    color = if (subtitleColor == Color.Unspecified) tokens.textSecondary else subtitleColor,
                )
            }
        }
        if (!value.isNullOrEmpty()) Text(value, style = type.body.copy(fontSize = 14.sp), color = tokens.textSecondary)
        trailing?.invoke()
        if (chevron) PixelIcon("chevron-right", size = 12.dp, tint = tokens.textTertiary)
    }
}

/** A sheet of one choice among a few: the chosen row is checked; a tap picks and closes. */
@Composable
private fun <T> ChoiceSheet(
    options: List<Pair<T, String>>,
    selected: T,
    onPick: (T) -> Unit,
    onDismiss: () -> Unit,
) {
    SheetFrame(onDismiss) { ChoiceRows(options, selected, onPick, onDismiss) }
}

@Composable
private fun <T> ColumnScope.ChoiceRows(
    options: List<Pair<T, String>>,
    selected: T,
    onPick: (T) -> Unit,
    onDismiss: () -> Unit,
) {
    for ((value, label) in options) {
        SheetRow(
            label = label,
            checked = value == selected,
            selected = value == selected,
            onClick = {
                onDismiss()
                if (value != selected) onPick(value)
            },
        )
    }
}
