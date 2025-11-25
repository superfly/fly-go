package flaps

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"slices"

	"github.com/cenkalti/backoff/v4"
)

type CreateAppRequest struct {
	Name      string `json:"name"`
	Org       string `json:"org_slug"`
	Network   string `json:"network"`
	AppRoleID string `json:"app_role_id"`
}

func (f *Client) CreateApp(ctx context.Context, in CreateAppRequest) (app *App, err error) {
	ctx = contextWithAction(ctx, appCreate)
	err = f._sendRequest(ctx, http.MethodPost, "/apps?wait=true", in, &app, nil)
	return
}

type AppOrganizationInfo struct {
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	InternalNumericID int32  `json:"internal_numeric_id"`
}

type App struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	InternalNumericID int32  `json:"internal_numeric_id"`
	Network           string `json:"network"`
	Status            string `json:"status"`

	MachineCount int64 `json:"machine_count"`
	VolumeCount  int64 `json:"volume_count"`

	Organization AppOrganizationInfo `json:"organization"`

	// hashid.appname.fly.dev, for ACME HTTP-01 pointing only to v6
	CnameTarget string `json:"cname_target"`
	// a role like "postgres_cluster" or "remote-docker-builder"
	AppRole string `json:"app_role"`
}

func (a *App) Deployed() bool {
	return a.Status == "deployed" || a.Status == "suspended"
}

func (a *App) IsPostgresApp() bool {
	return a.AppRole == "postgres_cluster"
}

func (f *Client) GetApp(ctx context.Context, name string) (app *App, err error) {
	err = f._sendRequest(ctx, http.MethodGet, "/apps/"+url.PathEscape(name), nil, &app, nil)
	return
}

type ListAppsRequest struct {
	OrgSlug string
	// AppRole is optional
	AppRole string
}

func (f *Client) ListApps(ctx context.Context, req ListAppsRequest) (apps []App, err error) {
	var res struct {
		Apps []App `json:"apps"`
	}

	query := url.Values{}
	query.Set("org_slug", req.OrgSlug)
	if req.AppRole != "" {
		query.Set("app_role", req.AppRole)
	}

	err = f._sendRequest(ctx, http.MethodGet, "/apps?"+query.Encode(), nil, &res, nil)
	if err == nil {
		apps = res.Apps
	}
	return
}

func (f *Client) DeleteApp(ctx context.Context, name string) error {
	return f._sendRequest(ctx, http.MethodDelete, "/apps/"+name, nil, nil, nil)
}

func (f *Client) AppNameAvailable(ctx context.Context, name string) (ok bool, err error) {
	_, err = f.GetApp(ctx, name)
	log.Println(err.Error())
	switch {
	case err == nil:
		// app was found
		return
	case err.Error() == "unauthorized":
		// app exists, but in an org we do not have access to
		err = nil
	case err.Error() == "app not found":
		ok = true
		err = nil
	}
	return
}

func (f *Client) WaitForApp(ctx context.Context, name string) error {
	ctx = contextWithAction(ctx, appGet)

	var op = func() error {
		err := f._sendRequest(ctx, http.MethodGet, "/apps/"+url.PathEscape(name), nil, nil, nil)
		if err == nil {
			return nil
		}
		if ferr, ok := err.(*FlapsError); ok && slices.Contains([]int{404, 401}, ferr.ResponseStatusCode) {
			return err
		}
		return backoff.Permanent(err)
	}
	return Retry(ctx, op)
}
