package flaps

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ListIPAssignmentsResponse struct {
	IPs []IPAssignment `json:"ips"`
}

type IPAssignment struct {
	IP          string    `json:"ip"`
	Region      string    `json:"region"`
	ServiceName string    `json:"service_name"`
	Shared      bool      `json:"shared"`
	CreatedAt   time.Time `json:"created_at"`
}

func (ip IPAssignment) IsFlycast() bool {
	return strings.HasPrefix(ip.IP, "fdaa:")
}

type AssignIPRequest struct {
	Type         string `json:"type"`
	Region       string `json:"region"`
	Organization string `json:"org_slug"`
	Network      string `json:"network"`
	ServiceName  string `json:"service_name"`
}

func (f *Client) sendRequestIpAssignments(ctx context.Context, appName, method, endpoint string, in, out any, qs url.Values, headers map[string][]string) error {
	endpoint = fmt.Sprintf("/apps/%s/ip_assignments%s", url.PathEscape(appName), endpoint)
	if len(qs) > 0 {
		endpoint += "?" + qs.Encode()
	}
	return f._sendRequest(ctx, method, endpoint, in, out, headers)
}

func (f *Client) GetIPAssignments(ctx context.Context, appName string) (res *ListIPAssignmentsResponse, err error) {
	ctx = contextWithAction(ctx, ipAssignmentList)

	if err := f.sendRequestIpAssignments(ctx, appName, http.MethodGet, "", nil, &res, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to list app ip assignments: %w", err)
	}

	return
}

func (f *Client) AssignIP(ctx context.Context, appName string, req AssignIPRequest) (res *IPAssignment, err error) {
	ctx = contextWithAction(ctx, ipAssignmentCreate)

	if err := f.sendRequestIpAssignments(ctx, appName, http.MethodPost, "", req, &res, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to add ip to app: %w", err)
	}

	return
}

func (f *Client) DeleteIPAssignment(ctx context.Context, appName, ip string) (err error) {
	ctx = contextWithAction(ctx, ipAssignmentDelete)

	if err := f.sendRequestIpAssignments(ctx, appName, http.MethodDelete, fmt.Sprintf("/%s", ip), nil, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to remove ip from app: %w", err)
	}

	return
}
