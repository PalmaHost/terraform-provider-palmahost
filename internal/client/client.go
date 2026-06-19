// Package client is a small typed HTTP client for the PalmaHost Cloud API,
// used by the Terraform provider. It is hand-written (rather than generated)
// to stay small and robust; the public SDKs are generated from the same spec.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to the PalmaHost API with a Bearer token.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New builds a client. baseURL includes the version prefix
// (e.g. https://api.palmahost.sh/v1).
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// APIError is a non-2xx response from the API.
type APIError struct {
	Status  int
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("palmahost API %d: %s (%s)", e.Status, e.Message, e.Code)
	}
	return fmt.Sprintf("palmahost API %d", e.Status)
}

// Do performs a request. The API wraps successful payloads in {"data": …}; when
// out is non-nil the data envelope is unwrapped into it. A non-2xx status is
// returned as *APIError carrying the error envelope.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		ae := &APIError{Status: resp.StatusCode}
		var env struct {
			Error APIError `json:"error"`
		}
		if json.Unmarshal(raw, &env) == nil && env.Error.Message != "" {
			ae.Code, ae.Message = env.Error.Code, env.Error.Message
		}
		return ae
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	// Unwrap the {"data": …} envelope the API uses for single objects + lists.
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err == nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, out)
	}
	return json.Unmarshal(raw, out)
}
