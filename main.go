package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	cobracli "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/help"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clib/theme"
	"github.com/gechr/clog"
	clogfx "github.com/gechr/clog/fx"
	"github.com/matcra587/peerscout/internal/config"
	"github.com/matcra587/peerscout/internal/output"
	"github.com/matcra587/peerscout/internal/polkachu"
	"github.com/matcra587/peerscout/internal/version"
	"github.com/spf13/cobra"
)

var configPath string

func main() {
	root := &cobra.Command{
		Use:   "peerscout",
		Short: "Discover blockchain peers from the Polkachu API",
		Long:  "Fetch live peers for Cosmos SDK chains from the Polkachu API.",
		Example: `  # Fetch peers for cosmos
  $ peerscout find cosmos

  # Comma-separated for config files
  $ peerscout find cosmos -f csv

  # List all supported networks
  $ peerscout list`,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			setupLogging(cmd)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	pf := root.PersistentFlags()
	pf.StringVar(&configPath, "config", "", "Path to TOML config file")
	pf.StringP("format", "f", "plain", "Output format: plain, json, csv")
	pf.Bool("agent", false, "Force agent mode (JSON output, quiet mode)")
	pf.BoolP("quiet", "q", false, "Suppress non-data output (spinners, logs)")
	pf.Bool("no-color", false, "Disable coloured output")
	pf.String("log-format", "auto", "Log output format: auto, text, json")
	pf.Bool("debug", false, "Enable debug logging")

	cobracli.Extend(pf.Lookup("config"), cobracli.FlagExtra{
		Group:       "Global",
		Placeholder: "PATH",
		Hint:        "file",
		Terse:       "config file path",
	})
	cobracli.Extend(pf.Lookup("format"), cobracli.FlagExtra{
		Group:       "Output",
		Placeholder: "FORMAT",
		Enum:        []string{"plain", "json", "csv"},
		EnumTerse:   []string{"one per line", "JSON output", "comma-separated"},
		EnumDefault: "plain",
		Terse:       "output format",
	})
	cobracli.Extend(pf.Lookup("agent"), cobracli.FlagExtra{
		Group: "Output",
		Terse: "force agent mode",
	})
	cobracli.Extend(pf.Lookup("quiet"), cobracli.FlagExtra{
		Group: "Output",
		Terse: "suppress spinners and logs",
	})
	cobracli.Extend(pf.Lookup("no-color"), cobracli.FlagExtra{
		Group: "Global",
		Terse: "disable colour",
	})
	cobracli.Extend(pf.Lookup("log-format"), cobracli.FlagExtra{
		Group:       "Global",
		Placeholder: "FORMAT",
		Enum:        []string{"auto", "text", "json"},
		EnumTerse:   []string{"detect terminal", "human text", "structured JSON"},
		EnumDefault: "auto",
		Terse:       "log format",
	})
	cobracli.Extend(pf.Lookup("debug"), cobracli.FlagExtra{
		Group: "Global",
		Terse: "debug output",
	})

	// Command groups for themed help.
	root.AddGroup(
		&cobra.Group{ID: "peers", Title: "Peer Discovery"},
		&cobra.Group{ID: "config", Title: "Configuration"},
	)

	root.AddCommand(findCmd())
	root.AddCommand(listCmd())
	root.AddCommand(configCmd())
	root.AddCommand(versionCmd())

	// Themed help rendering.
	th := theme.New(
		theme.WithEnumStyle(theme.EnumStyleHighlightBoth),
		theme.WithHelpRepeatEllipsisEnabled(true),
	)
	renderer := help.NewRenderer(th)
	root.SetHelpFunc(cobracli.HelpFunc(renderer, cobracli.SectionsWithOptions(cobracli.WithSubcommandOptional())))

	// Shell completions.
	_ = cobracli.NewCompletion(root)

	if err := root.Execute(); err != nil {
		clog.Error().Err(err).Msg("fatal")
		os.Exit(exitCode(err))
	}
}

func setupLogging(cmd *cobra.Command) {
	clog.SetEnvPrefix("PEERSCOUT")

	// --agent and --quiet suppress all non-data output.
	agent, _ := cmd.Flags().GetBool("agent")
	quiet, _ := cmd.Flags().GetBool("quiet")
	if agent || quiet {
		clog.SetLevel(clog.LevelFatal)
		return
	}

	debug, _ := cmd.Flags().GetBool("debug")
	if debug {
		clog.SetVerbose(true)
	}

	logFormat, _ := cmd.Flags().GetString("log-format")
	switch logFormat {
	case "json":
		clog.SetHandler(clog.HandlerFunc(func(e clog.Entry) {
			data, _ := json.Marshal(e)
			fmt.Fprintln(os.Stderr, string(data))
		}))
	case "auto":
		if !clog.Default.Output().IsTTY() {
			clog.SetHandler(clog.HandlerFunc(func(e clog.Entry) {
				data, _ := json.Marshal(e)
				fmt.Fprintln(os.Stderr, string(data))
			}))
		}
	}

	noColor, _ := cmd.Flags().GetBool("no-color")
	if noColor {
		clog.SetColorMode(clog.ColorNever)
	}
}

// isQuiet returns true if spinners and logs should be suppressed.
func isQuiet(cmd *cobra.Command) bool {
	agent, _ := cmd.Flags().GetBool("agent")
	quiet, _ := cmd.Flags().GetBool("quiet")
	return agent || quiet
}

// outputFormat returns the resolved format. --agent forces json.
func outputFormat(cmd *cobra.Command) string {
	agent, _ := cmd.Flags().GetBool("agent")
	if agent {
		return "json"
	}
	format, _ := cmd.Flags().GetString("format")
	return format
}

func findCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "find <network>",
		Short:   "Fetch peers for a network",
		GroupID: "peers",
		Example: `  # Fetch peers for cosmos
  $ peerscout find cosmos

  # Return 10 peers
  $ peerscout find cosmos -n 10

  # Get the seed node instead
  $ peerscout find cosmos --seed-node

  # Comma-separated for config files
  $ peerscout find cosmos -f csv

  # Output as JSON
  $ peerscout find cosmos -f json`,
		Args: cobra.ExactArgs(1),
		RunE: runFind,
	}

	cmd.Flags().IntP("count", "n", 5, "Number of peers to return")
	cmd.Flags().Bool("seed-node", false, "Return the seed node instead of live peers")
	cmd.Flags().Bool("state-sync", false, "Return the state-sync RPC endpoint")
	cmd.Flags().Bool("addrbook", false, "Return the addrbook download URL")
	cobracli.Extend(cmd.Flags().Lookup("count"), cobracli.FlagExtra{
		Placeholder: "N",
		Terse:       "peer count",
	})
	cobracli.Extend(cmd.Flags().Lookup("seed-node"), cobracli.FlagExtra{
		Terse: "seed node instead of peers",
	})
	cobracli.Extend(cmd.Flags().Lookup("state-sync"), cobracli.FlagExtra{
		Terse: "state-sync RPC endpoint",
	})
	cobracli.Extend(cmd.Flags().Lookup("addrbook"), cobracli.FlagExtra{
		Terse: "addrbook download URL",
	})
	cmd.MarkFlagsMutuallyExclusive("seed-node", "state-sync", "addrbook")

	return cmd
}

