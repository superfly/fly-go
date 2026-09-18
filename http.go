package fly

import (
	"bytes"
	"context"
	"io"
	"math"
	"mime"
	"net/http"
	"strings"
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
		t.Logger.Debug(string(redactJSON(data)))
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

	// Wrap the body so reads are logged without buffering what the caller sees or closing early.
	// JSON bodies are logged whole once fully read so credential fields can be redacted;
	// anything else is logged as it streams.
	resp.Body = &loggingReadCloser{
		rc:         resp.Body,
		logger:     t.Logger,
		requestURL: resp.Request.URL.String(),
		json:       isJSONContentType(resp.Header.Get("Content-Type")),
	}
}

// maxLoggedResponseBody caps how much of a JSON response is held back for redacted logging.
const maxLoggedResponseBody = 1 << 20

// loggingReadCloser logs bytes as they are read from the underlying ReadCloser.
// The caller always receives the bytes immediately; only the log write is deferred for JSON bodies.
type loggingReadCloser struct {
	requestURL string
	rc         io.ReadCloser
	logger     Logger
	json       bool
	buf        bytes.Buffer
	truncated  bool
	logged     bool
}

func (l *loggingReadCloser) Read(p []byte) (int, error) {
	n, err := l.rc.Read(p)
	if n > 0 {
		if l.json {
			l.buffer(p[:n])
		} else {
			l.logger.Debugf("  <-- %s: %s", l.requestURL, string(p[:n]))
		}
	}
	if err != nil {
		l.flush()
	}

	return n, err
}

func (l *loggingReadCloser) buffer(data []byte) {
	if l.truncated {
		return
	}
	if l.buf.Len()+len(data) > maxLoggedResponseBody {
		l.truncated = true
		l.buf.Reset()

		return
	}
	l.buf.Write(data)
}

func (l *loggingReadCloser) flush() {
	if !l.json || l.logged {
		return
	}
	l.logged = true
	switch {
	case l.truncated:
		l.logger.Debugf("  <-- %s: [body over %d bytes not logged]", l.requestURL, maxLoggedResponseBody)
	case l.buf.Len() > 0:
		l.logger.Debugf("  <-- %s: %s", l.requestURL, string(redactJSON(l.buf.Bytes())))
	}
}

func (l *loggingReadCloser) Close() error {
	l.flush()

	return l.rc.Close()
}

func isJSONContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
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
