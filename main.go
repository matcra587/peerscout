package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	cobracli "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/complete"
	"github.com/gechr/clib/help"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clib/theme"
	"github.com/gechr/clog"
	clogfx "github.com/gechr/clog/fx"
	"github.com/matcra587/peerscout/internal/agent"
	"github.com/matcra587/peerscout/internal/config"
	"github.com/matcra587/peerscout/internal/output"
	"github.com/matcra587/peerscout/internal/polkachu"
	"github.com/matcra587/peerscout/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var configPath string

// contextKey is a package-local type for context value keys to avoid collisions.
type contextKey string

const agentKey contextKey = "agent"

// AgentFromContext retrieves the agent DetectionResult from the command context.
func AgentFromContext(cmd *cobra.Command) agent.DetectionResult {
	v, _ := cmd.Context().Value(agentKey).(agent.DetectionResult)
	return v
}

func main() {
	root := newRootCmd()

	// Handle completion flags before cobra parses, so completion
	// works even when a required subcommand is missing.
	if flags, positional, ok := cobracli.Preflight(); ok {
		gen := newCompletionGenerator(root)
		handled, err := flags.Handle(gen, completionHandler(), complete.WithArgs(positional))
		if err != nil {
			clog.Error().Err(err).Msg("completion")
			os.Exit(2)
		}
		if handled {
			os.Exit(0)
		}
	}

	if err := root.Execute(); err != nil {
		clog.Error().Err(err).Msg("fatal")
		os.Exit(exitCode(err))
	}
}

func newCompletionGenerator(root *cobra.Command) *complete.Generator {
	gen := complete.NewGenerator("peerscout").FromFlags(cobracli.FlagMeta(root))
	gen.Subs = cobracli.Subcommands(root)
	return gen
}

func newRootCmd() *cobra.Command {
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
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			agentFlag, _ := cmd.Flags().GetBool("agent")
			det := agent.DetectWithFlag(agentFlag)
			ctx := context.WithValue(cmd.Context(), agentKey, det)
			cmd.SetContext(ctx)

			setupLogging(cmd, det)
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
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
		&cobra.Group{ID: "agent", Title: "Agent"},
	)

	root.AddCommand(findCmd())
	root.AddCommand(listCmd())
	root.AddCommand(configCmd())
	root.AddCommand(versionCmd())
	root.AddCommand(agentCmd())

	// Themed help rendering.
	th := theme.New(
		theme.WithEnumStyle(theme.EnumStyleHighlightBoth),
		theme.WithHelpRepeatEllipsisEnabled(true),
	)
	renderer := help.NewRenderer(th)
	root.SetHelpFunc(cobracli.HelpFunc(renderer, cobracli.SectionsWithOptions(cobracli.WithSubcommandOptional())))

	// Shell completion subcommand (for Homebrew: peerscout completion <shell>).
	root.AddCommand(cobracli.CompletionCommand(root, func() *complete.Generator {
		return newCompletionGenerator(root)
	}))

	return root
}

func setupLogging(cmd *cobra.Command, det agent.DetectionResult) {
	clog.SetEnvPrefix("PEERSCOUT")

	quiet, _ := cmd.Flags().GetBool("quiet")
	noColor, _ := cmd.Flags().GetBool("no-color")
	tty := terminal.Is(os.Stdout)

	// Agent mode, --quiet, or non-TTY suppress all non-data output.
	if det.Active || quiet || !tty {
		clog.SetLevel(clog.LevelFatal)
	} else {
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
	}

	// Disable colour for agent mode, --no-color, or non-TTY.
	if det.Active || noColor || !tty {
		clog.SetColorMode(clog.ColorNever)
	}
}

