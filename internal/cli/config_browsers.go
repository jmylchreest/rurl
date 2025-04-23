package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// addBrowserCommands adds the browser subcommands to the parent command
func addBrowserCommands(parentCmd *cobra.Command) {
	browserCmd := &cobra.Command{
		Use:   "browser",
		Short: "Manage browser configurations",
		Long:  `Add, edit, delete, and list browser configurations.`,
	}

	browserListCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured browsers",
		Long:  `Display all configured browsers.`,
		Run:   runBrowserListCmd,
	}
	browserAddCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new browser configuration",
		Long:  `Interactively add a new browser configuration.`,
		Run:   runBrowserAddCmd,
	}
	browserEditCmd := &cobra.Command{
		Use:               "edit [browser-id]",
		Short:             "Edit a browser configuration",
		Long:              `Edit an existing browser configuration.`,
		Args:              cobra.MaximumNArgs(1),
		Run:               runBrowserEditCmd,
		ValidArgsFunction: completeBrowserIDs,
	}
	browserDeleteCmd := &cobra.Command{
		Use:               "delete [browser-id]",
		Short:             "Delete a browser configuration",
		Long:              `Delete an existing browser configuration.`,
		Args:              cobra.ExactArgs(1),
		Run:               runBrowserDeleteCmd,
		ValidArgsFunction: completeBrowserIDs,
	}

	browserCmd.AddCommand(browserListCmd)
	browserCmd.AddCommand(browserAddCmd)
	browserCmd.AddCommand(browserEditCmd)
	browserCmd.AddCommand(browserDeleteCmd)
	parentCmd.AddCommand(browserCmd)
}

// runBrowserListCmd displays all configured browsers
func runBrowserListCmd(cmd *cobra.Command, args []string) {
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load configuration")
		return
	}

	printBrowserList(cfg)
}

// runBrowserAddCmd adds a new browser configuration
func runBrowserAddCmd(cmd *cobra.Command, args []string) {
	if cfg == nil {
		log.Error().Msg("Configuration not loaded.")
		os.Exit(1)
	}

	var browser config.Browser

	// Prompt for browser details
	fmt.Println("Enter details for the new browser:")

	// Loop for non-empty Browser Name
	for {
		browser.Name = promptString("Browser Name", "")
		if browser.Name == "" {
			fmt.Fprintln(os.Stderr, "Error: Browser Name cannot be empty.")
		} else {
			break
		}
	}

	// Loop for non-empty and unique Browser ID
	defaultID := strings.ToLower(strings.ReplaceAll(browser.Name, " ", "-"))
	for {
		browser.BrowserID = strings.ToLower(strings.ReplaceAll(promptString("Browser ID", defaultID), " ", "-"))
		if browser.BrowserID == "" {
			fmt.Fprintln(os.Stderr, "Error: Browser ID cannot be empty.")
			continue
		}

		// Validate BrowserID uniqueness
		idExists := false
		for _, b := range cfg.Browsers {
			if b.BrowserID == browser.BrowserID {
				idExists = true
				break
			}
		}
		if idExists {
			fmt.Fprintf(os.Stderr, "Error: Browser with ID '%s' already exists.\n", browser.BrowserID)
		} else {
			break
		}
	}

	// Loop for valid Executable Path
	for {
		browser.Executable = promptString("Executable Path or Command", "")
		if browser.Executable == "" {
			fmt.Fprintln(os.Stderr, "Error: Executable path cannot be empty.")
			continue
		}
		if err := validateExecutable(browser.Executable); err != nil {
			fmt.Fprintf(os.Stderr, "Validation Error: %v\n", err)
		} else {
			break
		}
	}

	browser.ProfileArg = promptString("Profile Argument Template (use %s for profile dir)", "--profile-directory=%s")
	browser.IncognitoArg = promptString("Incognito Argument", "--incognito")

	// Add the browser to config
	cfg.Browsers = append(cfg.Browsers, browser)

	// Save the config
	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nBrowser '%s' (ID: %s) added successfully.\n", browser.Name, browser.BrowserID)
}

// runBrowserEditCmd edits an existing browser configuration
func runBrowserEditCmd(cmd *cobra.Command, args []string) {
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load configuration")
		return
	}

	var browserID string
	if len(args) > 0 {
		browserID = args[0]
	} else {
		// If no browser ID provided, prompt user to select one
		selectedID, err := promptSelectBrowser("Select browser to edit:", cfg.Browsers)
		if err != nil {
			log.Error().Err(err).Msg("Failed to select browser")
			return
		}
		if selectedID == "" {
			fmt.Println("Operation cancelled.")
			return
		}
		browserID = selectedID
	}

	// Find the browser to edit
	var browserIndex int
	var found bool
	for i, b := range cfg.Browsers {
		if b.BrowserID == browserID {
			browserIndex = i
			found = true
			break
		}
	}

	if !found {
		fmt.Fprintf(os.Stderr, "Error: Browser with ID '%s' not found.\n", browserID)
		return
	}

	browser := &cfg.Browsers[browserIndex]

	// Prompt for edits
	name := promptString("Browser Name", browser.Name)
	executable := promptString("Executable Path", browser.Executable)
	profileArg := promptString("Profile Argument", browser.ProfileArg)
	incognitoArg := promptString("Incognito Argument", browser.IncognitoArg)

	// Update browser
	browser.Name = name
	browser.Executable = executable
	browser.ProfileArg = profileArg
	browser.IncognitoArg = incognitoArg

	// Save configuration
	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		log.Error().Err(err).Msg("Failed to save configuration")
		return
	}

	fmt.Printf("Browser '%s' updated successfully.\n", browser.Name)
}

// runBrowserDeleteCmd deletes a browser configuration
func runBrowserDeleteCmd(cmd *cobra.Command, args []string) {
	browserID := args[0]

	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load configuration")
		return
	}

	// Find and remove the browser
	var found bool
	for i, browser := range cfg.Browsers {
		if browser.BrowserID == browserID {
			// Ask for confirmation
			confirm := promptString(fmt.Sprintf("Are you sure you want to delete browser '%s'? (y/N)", browser.Name), "N")
			if strings.ToLower(confirm) != "y" {
				fmt.Println("Operation cancelled.")
				return
			}

			cfg.Browsers = append(cfg.Browsers[:i], cfg.Browsers[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		fmt.Fprintf(os.Stderr, "Error: Browser with ID '%s' not found.\n", browserID)
		return
	}

	// Save configuration
	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		log.Error().Err(err).Msg("Failed to save configuration")
		return
	}

	fmt.Printf("Browser '%s' deleted successfully.\n", browserID)
}
