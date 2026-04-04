package agent

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccess_Structure(t *testing.T) {
	t.Parallel()
	env := Success("find", []string{"peer1", "peer2"}, []string{"use -n for more"})
	assert.True(t, env.OK)
	assert.Equal(t, "find", env.Command)
	assert.Equal(t, []string{"peer1", "peer2"}, env.Data)
	assert.Equal(t, []string{"use -n for more"}, env.Hints)
	assert.Nil(t, env.Err)
}

func TestSuccess_JSONMarshal(t *testing.T) {
	t.Parallel()
	env := Success("list", []string{"cosmos", "dydx"}, nil)
	data, err := json.Marshal(env)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Equal(t, true, m["success"])
	assert.Equal(t, "list", m["command"])
	assert.NotNil(t, m["data"])
	assert.Nil(t, m["hints"])
	assert.Nil(t, m["error"])
}

func TestError_Structure(t *testing.T) {
	t.Parallel()
	env := Error("find", 404, "unknown network", "run peerscout list")
	assert.False(t, env.OK)
	assert.Equal(t, "find", env.Command)
	assert.Nil(t, env.Data)
	require.NotNil(t, env.Err)
	assert.Equal(t, 404, env.Err.Code)
	assert.Equal(t, "unknown network", env.Err.Message)
	assert.Equal(t, "run peerscout list", env.Err.Suggestion)
}

func TestError_JSONMarshal(t *testing.T) {
	t.Parallel()
	env := Error("find", 1, "connection refused", "")
	data, err := json.Marshal(env)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Equal(t, false, m["success"])
	assert.Nil(t, m["data"])
	assert.NotNil(t, m["error"])
}

func TestSuccess_HintsOmittedWhenNil(t *testing.T) {
	t.Parallel()
	env := Success("list", "data", nil)
	data, err := json.Marshal(env)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	_, hasHints := m["hints"]
	assert.False(t, hasHints)
}
