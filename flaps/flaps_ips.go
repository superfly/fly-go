package flaps

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type IPAssignment struct {
	CreatedAt   time.Time `json:"created_at"`
	IP          string    `json:"ip"`
	Region      string    `json:"region"`
	ServiceName string    `json:"service_name"`
	Shared      bool      `json:"shared"`
	Type        string    `json:"type,omitempty"`
}

type ListIPAssignmentsResponse struct {
	IPs []IPAssignment `json:"ips"`
}

type AssignIPRequest struct {
	Type        string `json:"type"`
	Region      string `json:"region,omitempty"`
	OrgSlug     string `json:"org_slug,omitempty"`
	Network     string `json:"network,omitempty"`
	ServiceName string `json:"service_name,omitempty"`
}

func (f *Client) sendRequestIPAssignments(ctx context.Context, method, endpoint string, in, out interface{}, headers map[string][]string) error {
	endpoint = fmt.Sprintf("/apps/%s%s", f.appName, endpoint)
	return f._sendRequest(ctx, method, endpoint, in, out, headers)
}

// ListIPAssignments lists all IP assignments for the app
func (f *Client) ListIPAssignments(ctx context.Context) (*ListIPAssignmentsResponse, error) {
	ctx = contextWithAction(ctx, ipAssignmentsList)

	out := new(ListIPAssignmentsResponse)
	if err := f.sendRequestIPAssignments(ctx, http.MethodGet, "/ip_assignments", nil, out, nil); err != nil {
		return nil, fmt.Errorf("failed to list IP assignments: %w", err)
	}

	return out, nil
}

// AssignIPAddress assigns a new IP address to the app
func (f *Client) AssignIPAddress(ctx context.Context, req *AssignIPRequest) (*IPAssignment, error) {
	ctx = contextWithAction(ctx, ipAssignmentCreate)

	out := new(IPAssignment)
	if err := f.sendRequestIPAssignments(ctx, http.MethodPost, "/ip_assignments", req, out, nil); err != nil {
		return nil, fmt.Errorf("failed to assign IP address: %w", err)
	}

	return out, nil
}

// ReleaseIPAddress releases an IP address from the app
func (f *Client) ReleaseIPAddress(ctx context.Context, ip string) error {
	ctx = contextWithAction(ctx, ipAssignmentDelete)

	endpoint := fmt.Sprintf("/ip_assignments/%s", ip)
	if err := f.sendRequestIPAssignments(ctx, http.MethodDelete, endpoint, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to release IP address: %w", err)
	}

	return nil
}
