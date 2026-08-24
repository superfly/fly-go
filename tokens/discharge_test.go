package tokens

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/superfly/macaroon"
	"github.com/superfly/macaroon/flyio"
	"github.com/superfly/macaroon/tp"
)

// TestDischargePollingBackoff guards the cap on the discharge polling
// interval. The tp client's own backoff doubles without bound, so it stops
// asking the third party long before it runs out of time and a discharge that
// becomes ready in the meantime isn't collected until the next poll, if there
// is one. Interactive discharges wait on a person finishing a login, so the
// interval has to stay short enough to notice when they do.
func TestDischargePollingBackoff(t *testing.T) {
	var (
		bo      time.Duration
		elapsed time.Duration
	)

	for i := 0; i < 200; i++ {
		bo = dischargePollingBackoff(bo)

		if bo > maxDischargePollBackoff {
			t.Fatalf("poll %d: backoff %s exceeds cap %s", i, bo, maxDischargePollBackoff)
		}
		if bo <= 0 {
			t.Fatalf("poll %d: non-positive backoff %s", i, bo)
		}

		elapsed += bo
	}

	// with the cap in place, the whole default timeout is covered by polls
	// rather than by one long sleep.
	if elapsed < DefaultDischargeTimeout {
		t.Fatalf("200 polls only cover %s, less than the %s default timeout", elapsed, DefaultDischargeTimeout)
	}
}

// TestUpdateDischarge exercises Update against a real third party that answers
// with a poll URL and only discharges later, which is what an interactive
// login looks like from the client's side.
func TestUpdateDischarge(t *testing.T) {
	t.Run("collects a discharge that arrives between polls", func(t *testing.T) {
		// the discharge lands after the fourth poll of the capped schedule
		// (0, 100ms, 300ms, 700ms, ...) and well before the deadline. An
		// unbounded backoff would already be sleeping past the deadline by
		// then and would report a timeout for a discharge that was ready.
		hdr := newTestThirdParty(t, 1200*time.Millisecond)

		toks := Parse(hdr)

		updated, err := toks.Update(context.Background(), WithDischargeTimeout(2500*time.Millisecond))
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if !updated {
			t.Fatal("expected tokens to be updated with a discharge")
		}
		if n := len(Parse(toks.All()).macaroons); n != 2 {
			t.Fatalf("expected permission and discharge tokens, got %d macaroons", n)
		}
	})

	t.Run("gives up after the configured timeout", func(t *testing.T) {
		// never discharged: Update can only stop when its own deadline passes.
		hdr := newTestThirdParty(t, 0)

		toks := Parse(hdr)

		start := time.Now()
		_, err := toks.Update(context.Background(), WithDischargeTimeout(500*time.Millisecond))
		took := time.Since(start)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected a deadline error, got %v", err)
		}
		if took > 5*time.Second {
			t.Fatalf("Update took %s: the timeout isn't coming from WithDischargeTimeout", took)
		}
	})
}

// newTestThirdParty stands up a third party that responds to discharge
// requests with a poll URL, and discharges after the given delay if it is
// non-zero. It returns an authorization header holding a permission token with
// a third party caveat for that server.
func newTestThirdParty(tb testing.TB, dischargeAfter time.Duration) string {
	tb.Helper()

	var (
		thirdParty *tp.TP
		discharged = make(chan string, 1)
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch path := r.URL.EscapedPath(); {
		case path == tp.InitPath:
			thirdParty.InitRequestMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				discharged <- thirdParty.RespondPoll(w, r)
			})).ServeHTTP(w, r)
		case strings.HasPrefix(path, tp.PollPathPrefix):
			thirdParty.HandlePollRequest(w, r)
		default:
			tb.Errorf("unexpected request to %s", path)
		}
	}))
	tb.Cleanup(server.Close)

	store, err := tp.NewMemoryStore(tp.PrefixMunger("/user/"), 100)
	if err != nil {
		tb.Fatalf("NewMemoryStore: %v", err)
	}

	thirdParty = &tp.TP{
		Location: server.URL,
		Key:      macaroon.NewEncryptionKey(),
		Store:    store,
	}

	if dischargeAfter > 0 {
		go func() {
			pollSecret := <-discharged
			time.Sleep(dischargeAfter)

			if err := thirdParty.DischargePoll(context.Background(), pollSecret); err != nil {
				tb.Errorf("DischargePoll: %v", err)
			}
		}()
	}

	m, err := macaroon.New([]byte("kid"), flyio.LocationPermission, macaroon.NewSigningKey())
	if err != nil {
		tb.Fatalf("macaroon.New: %v", err)
	}
	if err := m.Add3P(thirdParty.Key, thirdParty.Location); err != nil {
		tb.Fatalf("Add3P: %v", err)
	}

	tok, err := m.Encode()
	if err != nil {
		tb.Fatalf("Encode: %v", err)
	}

	return macaroon.ToAuthorizationHeader(tok)
}
