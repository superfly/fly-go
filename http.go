package fly

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/PuerkitoBio/rehttp"
)

func NewHTTPClient(logger Logger, transport http.RoundTripper) (*http.Client, error) {
	retryTransport := rehttp.NewTransport(
		transport,
		rehttp.RetryAll(
			rehttp.RetryMaxRetries(3),
			rehttp.RetryAny(
				rehttp.RetryTemporaryErr(),
				rehttp.RetryStatuses(502, 503),
			),
		),
		rehttp.ExpJitterDelay(100*time.Millisecond, 1*time.Second),
	)

	if logger != nil {
		return &http.Client{
			Transport: &LoggingTransport{
				InnerTransport: retryTransport,
				Logger:         logger,
			},
		}, nil
	}

	return &http.Client{
		Transport: retryTransport,
	}, nil
}

type LoggingTransport struct {
	InnerTransport http.RoundTripper
	Logger         Logger
	mu             sync.Mutex
}

func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := context.WithValue(req.Context(), contextKeyRequestStart, time.Now())
	req = req.WithContext(ctx)

	t.logRequest(req)

	resp, err := t.InnerTransport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	t.logResponse(resp)

	return resp, err
}

func (t *LoggingTransport) logRequest(req *http.Request) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Logger.Debugf("--> %s %s\n", req.Method, req.URL)

	if req.Body == nil {
		return
	}

	defer func() { _ = req.Body.Close() }()

	data, err := io.ReadAll(req.Body)

	if err != nil {
		t.Logger.Debug("error reading request body:", err)
	} else {
		t.Logger.Debug(string(data))
	}

	if req.Body != nil {
		t.Logger.Debug(req.Body)
	}

	req.Body = io.NopCloser(bytes.NewReader(data))
}

func (t *LoggingTransport) logResponse(resp *http.Response) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ctx := resp.Request.Context()
	if start, ok := ctx.Value(contextKeyRequestStart).(time.Time); ok {
		t.Logger.Debugf("<-- %d %s (%s)\n", resp.StatusCode, resp.Request.URL, shiftedDuration(time.Since(start), 2))
	} else {
		t.Logger.Debugf("<-- %d %s\n", resp.StatusCode, resp.Request.URL)
	}

	// Wrap the body so reads are logged as they happen without buffering or closing early.
	resp.Body = &loggingReadCloser{rc: resp.Body, logger: t.Logger, requestURL: resp.Request.URL.String()}
}

// loggingReadCloser logs bytes as they are read from the underlying ReadCloser.
// This preserves streaming semantics by avoiding any pre-reading or buffering.
type loggingReadCloser struct {
	requestURL string
	rc         io.ReadCloser
	logger     Logger
}

func (l *loggingReadCloser) Read(p []byte) (int, error) {
	n, err := l.rc.Read(p)
	if n > 0 {
		l.logger.Debugf("  <-- %s: %s", l.requestURL, string(p[:n]))
	}
	return n, err
}

func (l *loggingReadCloser) Close() error {
	return l.rc.Close()
}

func shiftedDuration(d time.Duration, dicimal int) time.Duration {
	shift := int(math.Pow10(dicimal))

	units := []time.Duration{time.Second, time.Millisecond, time.Microsecond, time.Nanosecond}
	for _, u := range units {
		if d > u {
			div := u / time.Duration(shift)
			if div == 0 {
				break
			}
			d = d / div * div
			break
		}
	}

	return d
}