// isQuiet returns true if spinners and logs should be suppressed.
func isQuiet(cmd *cobra.Command) bool {
	det := AgentFromContext(cmd)
	quiet, _ := cmd.Flags().GetBool("quiet")
	return det.Active || quiet || !terminal.Is(os.Stdout)
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
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeNetworks,
		Annotations:       map[string]string{"clib": "dynamic-args='network'"},
		RunE:              runFind,
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
	det := AgentFromContext(cmd)
	w := cmd.OutOrStdout()
	quiet := isQuiet(cmd)

	var chains []string
	fetchChains := func(ctx context.Context) error {
		var err error
		chains, err = client.ListChains(ctx)
		return err
	}

	if quiet {
		if err := fetchChains(ctx); err != nil {
			if det.Active {
				return agentError(w, "find", err)
			}
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
		err := fmt.Errorf("unknown network %q - run 'peerscout list' to see all supported networks", network)
		if det.Active {
			return agentError(w, "find", err)
		}
		return err
	}

	seedNode, _ := cmd.Flags().GetBool("seed-node")
	stateSync, _ := cmd.Flags().GetBool("state-sync")
	addrbook, _ := cmd.Flags().GetBool("addrbook")

	format, _ := cmd.Flags().GetString("format")
	isTTY := terminal.Is(os.Stdout)

	if seedNode || stateSync || addrbook {
		detail, err := client.ChainDetail(ctx, network)
		if err != nil {
			if det.Active {
				return agentError(w, "find", err)
			}
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

		data := map[string]any{"network": network, key: value}
		switch output.DetectFormat(output.FormatOpts{AgentMode: det.Active, Format: format}) {
		case output.FormatAgentJSON:
			return output.RenderAgentJSON(w, "find", data, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, data, isTTY)
		default:
			fmt.Fprintln(w, value)
		}
		return nil
	}

	count := cfg.Count
	if cmd.Flags().Changed("count") {
		count, _ = cmd.Flags().GetInt("count")
	}
	if count < 1 {
		return fmt.Errorf("count must be a positive integer, got %d", count)
	}

	var result *polkachu.AccumulateResult
	var duplicates int
	if quiet {
		var err error
		result, err = client.AccumulatePeers(ctx, network, count, nil)
		if err != nil {
			return fmt.Errorf("fetching peers: %w", err)
		}
		duplicates = result.Duplicates
	} else {
		if err := clog.Shimmer("discovering peers").
			Elapsed("duration").
			Int("target", count).
			Progress(ctx, func(ctx context.Context, u *clogfx.Update) error {
				var err error
				result, err = client.AccumulatePeers(ctx, network, count, func(current int) {
					u.Int("found", current).Send()
				})
				if result != nil {
					duplicates = result.Duplicates
				}
				return err
			}).
			Int("duplicates", duplicates).
			Send(); err != nil {
			return fmt.Errorf("fetching peers: %w", err)
		}
	}

	allPeers := result.Peers
	if count > 0 && count < len(allPeers) {
		allPeers = allPeers[:count]
	}

	type peerResult struct {
		Network string   `json:"network"`
		Peers   []string `json:"peers"`
		Count   int      `json:"count"`
	}

	data := peerResult{
		Network: network,
		Peers:   allPeers,
		Count:   len(allPeers),
	}

	switch output.DetectFormat(output.FormatOpts{AgentMode: det.Active, Format: format}) {
	case output.FormatAgentJSON:
		return output.RenderAgentJSON(w, "find", data, nil)
	case output.FormatJSON:
		return output.RenderJSON(w, data, isTTY)
	case output.FormatCSV:
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
	det := AgentFromContext(cmd)
	w := cmd.OutOrStdout()

	var chains []string
	fetchChains := func(ctx context.Context) error {
		var err error
		chains, err = client.ListChains(ctx)
		return err
	}
	if isQuiet(cmd) {
		if err := fetchChains(ctx); err != nil {
			if det.Active {
				return agentError(w, "list", err)
			}
			return fmt.Errorf("unable to reach Polkachu API: %w", err)
		}
	} else {
		if err := clog.Shimmer("fetching networks").
			Elapsed("duration").
			Wait(ctx, fetchChains).Send(); err != nil {
			return fmt.Errorf("unable to reach Polkachu API: %w", err)
		}
	}

	format, _ := cmd.Flags().GetString("format")
	isTTY := terminal.Is(os.Stdout)

	switch output.DetectFormat(output.FormatOpts{AgentMode: det.Active, Format: format}) {
	case output.FormatAgentJSON:
		return output.RenderAgentJSON(w, "list", chains, nil)
	case output.FormatJSON:
		return output.RenderJSON(w, chains, isTTY)
	case output.FormatCSV:
		fmt.Fprintln(w, strings.Join(chains, ","))
	default:
		var th *theme.Theme
		noColor, _ := cmd.Flags().GetBool("no-color")
		if !det.Active && !noColor && isTTY {
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
		RunE:  runVersion,
	}

	cmd.Flags().Bool("short", false, "Print version number only")
	cobracli.Extend(cmd.Flags().Lookup("short"), cobracli.FlagExtra{
		Terse: "version number only",
	})

	return cmd
}

func runVersion(cmd *cobra.Command, _ []string) error {
	det := AgentFromContext(cmd)
	w := cmd.OutOrStdout()

	if det.Active {
		data := map[string]string{
			"version":  version.Version,
			"commit":   version.Commit,
			"branch":   version.Branch,
			"built":    version.BuildTime,
			"built_by": version.BuildBy,
		}
		return output.RenderAgentJSON(w, "version", data, nil)
	}

	short, _ := cmd.Flags().GetBool("short")
	if short {
		fmt.Fprintln(w, version.Version)
		return nil
	}

	noColor, _ := cmd.Flags().GetBool("no-color")
	var th *theme.Theme
	if !noColor && terminal.Is(os.Stdout) {
		th = theme.Default()
	}

	if th != nil {
		fmt.Fprintf(w, "%s %s\n", th.Bold.Render("peerscout"), th.Green.Render(version.Version))
		fmt.Fprintf(w, "  %s  %s\n", th.Dim.Render("commit:"), version.Commit)
		fmt.Fprintf(w, "  %s  %s\n", th.Dim.Render("branch:"), version.Branch)
		fmt.Fprintf(w, "  %s   %s\n", th.Dim.Render("built:"), version.BuildTime)
		fmt.Fprintf(w, "  %s %s\n", th.Dim.Render("built by:"), version.BuildBy)
	} else {
		fmt.Fprintf(w, "peerscout %s\n", version.Version)
		fmt.Fprintf(w, "  commit:  %s\n", version.Commit)
		fmt.Fprintf(w, "  branch:  %s\n", version.Branch)
		fmt.Fprintf(w, "  built:   %s\n", version.BuildTime)
		fmt.Fprintf(w, "  built by: %s\n", version.BuildBy)
	}
	return nil
}

func completionHandler() complete.Handler {
	return func(shell, kind string, _ []string) {
		client := polkachu.NewClient()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if kind == "network" {
			chains, err := client.ListChains(ctx)
			if err != nil {
				return
			}
			for _, c := range chains {
				fmt.Println(c)
			}
		}
	}
}

func completeNetworks(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client := polkachu.NewClient()
	chains, err := client.ListChains(cmd.Context())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return chains, cobra.ShellCompDirectiveNoFileComp
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

// --- Agent subcommand ---

func agentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent",
		Short:   "Agent discovery subcommands",
		GroupID: "agent",
	}
	cmd.AddCommand(agentSchemaCmd())
	cmd.AddCommand(agentGuideCmd())
	return cmd
}

type commandSchema struct {
	Use      string          `json:"use"`
	Short    string          `json:"short"`
	Long     string          `json:"long,omitempty"`
	Flags    []flagSchema    `json:"flags,omitempty"`
	Commands []commandSchema `json:"commands,omitempty"`
}

type flagSchema struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Type      string `json:"type"`
	Default   string `json:"default,omitempty"`
	Usage     string `json:"usage,omitempty"`
}

func agentSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Print command schema as JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			compact, _ := cmd.Flags().GetBool("compact")
			schema := buildSchema(cmd.Root(), compact)
			det := AgentFromContext(cmd)
			w := cmd.OutOrStdout()
			if det.Active {
				return output.RenderAgentJSON(w, "agent schema", schema, nil)
			}
			return output.RenderJSON(w, schema, terminal.Is(os.Stdout))
		},
	}
	cmd.Flags().Bool("compact", false, "Strip descriptions for smaller output")
	cobracli.Extend(cmd.Flags().Lookup("compact"), cobracli.FlagExtra{
		Terse: "strip descriptions",
	})
	return cmd
}

