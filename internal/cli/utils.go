package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"text/tabwriter"

	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/choose"
	"github.com/fatih/color"
	"github.com/jmylchreest/rurl/internal/config"
)

// --- Prompting Helpers ---

// promptString prompts for a simple string value with a default
func promptString(prompt string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	var input string
	reader := bufio.NewReader(os.Stdin)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

// promptSelectProfile prompts user to select a profile from a list
// Returns the selected profile ID, or an empty string if cancelled/error, or the default profile ID if Enter is pressed.
func promptSelectProfile(promptText string, availableProfiles []config.Profile, globalDefaultProfileID string, currentRuleProfileID string) (string, error) {
	if len(availableProfiles) == 0 {
		return "", fmt.Errorf("no available profiles to choose from")
	}

	choices := make([]choose.Choice, len(availableProfiles))
	for i, p := range availableProfiles {
		note := fmt.Sprintf("ID: %s, Browser: %s", p.ID, p.BrowserID)
		if p.ID == currentRuleProfileID {
			note += " (Current)"
		}
		if p.ID == globalDefaultProfileID {
			note += " [DEFAULT]"
		}
		choices[i] = choose.Choice{
			Text: p.Name,
			Note: note,
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

	// If user just pressed enter and we have a default
	if result == "" && globalDefaultProfileID != "" {
		fmt.Printf("Default profile selected: %s\n", globalDefaultProfileID)
		return globalDefaultProfileID, nil
	}

	// Find the matching profile
	for _, p := range availableProfiles {
		if p.Name == result {
			return p.ID, nil
		}
	}

	return "", nil
}

// promptSelectProfileOrDelete prompts user to select a profile or delete an item (like a rule)
// Returns selected profile ID, or delete=true if delete chosen, or empty string/false if cancelled/error
func promptSelectProfileOrDelete(promptText string, availableProfiles []config.Profile) (selectedID string, deleteItem bool, err error) {
	if len(availableProfiles) == 0 {
		// If no profiles left, the only option is effectively delete
		fmt.Println(promptText)
		fmt.Println("  No alternative profiles available.")
		confirmDelete := promptString("Delete this item? (yes/no)", "yes")
		if strings.EqualFold(confirmDelete, "yes") {
			return "", true, nil
		}
		return "", false, fmt.Errorf("operation cancelled, no profiles available and deletion declined")
	}

	choices := make([]choose.Choice, len(availableProfiles)+1) // +1 for delete option
	for i, p := range availableProfiles {
		choices[i] = choose.Choice{
			Text: p.Name,
			Note: fmt.Sprintf("ID: %s, Browser: %s", p.ID, p.BrowserID),
		}
	}
	// Add delete option as the last choice
	choices[len(availableProfiles)] = choose.Choice{
		Text: "Delete this item",
		Note: "Remove this item completely",
	}

	result, err := prompt.New().Ask(promptText).
		AdvancedChoose(choices, choose.WithHelp(true))
	if err != nil {
		if err == prompt.ErrUserQuit {
			return "", false, nil
		}
		return "", false, err
	}

	// Check if delete was selected
	if result == "Delete this item" {
		return "", true, nil
	}

	// Find the matching profile
	for _, p := range availableProfiles {
		if p.Name == result {
			return p.ID, false, nil
		}
	}

	return "", false, nil
}

// promptSelectBrowser prompts user to select a browser from a list
// Returns the selected browser ID, or an empty string if cancelled/error
func promptSelectBrowser(promptText string, availableBrowsers []config.Browser) (string, error) {
	if len(availableBrowsers) == 0 {
		return "", fmt.Errorf("no available browsers to choose from")
	}

	choices := make([]choose.Choice, len(availableBrowsers))
	for i, b := range availableBrowsers {
		choices[i] = choose.Choice{
			Text: b.Name,
			Note: fmt.Sprintf("Path: %s", b.Executable),
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

	// Find the matching browser
	for _, b := range availableBrowsers {
		if b.Name == result {
			return b.BrowserID, nil
		}
	}

	return "", nil
}

// promptSelectRule prompts user to select a rule from a list
// Returns the selected rule name, or an empty string if cancelled/error
func promptSelectRule(promptText string, availableRules []config.Rule) (string, error) {
	if len(availableRules) == 0 {
		return "", fmt.Errorf("no available rules to choose from")
	}

	choices := make([]choose.Choice, len(availableRules))
	for i, r := range availableRules {
		choices[i] = choose.Choice{
			Text: r.Name,
			Note: fmt.Sprintf("Pattern: %s, Profile: %s", r.Pattern, r.ProfileID),
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

	// Find the matching rule
	for _, r := range availableRules {
		if r.Name == result {
			return r.Name, nil
		}
	}

	return "", nil
}

// promptYesNo presents a yes/no choice using AdvancedChoose
// Returns true for yes, false for no or cancellation
func promptYesNo(promptText string, defaultYes bool) bool {
	choices := []choose.Choice{
		{Text: "Yes", Note: ""},
		{Text: "No", Note: ""},
	}

	defaultIndex := 1 // Default to No
	if defaultYes {
		defaultIndex = 0 // Default to Yes
	}

	result, err := prompt.New().Ask(promptText).
		AdvancedChoose(choices, choose.WithDefaultIndex(defaultIndex))
	if err != nil || result == "" {
		return false
	}

	return result == "Yes"
}

// --- Printing Helpers ---

// printBrowserList handles the actual printing of the browser list using tabwriter
func printBrowserList(cfg *config.Config) {
	if cfg == nil || len(cfg.Browsers) == 0 {
		fmt.Println("No browsers configured. Run 'rurl config detect-browsers --save' or 'rurl config browser add'.")
		return
	}

	fmt.Println("\n--- Browsers ---")

	// Initialize tabwriter
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0) // minwidth, tabwidth, padding, padchar, flags

	// Print header
	fmt.Fprintln(w, "ID\tName\tExecutable\tProfile Arg\tIncognito Arg")
	fmt.Fprintln(w, "--\t----\t----------\t------------\t--------------")

	// Print rows
	for _, b := range cfg.Browsers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			b.BrowserID,
			b.Name,
			b.Executable,
			b.ProfileArg,
			b.IncognitoArg,
		)
	}

	// Flush the writer to print the table
	w.Flush()
}

// printProfileList handles the actual printing of the profile list using tabwriter
func printProfileList(cfg *config.Config) {
	if cfg == nil || len(cfg.Profiles) == 0 {
		fmt.Println("No profiles configured. Run 'rurl config profile add'.")
		return
	}

	fmt.Println("\n--- Profiles ---")

	// Initialize tabwriter
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Create a color object for cyan
	cyan := color.New(color.FgCyan).SprintFunc()

	// Print header
	fmt.Fprintln(w, "ID\tName\tBrowser ID\tDirectory\tDefault")
	fmt.Fprintln(w, "--\t----\t----------\t----------\t-------")

	// Print rows
	for _, p := range cfg.Profiles {
		defaultMarker := ""
		if cfg.DefaultProfileID == p.ID {
			defaultMarker = cyan("[DEFAULT]")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			p.ID,
			p.Name,
			p.BrowserID,
			p.ProfileDir,
			defaultMarker,
		)
	}

	// Flush the writer to print the table
	w.Flush()
}

// printRuleList displays the configured rules using a tabwriter
func printRuleList(cfg *config.Config) {
	fmt.Println("\n--- Rules ---")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Name\tPattern\tScope\tProfile ID\tIncognito\tType")
	fmt.Fprintln(w, "----\t-------\t-----\t----------\t----------\t----")

	// Display the Default Rule first
	defaultProfileDisplay := "<none set>"
	if cfg.DefaultProfileID != "" {
		// Check if default profile actually exists, otherwise show it as invalid
		if _, err := cfg.FindProfileByID(cfg.DefaultProfileID); err == nil {
			defaultProfileDisplay = cfg.DefaultProfileID
		} else {
			defaultProfileDisplay = fmt.Sprintf("%s (invalid!)", cfg.DefaultProfileID)
		}
	}
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%s\n",
		defaultRuleName, // Assumes defaultRuleName is accessible (it's in config_rules.go)
		".*",            // Matches everything
		"url",           // Default rule always matches full URL
		defaultProfileDisplay,
		false, // Default rule is never incognito
		"Built-in",
	)

	// Display user-defined rules
	if len(cfg.Rules) == 0 {
		fmt.Fprintln(w, "(No user-defined rules)")
	} else {
		for _, r := range cfg.Rules {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%s\n",
				r.Name,
				r.Pattern,
				r.Scope,
				r.ProfileID,
				r.Incognito,
				"User",
			)
		}
	}
	w.Flush()
}

// --- Validation Helpers ---

// validateExecutable checks if a file exists and is executable
// Moved from config_browsers.go
func validateExecutable(path string) error {
	if path == "" {
		return fmt.Errorf("executable path cannot be empty")
	}

	// If path is not absolute, try to find it in PATH
	if !filepath.IsAbs(path) {
		fullPath, err := exec.LookPath(path)
		if err != nil {
			return fmt.Errorf("executable '%s' not found in PATH: %w", path, err)
		}
		path = fullPath
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("executable '%s' does not exist: %w", path, err)
		}
		return fmt.Errorf("cannot access executable '%s': %w", path, err)
	}

	if info.IsDir() {
		return fmt.Errorf("path '%s' is a directory, not an executable", path)
	}

	return nil
}
