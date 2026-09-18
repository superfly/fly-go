package fly

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type captureLogger struct {
	buf bytes.Buffer
}

func (l *captureLogger) Debug(v ...any) {
	fmt.Fprintln(&l.buf, v...)
}

func (l *captureLogger) Debugf(format string, v ...any) {
	fmt.Fprintf(&l.buf, format, v...)
}

func TestLoggingTransportRedactsJSONBodies(t *testing.T) {
	const (
		requestSecret  = "zzprobezz-request-value"
		responseSecret = "FlyV1 fm2_zzprobezz-response-token"
		responseValue  = "zzprobezz-response-value"
	)

	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"tokenHeader":%q,"secrets":[{"name":"DATABASE_URL","value":%q}],"version":3}`, responseSecret, responseValue)
	}))
	defer srv.Close()

	logger := &captureLogger{}
	client, err := NewHTTPClient(logger, http.DefaultTransport)
	if err != nil {
		t.Fatal(err)
	}

	requestBody := fmt.Sprintf(`{"values":{"DATABASE_URL":%q,"OTHER":null}}`, requestSecret)
	resp, err := client.Post(srv.URL+"/v1/apps/x/secrets", "application/json", strings.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(received) != requestBody {
		t.Errorf("server received altered request body: %s", received)
	}
	if !strings.Contains(string(got), responseSecret) || !strings.Contains(string(got), responseValue) {
		t.Errorf("client received altered response body: %s", got)
	}

	logged := logger.buf.String()
	for _, secret := range []string{requestSecret, responseSecret, responseValue} {
		if strings.Contains(logged, secret) {
			t.Errorf("secret %q written to debug log:\n%s", secret, logged)
		}
	}
	for _, want := range []string{"DATABASE_URL", "tokenHeader", `"version":3`, "[REDACTED]"} {
		if !strings.Contains(logged, want) {
			t.Errorf("expected %q in debug log:\n%s", want, logged)
		}
	}
}

func TestLoggingTransportLogsNonJSONResponses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "plain text response")
	}))
	defer srv.Close()

	logger := &captureLogger{}
	client, err := NewHTTPClient(logger, http.DefaultTransport)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "plain text response" {
		t.Errorf("client received altered response body: %s", got)
	}
	if !strings.Contains(logger.buf.String(), "plain text response") {
		t.Errorf("expected plain response in debug log:\n%s", logger.buf.String())
	}
}

func TestRedactJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "top level sensitive key",
			in:   `{"value":"s3cret","name":"A"}`,
			want: `{"name":"A","value":"[REDACTED]"}`,
		},
		{
			name: "map under sensitive key keeps names",
			in:   `{"values":{"A":"x","B":null}}`,
			want: `{"values":{"A":"[REDACTED]","B":null}}`,
		},
		{
			name: "nested in arrays and objects",
			in:   `{"data":{"secrets":[{"key":"A","value":"x"},{"key":"B","value":"y"}]}}`,
			want: `{"data":{"secrets":[{"key":"A","value":"[REDACTED]"},{"key":"B","value":"[REDACTED]"}]}}`,
		},
		{
			name: "case insensitive keys",
			in:   `{"Password":"p","ACCESS_TOKEN":"t","tokenHeader":"h"}`,
			want: `{"ACCESS_TOKEN":"[REDACTED]","Password":"[REDACTED]","tokenHeader":"[REDACTED]"}`,
		},
		{
			name: "non string leaves under sensitive key",
			in:   `{"token":{"id":7,"nested":["a",1,true]}}`,
			want: `{"token":{"id":7,"nested":["[REDACTED]",1,true]}}`,
		},
		{
			name: "extension environment block",
			in:   `{"data":{"addOn":{"name":"redis","environment":{"REDIS_URL":"redis://x"}}}}`,
			want: `{"data":{"addOn":{"environment":{"REDIS_URL":"[REDACTED]"},"name":"redis"}}}`,
		},
		{
			name: "invalid json passes through",
			in:   `not json {"value":"x"`,
			want: `not json {"value":"x"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := string(redactJSON([]byte(tc.in))); got != tc.want {
				t.Errorf("redactJSON(%s)\n got %s\nwant %s", tc.in, got, tc.want)
			}
		})
	}
}