func runFind(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configPath, cmd.Flags())
	if err != nil {
		return err
	}

	network := args[0]

	ctx := cmd.Context()
	client := polkachu.NewClient()

	quiet := isQuiet(cmd)

	var chains []string
	fetchChains := func(_ context.Context) error {
		var err error
		chains, err = client.ListChains(ctx)
		return err
	}
	if quiet {
		if err := fetchChains(ctx); err != nil {
			return fmt.Errorf("unable to reach Polkachu API: %w", err)
		}
	} else {
		if err := clog.Shimmer("fetching networks").
			Elapsed("duration").
			Wait(ctx, fetchChains).Send(); err != nil {
			return fmt.Errorf("unable to reach Polkachu API: %w", err)
		}
	}

	if !slices.Contains(chains, network) {
		return fmt.Errorf("unknown network %q - run 'peerscout list' to see all supported networks", network)
	}

	seedNode, _ := cmd.Flags().GetBool("seed-node")
	stateSync, _ := cmd.Flags().GetBool("state-sync")
	addrbook, _ := cmd.Flags().GetBool("addrbook")

	if seedNode || stateSync || addrbook {
		detail, err := client.ChainDetail(ctx, network)
		if err != nil {
			return fmt.Errorf("fetching chain detail: %w", err)
		}

		var key, value, label string
		switch {
		case seedNode:
			label = "seed node"
			if detail.Services.Seed.Active {
				key, value = "seed", detail.Services.Seed.Seed
			}
		case stateSync:
			label = "state-sync"
			if detail.Services.StateSync.Active {
				key, value = "state_sync", detail.Services.StateSync.Node
			}
		case addrbook:
			label = "addrbook"
			if detail.Services.Addrbook.Active {
				key, value = "addrbook", detail.Services.Addrbook.DownloadURL
			}
		}

		if value == "" {
			clog.Warn().Str("network", network).Msgf("%s not available", label)
			return nil
		}

		w := cmd.OutOrStdout()
		switch outputFormat(cmd) {
		case "json":
			return output.RenderJSON(w, map[string]any{
				"network": network,
				key:       value,
			})
		default:
			fmt.Fprintln(w, value)
		}
		return nil
	}

	count := cfg.Count
	if cmd.Flags().Changed("count") {
		count, _ = cmd.Flags().GetInt("count")
	}

	var result *polkachu.AccumulateResult
	if quiet {
		var err error
		result, err = client.AccumulatePeers(ctx, network, count, nil)
		if err != nil {
			return fmt.Errorf("fetching peers: %w", err)
		}
	} else {
		if err := clog.Shimmer("discovering peers").
			Elapsed("duration").
			Int("target", count).
			Progress(ctx, func(_ context.Context, u *clogfx.Update) error {
				var err error
				result, err = client.AccumulatePeers(ctx, network, count, func(current int) {
					u.Int("found", current).Send()
				})
				return err
			}).
			Int("duplicates", result.Duplicates).
			Send(); err != nil {
			return fmt.Errorf("fetching peers: %w", err)
		}
	}

	allPeers := result.Peers
	if count > 0 && count < len(allPeers) {
		allPeers = allPeers[:count]
	}

	w := cmd.OutOrStdout()

	switch outputFormat(cmd) {
	case "json":
		return output.RenderJSON(w, map[string]any{
			"network": network,
			"peers":   allPeers,
			"count":   len(allPeers),
		})
	case "csv":
		fmt.Fprintln(w, strings.Join(allPeers, ","))
	default:
		for _, p := range allPeers {
			fmt.Fprintln(w, p)
		}
	}
	return nil
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all supported networks",
		GroupID: "peers",
		Example: `  # List all networks
  $ peerscout list

  # List as JSON
  $ peerscout list -f json`,
		Args: cobra.NoArgs,
		RunE: runList,
	}
}

