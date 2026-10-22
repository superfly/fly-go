package flaps

import (
	"context"
	"net/http"
	"testing"
)

func TestMetadataKeyIsEscapedInPath(t *testing.T) {
	operations := []struct {
		name string
		run  func(*Client, string) error
	}{
		{name: "set", run: func(client *Client, key string) error {
			return client.SetMetadata(context.Background(), "my-app", "mach1", key, "v")
		}},
		{name: "delete", run: func(client *Client, key string) error {
			return client.DeleteMetadata(context.Background(), "my-app", "mach1", key)
		}},
	}
	keys := []struct {
		name string
		key  string
		want string
	}{
		{name: "slash", key: "team/env", want: "/v1/apps/my-app/machines/mach1/metadata/team%2Fenv"},
		{name: "question_mark", key: "what?", want: "/v1/apps/my-app/machines/mach1/metadata/what%3F"},
		{name: "hash", key: "a#b", want: "/v1/apps/my-app/machines/mach1/metadata/a%23b"},
		{name: "dot_segment", key: "../x", want: "/v1/apps/my-app/machines/mach1/metadata/..%2Fx"},
	}

	for _, operation := range operations {
		for _, key := range keys {
			t.Run(operation.name+"/"+key.name, func(t *testing.T) {
				transport := &managedPostgresRoundTripper{statusCode: http.StatusOK, body: `{}`}
				if err := operation.run(newTestFlapsClient(t, transport), key.key); err != nil {
					t.Fatalf("request error = %v", err)
				}
				if got := transport.req.URL.RequestURI(); got != key.want {
					t.Fatalf("request URI = %q, want %q", got, key.want)
				}
			})
		}
	}
}

func TestDeleteAppEscapesNameInPath(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusOK, body: `{}`}
	if err := newTestFlapsClient(t, transport).DeleteApp(context.Background(), "a/b?c"); err != nil {
		t.Fatalf("DeleteApp() error = %v", err)
	}
	if got, want := transport.req.URL.RequestURI(), "/v1/apps/a%2Fb%3Fc"; got != want {
		t.Fatalf("request URI = %q, want %q", got, want)
	}
}
