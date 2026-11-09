package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client is the Lucida API transport: bearer key, project-scoped paths, RFC 9457 errors, and a
// small retry on 429/5xx. It satisfies apiremote.API.
type Client struct {
	Host      string
	APIKey    string
	ProjectID string
	HTTP      *http.Client
	Backoff   time.Duration // base wait between retries (default 500ms)
}

// APIError is a non-2xx answer. Detail comes from the problem+json body; nothing else is echoed.
type APIError struct {
	Status int
	Detail string
}

func (e *APIError) Error() string { return fmt.Sprintf("lucida api %d: %s", e.Status, e.Detail) }

// IsNotFound reports a 404 (inaccessible resources are 404 too).
func IsNotFound(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.Status == http.StatusNotFound
}

// Do performs a project-scoped request ("/flags" -> /v1/projects/<id>/flags).
func (c *Client) Do(ctx context.Context, method, suffix string, q url.Values, body, out any) error {
	return c.Root(ctx, method, "/v1/projects/"+url.PathEscape(c.ProjectID)+suffix, q, body, out)
}

// Root performs a request on an absolute API path such as /v1/projects/<id>.
func (c *Client) Root(ctx context.Context, method, path string, q url.Values, body, out any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	u := strings.TrimRight(c.Host, "/") + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var err error
	for attempt := 0; attempt < 4; attempt++ {
		var wait time.Duration
		if err = c.once(ctx, method, u, payload, out, &wait); err == nil {
			return nil
		}
		var ae *APIError
		if errors.As(err, &ae) && ae.Status != 429 && ae.Status < 500 {
			return err
		}
		if wait == 0 {
			base := c.Backoff
			if base == 0 {
				base = 500 * time.Millisecond
			}
			wait = base << attempt
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return err
}

func (c *Client) once(ctx context.Context, method, u string, payload []byte, out any, wait *time.Duration) error {
	req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "terraform-provider-lucida")
	req.Header.Set("X-Lucida-Client", "terraform")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		var p struct{ Detail, Title string }
		_ = json.Unmarshal(b, &p)
		d := p.Detail
		if d == "" {
			d = p.Title
		}
		if d == "" {
			d = http.StatusText(resp.StatusCode)
		}
		if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && s >= 0 && s <= 30 {
			*wait = time.Duration(s) * time.Second
		}
		return &APIError{Status: resp.StatusCode, Detail: d}
	}
	if out == nil || len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, out)
}
