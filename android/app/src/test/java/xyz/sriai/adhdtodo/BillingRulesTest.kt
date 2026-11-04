package xyz.sriai.adhdtodo

import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import xyz.sriai.adhdtodo.data.Billing
import xyz.sriai.adhdtodo.data.BillingBackend
import xyz.sriai.adhdtodo.data.CheckoutFailure
import xyz.sriai.adhdtodo.data.PageJson
import xyz.sriai.adhdtodo.data.PlanKey
import xyz.sriai.adhdtodo.ui.BillingHolder
import xyz.sriai.adhdtodo.ui.SheetStage
import xyz.sriai.adhdtodo.ui.billingDate
import xyz.sriai.adhdtodo.ui.checkoutLanguage
import xyz.sriai.adhdtodo.ui.perMonth
import xyz.sriai.adhdtodo.ui.renewalLine
import java.time.Instant
import java.time.ZoneId

/** Withdo Pro's rules (v45): the JSON, the sheet's states, and the wording of prices and dates. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class BillingRulesTest {
    private val free = Billing(offered = true, plan = "free", monthly = "$6.99", yearly = "$58.72")
    private val pro = Billing(offered = true, plan = "pro", status = "active", periodEnd = "2027-11-11T09:00:00Z", trialUsed = true)
    private val now = Instant.parse("2026-11-03T01:41:00Z")
    private val shanghai = ZoneId.of("Asia/Shanghai")

    /** A backend with a script: what Checkout answers and what Billing reads next. */
    private class FakeBackend(var checkout: () -> String, var read: Billing) : BillingBackend {
        val asked = mutableListOf<String>()
        var refreshes = 0
        override fun billing(): Billing = read
        override fun refreshBilling() { refreshes++ }
        override fun checkout(plan: String, lang: String): String {
            asked += "$plan/$lang"
            return checkout()
        }
    }

    @Test
    fun `the core's Billing JSON decodes`() {
        val raw = """{"offered":true,"plan":"pro","status":"trialing","period_end":"2026-11-11T09:00:00Z",
            "trial_used":true,"monthly":"${'$'}6.99","yearly":"${'$'}58.72","manage_url":"https://p.example/x"}"""
        val b = PageJson.decodeFromString<Billing>(raw)
        assertTrue(b.shown && b.pro && b.trialUsed)
        assertEquals("trialing", b.status)
        assertEquals("https://p.example/x", b.manageUrl)
        assertFalse(b.canUpgrade)
    }

    @Test
    fun `only a known Free plan on a server that sells Pro can upgrade`() {
        assertTrue(free.canUpgrade)
        assertFalse(Billing().canUpgrade)
        assertFalse(free.copy(offered = false).canUpgrade)
        assertFalse(free.copy(plan = "").canUpgrade)
        assertFalse(pro.canUpgrade)
    }

    @Test
    fun `the yearly price over twelve months keeps its currency`() {
        assertEquals("$4.89", perMonth("$58.72"))
        assertEquals("€5,00", perMonth("€60,00"))
        assertEquals("49.99 USD", perMonth("599.88 USD"))
        assertNull(perMonth("free"))
    }

    @Test
    fun `a date drops its year in this year only`() {
        val res = RuntimeEnvironment.getApplication().resources
        assertEquals("Nov 11", billingDate(res, "2026-11-11T09:00:00Z", now, shanghai))
        assertEquals("Nov 11, 2027", billingDate(res, "2027-11-11T09:00:00Z", now, shanghai))
        assertEquals("", billingDate(res, "", now, shanghai))
        assertEquals("", billingDate(res, "not a date", now, shanghai))
    }

    @Test
    fun `the renewal line follows the subscription's status`() {
        val res = RuntimeEnvironment.getApplication().resources
        fun line(status: String, end: String) = renewalLine(res, pro.copy(status = status, periodEnd = end), now, shanghai)
        assertEquals("Free trial until Nov 11, then it renews", line("trialing", "2026-11-11T09:00:00Z"))
        assertEquals("Renews Nov 11, 2027", line("active", "2027-11-11T09:00:00Z"))
        assertEquals("Ends Nov 11", line("canceling", "2026-11-11T09:00:00Z"))
        assertTrue(line("past_due", "2026-11-11T09:00:00Z").startsWith("Payment didn"))
        assertEquals("", line("active", ""))
    }

    @Test
    fun `the renewal line leads with the billing interval when the server says it`() {
        val res = RuntimeEnvironment.getApplication().resources
        fun line(status: String, interval: String, end: String) =
            renewalLine(res, pro.copy(status = status, interval = interval, periodEnd = end), now, shanghai)
        assertEquals("Yearly · Free trial until Nov 11, then it renews", line("trialing", "yearly", "2026-11-11T09:00:00Z"))
        assertEquals("Yearly · Renews Nov 11, 2027", line("active", "yearly", "2027-11-11T09:00:00Z"))
        assertEquals("Monthly · Renews Nov 11", line("active", "monthly", "2026-11-11T09:00:00Z"))
        assertEquals("Monthly · Ends Nov 11", line("canceling", "monthly", "2026-11-11T09:00:00Z"))
        assertEquals("Renews Nov 11, 2027", line("active", "", "2027-11-11T09:00:00Z"))
        RuntimeEnvironment.setQualifiers("zh-rCN")
        val zh = RuntimeEnvironment.getApplication().resources
        assertEquals(
            "年付 · 试用到 11月11日，之后自动续费",
            renewalLine(zh, pro.copy(status = "trialing", interval = "yearly", periodEnd = "2026-11-11T09:00:00Z"), now, shanghai),
        )
        assertEquals(
            "年付 · 2027年11月11日续费",
            renewalLine(zh, pro.copy(status = "active", interval = "yearly", periodEnd = "2027-11-11T09:00:00Z"), now, shanghai),
        )
    }

    @Test
    fun `the language of the checkout page`() {
        RuntimeEnvironment.setQualifiers("zh-rCN")
        assertEquals("zh-Hans", checkoutLanguage(RuntimeEnvironment.getApplication().resources))
        RuntimeEnvironment.setQualifiers("en")
        assertEquals("en", checkoutLanguage(RuntimeEnvironment.getApplication().resources))
    }

    @Test
    fun `the core's words map to failures`() {
        assertEquals(CheckoutFailure.RATE_LIMITED, CheckoutFailure.of(Exception("rate_limited")))
        assertEquals(CheckoutFailure.UPSTREAM, CheckoutFailure.of(Exception("upstream")))
        assertEquals(CheckoutFailure.ALREADY_SUBSCRIBED, CheckoutFailure.of(Exception("already_subscribed")))
        assertEquals(CheckoutFailure.SIGNED_OUT, CheckoutFailure.of(Exception("signed_out")))
        assertEquals(CheckoutFailure.NETWORK, CheckoutFailure.of(Exception("something else")))
    }

    @Test
    fun `the sheet opens on Yearly and only for a Free account`() {
        val holder = BillingHolder()
        holder.open()
        assertEquals(SheetStage.CLOSED, holder.stage)
        holder.update(free)
        holder.plan = PlanKey.MONTHLY
        holder.open()
        assertEquals(SheetStage.CHOOSING, holder.stage)
        assertEquals(PlanKey.YEARLY, holder.plan)
    }

    @Test
    fun `the button asks for the chosen plan, opens the page and waits`() = runBlocking {
        val holder = BillingHolder().apply { update(free); open(); plan = PlanKey.MONTHLY }
        val backend = FakeBackend({ "https://pay.example/c/1" }, free)
        val opened = mutableListOf<String>()
        holder.start(backend, "zh-Hans") { opened += it }
        assertEquals(listOf("monthly/zh-Hans"), backend.asked)
        assertEquals(listOf("https://pay.example/c/1"), opened)
        assertEquals(SheetStage.WAITING, holder.stage)
        assertEquals("https://pay.example/c/1", holder.checkoutUrl)
    }

    @Test
    fun `a failed try stays in the sheet with its reason`() = runBlocking {
        val holder = BillingHolder().apply { update(free); open() }
        val backend = FakeBackend({ throw Exception("upstream") }, free)
        val opened = mutableListOf<String>()
        holder.start(backend, "en") { opened += it }
        assertEquals(SheetStage.CHOOSING, holder.stage)
        assertEquals(CheckoutFailure.UPSTREAM, holder.failure)
        assertTrue(opened.isEmpty())
        // The next try clears the reason.
        backend.checkout = { "https://pay.example/c/2" }
        holder.start(backend, "en") { opened += it }
        assertNull(holder.failure)
        assertEquals(SheetStage.WAITING, holder.stage)
    }

    @Test
    fun `an account that is Pro already reads the plan again and closes the sheet`() = runBlocking {
        val holder = BillingHolder().apply { update(free); open() }
        val backend = FakeBackend({ throw Exception("already_subscribed") }, pro)
        holder.start(backend, "en") {}
        assertEquals(1, backend.refreshes)
        assertEquals(SheetStage.CLOSED, holder.stage)
        assertTrue(holder.billing.pro)
    }

    @Test
    fun `coming back Pro closes the waiting sheet, coming back Free leaves it`() = runBlocking {
        val holder = BillingHolder().apply { update(free); open() }
        val backend = FakeBackend({ "https://pay.example/c/3" }, free)
        holder.start(backend, "en") {}
        holder.refresh(backend)
        assertEquals(SheetStage.WAITING, holder.stage)
        backend.read = pro
        holder.refresh(backend)
        assertEquals(SheetStage.CLOSED, holder.stage)
        assertTrue(holder.billing.pro)
    }

    @Test
    fun `a sheet dismissed while the page is asked for never opens the browser`() = runBlocking {
        val holder = BillingHolder().apply { update(free); open() }
        val opened = mutableListOf<String>()
        val backend = FakeBackend({ holder.close(); "https://pay.example/c/4" }, free)
        holder.start(backend, "en") { opened += it }
        assertTrue(opened.isEmpty())
        assertEquals(SheetStage.CLOSED, holder.stage)
    }

    @Test
    fun `Manage subscription opens a link made fresh`() = runBlocking {
        val holder = BillingHolder().apply { update(pro.copy(manageUrl = "https://old.example/")) }
        val backend = FakeBackend({ "" }, pro.copy(manageUrl = "https://fresh.example/"))
        val opened = mutableListOf<String>()
        holder.manage(backend) { opened += it }
        assertEquals(1, backend.refreshes)
        assertEquals(listOf("https://fresh.example/"), opened)
        // No link (an account that never subscribed): nothing opens.
        backend.read = free
        holder.manage(backend) { opened += it }
        assertEquals(1, opened.size)
    }
}
