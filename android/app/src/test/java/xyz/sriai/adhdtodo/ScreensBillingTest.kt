package xyz.sriai.adhdtodo

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import com.github.takahirom.roborazzi.captureRoboImage
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.ParameterizedRobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.Account
import xyz.sriai.adhdtodo.data.Billing
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.Device
import xyz.sriai.adhdtodo.data.ModelPhase
import xyz.sriai.adhdtodo.data.ModelState
import xyz.sriai.adhdtodo.data.QuickDraft
import xyz.sriai.adhdtodo.data.QuickParse
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.data.UpdatePhase
import xyz.sriai.adhdtodo.data.UpdateUi
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.testing.screenshotFile
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.BackBar
import xyz.sriai.adhdtodo.ui.BillingHolder
import xyz.sriai.adhdtodo.ui.BottomNav
import xyz.sriai.adhdtodo.ui.QuietNote
import xyz.sriai.adhdtodo.ui.SettingsData
import xyz.sriai.adhdtodo.ui.SettingsScreen
import xyz.sriai.adhdtodo.ui.SheetStage
import xyz.sriai.adhdtodo.ui.Tab
import xyz.sriai.adhdtodo.ui.UpgradeSheet
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.localCard
import xyz.sriai.adhdtodo.ui.theme.AppearanceMode
import xyz.sriai.adhdtodo.ui.theme.LocalTokens

/**
 * Withdo Pro on the phone (v45 section two) as full phone pictures, with fake plans and no
 * network: Settings > Assistant as the Free plan card, the upgrade sheet over it, the wait for the
 * payment, the Pro card, and the quiet line past the day's turns. Today is Tuesday 3 November
 * 2026, as in docs/design/mockups/v45.html. Render with
 * `scripts/android-screens.sh 'xyz.sriai.adhdtodo.ScreensBillingTest'`.
 */
@RunWith(ParameterizedRobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class ScreensBillingTest(private val case: Case) {
    @get:Rule
    val compose = createComposeRule()

    @Test
    fun screenshot() {
        compose.setThemedContent(night = case.night, zh = case.zh) { Picture(case.shot) }
        compose.waitForIdle()
        compose.onRoot().captureRoboImage(screenshotFile(case.name))
    }

    data class Case(val name: String, val night: Boolean, val zh: Boolean, val shot: Shot)

    enum class Shot { SETTINGS_FREE, SHEET, WAITING, SETTINGS_PRO, QUOTA_USED_UP_UPGRADE }

    companion object {
        @JvmStatic
        @ParameterizedRobolectricTestRunner.Parameters(name = "{0}")
        fun cases(): List<Case> = listOf(
            "settings-pro-free" to Shot.SETTINGS_FREE,
            "pro-sheet" to Shot.SHEET,
            "pro-waiting" to Shot.WAITING,
            "settings-pro" to Shot.SETTINGS_PRO,
            "quota-used-up-upgrade" to Shot.QUOTA_USED_UP_UPGRADE,
        ).flatMap { (name, shot) ->
            listOf(
                Case("$name-light", night = false, zh = false, shot = shot),
                Case("$name-dark", night = true, zh = false, shot = shot),
                Case("$name-light-zh", night = false, zh = true, shot = shot),
                Case("$name-dark-zh", night = true, zh = true, shot = shot),
            )
        }
    }
}

private val FreePlan = Billing(offered = true, plan = "free", monthly = "$6.99", yearly = "$58.72")
private val ProPlan = Billing(
    offered = true, plan = "pro", status = "trialing", periodEnd = "2026-11-11T09:00:00Z",
    trialUsed = true, monthly = "$6.99", yearly = "$58.72", manageUrl = "https://portal.example/session",
)

@Composable
private fun Picture(shot: ScreensBillingTest.Shot) {
    val resources = LocalContext.current.resources
    val configuration = LocalConfiguration.current
    when (shot) {
        ScreensBillingTest.Shot.SETTINGS_FREE -> SettingsFrame(ModelState("official", ModelPhase.READY, 14, 20, "free"), FreePlan)
        ScreensBillingTest.Shot.SETTINGS_PRO -> SettingsFrame(ModelState("official", ModelPhase.READY, 86, 100, "pro"), ProPlan)
        ScreensBillingTest.Shot.SHEET, ScreensBillingTest.Shot.WAITING -> {
            val holder = remember { BillingHolder() }
            val waiting = shot == ScreensBillingTest.Shot.WAITING
            holder.billing = FreePlan
            holder.stage = if (waiting) SheetStage.WAITING else SheetStage.CHOOSING
            SettingsFrame(ModelState("official", ModelPhase.READY, 14, 20, "free"), FreePlan)
            UpgradeSheet(holder, onStart = {}, onReopen = {}, onDismiss = {})
        }
        ScreensBillingTest.Shot.QUOTA_USED_UP_UPGRADE -> {
            val words = remember(configuration) { Words(resources, { Tuesday }, Shanghai) }
            val categories = remember { Golden.load<List<Category>>("categories.golden.json") }
            AssistantFrame(
                words = words,
                categories = categories,
                card = localCard(
                    resources, words,
                    QuickParse(listOf(QuickDraft(title = "打给房东", remindAt = "2026-11-04T09:00:00+08:00", ranges = listOf(listOf(0, 5))))),
                    titles = listOf("打给房东"), ids = listOf(2), note = QuietNote.USED_UP,
                ),
                thinking = false,
                model = ModelState("none", ModelPhase.USED_UP, 0, 20, "free"),
                draft = if (configuration.locales[0].language == "zh") "明天 9 点打给房东，再让 Claude 修 bug" else "Call the landlord at 9 tomorrow, have Claude fix the bug",
                placeholder = stringResource(R.string.composer_placeholder_mine),
                upgrade = true,
            )
        }
    }
}

/** Settings under the account with the Assistant group as the plan card (the pushed page, over Projects' bar). */
@Composable
private fun SettingsFrame(model: ModelState, billing: Billing) {
    val tokens = LocalTokens.current
    Column(Modifier.fillMaxSize().background(tokens.bg)) {
        Column(Modifier.weight(1f).statusBarsPadding()) {
            BackBar(title = stringResource(R.string.title_projects), onBack = {})
            Box(Modifier.weight(1f)) {
                SettingsScreen(
                    data = SettingsData(
                        account = Golden.load<Account>("account.golden.json").copy(email = "cyan@example.com", name = ""),
                        devices = Golden.load<List<Device>>("devices.golden.json").take(3),
                        model = model,
                        billing = billing,
                    ),
                    status = SyncStatus(state = "idle", linked = true, signedIn = true),
                    language = "",
                    appearance = AppearanceMode.System,
                    update = UpdateUi(UpdatePhase.UP_TO_DATE, current = "0.13.0"),
                    onOpenSync = {},
                    onLanguage = {},
                    onAppearance = {},
                    onNotifications = {},
                    onCategories = {},
                    onCheckUpdate = {},
                    onUpdate = {},
                    onOpenUrl = {},
                    now = Tuesday,
                    zone = Shanghai,
                )
            }
        }
        BottomNav(tab = Tab.PROJECTS, count = 2, onSelect = {})
    }
}
