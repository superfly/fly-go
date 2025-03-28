package tokens

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"github.com/superfly/macaroon"
	"github.com/superfly/macaroon/flyio"
	"github.com/superfly/macaroon/tp"
	"golang.org/x/exp/slices"
)

// Tokens is a collection of tokens belonging to the user. This includes
// macaroon tokens (per-org) and OAuth tokens (per-user).
//
// It is normal for this to include just macaroons, just oauth tokens, or a
// combination of the two. The GraphQL API is the only service that accepts
// macaroons and OAuth tokens in the same request. For other service, macaroons
// are preferred.
type Tokens struct {
	macaroons []string
	oauths    []string
	fromFile  string
	m         sync.RWMutex
}

// Parse extracts individual tokens from a token string. The input token may
// include an authorization scheme (`Bearer` or `FlyV1`) and/or a set of
// comma-separated macaroon and user tokens.
func Parse(token string) *Tokens {
	token = StripAuthorizationScheme(token)
	ret := &Tokens{}

	for _, tok := range strings.Split(token, ",") {
		tok = strings.TrimSpace(tok)
		switch pfx, _, _ := strings.Cut(tok, "_"); pfx {
		case "fm1r", "fm1a", "fm2":
			ret.macaroons = append(ret.macaroons, tok)
		default:
			ret.oauths = append(ret.oauths, tok)
		}
	}

	return ret
}

// ParseFromFile is like Parse but also records the file path that the tokens
// came from.
func ParseFromFile(token, fromFile string) *Tokens {
	ret := Parse(token)
	ret.fromFile = fromFile
	return ret

}

// FromFile returns the file path that was provided to ParseFromFile().
func (t *Tokens) FromFile() string {
	t.m.RLock()
	defer t.m.RUnlock()

	return t.fromFile
}

// Copy returns a deep copy of t.
func (t *Tokens) Copy() *Tokens {
	t.m.RLock()
	defer t.m.RUnlock()

	return &Tokens{
		macaroons: append([]string(nil), t.macaroons...),
		oauths:    append([]string(nil), t.oauths...),
		fromFile:  t.fromFile,
	}
}

// MacaroonsOnly returns a copy of t with only macaroon tokens.
func (t *Tokens) MacaroonsOnly() *Tokens {
	t.m.RLock()
	defer t.m.RUnlock()

	return &Tokens{
		macaroons: append([]string(nil), t.macaroons...),
		fromFile:  t.fromFile,
	}
}

// UserTokenOnly returns a copy of t with only user tokens.
func (t *Tokens) UserTokenOnly() *Tokens {
	t.m.RLock()
	defer t.m.RUnlock()

	return &Tokens{
		oauths:   append([]string(nil), t.oauths...),
		fromFile: t.fromFile,
	}
}

// GetMacaroonTokens returns the macaroon tokens.
func (t *Tokens) GetMacaroonTokens() []string {
	t.m.RLock()
	defer t.m.RUnlock()

	return append([]string(nil), t.macaroons...)
}

// GetUserTokens returns the user tokens.
func (t *Tokens) GetUserTokens() []string {
	t.m.RLock()
	defer t.m.RUnlock()

	return append([]string(nil), t.oauths...)
}

// AddTokens adds one or more tokens to t.
func (t *Tokens) AddTokens(toks ...string) *Tokens {
	t.m.Lock()
	defer t.m.Unlock()

	for _, tok := range toks {
		tok = strings.TrimSpace(tok)
		switch pfx, _, _ := strings.Cut(tok, "_"); pfx {
		case "fm1r", "fm1a", "fm2":
			t.macaroons = append(t.macaroons, tok)
		default:
			t.oauths = append(t.oauths, tok)
		}
	}

	return t
}

