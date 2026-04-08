package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfigValue_Country(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    any
		wantErr bool
	}{
		{name: "valid single", input: "GB", want: []string{"GB"}},
		{name: "valid multiple", input: "gb,us", want: []string{"GB", "US"}},
		{name: "trims spaces", input: " DE , FR ", want: []string{"DE", "FR"}},
		{name: "invalid length", input: "GBR", wantErr: true},
		{name: "invalid chars", input: "G1", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseConfigValue("country", tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseConfigValue_MaxRetries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    any
		wantErr bool
	}{
		{name: "valid", input: "10", want: 10},
		{name: "zero", input: "0", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
		{name: "not a number", input: "abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseConfigValue("max_retries", tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

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
