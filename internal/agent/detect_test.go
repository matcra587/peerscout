package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// clearAgentEnv unsets all known agent env vars for the test.
func clearAgentEnv(t *testing.T) {
	t.Helper()
	for _, e := range envVars {
		t.Setenv(e.key, "")
	}
}

func TestDetect_NoAgent(t *testing.T) {
	clearAgentEnv(t)
	r := Detect()
	assert.False(t, r.Active)
	assert.Empty(t, r.Name)
}

func TestDetect_ClaudeCode(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDE_CODE", "1")
	r := Detect()
	assert.True(t, r.Active)
	assert.Equal(t, "Claude Code", r.Name)
}

func TestDetect_ClaudeCodeAlt(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDECODE", "true")
	r := Detect()
	assert.True(t, r.Active)
	assert.Equal(t, "Claude Code", r.Name)
}

func TestDetect_Cursor(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CURSOR_AGENT", "1")
	r := Detect()
	assert.True(t, r.Active)
	assert.Equal(t, "Cursor", r.Name)
}

func TestDetect_FalseValue(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDE_CODE", "0")
	r := Detect()
	assert.False(t, r.Active)
}

func TestDetect_FalseString(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDE_CODE", "false")
	r := Detect()
	assert.False(t, r.Active)
}

func TestDetect_NoString(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDE_CODE", "no")
	r := Detect()
	assert.False(t, r.Active)
}

func TestDetect_ForceAgentMode(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("FORCE_AGENT_MODE", "1")
	r := Detect()
	assert.True(t, r.Active)
	assert.Equal(t, "Unknown", r.Name)
}

func TestDetectWithFlag_FlagTrue_NoEnv(t *testing.T) {
	clearAgentEnv(t)
	r := DetectWithFlag(true)
	assert.True(t, r.Active)
	assert.Equal(t, "manual", r.Name)
}

func TestDetectWithFlag_FlagTrue_WithEnv(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDE_CODE", "1")
	r := DetectWithFlag(true)
	assert.True(t, r.Active)
	assert.Equal(t, "Claude Code", r.Name)
}

func TestDetectWithFlag_FlagFalse_NoEnv(t *testing.T) {
	clearAgentEnv(t)
	r := DetectWithFlag(false)
	assert.False(t, r.Active)
}
