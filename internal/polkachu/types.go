package polkachu

import "time"

// ChainDetail holds the detailed information for a single chain.
type ChainDetail struct {
	Network  string           `json:"network"`
	Name     string           `json:"name"`
	ChainID  string           `json:"chain_id"`
	Services PolkachuServices `json:"polkachu_services"`
}

// PolkachuServices describes the services Polkachu offers for a chain.
type PolkachuServices struct {
	LivePeers ServiceStatus `json:"live_peers"`
}

// ServiceStatus indicates whether a Polkachu service is active.
type ServiceStatus struct {
	Active bool `json:"active"`
}

// ChainLivePeers holds the live peers response for a chain.
type ChainLivePeers struct {
	Network      string   `json:"network"`
	PolkachuPeer string   `json:"polkachu_peer"`
	LivePeers    []string `json:"live_peers"`
}

// ErrorResponse is returned on 404/500 errors.
type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RateLimitError is returned when the API responds with 429.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return "rate limited by Polkachu API"
}
