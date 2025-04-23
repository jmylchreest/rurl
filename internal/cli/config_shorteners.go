package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/jmylchreest/rurl/internal/config"
	// "github.com/jmylchreest/rurl/internal/logging"
	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/choose"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// Use the global logger from zerolog/log, assuming InitLogging was called.
// var cliLogger = logging.GetLogger() // Removed, use log.Logger directly

// addShortURLCommands adds the commands for managing short URL domains.
func addShortURLCommands(parentCmd *cobra.Command) {
	shorturlCmd := &cobra.Command{
		Use:     "shorturl",
		Aliases: []string{"short", "su"},
		Short:   "Manage short URL domain configurations",
		Long:    `Add, edit, delete, and list manually added or view built-in URL shortener domains recognized by rurl.`,
	}

	// --- List Short URLs Command ---
	listShortURLsCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured short URL domains",
		Long:    `Displays manually added short URL domains. Use the --builtin flag to also show built-in domains.`,
		Run:     runListShortURLsCmd,
	}
	listShortURLsCmd.Flags().BoolP("builtin", "b", false, "Include built-in shortener domains in the list")
	shorturlCmd.AddCommand(listShortURLsCmd)

	// --- Add Manual Short URL Command ---
	addShortURLCmd := &cobra.Command{
		Use:   "add [domain]",
		Short: "Add a new domain to the manual short URL list",
		Long: `Adds a new domain to the list of known shortener services.
URLs from this domain will be resolved before rule matching.
You can optionally set the --safelink flag.`,
		Args: cobra.ExactArgs(1),
		Run:  runAddManualShortURLCmd,
	}
	addShortURLCmd.Flags().BoolP("safelink", "s", false, "Mark this domain as a safelink (launch original URL after matching)")
	shorturlCmd.AddCommand(addShortURLCmd)

	// --- Edit Manual Short URL Command ---
	editShortURLCmd := &cobra.Command{
		Use:               "edit [domain]",
		Short:             "Edit settings for a manually added short URL domain",
		Long:              `Edits settings (currently only the IsSafelink flag) for a manually added short URL domain. Prompts for domain if not provided.`,
		Args:              cobra.MaximumNArgs(1),
		Run:               runEditManualShortURLCmd,
		ValidArgsFunction: completeManualShortURLDomains,
	}
	editShortURLCmd.Flags().BoolP("safelink", "s", false, "Mark this domain as a safelink (launch original URL after matching)")
	shorturlCmd.AddCommand(editShortURLCmd)

	// --- Delete Manual Short URL Command ---
	deleteShortURLCmd := &cobra.Command{
		Use:               "delete [domain]",
		Aliases:           []string{"rm", "del"},
		Short:             "Delete a domain from the manual short URL list",
		Long:              `Deletes the configuration for the specified manually added short URL domain. Prompts for domain if not provided. Built-in domains cannot be deleted.`,
		Args:              cobra.MaximumNArgs(1),
		Run:               runDeleteManualShortURLCmd,
		ValidArgsFunction: completeManualShortURLDomains,
	}
	shorturlCmd.AddCommand(deleteShortURLCmd)

	// Add the main 'shorturl' command to the parent ('config')
	parentCmd.AddCommand(shorturlCmd)
}

// --- Command Implementations ---

func runListShortURLsCmd(cmd *cobra.Command, args []string) {
	if cfg == nil {
		log.Logger.Error().Msg("Configuration not loaded.")
		os.Exit(1)
	}
	showBuiltin, _ := cmd.Flags().GetBool("builtin")
	printShortURLList(cfg, showBuiltin)
}

