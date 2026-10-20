package core

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestStartSyncWhileLinkRunsStartsNoRunner is A3-fix 4: the runner must wait for SignIn and Link to
// finish leasing ids, so StartSync during a slow device create is a no-op.
func TestStartSyncWhileLinkRunsStartsNoRunner(t *testing.T) {
	f := newAccountFixture(t)
	phone, _ := f.phone(t)
	createPhoneTask(t, phone, "Phone task")

	reached := make(chan struct{})
	release := make(chan struct{})
	var reachedOnce, releaseOnce sync.Once
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/devices" {
			reachedOnce.Do(func() { close(reached) })
			<-release
		}
		f.sync.Config.Handler.ServeHTTP(w, r)
	}))
	defer slow.Close()
	defer releaseOnce.Do(func() { close(release) })
	if err := phone.SyncConfigure(slow.URL, f.oidc.URL, "mobile-client", ""); err != nil {
		t.Fatalf("SyncConfigure: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		state, err := phone.SignIn("ada@example.com", "secret", "phone")
		if err == nil && !strings.Contains(state, `"choose"`) {
			err = fmt.Errorf("SignIn = %s, want choose", state)
		}
		done <- err
	}()

	select {
	case <-reached:
	case <-time.After(10 * time.Second):
		t.Fatal("the link never reached the server's device create")
	}

	phone.mu.Lock()
	links := phone.links
	phone.mu.Unlock()
	if links == 0 {
		t.Fatal("the link is not marked as running")
	}
	if err := phone.StartSync(); err != nil {
		t.Fatalf("StartSync during the link: %v", err)
	}
	phone.mu.Lock()
	runner := phone.runner
	phone.mu.Unlock()
	if runner != nil {
		t.Fatal("StartSync started a runner while SignIn was still linking")
	}

	releaseOnce.Do(func() { close(release) })
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("SignIn after the slow link: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("SignIn did not return after the device create was released")
	}
}
