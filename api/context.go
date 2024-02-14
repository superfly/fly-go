package api

import "context"

type contextKey string

const (
	contextKeyClient        = contextKey("client")
	contextKeyAuthorization = contextKey("authorization")
	contextKeyRequestStart  = contextKey("RequestStart")
)

// NewContextWithClient derives a Context that carries c from ctx.
func NewContextWithClient(ctx context.Context, c *Client) context.Context {
	return context.WithValue(ctx, contextKeyClient, c)
}

// FromContext returns the Client ctx carries. Panics if ctx carries no Client.
func ClientFromContext(ctx context.Context) *Client {
	return ctx.Value(contextKeyClient).(*Client)
}

// WithAuthorizationHeader returns a context that instructs the client to use
// the specified Authorization header value.
func WithAuthorizationHeader(ctx context.Context, hdr string) context.Context {
	return context.WithValue(ctx, contextKeyAuthorization, hdr)
}
