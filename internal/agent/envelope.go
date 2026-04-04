package agent

// Envelope is the top-level JSON structure returned in agent mode.
type Envelope struct {
	Success bool           `json:"success"`
	Command string         `json:"command"`
	Data    any            `json:"data,omitempty"`
	Hints   []string       `json:"hints,omitempty"`
	Err     *EnvelopeError `json:"error,omitempty"`
}

// EnvelopeError describes a failed operation.
type EnvelopeError struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Success builds an Envelope for a successful operation.
func Success(command string, data any, hints []string) Envelope {
	return Envelope{
		Success: true,
		Command: command,
		Data:    data,
		Hints:   hints,
	}
}

// Error builds an Envelope for a failed operation.
func Error(command string, code int, message, suggestion string) Envelope {
	return Envelope{
		Command: command,
		Err: &EnvelopeError{
			Code:       code,
			Message:    message,
			Suggestion: suggestion,
		},
	}
}
