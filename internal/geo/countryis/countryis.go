// Package countryis implements the country.is geolocation provider.
package countryis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gechr/clog"
	"github.com/matcra587/peerscout/internal/geo"
	"golang.org/x/time/rate"
)

const defaultBaseURL = "https://api.country.is"

// Client is a country.is geolocation provider.
type Client struct {
	httpClient *http.Client
	baseURL    string
	limiter    *rate.Limiter
}

// New creates a country.is client with default settings.
func New() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    defaultBaseURL,
		limiter:    rate.NewLimiter(rate.Limit(10), 10),
	}
}

// NewWithHTTP creates a client with a custom HTTP client and base URL (for testing).
func NewWithHTTP(httpClient *http.Client, baseURL string) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		limiter:    rate.NewLimiter(rate.Inf, 0),
	}
}

type countryResponse struct {
	Country string `json:"country"`
}

// Locate looks up country codes for the given IPs. Returns partial
// results on partial failure, empty map on total failure. Never
// returns an error.
func (c *Client) Locate(ctx context.Context, ips []string) map[string]geo.Location {
	if len(ips) == 0 {
		return nil
	}

	result := make(map[string]geo.Location, len(ips))
	for _, ip := range ips {
		if err := ctx.Err(); err != nil {
			break
		}
		if err := c.limiter.Wait(ctx); err != nil {
			break
		}
		loc, err := c.lookup(ctx, ip)
		if err != nil {
			clog.Debug().Err(err).Str("ip", ip).Msg("geo lookup failed")
			continue
		}
		result[ip] = loc
	}

	return result
}

func (c *Client) lookup(ctx context.Context, ip string) (geo.Location, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/"+url.PathEscape(ip), nil)
	if err != nil {
		return geo.Location{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return geo.Location{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return geo.Location{}, fmt.Errorf("country.is returned HTTP %d for %s", resp.StatusCode, ip)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return geo.Location{}, err
	}

	var cr countryResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return geo.Location{}, err
	}

	return geo.Location{
		CountryCode: cr.Country,
		Country:     geo.CountryName(cr.Country),
	}, nil
}
