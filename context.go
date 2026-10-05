package fly

import "context"

type contextKey string

const (
	contextKeyAuthorization     = contextKey("authorization")
	contextKeyRequestStart      = contextKey("RequestStart")
	contextKeySensitiveResponse = contextKey("SensitiveResponse")
	contextKeyNoRetry           = contextKey("NoRetry")
)

// WithAuthorizationHeader returns a context that instructs the client to use
// the specified Authorization header value.
func WithAuthorizationHeader(ctx context.Context, hdr string) context.Context {
	return context.WithValue(ctx, contextKeyAuthorization, hdr)
}

// WithSensitiveResponseBody marks the request's response body as containing
// secrets (e.g. credentials) so debug logging must not record it.
func WithSensitiveResponseBody(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKeySensitiveResponse, true)
}

func hasSensitiveResponseBody(ctx context.Context) bool {
	v, _ := ctx.Value(contextKeySensitiveResponse).(bool)
	return v
}

// WithoutHTTPRetries disables the shared transport's automatic retries
// (including on 502/503) for requests made with this context. Use for
// non-idempotent requests with side effects, e.g. credential rotation.
//
// This only governs the rehttp-based transport retry. flaps' own ECONNRESET
// backoff in flaps.go's do() is a separate layer gated on method == GET, so
// it does not need (and does not respect) this flag.
func WithoutHTTPRetries(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKeyNoRetry, true)
}

func retriesDisabled(ctx context.Context) bool {
	v, _ := ctx.Value(contextKeyNoRetry).(bool)
	return v
}
