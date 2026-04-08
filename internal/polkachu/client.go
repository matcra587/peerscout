// Package polkachu provides an HTTP client for the Polkachu API v2.
package polkachu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gechr/clog"
	"github.com/matcra587/peerscout/internal/version"
)

const baseURL = "https://polkachu.com/api/v2"

// Client communicates with the Polkachu API v2.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a Polkachu API client with sensible defaults.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse // API should not redirect
			},
		},
		baseURL: baseURL,
	}
}

// NewClientWithHTTP creates a client with a custom HTTP client (for testing).
func NewClientWithHTTP(httpClient *http.Client, base string) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    base,
	}
}

// ListChains returns the list of all supported chain names.
func (c *Client) ListChains(ctx context.Context) ([]string, error) {
	var chains []string
	if err := c.get(ctx, "/chains", &chains); err != nil {
		return nil, fmt.Errorf("listing chains: %w", err)
	}
	return chains, nil
}

// ChainDetail returns detailed information for a specific chain.
func (c *Client) ChainDetail(ctx context.Context, network string) (ChainDetail, error) {
	var detail ChainDetail
	if err := c.get(ctx, "/chains/"+url.PathEscape(network), &detail); err != nil {
		return ChainDetail{}, fmt.Errorf("getting chain detail for %q: %w", network, err)
	}
	return detail, nil
}

// FetchLivePeers returns live peers for a specific network.
func (c *Client) FetchLivePeers(ctx context.Context, network string) (ChainLivePeers, error) {
	var peers ChainLivePeers
	if err := c.get(ctx, "/chains/"+url.PathEscape(network)+"/live_peers", &peers); err != nil {
		return ChainLivePeers{}, fmt.Errorf("fetching live peers for %q: %w", network, err)
	}
	return peers, nil
}

// CheckLivePeersActive returns true if the given network has live peers available.
func (c *Client) CheckLivePeersActive(ctx context.Context, network string) (bool, error) {
	detail, err := c.ChainDetail(ctx, network)
	if err != nil {
		return false, err
	}
	return detail.Services.LivePeers.Active, nil
}

const maxRetries = 3

func (c *Client) get(ctx context.Context, path string, target any) error {
	for attempt := range maxRetries {
		err := c.doGet(ctx, path, target)
		if err == nil {
			return nil
		}

		rateLimited, ok := errors.AsType[*RateLimitError](err)
		if !ok {
			return err
		}

		wait := rateLimited.RetryAfter
		if wait <= 0 {
			wait = time.Duration(1<<attempt) * time.Second
		}
		if wait > 60*time.Second {
			wait = 60 * time.Second
		}

		clog.Debug().Int("status_code", http.StatusTooManyRequests).Duration("retry_after", wait).Int("attempt", attempt+1).Msg("rate limited")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return fmt.Errorf("rate limited after %d retries", maxRetries)
}

func (c *Client) doGet(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "peerscout/"+version.Version)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return &RateLimitError{RetryAfter: retryAfter}
	}

	if resp.StatusCode == http.StatusNotFound {
		var errResp ErrorResponse
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil {
			return &NotFoundError{Network: path, Message: errResp.Message}
		}
		return &NotFoundError{Network: path, Message: "not found"}
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil {
			return fmt.Errorf("api error (HTTP %d): %s", resp.StatusCode, errResp.Message)
		}
		return fmt.Errorf("api error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	seconds, err := strconv.Atoi(header)
	if err != nil {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// NotFoundError is returned when a chain is not found.
type NotFoundError struct {
	Network string
	Message string
}

func (e *NotFoundError) Error() string {
	return "network not found: " + e.Message
}
