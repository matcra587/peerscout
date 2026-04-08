// Package discovery owns the peer discovery pipeline:
// fetch → deduplicate → enrich → filter → retry.
package discovery

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/matcra587/peerscout/internal/geo"
	"github.com/matcra587/peerscout/internal/polkachu"
)

// EnrichedPeer is a peer address with optional geolocation data.
type EnrichedPeer struct {
	Address     string `json:"address"`
	CountryCode string `json:"country_code,omitempty"`
	Country     string `json:"country,omitempty"`
}

// Fetcher abstracts peer fetching for testability.
// *polkachu.Client satisfies this interface.
type Fetcher interface {
	FetchLivePeers(ctx context.Context, network string) (polkachu.ChainLivePeers, error)
}

// Locator looks up geographic locations for IP addresses.
type Locator interface {
	Locate(ctx context.Context, ips []string) map[string]geo.Location
}

// ProgressFunc reports the current count of matched peers and rounds completed.
type ProgressFunc func(found, rounds int)

// Opts configures a discovery run.
type Opts struct {
	Fetcher    Fetcher
	Network    string
	Count      int
	MaxRounds  int
	Locator    Locator
	Countries  []string
	OnProgress ProgressFunc
}

// Result holds the outcome of a discovery run.
type Result struct {
	Peers        []EnrichedPeer
	Duplicates   int
	Rounds       int
	APIExhausted bool
}

// Run executes the discovery pipeline: fetch → dedup → enrich → filter → retry.
func Run(ctx context.Context, opts Opts) (*Result, error) {
	seen := make(map[string]struct{})
	var matched []EnrichedPeer
	var duplicates int
	var apiExhausted bool
	rounds := 0

	for rounds < opts.MaxRounds {
		if err := ctx.Err(); err != nil {
			break
		}
		rounds++

		// Two fetches per round.
		var (
			mu      sync.Mutex
			results []polkachu.ChainLivePeers
			errs    []error
			wg      sync.WaitGroup
		)

		for range 2 {
			wg.Go(func() {
				resp, err := opts.Fetcher.FetchLivePeers(ctx, opts.Network)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					errs = append(errs, err)
					return
				}
				results = append(results, resp)
			})
		}
		wg.Wait()

		// If all fetches failed, return what we have or the error.
		if len(results) == 0 {
			if len(matched) > 0 {
				break
			}
			return nil, errors.Join(errs...)
		}

		// Deduplicate raw peers.
		var newPeers []string
		for _, r := range results {
			// Iterate LivePeers and PolkachuPeer separately to avoid
			// mutating the LivePeers backing array via append.
			for _, p := range r.LivePeers {
				if p == "" {
					continue
				}
				if _, ok := seen[p]; ok {
					duplicates++
					continue
				}
				seen[p] = struct{}{}
				newPeers = append(newPeers, p)
			}
			if p := r.PolkachuPeer; p != "" {
				if _, ok := seen[p]; ok {
					duplicates++
				} else {
					seen[p] = struct{}{}
					newPeers = append(newPeers, p)
				}
			}
		}

		// API exhausted — no new unique peers this round.
		if len(newPeers) == 0 {
			apiExhausted = true
			if opts.OnProgress != nil {
				opts.OnProgress(min(len(matched), opts.Count), rounds)
			}
			break
		}

		// Enrich with geo data.
		var enriched []EnrichedPeer
		if opts.Locator != nil {
			ips := geo.ExtractIPs(newPeers)
			locations := opts.Locator.Locate(ctx, ips)
			enriched = buildEnrichedPeers(newPeers, locations)
		} else {
			enriched = buildEnrichedPeers(newPeers, nil)
		}

		// Filter by country if requested.
		if len(opts.Countries) > 0 {
			enriched = filterByCountry(enriched, opts.Countries)
		}

		matched = append(matched, enriched...)

		if opts.OnProgress != nil {
			opts.OnProgress(min(len(matched), opts.Count), rounds)
		}

		if len(matched) >= opts.Count {
			break
		}
	}

	// Truncate to requested count.
	if opts.Count > 0 && len(matched) > opts.Count {
		matched = matched[:opts.Count]
	}

	return &Result{
		Peers:        matched,
		Duplicates:   duplicates,
		Rounds:       rounds,
		APIExhausted: apiExhausted,
	}, nil
}

// buildEnrichedPeers creates EnrichedPeer values from raw peer strings
// and an optional locations map.
func buildEnrichedPeers(peers []string, locations map[string]geo.Location) []EnrichedPeer {
	enriched := make([]EnrichedPeer, 0, len(peers))
	for _, p := range peers {
		ep := EnrichedPeer{Address: p}
		if _, hostPort, ok := strings.Cut(p, "@"); ok {
			if host, _, err := net.SplitHostPort(hostPort); err == nil {
				if loc, ok := locations[host]; ok {
					ep.CountryCode = loc.CountryCode
					ep.Country = loc.Country
				}
			}
		}
		enriched = append(enriched, ep)
	}
	return enriched
}

// filterByCountry returns peers whose CountryCode is in the allowed set.
// Peers with an empty CountryCode are excluded.
// The codes parameter must be pre-normalised to uppercase.
func filterByCountry(peers []EnrichedPeer, codes []string) []EnrichedPeer {
	if len(codes) == 0 {
		return nil
	}

	allowed := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		if c != "" {
			allowed[c] = struct{}{}
		}
	}

	var matched []EnrichedPeer
	for _, p := range peers {
		if p.CountryCode == "" {
			continue
		}
		if _, ok := allowed[p.CountryCode]; ok {
			matched = append(matched, p)
		}
	}
	return matched
}
