package flaps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewWithOptionsSetsCookieJar(t *testing.T) {
	client, err := NewWithOptions(context.Background(), NewClientOpts{})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	if client.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
	if client.httpClient.Jar == nil {
		t.Fatal("httpClient.Jar is nil")
	}
}

func TestFlapsClientPersistsPathScopedCookiesPerApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/apps":
			if r.Method != http.MethodPost {
				http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
				return
			}

			var req struct {
				AppName string `json:"app_name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}

			switch req.AppName {
			case "app-a":
				http.SetCookie(w, &http.Cookie{
					Name:  "flaps_app_affinity",
					Value: "creator-node-a",
					Path:  "/v1/apps/app-a",
				})
			case "app-b":
				http.SetCookie(w, &http.Cookie{
					Name:  "flaps_app_affinity",
					Value: "creator-node-b",
					Path:  "/v1/apps/app-b",
				})
			default:
				http.Error(w, "unexpected app", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "/v1/apps/app-a/status":
			assertCookieHeader(t, r, "flaps_app_affinity=creator-node-a")
			w.WriteHeader(http.StatusOK)
		case "/v1/apps/app-b/status":
			assertCookieHeader(t, r, "flaps_app_affinity=creator-node-b")
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("FLY_FLAPS_BASE_URL", server.URL)

	client, err := NewWithOptions(context.Background(), NewClientOpts{})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}

	ctx := context.Background()
	for _, req := range []struct {
		method string
		path   string
		body   interface{}
	}{
		{method: http.MethodPost, path: "/apps", body: map[string]string{"app_name": "app-a"}},
		{method: http.MethodPost, path: "/apps", body: map[string]string{"app_name": "app-b"}},
		{method: http.MethodGet, path: "/apps/app-a/status"},
		{method: http.MethodGet, path: "/apps/app-b/status"},
	} {
		if err := client._sendRequest(ctx, req.method, req.path, req.body, nil, nil); err != nil {
			t.Fatalf("request %s %s error = %v", req.method, req.path, err)
		}
	}
}

func assertCookieHeader(t *testing.T, r *http.Request, want string) {
	t.Helper()

	got := r.Header.Get("Cookie")
	if got != want {
		t.Fatalf("cookie header = %q, want %q", got, want)
	}
}

func TestSnakeCase(t *testing.T) {
	type testcase struct {
		name string
		in   string
		want string
	}

	cases := []testcase{
		{name: "case1", in: "fooBar", want: "foo_bar"},
		{name: "case2", in: appCreate.String(), want: "app_create"},
	}
	for _, tc := range cases {
		got := snakeCase(tc.in)
		if got != tc.want {
			t.Errorf("%s, got '%v', want '%v'", tc.name, got, tc.want)
		}
	}
}
