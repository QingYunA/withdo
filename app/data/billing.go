package data

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// Withdo Pro on this computer (M16): Settings > Model's plan pill, the footer row of the official
// card, the upgrade sheet and the composer's quiet line all read BillingInfo. It sits beside
// OfficialInfo and is kept the same way: asked for in the background, never on the UI thread,
// and asked again on the runner's "billing" event and when the Settings window comes back.

// The plans StartCheckout takes.
const (
	PlanMonthly = syncclient.PlanMonthly
	PlanYearly  = syncclient.PlanYearly
)

// The failures StartCheckout and ManageSubscription word for the person; any other error is
// "could not open the payment page, try again".
var (
	// ErrAlreadySubscribed says the account is Pro already (maybe from another device): the plan is
	// asked for again and the sheet closes.
	ErrAlreadySubscribed = errors.New("data: the account already has Withdo Pro")
	// ErrCheckoutThrottled says too many payment pages were asked for; try again in a while.
	ErrCheckoutThrottled = errors.New("data: too many checkout attempts")
)

// BillingInfo is the plan in one frame.
type BillingInfo struct {
	// Offered says the server sells Pro and an account is signed in: the card shows a plan. False
	// for a server that does not, signed out, and until the server has answered.
	Offered bool
	// Known says GET /v1/billing answered: Status, PeriodEnd, TrialUsed and the prices are real.
	Known bool
	// Pro says the account is on Withdo Pro now. Before Known it follows the quota's plan.
	Pro bool
	// Status is the subscription's: trialing, active, past_due, canceling, canceled; empty for an
	// account that never subscribed.
	Status string
	// Interval is "monthly" or "yearly", empty when the server does not say.
	Interval  string
	PeriodEnd time.Time
	// TrialUsed says the 7 days were taken: the sheet offers Subscribe, not the trial.
	TrialUsed bool
	// Monthly and Yearly are the prices as the server words them ("$6.99", "$58.72").
	Monthly, Yearly string
}

// Subscribed says the account has a subscription to manage (a trial, a paid one, or one ending).
func (b BillingInfo) Subscribed() bool { return b.Pro && b.Status != "" }

// Trialing says the account is in its free days.
func (b BillingInfo) Trialing() bool { return b.Status == "trialing" }

// Canceling says the subscription ends at PeriodEnd and does not renew.
func (b BillingInfo) Canceling() bool { return b.Status == "canceling" }

// BillingSource is where Settings and the composer learn about Pro. *Real implements it.
type BillingSource interface {
	// BillingInfo is the current plan. It never blocks on the network: it starts a refresh in the
	// background when its answer is old.
	BillingInfo() BillingInfo
	// RefreshBilling asks the server again, soon.
	RefreshBilling()
	// StartCheckout asks for the payment page of plan (PlanMonthly or PlanYearly), in the app's
	// language lang, and returns its URL for the browser.
	StartCheckout(ctx context.Context, plan, lang string) (string, error)
	// ManageURL is a fresh link to the customer portal. It lives 15 minutes, so it is asked for on
	// every click and opened at once.
	ManageURL(ctx context.Context) (string, error)
}

// billingTask is the state behind BillingInfo.
type billingTask struct {
	mu        sync.Mutex
	server    string
	signedIn  bool
	known     bool // offered is the server's answer
	offered   bool
	bill      syncclient.Billing
	haveBill  bool
	refreshAt time.Time // no refresh before this
	busy      bool
	again     bool // a refresh was asked for while one ran
}

const (
	billingRefreshEvery = 5 * time.Minute
	billingRetryAfter   = 30 * time.Second
	billingRefreshTO    = 15 * time.Second
)

var _ BillingSource = (*Real)(nil)

// BillingInfo implements BillingSource.
func (r *Real) BillingInfo() BillingInfo {
	server, signedIn := r.officialSession()
	b := &r.bill
	b.mu.Lock()
	if b.server != server || b.signedIn != signedIn { // another server or account: ask again
		b.server, b.signedIn, b.known, b.haveBill, b.refreshAt = server, signedIn, false, false, time.Time{}
	}
	due := !b.busy && !time.Now().Before(b.refreshAt)
	b.busy = b.busy || due
	info := b.info(r.llm.plan())
	b.mu.Unlock()
	if due {
		go r.refreshBilling()
	}
	return info
}

// info is the plan for the state held; quotaPlan is what the quota said ("" if nothing yet). Call
// with mu held.
func (b *billingTask) info(quotaPlan string) BillingInfo {
	if !b.known || !b.offered || !b.signedIn {
		return BillingInfo{}
	}
	out := BillingInfo{Offered: true, Pro: quotaPlan == "pro"}
	if b.haveBill {
		bl := b.bill
		out = BillingInfo{
			Offered: true, Known: true, Pro: bl.Pro(), Status: bl.Status, Interval: bl.Interval, PeriodEnd: bl.PeriodEnd,
			TrialUsed: bl.TrialUsed, Monthly: bl.Monthly, Yearly: bl.Yearly,
		}
	}
	return out
}