func runAddManualShortURLCmd(cmd *cobra.Command, args []string) {
	if cfg == nil {
		log.Logger.Error().Msg("Configuration not loaded.")
		os.Exit(1)
	}
	domain := args[0]

	// Validate domain uniqueness across *both* built-in and manual lists
	for _, s := range cfg.Shorteners {
		if s.Domain == domain {
			fmt.Fprintf(os.Stderr, "Error: Domain '%s' is already present in the built-in shortener list.\n", domain)
			os.Exit(1)
		}
	}
	for _, s := range cfg.ManualShorteners {
		if s.Domain == domain {
			fmt.Fprintf(os.Stderr, "Error: Domain '%s' has already been manually added.\n", domain)
			os.Exit(1)
		}
	}

	// Basic domain format check (could be more robust)
	if !strings.Contains(domain, ".") || strings.ContainsAny(domain, "/:?#@") {
		fmt.Fprintf(os.Stderr, "Error: Invalid domain format '%s'. Please provide just the domain name (e.g., my.shortener.com).\n", domain)
		os.Exit(1)
	}

	// Check if safelink flag was provided
	isSafelink := false
	if cmd.Flags().Changed("safelink") {
		isSafelink, _ = cmd.Flags().GetBool("safelink")
	} else {
		// If flag not provided, prompt for it
		isSafelink = promptSafelink(fmt.Sprintf("Should '%s' be treated as a safelink?", domain), false)
	}

	newShortener := config.ShortenerService{
		Domain:     domain,
		IsSafelink: isSafelink,
	}
	cfg.ManualShorteners = append(cfg.ManualShorteners, newShortener)

	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		log.Logger.Error().Err(err).Str("domain", domain).Msg("Failed to save config after adding manual short URL domain")
		fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	log.Logger.Info().Str("domain", domain).Bool("is_safelink", isSafelink).Msg("Manual short URL domain added successfully.")
	fmt.Printf("Manual short URL domain '%s' added successfully (IsSafelink: %t).\n", domain, isSafelink)
}

func runEditManualShortURLCmd(cmd *cobra.Command, args []string) {
	if cfg == nil {
		log.Logger.Error().Msg("Configuration not loaded.")
		os.Exit(1)
	}

	var domainName string
	var err error

	if len(args) == 0 {
		if len(cfg.ManualShorteners) == 0 {
			fmt.Fprintln(os.Stderr, "Error: No manual short URLs configured to edit.")
			os.Exit(1)
		}
		domainName, err = promptSelectManualShortURL("Select the manual short URL domain to edit:", cfg.ManualShorteners)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error selecting short URL domain: %v\n", err)
			os.Exit(1)
		}
		if domainName == "" { // User cancelled
			fmt.Println("Edit cancelled.")
			os.Exit(0)
		}
	} else {
		domainName = args[0]
	}

	shortenerToEdit, index, err := cfg.FindManualShortenerByDomain(domainName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check if safelink flag was provided
	var newValue bool
	if cmd.Flags().Changed("safelink") {
		newValue, _ = cmd.Flags().GetBool("safelink")
	} else {
		// If flag not provided, prompt for it
		newValue = promptSafelink(fmt.Sprintf("Should '%s' be treated as a safelink?", domainName), shortenerToEdit.IsSafelink)
	}

	// Only update if the value is different
	if newValue == shortenerToEdit.IsSafelink {
		fmt.Println("Setting unchanged. No edit necessary.")
		os.Exit(0)
	}

	// Update the shortener in the slice
	cfg.ManualShorteners[index].IsSafelink = newValue

	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		log.Logger.Error().Err(err).Str("domain", domainName).Msg("Failed to save config after editing manual short URL domain")
		fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	log.Logger.Info().Str("domain", domainName).Bool("is_safelink", newValue).Msg("Manual short URL domain updated successfully.")
	fmt.Printf("Manual short URL domain '%s' updated successfully (IsSafelink: %t).\n", domainName, newValue)
}

func runDeleteManualShortURLCmd(cmd *cobra.Command, args []string) {
	if cfg == nil {
		log.Logger.Error().Msg("Configuration not loaded.")
		os.Exit(1)
	}

	var domainName string
	var err error

	if len(args) == 0 {
		if len(cfg.ManualShorteners) == 0 {
			fmt.Fprintln(os.Stderr, "Error: No manual short URLs configured to delete.")
			os.Exit(1)
		}
		domainName, err = promptSelectManualShortURL("Select the manual short URL domain to delete:", cfg.ManualShorteners)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error selecting short URL domain: %v\n", err)
			os.Exit(1)
		}
		if domainName == "" { // User cancelled
			fmt.Println("Delete cancelled.")
			os.Exit(0)
		}
	} else {
		domainName = args[0]
	}

	// --- Check if it's a built-in shortener first ---
	isBuiltIn := false
	for _, s := range cfg.Shorteners {
		if s.Domain == domainName {
			isBuiltIn = true
			break
		}
	}
	if isBuiltIn {
		fmt.Fprintf(os.Stderr, "Error: Domain '%s' is a built-in shortener and cannot be deleted.\n", domainName)
		os.Exit(1)
	}
	// --- End check ---

	_, index, err := cfg.FindManualShortenerByDomain(domainName)
	if err != nil {
		// This error now implies it's not a manual domain *either*
		fmt.Fprintf(os.Stderr, "Error: Manual short URL domain '%s' not found.\n", domainName)
		os.Exit(1)
	}

	confirm := promptString(fmt.Sprintf("Are you sure you want to delete the manual short URL domain '%s'? (yes/no)", domainName), "no")
	if !strings.EqualFold(confirm, "yes") {
		fmt.Println("Deletion cancelled.")
		os.Exit(0)
	}

	// Remove the shortener from the slice
	cfg.ManualShorteners = append(cfg.ManualShorteners[:index], cfg.ManualShorteners[index+1:]...)

	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		log.Logger.Error().Err(err).Str("domain", domainName).Msg("Failed to save config after deleting manual short URL domain")
		fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	log.Logger.Info().Str("domain", domainName).Msg("Manual short URL domain deleted successfully.")
	fmt.Printf("Manual short URL domain '%s' deleted successfully.\n", domainName)
}

