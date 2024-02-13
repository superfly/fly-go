package client

import (
	"errors"

	"github.com/superfly/fly-go/api"
	"github.com/superfly/fly-go/api/tokens"
	//"github.com/superfly/fly-go/api/tokens"
	//"github.com/superfly/fly-go/internal/logger"
	//"github.com/superfly/fly-go/iostreams"
)

var ErrNoAuthToken = errors.New("No access token available. Please login with 'flyctl auth login'")

type Client struct {
	api *api.Client
}

func (c *Client) API() *api.Client {
	return c.api
}

func (c *Client) Authenticated() bool {
	return c.api.Authenticated()
}

type NewClientOpts struct {
	Token         string
	Tokens        *tokens.Tokens
	ClientName    string
	ClientVersion string
	Logger        api.Logger
}

// NewClientWithOptions returns a new instance of Client.
func NewClientWithOptions(opts *NewClientOpts) *Client {
	return &Client{
		api: api.NewClientFromOptions(api.ClientOptions{
			Name:        opts.ClientName,
			Version:     opts.ClientVersion,
			AccessToken: opts.Token,
			Tokens:      opts.Tokens,
			Logger:      opts.Logger,
		}),
	}
}
