package flaps

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	fly "github.com/superfly/fly-go"
)

func (f *Client) sendRequestCertificates(ctx context.Context, appName, method, endpoint string, in, out any, headers map[string][]string) error {
	endpoint = fmt.Sprintf("/apps/%s/certificates%s", url.PathEscape(appName), endpoint)
	return f._sendRequest(ctx, method, endpoint, in, out, headers)
}

func (f *Client) ListCertificates(ctx context.Context, appName string) (*fly.ListCertificatesResponse, error) {
	ctx = contextWithAction(ctx, certificateList)

	out := new(fly.ListCertificatesResponse)
	if err := f.sendRequestCertificates(ctx, appName, http.MethodGet, "", nil, out, nil); err != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", err)
	}
	return out, nil
}

// Add a hostname and start ACME issuance.
func (f *Client) CreateACMECertificate(ctx context.Context, appName string, req fly.CreateCertificateRequest) (*fly.CertificateDetailResponse, error) {
	ctx = contextWithAction(ctx, certificateCreateACME)

	out := new(fly.CertificateDetailResponse)
	if err := f.sendRequestCertificates(ctx, appName, http.MethodPost, "/acme", req, out, nil); err != nil {
		return nil, fmt.Errorf("failed to create ACME certificate: %w", err)
	}
	return out, nil
}

// Add a custom certificate. Will not start ACME issuance if not already present.
func (f *Client) CreateCustomCertificate(ctx context.Context, appName string, req fly.ImportCertificateRequest) (*fly.CertificateDetailResponse, error) {
	ctx = contextWithAction(ctx, certificateCreateCustom)

	out := new(fly.CertificateDetailResponse)
	if err := f.sendRequestCertificates(ctx, appName, http.MethodPost, "/custom", req, out, nil); err != nil {
		return nil, fmt.Errorf("failed to import certificate: %w", err)
	}
	return out, nil
}

func (f *Client) GetCertificate(ctx context.Context, appName, hostname string) (*fly.CertificateDetailResponse, error) {
	ctx = contextWithAction(ctx, certificateGet)

	out := new(fly.CertificateDetailResponse)
	endpoint := fmt.Sprintf("/%s", url.PathEscape(hostname))
	if err := f.sendRequestCertificates(ctx, appName, http.MethodGet, endpoint, nil, out, nil); err != nil {
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}
	return out, nil
}

// Triggers DNS validation + ACME issuance if required.
func (f *Client) CheckCertificate(ctx context.Context, appName, hostname string) (*fly.CertificateDetailResponse, error) {
	ctx = contextWithAction(ctx, certificateCheck)

	out := new(fly.CertificateDetailResponse)
	endpoint := fmt.Sprintf("/%s/check", url.PathEscape(hostname))
	if err := f.sendRequestCertificates(ctx, appName, http.MethodPost, endpoint, nil, out, nil); err != nil {
		return nil, fmt.Errorf("failed to check certificate: %w", err)
	}
	return out, nil
}

// Removes hostname and all certificates.
func (f *Client) DeleteCertificate(ctx context.Context, appName, hostname string) error {
	ctx = contextWithAction(ctx, certificateDelete)

	endpoint := fmt.Sprintf("/%s", url.PathEscape(hostname))
	if err := f.sendRequestCertificates(ctx, appName, http.MethodDelete, endpoint, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to delete certificate: %w", err)
	}
	return nil
}

// Removes ACME certificates and stops renewals, leaving a custom certificate in place.
func (f *Client) DeleteACMECertificate(ctx context.Context, appName, hostname string) error {
	ctx = contextWithAction(ctx, certificateDeleteACME)

	endpoint := fmt.Sprintf("/%s/acme", url.PathEscape(hostname))
	if err := f.sendRequestCertificates(ctx, appName, http.MethodDelete, endpoint, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to stop ACME certificate: %w", err)
	}
	return nil
}

// Removes a custom certificate, leaving the ACME certificates in place.
func (f *Client) DeleteCustomCertificate(ctx context.Context, appName, hostname string) error {
	ctx = contextWithAction(ctx, certificateDeleteCustom)

	endpoint := fmt.Sprintf("/%s/custom", url.PathEscape(hostname))
	if err := f.sendRequestCertificates(ctx, appName, http.MethodDelete, endpoint, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to delete custom certificate: %w", err)
	}
	return nil
}
