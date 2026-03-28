// Package dirs provides platform-aware directory helpers for peerscout.
package dirs

import (
	"os"
	"path/filepath"
	"runtime"
)

const appName = "peerscout"

// ConfigDir returns the configuration directory for peerscout.
//
// Resolution:
//   - $XDG_CONFIG_HOME/peerscout if set (all platforms)
//   - ~/.config/peerscout on macOS (override Library path)
//   - os.UserConfigDir()/peerscout elsewhere (honours XDG on Linux)
func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appName), nil
	}

	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", appName), nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}

// DefaultConfigPath returns the full path to the default config file.
func DefaultConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// CacheDir returns the cache directory for peerscout.
//
// Resolution:
//   - $XDG_CACHE_HOME/peerscout if set (all platforms)
//   - ~/.cache/peerscout on macOS (override Library path)
//   - os.UserCacheDir()/peerscout elsewhere (honours XDG on Linux)
func CacheDir() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, appName), nil
	}

	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".cache", appName), nil
	}

	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}
