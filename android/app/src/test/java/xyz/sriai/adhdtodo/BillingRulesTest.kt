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
import xyz.sriai.adhdtodo.data.ProviderKey
import xyz.sriai.adhdtodo.ui.BillingHolder
import xyz.sriai.adhdtodo.ui.SheetStage
import xyz.sriai.adhdtodo.ui.billingDate
import xyz.sriai.adhdtodo.ui.checkoutLanguage
import xyz.sriai.adhdtodo.ui.payPalLabel
import xyz.sriai.adhdtodo.ui.perMonth
import xyz.sriai.adhdtodo.ui.planMeta
import xyz.sriai.adhdtodo.ui.planNote
import xyz.sriai.adhdtodo.ui.planRows
import xyz.sriai.adhdtodo.ui.planTag
import java.time.Instant
import java.time.ZoneId

/** Withdo Pro's rules (v45, v47): the JSON, the sheet's states, and the wording of prices and dates. */
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
        override fun checkout(plan: String, provider: String, lang: String): String {
            asked += "$plan/$provider/$lang"
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
    fun `the plan row's meta line follows the subscription's status and says the interval`() {
        val res = RuntimeEnvironment.getApplication().resources
        fun meta(status: String, interval: String, end: String) =
            planMeta(res, pro.copy(status = status, interval = interval, periodEnd = end), now, shanghai)
        assertEquals("Yearly · First charge Nov 11", meta("trialing", "yearly", "2026-11-11T09:00:00Z"))
        assertEquals("Yearly · Renews Nov 11, 2027", meta("active", "yearly", "2027-11-11T09:00:00Z"))
        assertEquals("Monthly · Renews Nov 11", meta("active", "monthly", "2026-11-11T09:00:00Z"))
        assertEquals("Pro until Nov 11", meta("canceling", "yearly", "2026-11-11T09:00:00Z"))
        assertEquals("The renewal didn’t go through", meta("past_due", "yearly", "2026-11-11T09:00:00Z"))
        // No interval: the date alone; no date: the interval alone.
        assertEquals("First charge Nov 11", meta("trialing", "", "2026-11-11T09:00:00Z"))
        assertEquals("Renews Nov 11, 2027", meta("active", "", "2027-11-11T09:00:00Z"))
        assertEquals("Yearly", meta("active", "yearly", ""))
        assertEquals("", meta("active", "", ""))
    }

    @Test
    fun `the plan row's meta line in Chinese`() {
        RuntimeEnvironment.setQualifiers("zh-rCN")
        val zh = RuntimeEnvironment.getApplication().resources
        fun meta(status: String, end: String) = planMeta(zh, pro.copy(status = status, interval = "yearly", periodEnd = end), now, shanghai)
        assertEquals("年付 · 11月11日开始扣费", meta("trialing", "2026-11-11T09:00:00Z"))
        assertEquals("年付 · 2027年11月11日续费", meta("active", "2027-11-11T09:00:00Z"))
        assertEquals("Pro 用到 11月11日", meta("canceling", "2026-11-11T09:00:00Z"))
        assertEquals("续费没扣成功", meta("past_due", "2026-11-11T09:00:00Z"))
        assertEquals("试用中", planTag(zh, pro.copy(status = "trialing")))
        assertEquals("已取消", planTag(zh, pro.copy(status = "canceling")))
        assertEquals("续费失败", planTag(zh, pro.copy(status = "past_due")))
    }

    @Test
    fun `the status tag`() {
        val res = RuntimeEnvironment.getApplication().resources
        assertEquals("Free trial", planTag(res, pro.copy(status = "trialing")))
        assertEquals("Pro", planTag(res, pro.copy(status = "active")))
        assertEquals("Canceled", planTag(res, pro.copy(status = "canceling")))
        assertEquals("Payment failed", planTag(res, pro.copy(status = "past_due")))
    }

    @Test
    fun `the plan sheet's rows hold the price, the payment method and the date of the status`() {
        val res = RuntimeEnvironment.getApplication().resources
        val yearly = pro.copy(interval = "yearly", monthly = "$6.99", yearly = "$58.72", periodEnd = "2027-11-11T09:00:00Z")
        assertEquals(
            listOf("Plan" to "Yearly $58.72", "Payment" to "Waffo", "Renews" to "Nov 11, 2027"),
            planRows(res, yearly, now, shanghai),
        )
        val monthlyPayPal = yearly.copy(status = "trialing", interval = "monthly", provider = ProviderKey.PAYPAL, periodEnd = "2026-11-11T09:00:00Z")
        assertEquals(
            listOf("Plan" to "Monthly $6.99", "Payment" to "PayPal", "First charge" to "Nov 11"),
            planRows(res, monthlyPayPal, now, shanghai),
        )
        assertEquals("Pro until", planRows(res, yearly.copy(status = "canceling"), now, shanghai).last().first)
        // No interval: no price; no date: no third row.
        assertEquals("Pro", planRows(res, yearly.copy(interval = ""), now, shanghai).first().second)
        assertEquals(2, planRows(res, yearly.copy(periodEnd = ""), now, shanghai).size)
        // Past due: the period end is in the past, so no date row.
        assertEquals(
            listOf("Plan" to "Yearly $58.72", "Payment" to "Waffo"),
            planRows(res, yearly.copy(status = "past_due", periodEnd = "2026-11-01T09:00:00Z"), now, shanghai),
        )
    }

    @Test
    fun `the plan sheet's rows and paragraphs in Chinese`() {
        RuntimeEnvironment.setQualifiers("zh-rCN")
        val zh = RuntimeEnvironment.getApplication().resources
        val canceling = pro.copy(status = "canceling", interval = "yearly", yearly = "$58.72", provider = ProviderKey.PAYPAL, periodEnd = "2026-11-12T09:00:00Z")
        assertEquals(
            listOf("方案" to "年付 $58.72", "付款方式" to "PayPal", "Pro 用到" to "11月12日"),
            planRows(zh, canceling, now, shanghai),
        )
        assertEquals("到期后回到免费版，官方模型每天 20 次，不会再扣费。之后想继续用，在这里重新订阅。", planNote(zh, canceling))
        assertEquals("试用结束前随时可以取消，不会扣款。", planNote(zh, pro.copy(status = "trialing")))
        assertEquals("随时可以取消，Pro 用到这一期结束。", planNote(zh, pro.copy(status = "active")))
        assertEquals("开始扣费", planRows(zh, pro.copy(status = "trialing", periodEnd = "2026-11-12T09:00:00Z"), now, shanghai).last().first)
        assertEquals("下次续费", planRows(zh, pro.copy(periodEnd = "2026-11-12T09:00:00Z"), now, shanghai).last().first)
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
        assertEquals(listOf("monthly/waffo/zh-Hans"), backend.asked)
        assertEquals(listOf("https://pay.example/c/1"), opened)
        assertEquals(SheetStage.WAITING, holder.stage)
        assertEquals("https://pay.example/c/1", holder.checkoutUrl)
    }

    @Test
    fun `the PayPal button asks for the chosen plan at PayPal and the wait is PayPal's`() = runBlocking {
        val holder = BillingHolder().apply { update(free.copy(paypal = true)); open(); plan = PlanKey.YEARLY }
        val backend = FakeBackend({ "https://paypal.example/c/1" }, free)
        val opened = mutableListOf<String>()
        holder.start(backend, "en", ProviderKey.PAYPAL) { opened += it }
        assertEquals(listOf("yearly/paypal/en"), backend.asked)
        assertEquals(listOf("https://paypal.example/c/1"), opened)
        assertEquals(SheetStage.WAITING, holder.stage)
        assertEquals(ProviderKey.PAYPAL, holder.provider)
        // Opening the sheet again starts from the main button's provider.
        holder.update(free.copy(paypal = true))
        holder.open()
        assertEquals(ProviderKey.WAFFO, holder.provider)
    }

    @Test
    fun `the core's PayPal fields decode and an older core's JSON still does`() {
        val raw = """{"offered":true,"paypal":true,"provider":"paypal","plan":"pro","status":"active","period_end":"2027-11-11T09:00:00Z"}"""
        val b = PageJson.decodeFromString<Billing>(raw)
        assertTrue(b.paypal && b.viaPayPal)
        val old = PageJson.decodeFromString<Billing>("""{"offered":true,"plan":"free"}""")
        assertFalse(old.paypal || old.viaPayPal)
    }

    @Test
    fun `the wordmark colors Pay and Pal and keeps the words around`() {
        val label = payPalLabel("Subscribe with PayPal")
        assertEquals("Subscribe with PayPal", label.text)
        assertEquals(listOf("Pay", "Pal"), label.spanStyles.filter { it.item.fontStyle != null }.map { label.text.substring(it.start, it.end) })
        assertEquals("用 PayPal 订阅", payPalLabel("用 PayPal 订阅").text)
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
