package flaps

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/cenkalti/backoff/v4"
	fly "github.com/superfly/fly-go"
	"github.com/superfly/fly-go/internal/tracing"
	"github.com/superfly/fly-go/tokens"
	"github.com/superfly/macaroon"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const headerFlyRequestId = "fly-request-id"

type Client struct {
	appName    string
	baseUrl    *url.URL
	tokens     *tokens.Tokens
	httpClient *http.Client
	userAgent  string
}

type NewClientOpts struct {
	// required:
	AppName string

	// optional, avoids API roundtrip when connecting to flaps by wireguard:
	AppCompact *fly.AppCompact

	// optional, sent with requests
	UserAgent string

	// optional, used to connect to machines API
	DialContext func(ctx context.Context, network, address string) (net.Conn, error)
	OrgSlug     string // required if DialContext set

	// URL used when connecting via usermode wireguard.
	BaseURL *url.URL

	Tokens *tokens.Tokens

	// optional:
	Logger fly.Logger

	// optional, used to construct the underlying HTTP client
	Transport http.RoundTripper
}

func NewWithOptions(ctx context.Context, opts NewClientOpts) (*Client, error) {
	var err error
	flapsBaseURL := os.Getenv("FLY_FLAPS_BASE_URL")
	if flapsBaseURL == "" {
		flapsBaseURL = "https://api.machines.dev"
	}

	if opts.DialContext != nil {
		return newWithUsermodeWireguard(wireguardConnectionParams{
			appName:     opts.AppName,
			orgSlug:     opts.OrgSlug,
			dialContext: opts.DialContext,
			baseURL:     opts.BaseURL,
			userAgent:   opts.UserAgent,
		}, opts.Logger)
	} else if flapsBaseURL == "" {
		flapsBaseURL = "https://api.machines.dev"
	}
	flapsUrl, err := url.Parse(flapsBaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid FLY_FLAPS_BASE_URL '%s' with error: %w", flapsBaseURL, err)
	}

	transport := http.DefaultTransport
	if opts.Transport != nil {
		transport = opts.Transport
	}
	otelTransport := otelhttp.NewTransport(transport)
	httpClient, err := fly.NewHTTPClient(opts.Logger, otelTransport)
	if err != nil {
		return nil, fmt.Errorf("flaps: can't setup HTTP client to %s: %w", flapsUrl.String(), err)
	}

	userAgent := "fly-go"
	if opts.UserAgent != "" {
		userAgent = opts.UserAgent
	}

	return &Client{
		appName:    opts.AppName,
		baseUrl:    flapsUrl,
		tokens:     opts.Tokens,
		httpClient: httpClient,
		userAgent:  userAgent,
	}, nil
}

type wireguardConnectionParams struct {
	appName     string
	orgSlug     string
	userAgent   string
	dialContext func(ctx context.Context, network, address string) (net.Conn, error)
	baseURL     *url.URL
	tokens      *tokens.Tokens
}

func newWithUsermodeWireguard(params wireguardConnectionParams, logger fly.Logger) (*Client, error) {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return params.dialContext(ctx, network, addr)
		},
	}
	instrumentedTransport := otelhttp.NewTransport(transport)

	httpClient, err := fly.NewHTTPClient(logger, instrumentedTransport)
	if err != nil {
		return nil, fmt.Errorf("flaps: can't setup HTTP client for %s: %w", params.orgSlug, err)
	}

	return &Client{
		appName:    params.appName,
		baseUrl:    params.baseURL,
		tokens:     params.tokens,
		httpClient: httpClient,
		userAgent:  params.userAgent,
	}, nil
}

func (f *Client) CreateApp(ctx context.Context, name string, org string) (err error) {
	in := map[string]interface{}{
		"app_name": name,
		"org_slug": org,
	}

	ctx = contextWithAction(ctx, appCreate)

	err = f._sendRequest(ctx, http.MethodPost, "/apps", in, nil, nil)
	return
}

func (f *Client) WaitForApp(ctx context.Context, name string) error {
	ctx = contextWithAction(ctx, machineGet)

	var op = func() error {
		err := f._sendRequest(ctx, http.MethodGet, "/apps/"+url.PathEscape(name), nil, nil, nil)
		if err == nil {
			return nil
		}
		if ferr, ok := err.(*FlapsError); ok && slices.Contains([]int{404, 401}, ferr.ResponseStatusCode) {
			return err
		}
		return backoff.Permanent(err)
	}
	return Retry(ctx, op)
}

var snakeCasePattern = regexp.MustCompile("[A-Z]")

func snakeCase(s string) string {
	return snakeCasePattern.ReplaceAllStringFunc(s, func(m string) string {
		return "_" + strings.ToLower(m)
	})
}

