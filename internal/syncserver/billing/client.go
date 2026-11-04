package billing

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is Waffo Pancake's API root. The key, not the host, picks test or production.
const DefaultBaseURL = "https://api.waffo.ai"

const (
	pathCheckout     = "/v1/actions/checkout/create-session"
	pathSessionToken = "/v1/actions/auth/issue-session-token"
	apiTimeout       = 15 * time.Second
	apiBodyMax       = 1 << 20
)

// ErrUpstream says Waffo refused a request or could not be reached. The text carries the status
// and Waffo's error message, never a request body, key or signature.
var ErrUpstream = errors.New("billing: the payment provider failed")

// upstreamError is an answer of Waffo with a status other than 2xx; it is an ErrUpstream.
type upstreamError struct {
	status  int
	message string
}

func (e *upstreamError) Error() string {
	return fmt.Sprintf("%s: status %d: %s", ErrUpstream.Error(), e.status, e.message)
}

func (e *upstreamError) Unwrap() error { return ErrUpstream }

// api signs and sends the calls Withdo makes to Waffo.
type api struct {
	base       string
	merchantID string
	key        *rsa.PrivateKey
	http       *http.Client
	now        func() time.Time
}

// sign is X-Signature: Base64(RSA-SHA256, PKCS#1 v1.5, over
// METHOD "\n" PATH "\n" TIMESTAMP "\n" Base64(SHA-256(body))).
func (a *api) sign(method, path, timestamp string, body []byte) (string, error) {
	sum := sha256.Sum256(body)
	canonical := method + "\n" + path + "\n" + timestamp + "\n" + base64.StdEncoding.EncodeToString(sum[:])
	digest := sha256.Sum256([]byte(canonical))
	sig, err := rsa.SignPKCS1v15(rand.Reader, a.key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("billing: sign the request: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// post sends one signed JSON POST and decodes the "data" of Waffo's {"data":…,"errors":[…]} answer.
func (a *api) post(ctx context.Context, path string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("billing: encode the request: %w", err)
	}
	ts := strconv.FormatInt(a.now().Unix(), 10)
	sig, err := a.sign(http.MethodPost, path, ts, body)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.base+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-Id", a.merchantID)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Signature", sig)
	resp, err := a.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %v", ErrUpstream, ctx.Err())
		}
		return fmt.Errorf("%w: unreachable", ErrUpstream)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, apiBodyMax))
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	_ = json.Unmarshal(raw, &envelope)
	if resp.StatusCode/100 != 2 {
		msg := "no message"
		if len(envelope.Errors) > 0 && envelope.Errors[0].Message != "" {
			msg = envelope.Errors[0].Message
		}
		return &upstreamError{status: resp.StatusCode, message: strings.TrimSpace(msg)}
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("%w: unreadable answer", ErrUpstream)
	}
	return nil
}

type checkoutRequest struct {
	ProductID               string            `json:"productId"`
	Currency                string            `json:"currency"`
	BuyerEmail              string            `json:"buyerEmail,omitempty"`
	Metadata                map[string]string `json:"metadata"`
	OrderMerchantExternalID string            `json:"orderMerchantExternalId"`
	WithTrial               bool              `json:"withTrial"`
	SuccessURL              string            `json:"successUrl"`
	Language                string            `json:"language,omitempty"`
}

// createCheckout calls create-session and returns the hosted checkout page's URL.
func (a *api) createCheckout(ctx context.Context, in checkoutRequest) (string, error) {
	var out struct {
		CheckoutURL string `json:"checkoutUrl"`
	}
	if err := a.post(ctx, pathCheckout, in, &out); err != nil {
		return "", err
	}
	if out.CheckoutURL == "" {
		return "", fmt.Errorf("%w: no checkout URL", ErrUpstream)
	}
	return out.CheckoutURL, nil
}

// portalLink mints a customer-portal session for the buyer and returns the URL with its token in
// the fragment. The link is a password for 15 minutes: callers return it once and never keep it.
func (a *api) portalLink(ctx context.Context, storeID, buyerIdentity string) (string, error) {
	var out struct {
		Token     string `json:"token"`
		PortalURL string `json:"portalUrl"`
	}
	in := map[string]string{"storeId": storeID, "buyerIdentity": buyerIdentity, "purpose": "portal"}
	if err := a.post(ctx, pathSessionToken, in, &out); err != nil {
		return "", err
	}
	if out.PortalURL == "" || out.Token == "" {
		return "", fmt.Errorf("%w: no portal link", ErrUpstream)
	}
	return out.PortalURL + "#token=" + out.Token, nil
}