func buildSchema(cmd *cobra.Command, compact bool) commandSchema {
	s := commandSchema{Use: cmd.Use}
	if !compact {
		s.Short = cmd.Short
		s.Long = cmd.Long
	}

	cmd.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		fs := flagSchema{
			Name: f.Name,
			Type: f.Value.Type(),
		}
		if f.Shorthand != "" {
			fs.Shorthand = f.Shorthand
		}
		if f.DefValue != "" && f.DefValue != "false" {
			fs.Default = f.DefValue
		}
		if !compact {
			fs.Usage = f.Usage
		}
		s.Flags = append(s.Flags, fs)
	})

	for _, child := range cmd.Commands() {
		if !child.IsAvailableCommand() || child.Name() == "help" {
			continue
		}
		s.Commands = append(s.Commands, buildSchema(child, compact))
	}
	return s
}

func agentGuideCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "guide <name>",
		Short:     "Print an agent guide (" + strings.Join(agent.GuideNames, ", ") + ")",
		Args:      cobra.ExactArgs(1),
		ValidArgs: agent.GuideNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := agent.Guide(args[0])
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), content)
			return nil
		},
	}
}

// agentError writes a structured error envelope to w and returns the
// original error so the process exits with a non-zero code.
func agentError(w io.Writer, command string, err error) error {
	code := 1
	if _, ok := errors.AsType[*polkachu.NotFoundError](err); ok {
		code = 404
	}
	env := agent.Error(command, code, err.Error(), "")
	data, _ := json.Marshal(env)
	_, _ = w.Write(append(data, '\n'))
	return err
}