func (f *Client) _sendRequest(ctx context.Context, method, endpoint string, in, out interface{}, headers map[string][]string) error {
	actionName := snakeCase(actionFromContext(ctx).String())
	var caveats []string
	caveatNames, err := f.getCaveatNames()
	if err == nil {
		caveats = caveatNames
	}

	ctx, span := tracing.GetTracer().Start(ctx, fmt.Sprintf("flaps.%s", actionName), trace.WithAttributes(
		attribute.String("request.action", actionName),
		attribute.String("request.endpoint", endpoint),
		attribute.String("request.method", method),
		attribute.String("request.machine_id", machineIDFromContext(ctx)),
		attribute.StringSlice("request.caveats", caveats),
	))
	defer span.End()

	// timing := instrument.Flaps.Begin()
	// defer timing.End()

	req, err := f.NewRequest(ctx, method, endpoint, in, headers)
	if err != nil {
		tracing.RecordError(span, err, "failed to prepare request")
		return err
	}
	req.Header.Set("User-Agent", f.userAgent)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		tracing.RecordError(span, err, "failed to do request")
		return err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error closing response body:", err)
		}
	}()

	span.SetAttributes(attribute.Int("request.status_code", resp.StatusCode))
	span.SetAttributes(attribute.String("request.id", resp.Header.Get(headerFlyRequestId)))

	span.AddLink(trace.Link{SpanContext: tracing.SpanContextFromHeaders(resp)})

	if resp.StatusCode > 299 {
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			responseBody = make([]byte, 0)
		}
		return &FlapsError{
			OriginalError:      handleAPIError(resp.StatusCode, responseBody),
			ResponseStatusCode: resp.StatusCode,
			ResponseBody:       responseBody,
			FlyRequestId:       resp.Header.Get(headerFlyRequestId),
			TraceID:            span.SpanContext().TraceID().String(),
		}
	}
	if out != nil {
		if strOut, ok := out.(*string); ok {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed reading response body: %w", err)
			}
			*strOut = string(bodyBytes)
		} else if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("failed decoding response: %w", err)
		}
	}
	return nil
}

func (f *Client) urlFromBaseUrl(pathAndQueryString string) (*url.URL, error) {
	newUrl := *f.baseUrl // this does a copy: https://github.com/golang/go/issues/38351#issue-597797864
	newPath, err := url.Parse(pathAndQueryString)
	if err != nil {
		return nil, fmt.Errorf("failed parsing flaps path '%s' with error: %w", pathAndQueryString, err)
	}
	return newUrl.ResolveReference(&url.URL{Path: newPath.Path, RawQuery: newPath.RawQuery}), nil
}

func (f *Client) NewRequest(ctx context.Context, method, path string, in interface{}, headers map[string][]string) (*http.Request, error) {
	var body io.Reader

	if headers == nil {
		headers = make(map[string][]string)
	}

	targetEndpoint, err := f.urlFromBaseUrl(fmt.Sprintf("/v1%s", path))
	if err != nil {
		return nil, err
	}

	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		headers["Content-Type"] = []string{"application/json"}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, targetEndpoint.String(), body)
	if err != nil {
		return nil, fmt.Errorf("could not create new request, %w", err)
	}
	req.Header = headers
	req.Header.Add("Authorization", f.tokens.FlapsHeader())

	return req, nil
}

func (f *Client) getCaveatNames() ([]string, error) {
	tok := f.tokens.MacaroonsOnly().All()
	raws, err := macaroon.Parse(tok)
	if err != nil {
		return nil, err
	}

	m, err := macaroon.Decode(raws[0])
	if err != nil {
		return nil, err
	}

	caveats := m.UnsafeCaveats.Caveats
	caveatNames := make([]string, len(caveats))

	for i, c := range caveats {
		caveatNames[i] = c.Name()
	}

	return caveatNames, nil
}

// handleAPIError returns an error based on the status code and response body.
func handleAPIError(statusCode int, responseBody []byte) error {
	switch statusCode / 100 {
	case 1, 3:
		return fmt.Errorf("API returned unexpected status, %d", statusCode)
	case 4, 5:
		apiErr := struct {
			Error   string `json:"error"`
			Message string `json:"message,omitempty"`
		}{}
		jsonErr := json.Unmarshal(responseBody, &apiErr)
		if jsonErr != nil {
			return fmt.Errorf("request returned non-2xx status: %d: %s", statusCode, string(responseBody))
		} else if apiErr.Message != "" {
			return fmt.Errorf("%s", apiErr.Message)
		}
		return errors.New(apiErr.Error)
	default:
		return errors.New("something went terribly wrong")
	}
}

type CreateOIDCTokenRequest struct {
	Audience         string `json:"aud"`
	AWSPrincipalTags bool   `json:"aws_principal_tags"`
}

func (f *Client) GetOIDCToken(ctx context.Context, aud string, aws bool) (string, error) {
	ctx = contextWithAction(ctx, getOIDCToken)
	var token string
	err := f._sendRequest(ctx, http.MethodPost, "/tokens/oidc", &CreateOIDCTokenRequest{
		Audience:         aud,
		AWSPrincipalTags: aws,
	}, &token, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get OIDC token: %w", err)
	}
	return token, nil
}