// --- Helper Functions ---

// printShortURLList prints the list of configured shortener domains using tabwriter.
// If showBuiltin is true, it includes both manual and built-in domains.
func printShortURLList(cfg *config.Config, showBuiltin bool) {
	fmt.Println("--- Short URLs ---")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0) // minwidth, tabwidth, padding, padchar, flags

	// Print header
	fmt.Fprintln(w, "Domain\tIsSafelink\tType")
	fmt.Fprintln(w, "------\t----------\t----")

	manualCount := 0
	for _, s := range cfg.ManualShorteners {
		fmt.Fprintf(w, "%s\t%t\t%s\n", s.Domain, s.IsSafelink, "Manual")
		manualCount++
	}

	builtinCount := 0
	if showBuiltin {
		// Add a separator if both lists are shown and manual list wasn't empty
		if manualCount > 0 && len(cfg.Shorteners) > 0 {
			fmt.Fprintln(w, "------\t----------\t----") // Separator line
		}
		for _, s := range cfg.Shorteners {
			fmt.Fprintf(w, "%s\t%t\t%s\n", s.Domain, s.IsSafelink, "Built-in")
			builtinCount++
		}
		if builtinCount == 0 {
			// This case is unlikely with defaults, but handle it
			fmt.Fprintln(w, "(No built-in shorteners found)\t\t") // Add tabs for alignment
		}
	}

	w.Flush()
}

// completeManualShortURLDomains provides completion for manually added short URL domains.
func completeManualShortURLDomains(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if cfg == nil {
		log.Logger.Warn().Msg("Completion: Configuration not loaded.")
		return nil, cobra.ShellCompDirectiveError
	}
	if len(args) > 0 { // Don't complete if domain is already provided
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var domains []string
	for _, s := range cfg.ManualShorteners {
		if strings.HasPrefix(s.Domain, toComplete) {
			domains = append(domains, fmt.Sprintf("%s\tIsSafelink: %t", s.Domain, s.IsSafelink))
		}
	}
	return domains, cobra.ShellCompDirectiveNoFileComp
}

// promptSelectManualShortURL prompts the user to select a manually added short URL domain from a list.
func promptSelectManualShortURL(promptText string, shorteners []config.ShortenerService) (string, error) {
	if len(shorteners) == 0 {
		return "", fmt.Errorf("no manual short URL domains configured")
	}

	choices := make([]choose.Choice, len(shorteners))
	for i, s := range shorteners {
		choices[i] = choose.Choice{
			Text: s.Domain,
			Note: fmt.Sprintf("IsSafelink: %t", s.IsSafelink),
		}
	}

	result, err := prompt.New().Ask(promptText).
		AdvancedChoose(choices, choose.WithHelp(true))
	if err != nil {
		if err == prompt.ErrUserQuit {
			return "", nil
		}
		return "", err
	}

	// Find the matching domain
	for _, s := range shorteners {
		if s.Domain == result {
			return s.Domain, nil
		}
	}

	return "", nil
}

// promptSafelink presents a yes/no choice for safelink setting using AdvancedChoose
func promptSafelink(promptText string, currentValue bool) bool {
	choices := []choose.Choice{
		{Text: "Yes", Note: "URLs will be resolved before rule matching, then original URL launched"},
		{Text: "No", Note: "URLs will be resolved before rule matching, then resolved URL launched"},
	}

	defaultIndex := 1 // Default to No
	if currentValue {
		defaultIndex = 0 // Default to Yes if currently set
	}

	result, err := prompt.New().Ask(promptText).
		AdvancedChoose(choices, choose.WithDefaultIndex(defaultIndex))
	if err != nil || result == "" {
		return currentValue // Keep current value on error or cancel
	}

	return result == "Yes"
}

// registerShortURLCommands adds the shorturl subcommands to the given parent command.
func registerShortURLCommands(parentCmd *cobra.Command) {
	addShortURLCommands(parentCmd)
}
