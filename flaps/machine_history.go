package flaps

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetMachineVersion decodes the requested Machine version into out without following redirects.
func (f *Client) GetMachineVersion(ctx context.Context, appName, machineID, version string, out any) error {
	if version == "" {
		return fmt.Errorf("missing Machine version")
	}

	return f.readMachineHistory(ctx, appName, machineID, "?version="+url.QueryEscape(version), out)
}

// GetMachineEvents decodes up to limit events into out without following redirects.
func (f *Client) GetMachineEvents(ctx context.Context, appName, machineID string, limit int, out any) error {
	if limit <= 0 {
		return fmt.Errorf("event limit must be positive")
	}

	return f.readMachineHistory(ctx, appName, machineID, fmt.Sprintf("/events?limit=%d", limit), out)
}

func (f *Client) readMachineHistory(ctx context.Context, appName, machineID, suffix string, out any) error {
	for _, id := range []string{appName, machineID} {
		if id == "" || id == "." || id == ".." || strings.Contains(id, "/") {
			return fmt.Errorf("invalid app name or Machine ID")
		}
	}
	path := "/apps/" + url.PathEscape(appName) + "/machines/" + url.PathEscape(machineID) + suffix
	reader := *f
	reader.httpClient = &http.Client{
		Transport:     f.httpClient.Transport,
		Jar:           f.httpClient.Jar,
		Timeout:       f.httpClient.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	ctx = contextWithAction(ctx, machineGet)
	ctx = contextWithMachineID(ctx, machineID)

	return reader.sendRequestWithStatus(ctx, http.MethodGet, path, nil, out, nil, http.StatusOK)
}
