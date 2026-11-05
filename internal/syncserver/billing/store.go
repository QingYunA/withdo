package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoAccount says an event names an account that does not exist.
var ErrNoAccount = errors.New("billing: no such account")

// eventKeep is how long a processed event id is remembered for deduplication. Waffo retries for
// about an hour and PayPal for three days; the rest is slack.
const eventKeep = 30 * 24 * time.Hour

// Store keeps the accounts' subscription rows. PGStore is the Postgres one; tests use a fake.
type Store interface {
	// Get reads an account's row.
	Get(ctx context.Context, account int64) (Subscription, bool, error)
	// GetBySubject reads the row of the account a sign-in subject (a user's own or an alias) maps to.
	GetBySubject(ctx context.Context, subject string) (Subscription, bool, error)
	// Apply handles one event atomically: it records eventID and, unless that id was seen before
	// (dup), reads the account's row, runs update on it and saves the result when update says it
	// changed. It returns ErrNoAccount, recording nothing, when the account does not exist.
	Apply(ctx context.Context, eventID, eventType string, account int64,
		update func(cur Subscription, exists bool) (next Subscription, changed bool)) (before, after Subscription, dup, saved bool, err error)
	// SetPendingPayPal remembers the PayPal subscription a checkout made for the account, replacing
	// the one it remembered before.
	SetPendingPayPal(ctx context.Context, account int64, subscriptionID string, at time.Time) error
	// PendingPayPal reads the account's remembered PayPal subscription and when it was made.
	PendingPayPal(ctx context.Context, account int64) (subscriptionID string, at time.Time, ok bool, err error)
	// ClearPendingPayPal forgets the account's remembered subscription, but only if it is still
	// subscriptionID: a newer checkout's is kept.
	ClearPendingPayPal(ctx context.Context, account int64, subscriptionID string) error
}

// PGStore is Store over the billing_subscriptions, billing_events and billing_paypal_pending tables (migrations 005 to 008).
type PGStore struct{ pool *pgxpool.Pool }

// NewPGStore returns the Postgres store.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

const subscriptionColumns = `user_id, order_id, product_name, billing_interval, status, period_start, period_end, trial_used, mode, event_at, provider`

type rowScanner interface{ Scan(dest ...any) error }

func scanSubscription(row rowScanner) (Subscription, error) {
	var s Subscription
	var start, end *time.Time
	if err := row.Scan(&s.Account, &s.OrderID, &s.ProductName, &s.Interval, &s.Status, &start, &end, &s.TrialUsed, &s.Mode, &s.EventAt, &s.Provider); err != nil {
		return Subscription{}, err
	}
	if start != nil {
		s.PeriodStart = start.UTC()
	}
	if end != nil {
		s.PeriodEnd = end.UTC()
	}
	s.EventAt = s.EventAt.UTC()
	return s, nil
}

// Get implements Store.
func (p *PGStore) Get(ctx context.Context, account int64) (Subscription, bool, error) {
	s, err := scanSubscription(p.pool.QueryRow(ctx,
		`SELECT `+subscriptionColumns+` FROM billing_subscriptions WHERE user_id = $1`, account))
	return orNone(s, err)
}

// GetBySubject implements Store.
func (p *PGStore) GetBySubject(ctx context.Context, subject string) (Subscription, bool, error) {
	s, err := scanSubscription(p.pool.QueryRow(ctx,
		`SELECT `+subscriptionColumns+` FROM billing_subscriptions WHERE user_id =
			(SELECT id FROM users WHERE subject = $1
			 UNION ALL SELECT user_id FROM subject_aliases WHERE subject = $1 LIMIT 1)`, subject))
	return orNone(s, err)
}

func orNone(s Subscription, err error) (Subscription, bool, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, false, nil
	}
	if err != nil {
		return Subscription{}, false, fmt.Errorf("billing: read the subscription: %w", err)
	}
	return s, true, nil
}

