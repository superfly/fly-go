package flaps

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"slices"

	"github.com/cenkalti/backoff/v4"
	"github.com/superfly/fly-go"
)

func (f *Client) CreateApp(ctx context.Context, name string, org string) (err error) {
	in := map[string]any{
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
		if ferr, ok := err.(*FlapsError); ok && slices.Contains([]int{404, 401, 403}, ferr.ResponseStatusCode) {
			return err
		}
		return backoff.Permanent(err)
	}
	return Retry(ctx, op)
}

// todo: proper types / export
func (f *Client) getApp(ctx context.Context, name string) (app *fly.AppBasic, err error) {
	err = f._sendRequest(ctx, http.MethodGet, "/apps/"+url.PathEscape(name), nil, &app, nil)
	return
}

func (f *Client) AppNameAvailable(ctx context.Context, name string) (ok bool, err error) {
	_, err = f.getApp(ctx, name)
	log.Println(err.Error())
	switch {
	case err == nil:
		return
	case err.Error() == "unauthorized":
		ok = false
		err = nil
	case err.Error() == "app not found":
		ok = true
		err = nil
	}
	return
}
