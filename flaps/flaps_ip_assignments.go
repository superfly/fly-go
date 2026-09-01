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

// IPAssignmentType is the type of IP address to allocate.
type IPAssignmentType string

const (
	IPAssignmentTypeV4        IPAssignmentType = "v4"
	IPAssignmentTypeV6        IPAssignmentType = "v6"
	IPAssignmentTypeSharedV4  IPAssignmentType = "shared_v4"
	IPAssignmentTypePrivateV6 IPAssignmentType = "private_v6"
	IPAssignmentTypeEgressV4  IPAssignmentType = "egress_v4"
	IPAssignmentTypeEgressV6  IPAssignmentType = "egress_v6"
	// IPAssignmentTypeEgressPair allocates both an egress v4 and v6 address at once.
	IPAssignmentTypeEgressPair IPAssignmentType = "egress_pair"
)

type IPAssignment struct {
	IP          string    `json:"ip"`
	Region      string    `json:"region"`
	ServiceName string    `json:"service_name"`
	Shared      bool      `json:"shared"`
	CreatedAt   time.Time `json:"created_at"`
	Egress      bool      `json:"egress"`
}

func (ip IPAssignment) IsFlycast() bool {
	return strings.HasPrefix(ip.IP, "fdaa:")
}

func (ip IPAssignment) IsV6() bool {
	return strings.Contains(ip.IP, ":")
}

func (ip IPAssignment) Type() IPAssignmentType {
	switch {
	case ip.Egress && ip.IsV6():
		return IPAssignmentTypeEgressV6
	case ip.Egress:
		return IPAssignmentTypeEgressV4
	case ip.IsFlycast():
		return IPAssignmentTypePrivateV6
	case ip.Shared:
		return IPAssignmentTypeSharedV4
	case ip.IsV6():
		return IPAssignmentTypeV6
	default:
		return IPAssignmentTypeV4
	}
}

type AssignIPRequest struct {
	Type         IPAssignmentType `json:"type"`
	Region       string           `json:"region"`
	Organization string           `json:"org_slug"`
	Network      string           `json:"network"`
	ServiceName  string           `json:"service_name"`
}

// IPPair is the pair of addresses returned when an IPAssignmentTypeEgressPair is allocated.
type IPPair struct {
	V4 string `json:"v4"`
	V6 string `json:"v6"`
}

type AssignIPResponse struct {
	IP *string `json:"ip"`
	// IPPair is set when EgressPair IP type was requested; in that case IP is nil.
	IPPair      *IPPair   `json:"ip_pair"`
	Region      string    `json:"region"`
	ServiceName string    `json:"service_name"`
	Shared      bool      `json:"shared"`
	CreatedAt   time.Time `json:"created_at"`
	Egress      bool      `json:"egress"`
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

func (f *Client) AssignIP(ctx context.Context, appName string, req AssignIPRequest) (res *AssignIPResponse, err error) {
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