// Replace replaces t with other.
func (t *Tokens) Replace(other *Tokens) {
	t.m.Lock()
	defer t.m.Unlock()

	other.m.Lock()
	defer other.m.Unlock()

	if t.equalUnlocked(other) {
		return
	}

	t.macaroons = append([]string(nil), other.macaroons...)
	t.oauths = append([]string(nil), other.oauths...)
	t.fromFile = other.fromFile
}

// ReplaceMacaroonTokens replaces the macaroon tokens with macs.
func (t *Tokens) ReplaceMacaroonTokens(macs []string) {
	t.m.Lock()
	defer t.m.Unlock()

	t.macaroons = append([]string(nil), macs...)
}

// Equal returns true if t and other are equal.
func (t *Tokens) Equal(other *Tokens) bool {
	t.m.RLock()
	defer t.m.RUnlock()

	other.m.RLock()
	defer other.m.RUnlock()

	return t.equalUnlocked(other)
}

func (t *Tokens) equalUnlocked(other *Tokens) bool {

	return slices.Equal(t.macaroons, other.macaroons) && slices.Equal(t.oauths, other.oauths) && t.fromFile == other.fromFile
}

// Empty returns true if t has no tokens.
func (t *Tokens) Empty() bool {
	return len(t.macaroons)+len(t.oauths) == 0
}

// Update prunes any invalid/expired macaroons and fetches needed third party
// discharges
func (t *Tokens) Update(ctx context.Context, opts ...UpdateOption) (bool, error) {
	options := &updateOptions{debugger: noopDebugger{}, advancePrune: 1 * time.Minute}
	for _, o := range opts {
		o(options)
	}

	pruned := t.pruneBadMacaroons(options)
	discharged, err := t.dischargeThirdPartyCaveats(ctx, options)

	return pruned || discharged, err
}

func (t *Tokens) Flaps() string {
	return t.normalized(false, false)
}

func (t *Tokens) FlapsHeader() string {
	return t.normalized(false, true)
}

func (t *Tokens) Docker() string {
	return t.normalized(false, false)
}

func (t *Tokens) NATS() string {
	return t.normalized(false, false)
}

func (t *Tokens) Bubblegum() string {
	return t.normalized(false, false)
}

func (t *Tokens) BubblegumHeader() string {
	return t.normalized(false, true)
}

func (t *Tokens) GraphQL() string {
	return t.normalized(true, false)
}

func (t *Tokens) GraphQLHeader() string {
	return t.normalized(true, true)
}

func (t *Tokens) All() string {
	return t.normalized(true, false)
}

func (t *Tokens) normalized(macaroonsAndUserTokens, includeScheme bool) string {
	t.m.RLock()
	defer t.m.RUnlock()

	scheme := ""
	if includeScheme {
		scheme = "Bearer "
		if len(t.macaroons) > 0 {
			scheme = "FlyV1 "
		}
	}

	if macaroonsAndUserTokens {
		return scheme + strings.Join(append(t.macaroons, t.oauths...), ",")
	}
	if len(t.macaroons) == 0 {
		return scheme + strings.Join(t.oauths, ",")
	}
	return scheme + strings.Join(t.macaroons, ",")
}

// pruneBadMacaroons removes expired and invalid macaroon tokens as well as
// discharge tokens that are no longer needed.
func (t *Tokens) pruneBadMacaroons(options *updateOptions) bool {
	t.m.Lock()
	defer t.m.Unlock()

	var (
		updated   bool
		tpTickets = make(map[string]bool)
		parsed    = make(map[string]*macaroon.Macaroon)
	)

	for _, tok := range t.macaroons {
		raws, err := macaroon.Parse(tok)
		if err != nil {
			continue
		}

		m, err := macaroon.Decode(raws[0])
		if err != nil {
			continue
		}

		if time.Now().After(m.Expiration()) {
			continue
		}

		parsed[tok] = m

		if m.Location != flyio.LocationPermission {
			continue
		}

		for _, tp := range macaroon.GetCaveats[*macaroon.Caveat3P](&m.UnsafeCaveats) {
			tpTickets[string(tp.Ticket)] = true
		}
	}

	t.macaroons = slices.DeleteFunc(t.macaroons, func(tok string) bool {
		m, ok := parsed[tok]
		if !ok {
			updated = true
			return true
		}

		if m.Location == flyio.LocationPermission {
			return false
		}

		if !tpTickets[string(m.Nonce.KID)] {
			updated = true
			return true
		}

		// preemptively prune auth tokens according to the advancePrune option.
		// The hope is that we can replace discharge tokens *before* they expire
		// so requests don't fail.
		//
		// TODO: this is hacky
		if (m.Location == flyio.LocationAuthentication || m.Location == flyio.LocationNewAuthentication) &&
			time.Now().Add(options.advancePrune).After(m.Expiration()) {
			updated = true
			return true
		}

		return false
	})

	return updated
}

