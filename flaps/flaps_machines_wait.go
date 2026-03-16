package flaps

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-querystring/query"
)

type waitQuerystring struct {
	Version        string   `url:"version,omitempty"`
	FromEventID    string   `url:"from_event_id,omitempty"`
	TimeoutSeconds int      `url:"timeout,omitempty"`
	State          []string `url:"state,omitempty"`
}

const proxyTimeoutThreshold = 60 * time.Second

type WaitOption func(*waitOptions)

type waitOptions struct {
	states      []string
	timeout     time.Duration
	version     string
	fromEventID string
}

func WithWaitStates(states ...string) WaitOption {
	return func(opts *waitOptions) {
		opts.states = append(opts.states, states...)
	}
}

func WithWaitTimeout(timeout time.Duration) WaitOption {
	return func(opts *waitOptions) {
		opts.timeout = max(1*time.Second, min(timeout, proxyTimeoutThreshold))
	}
}

func WithWaitVersion(version string) WaitOption {
	return func(opts *waitOptions) {
		opts.version = version
	}
}

func WithWaitFromEventID(fromEventID string) WaitOption {
	return func(opts *waitOptions) {
		opts.fromEventID = fromEventID
	}
}

func (f *Client) Wait(ctx context.Context, appName string, machineID string, waitOpts ...WaitOption) (err error) {
	waitEndpoint := fmt.Sprintf("/%s/wait", machineID)

	opts := waitOptions{
		timeout: proxyTimeoutThreshold,
	}
	for _, waitOpt := range waitOpts {
		waitOpt(&opts)
	}

	if len(opts.states) == 0 {
		opts.states = []string{"started"}
	}
	waitQs := waitQuerystring{
		Version:        opts.version,
		FromEventID:    opts.fromEventID,
		TimeoutSeconds: int(opts.timeout.Seconds()),
		State:          opts.states,
	}
	qsVals, err := query.Values(waitQs)
	if err != nil {
		return fmt.Errorf("error making query string for wait request: %w", err)
	}
	ctx = contextWithAction(ctx, machineWait)
	ctx = contextWithMachineID(ctx, machineID)

	waitEndpoint += fmt.Sprintf("?%s", qsVals.Encode())
	if err := f.sendRequestMachines(ctx, appName, http.MethodGet, waitEndpoint, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to wait for VM %s in %v state: %w", machineID, opts.states, err)
	}
	return
}
