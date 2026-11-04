package syncserver

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
	"github.com/QingYunA/adhd-todo/internal/syncserver/billing"
)

// CodeBillingDisabled is the "code" of the 503 every billing route answers without a Waffo key.
const CodeBillingDisabled = "billing_disabled"

// Codes of the billing routes' refusals.
const (
	codeBadPlan           = "bad_plan"
	codeAlreadySubscribed = "already_subscribed"
	codeRateLimited       = "rate_limited"
	codeBillingUpstream   = "billing_upstream"
)

const (
	webhookMaxBody  = 1 << 20
	billingNotifyTO = 5 * time.Second
)

// EmailLookup finds the email of the account a primary sign-in subject belongs to, "" when it has
// none (a development token). Checkout pre-fills the page with it and the portal link is made for it.
type EmailLookup func(ctx context.Context, subject string) (string, error)

// EnableBilling turns on Withdo Pro (M16): the checkout, the plan and the Waffo webhook. Call it
// before serving. email may be nil. Without it every billing route answers 503 billing_disabled
// and GET /v1/config says "billing": false.
func (s *Server) EnableBilling(cfg billing.Config, email EmailLookup) error {
	svc, err := billing.New(cfg, billing.NewPGStore(s.pool), s.billingChanged)
	if err != nil {
		return err
	}
	s.billing, s.emailOf = svc, email
	s.llm.pro = svc.IsProSubject
	return nil
}

// billingChanged tells an account's devices that its plan changed, so open apps refresh at once.
func (s *Server) billingChanged(account int64) {
	ctx, cancel := context.WithTimeout(context.Background(), billingNotifyTO)
	defer cancel()
	var workspaceID int64
	if err := s.pool.QueryRow(ctx, "SELECT id FROM workspaces WHERE owner_id = $1", account).Scan(&workspaceID); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("billing: account=%d notify devices: %v", account, err)
		}
		return
	}
	s.hub.publishBilling(workspaceID)
}

// accountEmail is the email of the session's account, "" when unknown. Errors are logged, not
// returned: a missing email only costs the pre-filled page.
func (s *Server) accountEmail(ctx context.Context, sess *session) string {
	if s.emailOf == nil {
		return ""
	}
	var subject string
	if err := s.pool.QueryRow(ctx, "SELECT subject FROM users WHERE id = $1", sess.userID).Scan(&subject); err != nil {
		return ""
	}
	addr, err := s.emailOf(ctx, subject)
	if err != nil {
		log.Printf("billing: account=%d email lookup failed", sess.userID)
		return ""
	}
	return addr
}

// billingOff answers 503 billing_disabled and returns true when billing has no key.
func (s *Server) billingOff(w http.ResponseWriter) bool {
	if s.billing != nil {
		return false
	}
	plainError(w, http.StatusServiceUnavailable, CodeBillingDisabled)
	return true
}

// billingStatus is GET /v1/billing.
func (s *Server) billingStatus(w http.ResponseWriter, r *http.Request) {
	if s.billingOff(w) {
		return
	}
	sess := sessionFrom(r.Context())
	st, err := s.billing.Status(r.Context(), sess.userID, func() string { return s.accountEmail(r.Context(), sess) })
	if err != nil {
		writeError(w, err)
		return
	}
	out := syncproto.BillingResponse{
		Plan: st.Plan, Status: st.Status, TrialUsed: st.TrialUsed,
		Monthly: billing.PriceMonthly, Yearly: billing.PriceYearly, ManageURL: st.ManageURL,
	}
	if !st.PeriodEnd.IsZero() {
		out.PeriodEnd = st.PeriodEnd.UTC().Format(time.RFC3339)
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, out)
}

// billingCheckout is POST /v1/billing/checkout.
func (s *Server) billingCheckout(w http.ResponseWriter, r *http.Request) {
	if s.billingOff(w) {
		return
	}
	var req syncproto.CheckoutRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	sess := sessionFrom(r.Context())
	url, err := s.billing.Checkout(r.Context(), sess.userID, s.accountEmail(r.Context(), sess), req.Plan, req.Lang)
	switch {
	case errors.Is(err, billing.ErrUnknownPlan):
		plainError(w, http.StatusBadRequest, codeBadPlan)
	case errors.Is(err, billing.ErrAlreadySubscribed):
		plainError(w, http.StatusConflict, codeAlreadySubscribed)
	case errors.Is(err, billing.ErrRateLimited):
		plainError(w, http.StatusTooManyRequests, codeRateLimited)
	case errors.Is(err, billing.ErrUpstream):
		log.Printf("billing: account=%d checkout failed: %v", sess.userID, err)
		plainError(w, http.StatusBadGateway, codeBillingUpstream)
	case err != nil:
		writeError(w, err)
	default:
		writeJSON(w, http.StatusOK, syncproto.CheckoutResponse{URL: url})
	}
}

// billingWebhook is POST /v1/billing/waffo. It has no session: the signature is the credential.
// A genuine event is answered 200 whether or not it changed anything, so Waffo stops retrying;
// only a bad signature (401) or our own failure (500) makes it send the event again.
func (s *Server) billingWebhook(w http.ResponseWriter, r *http.Request) {
	if s.billingOff(w) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, webhookMaxBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		plainError(w, http.StatusBadRequest, "bad_request")
		return
	}
	ev, outcome, err := s.billing.HandleWebhook(r.Context(), r.Header.Get(billing.HeaderSignature), body)
	switch {
	case errors.Is(err, billing.ErrBadSignature), errors.Is(err, billing.ErrStaleSignature):
		log.Printf("billing: webhook rejected: %v", err)
		plainError(w, http.StatusUnauthorized, "bad_signature")
	case err != nil:
		log.Printf("billing: webhook event=%s failed: %v", ev.Type, err)
		plainError(w, http.StatusInternalServerError, "internal")
	default:
		account, _ := ev.Data.Account()
		log.Printf("billing: webhook event=%s account=%d outcome=%s", ev.Type, account, outcome)
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}
