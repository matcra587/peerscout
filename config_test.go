package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModifyConfigFile_Roundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	err := modifyConfigFile(cfgPath, func(doc map[string]any) {
		doc["count"] = 10
	})
	require.NoError(t, err)

	data, err := os.ReadFile(cfgPath) //nolint:gosec // test reads from t.TempDir()
	require.NoError(t, err)

	var got map[string]any
	_, err = toml.Decode(string(data), &got)
	require.NoError(t, err)
	assert.Equal(t, int64(10), got["count"])
}

func TestModifyConfigFile_CreatesDirectory(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nested", "dir")
	cfgPath := filepath.Join(dir, "config.toml")

	err := modifyConfigFile(cfgPath, func(doc map[string]any) {
		doc["count"] = 5
	})
	require.NoError(t, err)

	_, err = os.Stat(cfgPath)
	assert.NoError(t, err)
}

func TestModifyConfigFile_PreservesExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	require.NoError(t, modifyConfigFile(cfgPath, func(doc map[string]any) {
		doc["count"] = 5
	}))

	require.NoError(t, modifyConfigFile(cfgPath, func(doc map[string]any) {
		doc["other"] = "test"
	}))

	data, err := os.ReadFile(cfgPath) //nolint:gosec // test reads from t.TempDir()
	require.NoError(t, err)

	var got map[string]any
	_, err = toml.Decode(string(data), &got)
	require.NoError(t, err)
	assert.Equal(t, int64(5), got["count"])
	assert.Equal(t, "test", got["other"])
}
