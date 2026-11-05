package xyz.sriai.adhdtodo

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.Billing
import xyz.sriai.adhdtodo.data.CheckoutFailure
import xyz.sriai.adhdtodo.data.ModelPhase
import xyz.sriai.adhdtodo.data.ModelState
import xyz.sriai.adhdtodo.data.PlanKey
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.AccountPlanRow
import xyz.sriai.adhdtodo.ui.AnswerCard
import xyz.sriai.adhdtodo.ui.AssistantAbove
import xyz.sriai.adhdtodo.ui.BillingHolder
import xyz.sriai.adhdtodo.ui.PlanCard
import xyz.sriai.adhdtodo.ui.PlanSheet
import xyz.sriai.adhdtodo.ui.QuietNote
import xyz.sriai.adhdtodo.ui.SheetStage
import xyz.sriai.adhdtodo.ui.UpgradeSheet
import xyz.sriai.adhdtodo.ui.showsPlanCard
import java.time.Instant
import java.time.ZoneId

/** Withdo Pro's touches (v45, v47): the plan card, the account card's plan row and sheet, and the upgrade sheet's states. */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class BillingInteractionTest {
    @get:Rule
    val compose = createComposeRule()

    private val free = Billing(offered = true, plan = "free", monthly = "$6.99", yearly = "$58.72")
    private val pro = Billing(
        offered = true, plan = "pro", status = "trialing", periodEnd = "2026-11-11T09:00:00Z",
        trialUsed = true, monthly = "$6.99", yearly = "$58.72", manageUrl = "https://portal.example/",
    )
    private val ready = ModelState("official", ModelPhase.READY, left = 14, limit = 20, plan = "free")
    private val now = Instant.parse("2026-11-03T01:41:00Z")
    private val zone = ZoneId.of("Asia/Shanghai")

    @Test
    fun `a Free card shows the pill, the quota and the bar, and nothing to press`() {
        compose.setThemedContent { PlanCard(ready, free) }
        compose.onNodeWithTag("plan-free").assertIsDisplayed()
        compose.onNodeWithText("14 / 20 left today", substring = true).assertIsDisplayed()
        compose.onNodeWithTag("plan-bar").assertIsDisplayed()
        compose.onNodeWithText("Upgrade to Pro", substring = true).assertDoesNotExist()
    }

    @Test
    fun `a Pro card has the pill and the quota, without a renewal line or Manage subscription`() {
        val model = ready.copy(left = 86, limit = 100, plan = "pro")
        compose.setThemedContent { PlanCard(model, pro) }
        compose.onNodeWithTag("plan-pro").assertIsDisplayed()
        compose.onNodeWithTag("plan-free").assertDoesNotExist()
        compose.onNodeWithText("86 / 100 left today", substring = true).assertIsDisplayed()
        compose.onNodeWithText("Manage subscription").assertDoesNotExist()
        compose.onNodeWithText("Free trial", substring = true).assertDoesNotExist()
    }

    @Test
    fun `the account card's Free row says the day's quota and Upgrade to Pro opens the upgrade sheet`() {
        var tapped = 0
        compose.setThemedContent { AccountPlanRow(free, onClick = { tapped++ }, now = now, zone = zone) }
        compose.onNodeWithText("Free").assertIsDisplayed()
        compose.onNodeWithText("The official model, 20 a day").assertIsDisplayed()
        compose.onNodeWithTag("plan-row-upgrade", useUnmergedTree = true).assertIsDisplayed()
        compose.onNodeWithTag("plan-row").performClick()
        assertEquals(1, tapped)
    }

    @Test
    fun `the account card's Pro row has the name, the status tag and one meta line`() {
        var tapped = 0
        compose.setThemedContent {
            AccountPlanRow(pro.copy(interval = "yearly"), onClick = { tapped++ }, now = now, zone = zone)
        }
        compose.onNodeWithText("Withdo Pro").assertIsDisplayed()
        compose.onNodeWithTag("plan-tag", useUnmergedTree = true).assertIsDisplayed()
        compose.onNodeWithText("Free trial").assertIsDisplayed()
        compose.onNodeWithText("Yearly · First charge Nov 11").assertIsDisplayed()
        compose.onNodeWithTag("plan-row-upgrade", useUnmergedTree = true).assertDoesNotExist()
        compose.onNodeWithTag("plan-row").performClick()
        assertEquals(1, tapped)
    }

    @Test
    fun `the plan sheet has the three rows, the status paragraph and Manage subscription`() {
        var managed = 0
        val canceling = pro.copy(status = "canceling", interval = "yearly", provider = "paypal")
        compose.setThemedContent { PlanSheet(canceling, onManage = { managed++ }, onDismiss = {}, now = now, zone = zone) }
        compose.onNodeWithText("Withdo Pro").assertIsDisplayed()
        compose.onNodeWithText("Canceled").assertIsDisplayed()
        compose.onNodeWithText("Yearly $58.72").assertIsDisplayed()
        compose.onNodeWithText("PayPal").assertIsDisplayed()
        compose.onNodeWithText("Pro until").assertIsDisplayed()
        compose.onNodeWithText("Nov 11").assertIsDisplayed()
        compose.onNodeWithTag("plan-note").assertIsDisplayed()
        compose.onNodeWithTag("plan-manage").performClick()
        assertEquals(1, managed)
    }

    @Test
    fun `the plan row opens the plan sheet for Pro and the upgrade sheet for Free`() {
        val h = BillingHolder().apply { update(pro) }
        h.openPlan()
        assertEquals(true, h.planSheet)
        assertEquals(SheetStage.CLOSED, h.stage)
        h.update(free) // the subscription ended while the sheet was open
        assertEquals(false, h.planSheet)
        h.openPlan()
        assertEquals(false, h.planSheet)
        assertEquals(SheetStage.CHOOSING, h.stage)
    }

    @Test
    fun `the plan card shows only for a signed-in account on a server that sells Pro`() {
        assertEquals(true, showsPlanCard(free, ready))
        assertEquals(true, showsPlanCard(free, ready.copy(phase = ModelPhase.USED_UP)))
        assertEquals(false, showsPlanCard(free, ready.copy(phase = ModelPhase.SIGNED_OUT)))
        assertEquals(false, showsPlanCard(Billing(), ready))
        assertEquals(false, showsPlanCard(free.copy(plan = ""), ready))
    }

    private fun holder(stage: SheetStage = SheetStage.CHOOSING, billing: Billing = free) =
        BillingHolder().apply { this.billing = billing; this.stage = stage }

    @Test
    fun `the sheet opens on Yearly with its price, four ticks and the trial button`() {
        val h = holder()
        compose.setThemedContent { UpgradeSheet(h, onStart = {}, onReopen = {}, onDismiss = {}) }
        compose.onNodeWithText("Upgrade to Withdo Pro").assertIsDisplayed()
        compose.onNodeWithTag("billing-price").assertIsDisplayed()
        compose.onNodeWithText("$58.72").assertIsDisplayed()
        compose.onNodeWithText("/ year, about $4.89 / month").assertIsDisplayed()
        compose.onNodeWithText("The official model 100 times a day (20 on Free)").assertIsDisplayed()
        compose.onNodeWithText("Sync is always free").assertIsDisplayed()
        compose.onNodeWithText("Start 7-day free trial").assertIsDisplayed()
        compose.onNodeWithText("You won’t be charged before the trial ends. Billed by Waffo, tax included.").assertIsDisplayed()
    }

    @Test
    fun `the segment changes the plan and the price`() {
        val h = holder()
        compose.setThemedContent { UpgradeSheet(h, onStart = {}, onReopen = {}, onDismiss = {}) }
        compose.onNodeWithTag("billing-seg-monthly").performClick()
        compose.waitForIdle()
        assertEquals(PlanKey.MONTHLY, h.plan)
        compose.onNodeWithText("$6.99").assertIsDisplayed()
        compose.onNodeWithText("/ month").assertIsDisplayed()
        compose.onNodeWithTag("billing-seg-yearly").performClick()
        compose.waitForIdle()
        assertEquals(PlanKey.YEARLY, h.plan)
        compose.onNodeWithText("$58.72").assertIsDisplayed()
    }

    @Test
    fun `a used trial turns the button into Subscribe`() {
        val h = holder(billing = free.copy(trialUsed = true))
        compose.setThemedContent { UpgradeSheet(h, onStart = {}, onReopen = {}, onDismiss = {}) }
        compose.onNodeWithText("Subscribe").assertIsDisplayed()
        compose.onNodeWithText("Billed by Waffo, tax included.").assertIsDisplayed()
    }

    @Test
    fun `the button starts the checkout once, and is off while the page is asked for`() {
        var started = 0
        val h = holder()
        compose.setThemedContent { UpgradeSheet(h, onStart = { started++ }, onReopen = {}, onDismiss = {}) }
        compose.onNodeWithTag("billing-cta").performClick()
        assertEquals(1, started)
        h.stage = SheetStage.STARTING
        compose.waitForIdle()
        compose.onNodeWithText("Opening the payment page…").assertIsDisplayed()
        compose.onNodeWithTag("billing-cta").performClick()
        assertEquals(1, started)
    }

    @Test
    fun `without PayPal the sheet has no PayPal button`() {
        val plain = holder()
        compose.setThemedContent { UpgradeSheet(plain, onStart = {}, onReopen = {}, onDismiss = {}) }
        compose.onNodeWithTag("billing-paypal").assertDoesNotExist()
        compose.onNodeWithText("Billed by Waffo, tax included.", substring = true).assertIsDisplayed()
    }

    @Test
    fun `the PayPal button checks out at PayPal and the main button at Waffo`() {
        val asked = mutableListOf<String>()
        val h = holder(billing = free.copy(paypal = true))
        compose.setThemedContent { UpgradeSheet(h, onStart = { asked += it }, onReopen = {}, onDismiss = {}) }
        compose.onNodeWithText("or").assertIsDisplayed()
        compose.onNodeWithText("Subscribe with PayPal").assertIsDisplayed()
        compose.onNodeWithText("Cards are billed by Waffo, tax included.", substring = true).assertIsDisplayed()
        compose.onNodeWithTag("billing-paypal").performClick()
        compose.onNodeWithTag("billing-cta").performClick()
        assertEquals(listOf("paypal", "waffo"), asked)
        h.stage = SheetStage.STARTING
        compose.waitForIdle()
        compose.onNodeWithTag("billing-paypal").performClick()
        assertEquals(2, asked.size)
    }

    @Test
    fun `waiting for PayPal says so and offers to reopen PayPal`() {
        var reopened = 0
        val h = holder(SheetStage.WAITING, billing = free.copy(paypal = true)).apply {
            provider = "paypal"
            checkoutUrl = "https://paypal.example/c/1"
        }
        compose.setThemedContent { UpgradeSheet(h, onStart = {}, onReopen = { reopened++ }, onDismiss = {}) }
        compose.onNodeWithText("Finish subscribing in PayPal").assertIsDisplayed()
        compose.onNodeWithText("Reopen PayPal").performClick()
        assertEquals(1, reopened)
        compose.onNodeWithText("Finish paying in the browser").assertDoesNotExist()
    }

    @Test
    fun `a failure shows in the sheet`() {
        val h = holder().apply { failure = CheckoutFailure.RATE_LIMITED }
        compose.setThemedContent { UpgradeSheet(h, onStart = {}, onReopen = {}, onDismiss = {}) }
        compose.onNodeWithTag("billing-error").assertIsDisplayed()
        compose.onNodeWithText("Too many tries. Wait a little and try again.").assertIsDisplayed()
    }

    @Test
    fun `waiting for the payment offers to reopen the page`() {
        var reopened = 0
        val h = holder(SheetStage.WAITING).apply { checkoutUrl = "https://pay.example/c/1" }
        compose.setThemedContent { UpgradeSheet(h, onStart = {}, onReopen = { reopened++ }, onDismiss = {}) }
        compose.onNodeWithText("Finish paying in the browser").assertIsDisplayed()
        compose.onNodeWithTag("billing-cta").assertDoesNotExist()
        compose.onNodeWithTag("billing-reopen").performClick()
        assertEquals(1, reopened)
    }

    @Test
    fun `the used-up line offers Pro only to an account that can upgrade`() {
        val card = AnswerCard(summary = "Added 1", quiet = QuietNote.USED_UP)
        compose.setThemedContent {
            AssistantAbove(card, onUndo = {}, onPick = {}, onSignIn = {}, onUpgrade = {})
        }
        compose.onNodeWithTag("assistant-quiet").assertIsDisplayed()
        compose.onNodeWithText("Upgrade to Pro", substring = true).assertIsDisplayed()
        compose.onNodeWithText("for 100 a day.", substring = true).assertIsDisplayed()
    }

    @Test
    fun `without an upgrade the used-up line only says it is used up`() {
        val card = AnswerCard(summary = "Added 1", quiet = QuietNote.USED_UP)
        compose.setThemedContent { AssistantAbove(card, onUndo = {}, onPick = {}, onSignIn = {}) }
        compose.onNodeWithText("Upgrade to Pro", substring = true).assertDoesNotExist()
        compose.onNodeWithText("used up", substring = true, ignoreCase = true).assertIsDisplayed()
    }
}
