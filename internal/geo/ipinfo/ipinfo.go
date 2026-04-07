// Package ipinfo implements the ipinfo.io geolocation provider.
package ipinfo

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gechr/clog"
	ipinfolib "github.com/ipinfo/go/v2/ipinfo"
	"github.com/matcra587/peerscout/internal/geo"
	"golang.org/x/time/rate"
)

// Client is an ipinfo.io geolocation provider.
type Client struct {
	sdk     *ipinfolib.Client
	limiter *rate.Limiter
}

// New creates an ipinfo.io client with default settings.
func New(token string) *Client {
	return &Client{
		sdk:     ipinfolib.NewClient(nil, nil, token),
		limiter: rate.NewLimiter(rate.Limit(10), 10),
	}
}

// NewWithHTTP creates a client with a custom HTTP client and base URL (for testing).
func NewWithHTTP(httpClient *http.Client, baseURL, token string) *Client {
	sdk := ipinfolib.NewClient(httpClient, nil, token)

	u, _ := url.Parse(baseURL)
	sdk.BaseURL = u

	return &Client{
		sdk:     sdk,
		limiter: rate.NewLimiter(rate.Inf, 0),
	}
}

// Locate looks up country codes for the given IPs using the ipinfo
// batch API. Returns partial results on partial failure, empty map
// on total failure. Never returns an error.
//
// Context cancellation is checked before the rate-limit wait only;
// GetIPStrInfoBatch does not support caller-supplied cancellation.
func (c *Client) Locate(ctx context.Context, ips []string) map[string]geo.Location {
	if len(ips) == 0 {
		return nil
	}

	if err := c.limiter.Wait(ctx); err != nil {
		return nil
	}

	batch, err := c.sdk.GetIPStrInfoBatch(ips, ipinfolib.BatchReqOpts{
		TimeoutPerBatch: int64((10 * time.Second).Seconds()),
	})
	if err != nil {
		clog.Warn().Err(err).Msg("ipinfo batch lookup failed")

		if len(batch) == 0 {
			return nil
		}
	}

	result := make(map[string]geo.Location, len(batch))

	for ip, core := range batch {
		if core == nil || core.Country == "" {
			clog.Debug().Str("ip", ip).Msg("ipinfo returned no country")

			continue
		}

		result[ip] = geo.Location{
			CountryCode: core.Country,
			Country:     geo.CountryName(core.Country),
		}
	}

	return result
}
