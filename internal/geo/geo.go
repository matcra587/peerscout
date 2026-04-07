// Package geo provides geolocation types and helpers for IP address lookup.
package geo

import (
	"net"
	"strings"
)

// Location holds the geolocation result for an IP address.
type Location struct {
	CountryCode string `json:"country_code,omitempty"`
	Country     string `json:"country,omitempty"`
}

// ExtractIPs parses peer addresses (nodeID@ip:port) and returns
// deduplicated IP strings. Malformed peers are silently skipped.
func ExtractIPs(peers []string) []string {
	seen := make(map[string]struct{})
	var ips []string
	for _, peer := range peers {
		_, hostPort, ok := strings.Cut(peer, "@")
		if !ok {
			continue
		}
		host, _, err := net.SplitHostPort(hostPort)
		if err != nil {
			continue
		}
		if _, exists := seen[host]; exists {
			continue
		}
		seen[host] = struct{}{}
		ips = append(ips, host)
	}
	return ips
}
