// Package config handles PeerScout configuration loading and precedence.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	koanfpflag "github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/matcra587/peerscout/internal/dirs"
	"github.com/spf13/pflag"
)

// Config holds all PeerScout configuration.
type Config struct {
	Count int `koanf:"count"`

	// Global
	Debug     bool   `koanf:"debug"`
	NoColor   bool   `koanf:"no_color"`
	LogFormat string `koanf:"log_format"`

	// Geolocation
	GeoProvider string `koanf:"geo_provider"`
	GeoToken    string `koanf:"geo_token"`
}

// Defaults returns a Config with compiled defaults.
func Defaults() Config {
	return Config{
		Count:       5,
		LogFormat:   "auto",
		GeoProvider: "countryis",
	}
}

// Load reads configuration from the given sources in precedence order:
// compiled defaults < TOML file < env vars < CLI flags.
func Load(configPath string, flags *pflag.FlagSet) (Config, error) {
	k := koanf.New(".")

	// 1. Compiled defaults
	cfg := Defaults()

	// 2. Config file (explicit path or auto-discovered default)
	cfgFile := configPath
	if cfgFile == "" {
		if p, err := defaultConfigPath(); err == nil {
			if _, err := os.Stat(p); err == nil {
				cfgFile = p
			}
		}
	}
	if cfgFile != "" {
		if err := k.Load(file.Provider(cfgFile), toml.Parser()); err != nil {
			return Config{}, fmt.Errorf("loading config file: %w", err)
		}
	}

	// 3. Environment variables (PEERSCOUT_ prefix)
	err := k.Load(env.Provider("PEERSCOUT_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "PEERSCOUT_"))
	}), nil)
	if err != nil {
		return Config{}, fmt.Errorf("loading env vars: %w", err)
	}

	// Also check NO_COLOR (standard convention, no prefix)
	err = k.Load(env.Provider("NO_COLOR", ".", func(s string) string {
		return "no_color"
	}), nil)
	if err != nil {
		return Config{}, fmt.Errorf("loading NO_COLOR env: %w", err)
	}

	// 4. CLI flags (only changed flags, mapping hyphens to underscores)
	if flags != nil {
		provider := koanfpflag.ProviderWithFlag(flags, ".", k, func(f *pflag.Flag) (string, any) {
			if !f.Changed {
				return "", nil
			}
			return strings.ReplaceAll(f.Name, "-", "_"), koanfpflag.FlagVal(flags, f)
		})
		if err := k.Load(provider, nil); err != nil {
			return Config{}, fmt.Errorf("loading CLI flags: %w", err)
		}
	}

	if err := k.Unmarshal("", &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshalling config: %w", err)
	}

	return cfg, nil
}

func defaultConfigPath() (string, error) {
	return dirs.DefaultConfigPath()
}