// Apply implements Store.
func (p *PGStore) Apply(ctx context.Context, eventID, eventType string, account int64,
	update func(cur Subscription, exists bool) (Subscription, bool)) (before, after Subscription, dup, saved bool, err error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return before, after, false, false, fmt.Errorf("billing: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var one int
	if err := tx.QueryRow(ctx, `SELECT 1 FROM users WHERE id = $1`, account).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return before, after, false, false, ErrNoAccount
		}
		return before, after, false, false, fmt.Errorf("billing: find the account: %w", err)
	}
	tag, err := tx.Exec(ctx, `INSERT INTO billing_events (id, event_type) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING`, eventID, eventType)
	if err != nil {
		return before, after, false, false, fmt.Errorf("billing: record the event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return before, after, true, false, nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM billing_events WHERE received_at < $1`, time.Now().Add(-eventKeep)); err != nil {
		return before, after, false, false, fmt.Errorf("billing: prune events: %w", err)
	}
	cur, err := scanSubscription(tx.QueryRow(ctx,
		`SELECT `+subscriptionColumns+` FROM billing_subscriptions WHERE user_id = $1 FOR UPDATE`, account))
	exists := true
	if errors.Is(err, pgx.ErrNoRows) {
		cur, exists = Subscription{}, false
	} else if err != nil {
		return before, after, false, false, fmt.Errorf("billing: read the subscription: %w", err)
	}
	next, changed := update(cur, exists)
	if changed {
		_, err := tx.Exec(ctx, `INSERT INTO billing_subscriptions
			(user_id, order_id, product_name, billing_interval, status, period_start, period_end, trial_used, mode, event_at, provider, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now())
			ON CONFLICT (user_id) DO UPDATE SET order_id = EXCLUDED.order_id, product_name = EXCLUDED.product_name,
				billing_interval = EXCLUDED.billing_interval,
				status = EXCLUDED.status, period_start = EXCLUDED.period_start, period_end = EXCLUDED.period_end,
				trial_used = EXCLUDED.trial_used, mode = EXCLUDED.mode, event_at = EXCLUDED.event_at,
				provider = EXCLUDED.provider, updated_at = now()`,
			account, next.OrderID, next.ProductName, next.Interval, next.Status, nullTime(next.PeriodStart), nullTime(next.PeriodEnd),
			next.TrialUsed, next.Mode, next.EventAt, next.providerOrDefault())
		if err != nil {
			return before, after, false, false, fmt.Errorf("billing: save the subscription: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return before, after, false, false, fmt.Errorf("billing: commit: %w", err)
	}
	if !changed {
		next = cur
	}
	return cur, next, false, changed, nil
}

// SetPendingPayPal implements Store.
func (p *PGStore) SetPendingPayPal(ctx context.Context, account int64, subscriptionID string, at time.Time) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO billing_paypal_pending (user_id, subscription_id, created_at) VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET subscription_id = EXCLUDED.subscription_id, created_at = EXCLUDED.created_at`,
		account, subscriptionID, at)
	if err != nil {
		return fmt.Errorf("billing: remember the PayPal checkout: %w", err)
	}
	return nil
}

// PendingPayPal implements Store.
func (p *PGStore) PendingPayPal(ctx context.Context, account int64) (string, time.Time, bool, error) {
	var id string
	var at time.Time
	err := p.pool.QueryRow(ctx, `SELECT subscription_id, created_at FROM billing_paypal_pending WHERE user_id = $1`, account).Scan(&id, &at)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, false, nil
	}
	if err != nil {
		return "", time.Time{}, false, fmt.Errorf("billing: read the PayPal checkout: %w", err)
	}
	return id, at.UTC(), true, nil
}

// ClearPendingPayPal implements Store.
func (p *PGStore) ClearPendingPayPal(ctx context.Context, account int64, subscriptionID string) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM billing_paypal_pending WHERE user_id = $1 AND subscription_id = $2`, account, subscriptionID)
	if err != nil {
		return fmt.Errorf("billing: forget the PayPal checkout: %w", err)
	}
	return nil
}

func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
