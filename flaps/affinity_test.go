package flaps

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

type memAffinityStore struct {
	mu     sync.Mutex
	tokens map[string]string
}

func (s *memAffinityStore) LoadAffinity(baseURL, app string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tokens[baseURL+" "+app]
}

func (s *memAffinityStore) StoreAffinity(baseURL, app, token string, _ time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tokens == nil {
		s.tokens = map[string]string{}
	}
	s.tokens[baseURL+" "+app] = token
}

func testAffinityToken(t *testing.T, app, instance string, expires time.Time) string {
	t.Helper()
	payload, err := json.Marshal(affinityClaims{AppName: app, InstanceID: instance, ExpiresAt: expires})
	if err != nil {
		t.Fatal(err)
	}

	return base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

// TestAffinityStoreCarriesTokenAcrossClients is the cross-process case: one
// client creates an app and a second, with no shared cookie jar, lists its
// machines.
func TestAffinityStoreCarriesTokenAcrossClients(t *testing.T) {
	token := testAffinityToken(t, "app-a", "flaps-1", time.Now().Add(time.Minute))

	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/apps":
			w.Header().Set(AffinityHeader, token)
			w.WriteHeader(http.StatusCreated)
		case "/v1/apps/app-a/machines":
			gotHeader = r.Header.Get(AffinityHeader)
			_, _ = w.Write([]byte("[]"))
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	defer server.Close()
	t.Setenv("FLY_FLAPS_BASE_URL", server.URL)

	store := &memAffinityStore{}
	ctx := context.Background()

	creator, err := NewWithOptions(ctx, NewClientOpts{AffinityStore: store})
	if err != nil {
		t.Fatal(err)
	}
	if err := creator._sendRequest(ctx, http.MethodPost, "/apps", map[string]string{"app_name": "app-a"}, nil, nil); err != nil {
		t.Fatal(err)
	}

	lister, err := NewWithOptions(ctx, NewClientOpts{AffinityStore: store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lister.List(ctx, "app-a", ""); err != nil {
		t.Fatal(err)
	}
	if gotHeader != token {
		t.Fatalf("%s = %q, want the token stored by the first client", AffinityHeader, gotHeader)
	}
}

func TestAttachAffinity(t *testing.T) {
	base, _ := url.Parse("https://flaps.example")
	live := testAffinityToken(t, "app-a", "flaps-1", time.Now().Add(time.Minute))
	nearlyExpired := testAffinityToken(t, "app-a", "flaps-1", time.Now().Add(time.Second))
	otherApp := testAffinityToken(t, "app-b", "flaps-1", time.Now().Add(time.Minute))

	for _, tc := range []struct {
		name   string
		stored string
		path   string
		preset string
		want   string
	}{
		{name: "app scoped", stored: live, path: "/v1/apps/app-a/machines", want: live},
		{name: "app itself", stored: live, path: "/v1/apps/app-a", want: live},
		{name: "not app scoped", stored: live, path: "/v1/apps", want: ""},
		{name: "nearly expired", stored: nearlyExpired, path: "/v1/apps/app-a/machines", want: ""},
		{name: "token for another app", stored: otherApp, path: "/v1/apps/app-a/machines", want: ""},
		{name: "unparseable", stored: "garbage", path: "/v1/apps/app-a/machines", want: ""},
		{name: "caller set header", stored: live, path: "/v1/apps/app-a/machines", preset: "mine", want: "mine"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &memAffinityStore{}
			store.StoreAffinity(base.String(), "app-a", tc.stored, time.Time{})
			c := &Client{baseUrl: base, affinity: store}

			req, _ := http.NewRequest(http.MethodGet, base.String()+tc.path, nil)
			if tc.preset != "" {
				req.Header.Set(AffinityHeader, tc.preset)
			}
			c.attachAffinity(req)
			if got := req.Header.Get(AffinityHeader); got != tc.want {
				t.Fatalf("header = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCaptureAffinityIgnoresMismatchedApp(t *testing.T) {
	base, _ := url.Parse("https://flaps.example")
	store := &memAffinityStore{}
	c := &Client{baseUrl: base, affinity: store}

	req, _ := http.NewRequest(http.MethodGet, base.String()+"/v1/apps/app-a", nil)
	resp := &http.Response{Header: http.Header{}, Request: req}
	resp.Header.Set(AffinityHeader, testAffinityToken(t, "app-b", "flaps-1", time.Now().Add(time.Minute)))
	c.captureAffinity(resp)

	if got := store.LoadAffinity(base.String(), "app-b"); got != "" {
		t.Fatalf("stored token for app-b from a response to app-a: %q", got)
	}
}

func TestAppNameFromPath(t *testing.T) {
	for path, want := range map[string]string{
		"/v1/apps/app-a":                     "app-a",
		"/v1/apps/app-a/machines/123":        "app-a",
		"/v1/apps":                           "",
		"/v1/apps/":                          "",
		"/v1/platform/regions":               "",
		"/v1/postgres/clusters/abc/apps/foo": "",
	} {
		u, _ := url.Parse("https://flaps.example" + path)
		if got := appNameFromPath(u); got != want {
			t.Errorf("appNameFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}
