package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// page is Bitbucket's pagination envelope. next is absent on the last page.
type page[T any] struct {
	Values []T    `json:"values"`
	Next   string `json:"next"`
}

// fetchAll GETs first and follows next links until the last page. A next link is
// only followed to the API's own host, so the credential is never sent elsewhere.
func fetchAll[T any](ctx context.Context, a *Adapter, first string) ([]T, error) {
	base, err := url.Parse(a.baseURL)
	if err != nil {
		return nil, err
	}
	var all []T
	for next := first; next != ""; {
		u, err := url.Parse(next)
		if err != nil || u.Host != base.Host {
			return all, &forge.Error{Kind: forge.ErrKindPaneScoped, Message: "Bitbucket returned an unexpected next page link."}
		}
		var p page[T]
		if err := a.get(ctx, next, &p); err != nil {
			return all, err
		}
		all = append(all, p.Values...)
		next = p.Next
	}
	return all, nil
}

// get performs one authenticated GET and decodes the JSON body into out.
func (a *Adapter) get(ctx context.Context, rawURL string, out any) error {
	cred, err := a.credential(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(cred.username, cred.password)
	req.Header.Set("Accept", "application/json")

	resp, err := a.http.Do(req)
	if err != nil {
		return &forge.Error{Kind: forge.ErrKindPaneScoped, Message: "could not reach Bitbucket.", Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "bitbucket request failed", "path", req.URL.Path, "status", resp.StatusCode)
		if resp.StatusCode == http.StatusUnauthorized {
			a.forgetCredential()
		}
		return statusError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &forge.Error{Kind: forge.ErrKindPaneScoped, Message: "could not read Bitbucket's response.", Err: err}
	}
	return nil
}

// statusError maps a non-200 response onto the failure surfaces in forge.ErrorKind.
func statusError(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &forge.Error{
			Kind:    forge.ErrKindFatal,
			Message: "Bitbucket rejected the credential (401). Check the email and API token.",
		}
	case http.StatusForbidden:
		return &forge.Error{
			Kind: forge.ErrKindPaneScoped,
			Message: "Bitbucket refused access (403). The API token needs the read:workspace:bitbucket " +
				"and read:repository:bitbucket scopes.",
		}
	case http.StatusTooManyRequests:
		secs, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
		return &forge.Error{
			Kind:       forge.ErrKindTransient,
			Message:    "Bitbucket rate limit reached.",
			RetryAfter: time.Duration(secs) * time.Second,
		}
	default:
		return &forge.Error{
			Kind:    forge.ErrKindPaneScoped,
			Message: fmt.Sprintf("Bitbucket request failed: %s.", resp.Status),
		}
	}
}

// credential resolves the credential once and keeps it for the session (ADR-0011).
func (a *Adapter) credential(ctx context.Context) (credential, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cred != nil {
		return *a.cred, nil
	}
	u, err := url.Parse(a.baseURL)
	if err != nil {
		return credential{}, err
	}
	c, err := resolveCredential(ctx, a.getenv, a.fill, u.Hostname())
	if err != nil {
		return credential{}, err
	}
	a.cred = &c
	return c, nil
}

// forgetCredential drops a rejected credential, so a reload after fixing it retries.
func (a *Adapter) forgetCredential() {
	a.mu.Lock()
	a.cred = nil
	a.mu.Unlock()
}