func runList(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	client := polkachu.NewClient()

	var chains []string
	fetchChains := func(_ context.Context) error {
		var err error
		chains, err = client.ListChains(ctx)
		return err
	}
	if isQuiet(cmd) {
		if err := fetchChains(ctx); err != nil {
			return fmt.Errorf("unable to reach Polkachu API: %w", err)
		}
	} else {
		if err := clog.Shimmer("fetching networks").
			Elapsed("duration").
			Wait(ctx, fetchChains).Send(); err != nil {
			return fmt.Errorf("unable to reach Polkachu API: %w", err)
		}
	}

	w := cmd.OutOrStdout()

	switch outputFormat(cmd) {
	case "json":
		return output.RenderJSON(w, chains)
	case "csv":
		fmt.Fprintln(w, strings.Join(chains, ","))
	default:
		isTTY := terminal.Is(os.Stdout)
		var th *theme.Theme
		if isTTY {
			th = theme.Default()
		}
		return output.RenderColumns(w, chains, terminal.Width(os.Stdout), th)
	}
	return nil
}

func versionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		Run:   runVersion,
	}

	cmd.Flags().Bool("short", false, "Print version number only")
	cobracli.Extend(cmd.Flags().Lookup("short"), cobracli.FlagExtra{
		Terse: "version number only",
	})

	return cmd
}

func runVersion(cmd *cobra.Command, _ []string) {
	w := cmd.OutOrStdout()
	short, _ := cmd.Flags().GetBool("short")
	if short {
		fmt.Fprintln(w, version.Version)
		return
	}

	fmt.Fprintf(w, "peerscout %s\n", version.Version)
	fmt.Fprintf(w, "  commit:  %s\n", version.Commit)
	fmt.Fprintf(w, "  branch:  %s\n", version.Branch)
	fmt.Fprintf(w, "  built:   %s\n", version.BuildTime)
	fmt.Fprintf(w, "  built by: %s\n", version.BuildBy)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if _, ok := errors.AsType[*polkachu.NotFoundError](err); ok {
		return 1
	}
	return 2
}
