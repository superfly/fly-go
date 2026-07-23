package fly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type CLISession struct {
	ID          string `json:"id"`
	URL         string `json:"auth_url,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	// PKCE reports that the server understood the code_challenge sent when the
	// session was created and will deliver the token via RedeemCLISessionToken
	// instead of polling. Clients that sent a challenge but get PKCE=false are
	// talking to an older server and must fall back to polling.
	PKCE     bool           `json:"pkce,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// StartCLISession starts a session with the platform via web
func StartCLISession(sessionName string, args map[string]any) (CLISession, error) {
	var result CLISession

	if args == nil {
		args = make(map[string]any)
	}
	args["name"] = sessionName

	postData, err := json.Marshal(args)
	if err != nil {
		return result, err
	}

	url := fmt.Sprintf("%s/api/v1/cli_sessions", baseURL)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(postData))
	if err != nil {
		return result, err
	}

	if resp.StatusCode != 201 {
		return result, ErrUnknown
	}

	defer resp.Body.Close() // skipcq: GO-S2307

	json.NewDecoder(resp.Body).Decode(&result)

	return result, nil
}

// RedeemCLISessionToken exchanges the one-time completion code (handed to the
// user's browser when they approved the session) plus the PKCE code verifier
// for the session's access token. The session is destroyed server-side on
// success, so this works exactly once.
func RedeemCLISessionToken(ctx context.Context, id, code, codeVerifier string) (CLISession, error) {
	var result CLISession

	postData, err := json.Marshal(map[string]string{
		"code":          code,
		"code_verifier": codeVerifier,
	})
	if err != nil {
		return result, fmt.Errorf("marshal CLI session redemption request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/cli_sessions/%s/redeem", baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(postData))
	if err != nil {
		return result, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}
	defer res.Body.Close() // skipcq: GO-S2307

	switch res.StatusCode {
	case http.StatusOK:
		if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
			return result, fmt.Errorf("failed to decode session, please try again: %w", err)
		}

		return result, nil
	case http.StatusNotFound:
		return result, ErrNotFound
	default:
		var apiErr struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(res.Body).Decode(&apiErr); err == nil && apiErr.Error != "" {
			return result, fmt.Errorf("failed to redeem login code: %s", apiErr.Error)
		}

		return result, ErrUnknown
	}
}

func GetCLISessionState(ctx context.Context, id string) (CLISession, error) {
	var value CLISession

	url := fmt.Sprintf("%s/api/v1/cli_sessions/%s", baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return value, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return value, err
	}
	defer res.Body.Close() // skipcq: GO-S2307

	switch res.StatusCode {
	case http.StatusOK:
		var auth CLISession
		if err = json.NewDecoder(res.Body).Decode(&auth); err != nil {
			return value, fmt.Errorf("failed to decode session, please try again: %w", err)
		}

		return auth, nil
	case http.StatusNotFound:
		return value, ErrNotFound
	default:
		return value, ErrUnknown
	}
}
