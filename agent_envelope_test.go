package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentMode_JSONEnvelope verifies that every command produces a
// valid JSON envelope when running in agent mode. Commands that talk
// to the Polkachu API (find, list) are excluded because they need a
// live server or httptest mock.
func TestAgentMode_JSONEnvelope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		command string
	}{
		{name: "version", args: []string{"version"}, command: "version"},
		{name: "version --short", args: []string{"version", "--short"}, command: "version"},
		{name: "config list", args: []string{"config", "list"}, command: "config list"},
		{name: "config get", args: []string{"config", "get", "count"}, command: "config get"},
		{name: "config path", args: []string{"config", "path"}, command: "config path"},
		{name: "agent schema", args: []string{"agent", "schema"}, command: "agent schema"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfgFile := filepath.Join(t.TempDir(), "config.toml")
			require.NoError(t, os.WriteFile(cfgFile, nil, 0o600))

			var buf bytes.Buffer
			root := newRootCmd()
			root.SetOut(&buf)
			root.SetArgs(append(tt.args, "--agent", "--config", cfgFile))

			err := root.Execute()
			require.NoError(t, err)

			var env map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &env),
				"output must be valid JSON envelope, got: %s", buf.String())
			assert.Equal(t, true, env["success"])
			assert.Equal(t, tt.command, env["command"])
			assert.NotNil(t, env["data"])
		})
	}
}

// TestAgentMode_ConfigSet_JSONEnvelope verifies config set produces
// an envelope confirming the write.
func TestAgentMode_ConfigSet_JSONEnvelope(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfgFile := filepath.Join(tmp, "config.toml")

	var buf bytes.Buffer
	root := newRootCmd()
	root.SetOut(&buf)
	root.SetArgs([]string{"config", "set", "count", "10", "--agent", "--config", cfgFile})

	err := root.Execute()
	require.NoError(t, err)

	var env map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &env),
		"output must be valid JSON envelope, got: %s", buf.String())
	assert.Equal(t, true, env["success"])
	assert.Equal(t, "config set", env["command"])
	assert.NotNil(t, env["data"])
}

// TestAgentMode_ConfigUnset_JSONEnvelope verifies config unset
// produces an envelope confirming the removal.
func TestAgentMode_ConfigUnset_JSONEnvelope(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfgFile := filepath.Join(tmp, "config.toml")

	// Seed a config file so there is something to unset.
	err := os.WriteFile(cfgFile, []byte("count = 10\n"), 0o600)
	require.NoError(t, err)

	var buf bytes.Buffer
	root := newRootCmd()
	root.SetOut(&buf)
	root.SetArgs([]string{"config", "unset", "count", "--agent", "--config", cfgFile})

	err = root.Execute()
	require.NoError(t, err)

	var env map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &env),
		"output must be valid JSON envelope, got: %s", buf.String())
	assert.Equal(t, true, env["success"])
	assert.Equal(t, "config unset", env["command"])
	assert.NotNil(t, env["data"])
}
