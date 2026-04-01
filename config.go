package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/BurntSushi/toml"
	"github.com/gechr/clib/theme"
	"github.com/gechr/clog"
	"github.com/matcra587/peerscout/internal/config"
	"github.com/matcra587/peerscout/internal/dirs"
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Manage peerscout configuration",
		GroupID: "config",
	}

	cmd.AddCommand(configListCmd())
	cmd.AddCommand(configGetCmd())
	cmd.AddCommand(configSetCmd())
	cmd.AddCommand(configUnsetCmd())
	cmd.AddCommand(configPathCmd())

	return cmd
}

// ------------------------------------------------------------------
// config list (aliases: ls)
// ------------------------------------------------------------------

func configListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Show current settings",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(configPath, nil)
			if err != nil {
				return err
			}

			resolved := configToMap(cfg)
			fileSources := configSourcesFromFile()

			th := theme.Default()

			rows := make([][]string, 0, len(configKeys))
			for _, key := range configKeys {
				val := resolved[key]
				source := "default"
				if _, ok := fileSources[key]; ok {
					source = configFilePath()
				}
				if envVal := envSource(key); envVal != "" {
					source = "PEERSCOUT_" + strings.ToUpper(key)
				}
				desc := configDescriptions[key]
				rows = append(rows, []string{key, val, source, desc})
			}

			headerStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1)
			keyStyle := th.Blue.Padding(0, 1)
			valueStyle := lipgloss.NewStyle().Padding(0, 1)
			dimStyle := th.Dim.Padding(0, 1)

			t := table.New().
				Border(lipgloss.HiddenBorder()).
				Headers("Key", "Value", "Source", "Description").
				StyleFunc(func(row, col int) lipgloss.Style {
					if row == table.HeaderRow {
						return headerStyle
					}
					switch col {
					case 0:
						return keyStyle
					case 2, 3:
						return dimStyle
					default:
						return valueStyle
					}
				}).
				Rows(rows...)

			fmt.Fprintln(cmd.OutOrStdout(), t.Render())
			return nil
		},
	}
}

// ------------------------------------------------------------------
// config get
// ------------------------------------------------------------------

func configGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "get <key>",
		Short:             "Show a current setting",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeConfigKeys,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath, nil)
			if err != nil {
				return err
			}

			m := configToMap(cfg)
			val, ok := m[args[0]]
			if !ok {
				return fmt.Errorf("unknown config key %s - run 'peerscout config list' to see all keys", args[0])
			}

			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}
}

// ------------------------------------------------------------------
// config set (aliases: create)
// ------------------------------------------------------------------

func configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "set <key> <value>",
		Aliases:           []string{"create"},
		Short:             "Add/update a setting",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: completeConfigKeys,
		RunE: func(_ *cobra.Command, args []string) error {
			key, val := args[0], args[1]
			if !slices.Contains(configKeys, key) {
				return fmt.Errorf("unknown config key %s - valid keys: %s", key, strings.Join(configKeys, ", "))
			}
			typed, err := parseConfigValue(key, val)
			if err != nil {
				return err
			}
			cfgPath := resolveConfigPath()
			return modifyConfigFile(cfgPath, func(doc map[string]any) {
				doc[key] = typed
			})
		},
	}
}

// ------------------------------------------------------------------
// config unset (aliases: rm, remove, delete, del)
// ------------------------------------------------------------------

func configUnsetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "unset <key>",
		Aliases:           []string{"rm", "remove", "delete", "del"},
		Short:             "Clear a setting",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeConfigKeys,
		RunE: func(_ *cobra.Command, args []string) error {
			if !slices.Contains(configKeys, args[0]) {
				return fmt.Errorf("unknown config key %s - valid keys: %s", args[0], strings.Join(configKeys, ", "))
			}
			cfgPath := resolveConfigPath()
			var remaining int
			err := modifyConfigFile(cfgPath, func(doc map[string]any) {
				delete(doc, args[0])
				remaining = len(doc)
			})
			if err != nil {
				return err
			}
			if remaining == 0 {
				if err := os.Remove(cfgPath); err == nil {
					clog.Info().Path("path", cfgPath).Msg("config file removed (empty)")
				}
			}
			return nil
		},
	}
}

// ------------------------------------------------------------------
// config path
// ------------------------------------------------------------------

func configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := resolveConfigPath()
			w := cmd.OutOrStdout()
			if _, err := os.Stat(path); err != nil {
				fmt.Fprintf(w, "%s (not found)\n", path)
			} else {
				fmt.Fprintln(w, path)
			}
			return nil
		},
	}
}

// ------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------

var configKeys = []string{
	"count",
}

var configDescriptions = map[string]string{
	"count": "Number of peers to return",
}

func parseConfigValue(key, val string) (any, error) {
	switch key {
	case "count":
		n, err := strconv.Atoi(val)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("count must be a positive integer, got %s", val)
		}
		return n, nil
	default:
		return val, nil
	}
}

func completeConfigKeys(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return configKeys, cobra.ShellCompDirectiveNoFileComp
}

func configToMap(cfg config.Config) map[string]string {
	return map[string]string{
		"count": fmt.Sprintf("%d", cfg.Count),
	}
}

func configSourcesFromFile() map[string]struct{} {
	path := resolveConfigPath()
	data, err := os.ReadFile(path) //nolint:gosec // path from --config or XDG default
	if err != nil {
		return nil
	}

	var raw map[string]any
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil
	}

	result := make(map[string]struct{}, len(raw))
	for k := range raw {
		result[k] = struct{}{}
	}
	return result
}

func envSource(key string) string {
	return os.Getenv("PEERSCOUT_" + strings.ToUpper(key))
}

func configFilePath() string {
	path := resolveConfigPath()
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func resolveConfigPath() string {
	if configPath != "" {
		return configPath
	}
	p, err := dirs.DefaultConfigPath()
	if err != nil {
		return "config.toml"
	}
	return p
}

func modifyConfigFile(cfgPath string, modify func(doc map[string]any)) error {
	var raw map[string]any

	data, err := os.ReadFile(cfgPath) //nolint:gosec // path is from --config flag or XDG default
	if err == nil {
		if _, err := toml.Decode(string(data), &raw); err != nil {
			return fmt.Errorf("parsing config file: %w", err)
		}
	}
	if raw == nil {
		raw = make(map[string]any)
	}

	modify(raw)

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	return encodeTOMLToFile(cfgPath, raw)
}

func encodeTOMLToFile(path string, doc map[string]any) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) //nolint:gosec // path from --config or XDG default
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := toml.NewEncoder(f).Encode(doc); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	clog.Info().Str("path", path).Msg("config updated")
	return nil
}