// dischargeThirdPartyCaveats attempts to fetch any necessary discharge tokens
// for 3rd party caveats found within macaroon tokens.
//
// See https://github.com/superfly/macaroon/blob/main/tp/README.md
func (t *Tokens) dischargeThirdPartyCaveats(ctx context.Context, options *updateOptions) (bool, error) {
	t.m.RLock()
	macaroons := strings.Join(t.macaroons, ",")
	oauths := strings.Join(t.oauths, ",")
	t.m.RUnlock()

	if macaroons == "" {
		return false, nil
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return false, err
	}

	h := &http.Client{
		Jar: jar,
		Transport: debugTransport{
			d: options.debugger,
			t: http.DefaultTransport,
		},
	}

	copts := options.clientOptions
	copts = append(copts, tp.WithHTTP(h))
	if oauths != "" {
		copts = append(copts,
			tp.WithBearerAuthentication("auth.fly.io", oauths),
			tp.WithBearerAuthentication(flyio.LocationAuthentication, oauths),
		)
	}
	c := flyio.DischargeClient(copts...)

	switch needDischarge, err := c.NeedsDischarge(macaroons); {
	case err != nil:
		return false, err
	case !needDischarge:
		return false, nil
	}

	toCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	options.debugger.Debug("Attempting to upgrade authentication token")
	withDischarges, err := c.FetchDischargeTokens(toCtx, macaroons)

	// withDischarges will be non-empty in the event of partial success
	if withDischarges != "" && withDischarges != macaroons {
		t.m.Lock()
		defer t.m.Unlock()

		t.macaroons = Parse(withDischarges).macaroons
		return true, err
	}

	return false, err
}

type UpdateOption func(*updateOptions)

type updateOptions struct {
	clientOptions []tp.ClientOption
	debugger      Debugger
	advancePrune  time.Duration
}

func WithUserURLCallback(cb func(ctx context.Context, url string) error) UpdateOption {
	return func(o *updateOptions) {
		o.clientOptions = append(o.clientOptions, tp.WithUserURLCallback(cb))
	}
}

func WithDebugger(d Debugger) UpdateOption {
	return func(o *updateOptions) {
		o.debugger = d
	}
}

type Debugger interface {
	Debug(...any)
}

type noopDebugger struct{}

func (noopDebugger) Debug(...any) {}

func WithAdvancePrune(advancePrune time.Duration) UpdateOption {
	return func(o *updateOptions) {
		o.advancePrune = advancePrune
	}
}

type debugTransport struct {
	d Debugger
	t http.RoundTripper
}

func (d debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	d.d.Debug("Request:", req.URL.String())
	return d.t.RoundTrip(req)
}

// StripAuthorizationScheme strips any FlyV1/Bearer schemes from token.
func StripAuthorizationScheme(token string) string {
	token = strings.TrimSpace(token)

	pfx, rest, found := strings.Cut(token, " ")
	if !found {
		return token
	}

	if pfx = strings.TrimSpace(pfx); strings.EqualFold(pfx, "Bearer") || strings.EqualFold(pfx, "FlyV1") {
		return StripAuthorizationScheme(rest)
	}

	return token
}
