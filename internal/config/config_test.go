package config_test

import (
	"os"
	"testing"

	"github.com/matcra587/peerscout/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_CountryFromEnv(t *testing.T) {
	t.Setenv("PEERSCOUT_COUNTRY", "GB,US")

	cfg, err := config.Load("", nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"GB", "US"}, cfg.Country)
}

func TestLoad_MaxRetriesDefault(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load("", nil)
	require.NoError(t, err)

	assert.Equal(t, 5, cfg.MaxRetries)
}

func TestLoad_MaxRetriesFromEnv(t *testing.T) {
	t.Setenv("PEERSCOUT_MAX_RETRIES", "10")

	cfg, err := config.Load("", nil)
	require.NoError(t, err)

	assert.Equal(t, 10, cfg.MaxRetries)
}

func TestLoad_CountryFromFile(t *testing.T) {
	t.Parallel()
	cfgFile := t.TempDir() + "/config.toml"
	err := os.WriteFile(cfgFile, []byte(`country = ["DE", "FR"]`), 0o600)
	require.NoError(t, err)

	cfg, err := config.Load(cfgFile, nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"DE", "FR"}, cfg.Country)
}
