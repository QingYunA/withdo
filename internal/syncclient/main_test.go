package syncclient

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestMain removes the real retry pauses from the package's tests. The delays themselves are
// checked by TestRetryTransientTriesThreeTimes; every other test only cares that a retry happened.
func TestMain(m *testing.M) {
	retrySleep = func(context.Context, time.Duration) error { return nil }
	os.Exit(m.Run())
}
