package flaps

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AffinityHeader carries a flaps app affinity token. Flaps returns it on
// responses that establish affinity (app create and get) and honours it on
// app-scoped requests by replaying them to the flaps instance that issued it,
// which already sees the app in its local state.
const AffinityHeader = "Fly-Flaps-Affinity"

// affinityExpiryMargin is how long before its stated expiry a token stops
// being sent. A token flaps considers expired is ignored for routing, but its
// presence also stops flaps from issuing a fresh one, so a token on the edge
// of expiry is worth less than none.
const affinityExpiryMargin = 5 * time.Second

// AffinityStore keeps flaps app affinity tokens beyond the life of a Client,
// typically across processes, so that a request made shortly after another
// process created or read an app reaches the flaps instance that already has
// it.
//
// Implementations must be safe for concurrent use. Both methods are best
// effort: a store that cannot read or write should behave as empty rather
// than fail the request.
type AffinityStore interface {
	// LoadAffinity returns the token stored for appName at the flaps
	// endpoint baseURL, or "" if there is none.
	LoadAffinity(baseURL, appName string) string
	// StoreAffinity records token for appName at baseURL. The token is of
	// no use after expiresAt and may be discarded then.
	StoreAffinity(baseURL, appName, token string, expiresAt time.Time)
}

// affinityClaims is the readable part of an affinity token. The token is
// "<base64url(json)>.<signature>"; only flaps can verify the signature, but
// the payload says which app the token is for and when it expires.
type affinityClaims struct {
	AppName    string    `json:"app_name"`
	InstanceID string    `json:"instance_id"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func parseAffinityClaims(token string) (affinityClaims, bool) {
	payloadEnc, _, ok := strings.Cut(token, ".")
	if !ok {
		return affinityClaims{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadEnc)
	if err != nil {
		return affinityClaims{}, false
	}
	var c affinityClaims
	if err := json.Unmarshal(payload, &c); err != nil || c.AppName == "" || c.ExpiresAt.IsZero() {
		return affinityClaims{}, false
	}
	return c, true
}

// appNameFromPath returns the app an app-scoped flaps path (/v1/apps/<app>
// or below it) refers to, or "" for any other path.
func appNameFromPath(u *url.URL) string {
	rest, ok := strings.CutPrefix(u.EscapedPath(), "/v1/apps/")
	if !ok {
		return ""
	}
	name, _, _ := strings.Cut(rest, "/")
	name, err := url.PathUnescape(name)
	if err != nil {
		return ""
	}
	return name
}

// attachAffinity adds the stored affinity token for the request's app, unless
// the caller already set one or the stored token is about to expire.
func (f *Client) attachAffinity(req *http.Request) {
	if f.affinity == nil || req.Header.Get(AffinityHeader) != "" {
		return
	}
	app := appNameFromPath(req.URL)
	if app == "" {
		return
	}
	token := f.affinity.LoadAffinity(f.baseUrl.String(), app)
	if token == "" {
		return
	}
	claims, ok := parseAffinityClaims(token)
	if !ok || claims.AppName != app || !time.Now().Add(affinityExpiryMargin).Before(claims.ExpiresAt) {
		return
	}
	req.Header.Set(AffinityHeader, token)
	if f.logger != nil {
		f.logger.Debugf("flaps: app affinity: sending stored token for app %s (instance %s, expires %s)",
			app, claims.InstanceID, claims.ExpiresAt.Format(time.RFC3339))
	}
}

// captureAffinity stores the affinity token a response carries. The app it
// belongs to comes from the token itself, since app creation is a POST to
// /v1/apps whose path names no app.
func (f *Client) captureAffinity(resp *http.Response) {
	if f.affinity == nil || resp == nil {
		return
	}
	token := resp.Header.Get(AffinityHeader)
	if token == "" {
		return
	}
	claims, ok := parseAffinityClaims(token)
	if !ok {
		return
	}
	if resp.Request != nil {
		if app := appNameFromPath(resp.Request.URL); app != "" && app != claims.AppName {
			return
		}
	}
	f.affinity.StoreAffinity(f.baseUrl.String(), claims.AppName, token, claims.ExpiresAt)
	if f.logger != nil {
		f.logger.Debugf("flaps: app affinity: stored token for app %s (instance %s, expires %s)",
			claims.AppName, claims.InstanceID, claims.ExpiresAt.Format(time.RFC3339))
	}
}