// plan is the plan the last quota named, "" until it did.
func (o *officialTask) plan() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.haveQuota {
		return ""
	}
	return o.quota.Plan
}

// RefreshBilling implements BillingSource.
func (r *Real) RefreshBilling() {
	b := &r.bill
	b.mu.Lock()
	start := !b.busy
	if b.busy {
		b.again = true // the answer in flight may be older than what was asked about
	}
	b.busy = true
	b.mu.Unlock()
	if start {
		go r.refreshBilling()
	}
}

// refreshBilling asks the server whether it sells Pro and, signed in, for the plan. The caller has
// set busy, which this clears.
func (r *Real) refreshBilling() {
	b := &r.bill
	ctx, cancel := context.WithTimeout(context.Background(), billingRefreshTO)
	defer cancel()
	server, signedIn := r.officialSession()
	offered, bill, haveBill, err := r.askBilling(ctx, server, signedIn)
	b.mu.Lock()
	wasPro := b.haveBill && b.bill.Pro()
	switch {
	case b.server != server || b.signedIn != signedIn: // the session changed while asking: that answer is old
		b.refreshAt = time.Time{}
	case err != nil:
		b.refreshAt = time.Now().Add(billingRetryAfter)
	default:
		b.known, b.offered, b.bill, b.haveBill = true, offered, bill, haveBill
		b.refreshAt = time.Now().Add(billingRefreshEvery)
	}
	again := b.again
	b.again, b.busy = false, false
	changed := err == nil && haveBill && bill.Pro() != wasPro
	b.mu.Unlock()
	if changed {
		r.RefreshOfficial() // the day's limit follows the plan
	}
	r.syncChanged() // the card and the sheet follow
	if again {
		r.RefreshBilling()
	}
}

func (r *Real) askBilling(ctx context.Context, server string, signedIn bool) (offered bool, bill syncclient.Billing, haveBill bool, err error) {
	cfg, err := syncclient.FetchConfig(ctx, server, nil)
	if err != nil {
		return false, bill, false, err
	}
	if !cfg.Billing || !signedIn {
		return cfg.Billing, bill, false, nil
	}
	client, err := r.officialClient()
	if err != nil {
		return cfg.Billing, bill, false, err
	}
	bill, err = client.Billing(ctx)
	switch {
	case err == nil:
		return true, bill, true, nil
	case errors.Is(err, syncclient.ErrBillingOff):
		return false, bill, false, nil
	case errors.Is(err, syncclient.ErrBillingNotSignedIn):
		return true, bill, false, nil // the plan is known when the session is
	}
	return false, bill, false, err
}

// StartCheckout implements BillingSource.
func (r *Real) StartCheckout(ctx context.Context, plan, lang string) (string, error) {
	client, err := r.officialClient()
	if err != nil {
		return "", err
	}
	url, err := client.Checkout(ctx, plan, syncclient.ProviderWaffo, lang)
	switch {
	case errors.Is(err, syncclient.ErrAlreadySubscribed):
		r.RefreshBilling()
		return "", ErrAlreadySubscribed
	case errors.Is(err, syncclient.ErrCheckoutThrottled):
		return "", ErrCheckoutThrottled
	}
	return url, err
}

// ManageURL implements BillingSource.
func (r *Real) ManageURL(ctx context.Context) (string, error) {
	client, err := r.officialClient()
	if err != nil {
		return "", err
	}
	bill, err := client.Billing(ctx)
	if err != nil {
		return "", err
	}
	if bill.ManageURL == "" {
		return "", errors.New("data: the account has no subscription to manage")
	}
	return bill.ManageURL, nil
}

// PerMonth is the monthly share of a yearly price in the server's own words: "$58.72" is "$4.89".
// Empty when the price is not a symbol and a number.
func PerMonth(yearly string) string {
	prefix, amount, ok := splitPrice(yearly)
	if !ok {
		return ""
	}
	return prefix + strconv.FormatFloat(math.Floor(amount/12*100+0.5)/100, 'f', 2, 64)
}

// SavingPercent is how much cheaper the yearly price is than twelve monthly ones, rounded: 30 for
// "$6.99" and "$58.72". Zero when either price cannot be read or the year saves nothing.
func SavingPercent(monthly, yearly string) int {
	_, m, ok1 := splitPrice(monthly)
	_, y, ok2 := splitPrice(yearly)
	if !ok1 || !ok2 || m <= 0 || y >= m*12 {
		return 0
	}
	return int(math.Floor((1-y/(m*12))*100 + 0.5))
}

// splitPrice reads "$58.72" into its currency prefix and amount.
func splitPrice(s string) (prefix string, amount float64, ok bool) {
	s = strings.TrimSpace(s)
	i := strings.IndexAny(s, "0123456789")
	if i < 0 {
		return "", 0, false
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s[i:], ",", ""), 64)
	if err != nil {
		return "", 0, false
	}
	return s[:i], v, true
}
