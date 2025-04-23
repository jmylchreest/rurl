package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/jmylchreest/rurl/internal/launcher"
	"github.com/jmylchreest/rurl/internal/logging"
	"github.com/jmylchreest/rurl/internal/rules"
	"github.com/jmylchreest/rurl/internal/urlhandler"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	logLevelStr string
	cfg         *config.Config
	detectSave  bool
	rootCmd     *cobra.Command
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd = &cobra.Command{
		Use:   "rurl [URL]",
		Short: "rurl routes URLs to the appropriate browser based on rules.",
		Long: `rurl (Route URL) acts as a smart default browser handler.

When a URL is provided as an argument or passed via OS default browser mechanism,
it routes the URL to the correct browser profile based on rules.`,
		Args: cobra.MaximumNArgs(1), // Accepts zero or one argument (the URL)
		Run:  runRootCmd,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is %s)", DefaultConfigPath()))
	rootCmd.PersistentFlags().StringVarP(&logLevelStr, "log-level", "l", "error", "set log level (trace, debug, info, warn, error, fatal, panic)")

	// Add config command and its subcommands
	addConfigCommands()

	// Add completion command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:

  $ source <(rurl completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ rurl completion bash > /etc/bash_completion.d/rurl
  # macOS:
  $ rurl completion bash > /usr/local/etc/bash_completion.d/rurl

Zsh:

  # If shell completion is not already enabled in your environment, you will need to enable it. You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ rurl completion zsh > "${fpath[1]}/_rurl"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ rurl completion fish | source

  # To load completions for each session, execute once:
  $ rurl completion fish > ~/.config/fish/completions/rurl.fish

PowerShell:

  PS> rurl completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> rurl completion powershell > rurl.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	})
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	var err error

	// Initialize logging first, using the flag value
	// Note: Log level might be limited until config is fully loaded if config loading itself logs
	logging.InitLogging(logLevelStr)

	cfg, err = config.LoadConfig(cfgFile)
	if err != nil {
		// Use Printf directly as logger might not be fully ready or might filter this out
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}
	log.Debug().Msg("Configuration loaded successfully")

	// Re-initialize logging in case config file specifies a different level?
	// For now, command-line flag takes precedence.
	// If config file should override, logic needs adjustment here.
	// logging.InitLogging(cfg.LogLevel, debug) // Example if config has level
}

// runRootCmd handles the main URL routing functionality
func runRootCmd(cmd *cobra.Command, args []string) {
	if cfg == nil {
		log.Fatal().Msg("Configuration not loaded (should not happen)")
	}

	if len(args) == 0 {
		cmd.Help()
		os.Exit(0)
	}

	urlInput := args[0]
	log.Info().Str("url", urlInput).Msg("Processing URL")

	// 1. Process URL (Resolve shorteners, check for safelinks)
	resolvedURL, originalURL, isSafelink, err := urlhandler.ProcessURL(cfg, urlInput)
	if err != nil {
		log.Error().Err(err).Str("input_url", urlInput).Msg("Failed to process URL")
		fmt.Fprintf(os.Stderr, "Error processing URL: %v\n", err)
		os.Exit(1)
	}

	// Determine which URL to actually launch
	urlToLaunch := resolvedURL
	if isSafelink {
		urlToLaunch = originalURL
		log.Info().Str("original_url", originalURL).Msg("Safelink detected, launching original URL after rule matching")
	}

	// Apply Rules based on the RESOLVED URL
	matchResult, err := rules.ApplyRules(cfg, resolvedURL)
	if err != nil {
		log.Error().Err(err).Str("url", resolvedURL).Msg("Failed to apply rules")
		fmt.Fprintf(os.Stderr, "Error applying rules: %v\n", err)
		os.Exit(1)
	}

	if matchResult.Rule != nil {
		log.Info().Str("rule_name", matchResult.Rule.Name).Str("profile_id", matchResult.ProfileID).Msg("Rule matched")
	} else {
		log.Info().Str("profile_id", matchResult.ProfileID).Msg("No specific rule matched, using default profile")
	}

	err = launcher.Launch(cfg, matchResult.ProfileID, urlToLaunch, matchResult.Incognito)
	if err != nil {
		log.Error().Err(err).Str("profile_id", matchResult.ProfileID).Str("url_launched", urlToLaunch).Msg("Failed to launch browser")
		fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
		os.Exit(1)
	}

	log.Info().Msg("Browser launched successfully")
}

// DefaultConfigPath helper for CLI flags.
func DefaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "$HOME/.config" // Fallback for display
		log.Warn().Err(err).Msg("Could not determine user config dir for help text")
	}
	return filepath.Join(dir, "rurl", "config.toml")
}
