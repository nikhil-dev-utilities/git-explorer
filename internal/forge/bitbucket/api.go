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
func fetchAll[T any](ctx context.Context, a *Adapter, host forge.Host, first string) ([]T, error) {
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
		if err := a.get(ctx, host, next, &p); err != nil {
			return all, err
		}
		all = append(all, p.Values...)
		next = p.Next
	}
	return all, nil
}

// get performs one authenticated GET against host's API and decodes the JSON body into out.
func (a *Adapter) get(ctx context.Context, host forge.Host, rawURL string, out any) error {
	cred, err := a.credential(ctx, host)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	if cred.username == "" {
		req.Header.Set("Authorization", "Bearer "+cred.password)
	} else {
		req.SetBasicAuth(cred.username, cred.password)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.http.Do(req)
	if err != nil {
		return &forge.Error{Kind: forge.ErrKindPaneScoped, Message: "could not reach Bitbucket.", Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "bitbucket request failed", "path", req.URL.Path, "status", resp.StatusCode)
		if resp.StatusCode == http.StatusUnauthorized {
			a.forgetCredential(host)
		}
		return statusError(resp, isDataCenter(host))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &forge.Error{Kind: forge.ErrKindPaneScoped, Message: "could not read Bitbucket's response.", Err: err}
	}
	return nil
}

// statusError maps a non-200 response onto the failure surfaces in forge.ErrorKind.
func statusError(resp *http.Response, dataCenter bool) error {
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		msg := "Bitbucket rejected the credential (401). Check the email and API token."
		if dataCenter {
			msg = "Bitbucket rejected the credential (401). Check the username and HTTP access token."
		}
		return &forge.Error{Kind: forge.ErrKindFatal, Message: msg}
	case http.StatusForbidden:
		msg := "Bitbucket refused access (403). The API token needs the read:workspace:bitbucket " +
			"and read:repository:bitbucket scopes."
		if dataCenter {
			msg = "Bitbucket refused access (403). The HTTP access token needs Project read and Repository read permission."
		}
		return &forge.Error{Kind: forge.ErrKindPaneScoped, Message: msg}
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

// credential resolves host's credential once and keeps it for the session (ADR-0011).
// Cloud asks git's store for its API host; Data Center serves REST and git from one host.
func (a *Adapter) credential(ctx context.Context, host forge.Host) (credential, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if c, ok := a.creds[host.Name]; ok {
		return c, nil
	}
	apiHost := host.Name
	if !isDataCenter(host) {
		u, err := url.Parse(a.baseURL)
		if err != nil {
			return credential{}, err
		}
		apiHost = u.Hostname()
	}
	c, err := resolveCredential(ctx, a.getenv, a.fill, apiHost, isDataCenter(host))
	if err != nil {
		return credential{}, err
	}
	if a.creds == nil {
		a.creds = map[string]credential{}
	}
	a.creds[host.Name] = c
	return c, nil
}

// forgetCredential drops a rejected credential, so a reload after fixing it retries.
func (a *Adapter) forgetCredential(host forge.Host) {
	a.mu.Lock()
	delete(a.creds, host.Name)
	a.mu.Unlock()
}
