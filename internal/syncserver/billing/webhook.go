package billing

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

// WebhookTolerance is how far a signature's timestamp may be from now. Waffo re-sends an event
// with its first timestamp and the last retry comes 31+ minutes after the first delivery.
const WebhookTolerance = 45 * time.Minute

// HeaderSignature is Waffo's webhook signature header: "t=<unix ms>,v1=<base64 signature>".
const HeaderSignature = "X-Waffo-Signature"

var (
	// ErrBadSignature says the webhook is not signed by Waffo (or not for the mode it claims).
	ErrBadSignature = errors.New("billing: webhook signature does not verify")
	// ErrStaleSignature says the signature is genuine but its timestamp is outside the tolerance.
	ErrStaleSignature = errors.New("billing: webhook timestamp is outside the tolerance")
)

// Event is the part of a webhook payload the entitlement rule reads.
type Event struct {
	ID        string    `json:"id"`
	Timestamp flexTime  `json:"timestamp"`
	Type      string    `json:"eventType"`
	EventID   string    `json:"eventId"`
	StoreID   string    `json:"storeId"`
	Mode      string    `json:"mode"`
	Data      EventData `json:"data"`
}

// EventData is the event's "data".
type EventData struct {
	OrderID            string                     `json:"orderId"`
	OrderStatus        string                     `json:"orderStatus"`
	OrderMetadata      map[string]json.RawMessage `json:"orderMetadata"`
	ProductName        string                     `json:"productName"`
	BillingPeriod      string                     `json:"billingPeriod"`
	PeriodNumber       *int                       `json:"periodNumber"`
	CurrentPeriodStart flexTime                   `json:"currentPeriodStart"`
	CurrentPeriodEnd   flexTime                   `json:"currentPeriodEnd"`
	PaymentStatus      string                     `json:"paymentStatus"`
}

// Account is the Withdo account the checkout was made for: metadata.account, a string or number.
func (d EventData) Account() (int64, bool) {
	raw, ok := d.OrderMetadata["account"]
	if !ok {
		return 0, false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		s = strings.TrimSpace(string(raw))
	}
	id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// dedupeKey is what identifies a delivery: the event "id", else its type and eventId.
func (e Event) dedupeKey() string {
	if e.ID != "" {
		return e.ID
	}
	if e.EventID != "" {
		return e.Type + ":" + e.EventID
	}
	return ""
}

// flexTime reads an ISO 8601 date or date-time; an empty or unreadable value is the zero time.
type flexTime struct{ time.Time }

func (t *flexTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil || s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(layout, s); err == nil {
			t.Time = parsed.UTC()
			return nil
		}
	}
	return nil
}

// verifier checks webhook signatures with the public key of the payload's mode.
type verifier struct {
	keys map[string]*rsa.PublicKey
	now  func() time.Time
}

// verify checks header against body and returns the decoded event. The key is chosen by the
// payload's "mode"; the signature covers "<t>.<raw body>" (RSA-SHA256, PKCS#1 v1.5). Nothing in
// the returned error carries the body or the signature.
func (v verifier) verify(header string, body []byte) (Event, error) {
	var ev Event
	t, sig, ok := parseSignatureHeader(header)
	if !ok {
		return ev, ErrBadSignature
	}
	if err := json.Unmarshal(body, &ev); err != nil {
		return ev, ErrBadSignature
	}
	key := v.keys[ev.Mode]
	if key == nil {
		return ev, ErrBadSignature
	}
	sum := sha256.Sum256([]byte(strconv.FormatInt(t, 10) + "." + string(body)))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], sig); err != nil {
		return ev, ErrBadSignature
	}
	age := v.now().Sub(time.UnixMilli(t))
	if age > WebhookTolerance || age < -WebhookTolerance {
		return ev, ErrStaleSignature
	}
	return ev, nil
}

// parseSignatureHeader splits "t=<ms>,v1=<b64>" (pairs split at the first "=").
func parseSignatureHeader(header string) (t int64, sig []byte, ok bool) {
	var haveT bool
	for _, part := range strings.Split(header, ",") {
		name, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found {
			continue
		}
		switch name {
		case "t":
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return 0, nil, false
			}
			t, haveT = n, true
		case "v1":
			raw, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				return 0, nil, false
			}
			sig = raw
		}
	}
	return t, sig, haveT && len(sig) > 0
}
