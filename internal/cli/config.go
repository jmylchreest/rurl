package cli

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/jmylchreest/rurl/internal/browser"
	"github.com/jmylchreest/rurl/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// addConfigCommands adds all configuration-related commands to the root command
func addConfigCommands() {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage rurl configuration",
		Long:  `Add, edit, delete, and list browser, profile, and rule configurations.`,
	}

	// --- Combined List Command ---
	configListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured browsers, profiles, and rules",
		Long:  `Displays all configured browsers, profiles, and rules.`,
		Run:   runConfigListCmd,
	}
	configCmd.AddCommand(configListCmd)

	// --- Detect Browsers Command ---
	detectBrowsersCmd := &cobra.Command{
		Use:   "detect-browsers",
		Short: "Detect installed browsers/profiles and optionally update config",
		Long: `Scans the system for known browser installations and their profiles.
Prints the detected browsers and profiles.
Use the --save flag to compare with current config, handle removals interactively, and save changes.`,
		Run: runDetectBrowsersCmd,
	}
	detectBrowsersCmd.Flags().BoolVar(&detectSave, "save", false, "Save detected browsers/profiles to config file (interactive update)")
	configCmd.AddCommand(detectBrowsersCmd)

	// --- Browser Commands (Moved to config_browsers.go) ---
	addBrowserCommands(configCmd)

	// --- Profile Commands (Moved to config_profiles.go) ---
	addProfileCommands(configCmd)

	// --- Set Default Profile Command ---
	setDefaultProfileCmd := &cobra.Command{
		Use:               "set-default [profile-id]",
		Short:             "Set the default profile",
		Long:              `Sets the profile with the given ID as the default fallback profile. If no ID is provided, it prompts for selection.`,
		Args:              cobra.MaximumNArgs(1),
		Run:               runSetDefaultProfileCmd,
		ValidArgsFunction: completeProfileIDs,
	}
	configCmd.AddCommand(setDefaultProfileCmd)

	// --- Rule Commands (Moved to config_rules.go) ---
	AddRuleCommands(configCmd)

	// --- Shortener Commands (Moved to config_shorteners.go) ---
	registerShortURLCommands(configCmd)

	// Add the main config command to the root command
	rootCmd.AddCommand(configCmd)
}

// --- Helper Functions ---

// promptString prompts for a simple string value with a default

// validateExecutable checks if a file exists and is executable

// Helper function for prompting user to select a profile from a list

// Helper function for prompting user to select a profile or delete an item (like a rule)

// --- Internal Helpers for runDetectBrowsersCmd ---

// performDetection calls the detector and gathers browser/profile info
func performDetection() ([]config.Browser, []config.Profile, map[string]config.Browser, map[string]config.Profile, error) {
	detector, err := browser.NewDetector()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create browser detector for this OS")
		return nil, nil, nil, nil, fmt.Errorf("creating browser detector: %w", err)
	}

	detectedBrowsers, err := detector.DiscoverBrowsers()
	if err != nil {
		// Log but don't necessarily fail the whole process
		log.Error().Err(err).Msg("Failed during browser discovery")
		fmt.Fprintf(os.Stderr, "Warning: Error during browser discovery: %v\n", err)
		// Continue with potentially empty list
	}

	detectedProfiles := make([]config.Profile, 0)
	detectedBrowserMap := make(map[string]config.Browser)
	detectedProfileMap := make(map[string]config.Profile)

	for _, b := range detectedBrowsers {
		detectedBrowserMap[b.BrowserID] = b
		profiles, err := detector.DiscoverProfiles(b)
		if err != nil {
			log.Warn().Err(err).Str("browser_id", b.BrowserID).Msg("Failed to discover profiles for browser")
			fmt.Fprintf(os.Stderr, "Warning: Failed to discover profiles for %s (ID: %s): %v\n", b.Name, b.BrowserID, err)
		} else {
			detectedProfiles = append(detectedProfiles, profiles...)
			for _, p := range profiles {
				detectedProfileMap[p.ID] = p
			}
		}
	}

	return detectedBrowsers, detectedProfiles, detectedBrowserMap, detectedProfileMap, nil
}

// compareDetectedWithConfig identifies items in config not found by detection
func compareDetectedWithConfig(cfg *config.Config, detectedBrowserMap map[string]config.Browser, detectedProfileMap map[string]config.Profile) (map[string]config.Browser, map[string]config.Profile, map[string]struct{}) {
	cfgBrowserMap := make(map[string]config.Browser)
	for _, b := range cfg.Browsers {
		cfgBrowserMap[b.BrowserID] = b
	}
	cfgProfileMap := make(map[string]config.Profile)
	for _, p := range cfg.Profiles {
		cfgProfileMap[p.ID] = p
	}

	browsersToRemove := make(map[string]config.Browser)
	profilesToRemove := make(map[string]config.Profile) // Profiles explicitly not found for existing browsers
	profileIDsToRemove := make(map[string]struct{})     // All profile IDs associated with removed items

	// Find browsers in config but not detected
	for browserID, browser := range cfgBrowserMap {
		if _, found := detectedBrowserMap[browserID]; !found {
			log.Warn().Str("browser_id", browserID).Str("name", browser.Name).Msg("Configured browser not detected.")
			browsersToRemove[browserID] = browser
			// Mark all its current profiles for removal
			for _, p := range cfg.Profiles {
				if p.BrowserID == browserID {
					if _, alreadyMarked := profilesToRemove[p.ID]; !alreadyMarked { // Avoid double marking
						profilesToRemove[p.ID] = p
						profileIDsToRemove[p.ID] = struct{}{}
						log.Debug().Str("profile_id", p.ID).Str("reason", "Parent browser removed").Msg("Marking profile for removal")
					}
				}
			}
		}
	}

	// Find profiles in config but not detected (for browsers that *were* detected)
	for profileID, profile := range cfgProfileMap {
		// Only check profiles whose browser WAS detected AND wasn't already marked for removal
		if _, browserDetected := detectedBrowserMap[profile.BrowserID]; browserDetected {
			if _, profileDetected := detectedProfileMap[profileID]; !profileDetected {
				// Check if it wasn't already marked due to parent browser removal
				if _, alreadyMarked := profileIDsToRemove[profileID]; !alreadyMarked {
					log.Warn().Str("profile_id", profileID).Str("name", profile.Name).Str("browser_id", profile.BrowserID).Msg("Configured profile not detected.")
					profilesToRemove[profileID] = profile
					profileIDsToRemove[profileID] = struct{}{}
					log.Debug().Str("profile_id", profileID).Str("reason", "Profile not found").Msg("Marking profile for removal")
				}
			}
		}
	}
	return browsersToRemove, profilesToRemove, profileIDsToRemove
}

// handleOrphanedDefaultProfile manages selection of a new default if needed
func handleOrphanedDefaultProfile(originalDefaultID, currentDefaultID string, profileIDsToRemove map[string]struct{}, profilesToKeep []config.Profile) string {
	newDefaultProfileID := currentDefaultID
	if _, removingDefault := profileIDsToRemove[originalDefaultID]; removingDefault && originalDefaultID != "" {
		log.Warn().Str("profile_id", originalDefaultID).Msg("Default profile is being removed.")

		if len(profilesToKeep) == 1 {
			newDefaultProfileID = profilesToKeep[0].ID
			log.Info().Str("profile_id", newDefaultProfileID).Msg("Automatically setting the only remaining profile as default.")
			fmt.Printf("Info: Default profile '%s' removed. Automatically setting '%s' as new default.\n", originalDefaultID, newDefaultProfileID)
		} else if len(profilesToKeep) > 1 {
			selectedID, err := promptSelectProfile(fmt.Sprintf("Default profile '%s' is being removed. Select a new default profile:", originalDefaultID), profilesToKeep, originalDefaultID, "")
			if err != nil || selectedID == "" {
				log.Error().Err(err).Msg("Failed to select a new default profile. Clearing default.")
				fmt.Fprintln(os.Stderr, "Error selecting default profile or selection cancelled. Default profile will be unset.")
				newDefaultProfileID = ""
			} else {
				newDefaultProfileID = selectedID
				log.Info().Str("profile_id", newDefaultProfileID).Msg("New default profile selected by user.")
			}
		} else { // len == 0
			log.Warn().Msg("No profiles remaining. Clearing default profile setting.")
			fmt.Printf("Info: Default profile '%s' removed. No other profiles remain. Default profile unset.\n", originalDefaultID)
			newDefaultProfileID = ""
		}
	}
	return newDefaultProfileID
}

// handleOrphanedRules manages selection/deletion for rules with removed profiles
func handleOrphanedRules(originalRules []config.Rule, profileIDsToRemove map[string]struct{}, profilesToKeep []config.Profile) (map[string]string, map[string]struct{}) {
	rulesToUpdate := make(map[string]string)
	rulesToDelete := make(map[string]struct{})

	log.Debug().Msg("Checking rules for orphaned profiles...")
	for _, rule := range originalRules {
		if _, removingRuleProfile := profileIDsToRemove[rule.ProfileID]; removingRuleProfile {
			log.Warn().Str("rule_name", rule.Name).Str("profile_id", rule.ProfileID).Msg("Rule references a profile being removed.")

			if len(profilesToKeep) == 1 {
				newProfileID := profilesToKeep[0].ID
				log.Info().Str("rule_name", rule.Name).Str("new_profile_id", newProfileID).Msg("Automatically updating rule to use the only remaining profile.")
				fmt.Printf("Info: Rule '%s' automatically updated to use profile '%s'.\n", rule.Name, newProfileID)
				rulesToUpdate[rule.Name] = newProfileID
			} else if len(profilesToKeep) > 1 {
				prompt := fmt.Sprintf("Rule '%s' uses profile '%s' which is being removed.", rule.Name, rule.ProfileID)
				selectedID, deleteRule, err := promptSelectProfileOrDelete(prompt+" Select replacement or delete rule:", profilesToKeep)

				if err != nil {
					log.Error().Err(err).Str("rule_name", rule.Name).Msg("Error during rule update prompt. Rule will be deleted.")
					fmt.Fprintf(os.Stderr, "Error processing rule '%s': %v. Rule will be deleted.\n", rule.Name, err)
					rulesToDelete[rule.Name] = struct{}{}
				} else if deleteRule {
					log.Info().Str("rule_name", rule.Name).Msg("User chose to delete rule.")
					rulesToDelete[rule.Name] = struct{}{}
				} else if selectedID != "" {
					log.Info().Str("rule_name", rule.Name).Str("new_profile_id", selectedID).Msg("User selected new profile for rule.")
					rulesToUpdate[rule.Name] = selectedID
				} else { // Cancelled prompt
					log.Warn().Str("rule_name", rule.Name).Msg("Rule update cancelled by user. Rule will be deleted.")
					fmt.Fprintf(os.Stderr, "Rule '%s' update cancelled. Rule will be deleted.\n", rule.Name)
					rulesToDelete[rule.Name] = struct{}{}
				}
			} else { // len == 0
				log.Warn().Str("rule_name", rule.Name).Msg("No remaining profiles. Deleting rule.")
				fmt.Printf("Info: Rule '%s' deleted because its profile '%s' was removed and no other profiles exist.\n", rule.Name, rule.ProfileID)
				rulesToDelete[rule.Name] = struct{}{}
			}
		}
	}
	return rulesToUpdate, rulesToDelete
}

// displayProposedChanges prints a summary of potential destructive changes
// (No longer used - using simplified summary now)

// confirmAndSaveChanges prompts the user and saves the final configuration
func confirmAndSaveChanges(finalCfg *config.Config, cfgFile string) bool {
	confirm := promptString("\nApply these changes and save the configuration? (yes/no)", "no")
	if !strings.EqualFold(confirm, "yes") {
		fmt.Println("Changes discarded.")
		log.Info().Msg("User cancelled configuration save.")
		return false
	}

	if err := config.SaveConfig(finalCfg, cfgFile); err != nil {
		log.Error().Err(err).Msg("Failed to save updated configuration")
		fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
		os.Exit(1) // Exit on save error
	}

	log.Info().Msg("Configuration updated successfully based on detection.")
	fmt.Println("\nConfiguration saved successfully.")
	return true
}

// --- Main Command Run Functions ---

// runConfigListCmd displays all configured browsers, profiles, and rules
func runConfigListCmd(cmd *cobra.Command, args []string) {
	if cfg == nil {
		log.Error().Msg("Configuration not loaded.")
		os.Exit(1)
	}
	fmt.Println("--- Configuration Summary ---")
	printBrowserList(cfg)
	printProfileList(cfg)
	printRuleList(cfg)
}

// runDetectBrowsersCmd is the CLI command to detect browsers and handle config updates
func runDetectBrowsersCmd(cmd *cobra.Command, args []string) {
	log.Info().Msg("Running browser detection...")
	if cfg == nil {
		log.Error().Msg("Configuration not loaded.")
		os.Exit(1)
	}

	// --- Store Original Config State --- (Needed for comparison if saving)
	originalBrowsers := make([]config.Browser, len(cfg.Browsers))
	copy(originalBrowsers, cfg.Browsers)
	originalProfiles := make([]config.Profile, len(cfg.Profiles))
	copy(originalProfiles, cfg.Profiles)
	originalRules := make([]config.Rule, len(cfg.Rules))
	copy(originalRules, cfg.Rules)
	originalDefaultProfileID := cfg.DefaultProfileID

	// --- Detection (using refactored browser package) ---
	discoveredBrowsers, discoveredProfiles, err := browser.DetectAll()
	if err != nil {
		// Log the error from the detector creation
		log.Error().Err(err).Msg("Failed to initialize browser detection")
		fmt.Fprintf(os.Stderr, "Error initializing browser detection: %v\n", err)
		os.Exit(1)
	}
	log.Info().Int("browser_count", len(discoveredBrowsers)).Int("profile_count", len(discoveredProfiles)).Msg("Detection complete")

	// --- Report Detected Items or Save ---
	if !detectSave { // Flag from root.go
		log.Info().Msg("Displaying detected browsers/profiles (use --save to update config).")
		// Print detected browsers using utils helper
		tempCfgBrowsers := &config.Config{Browsers: discoveredBrowsers}
		printBrowserList(tempCfgBrowsers) // Will print header and "(None detected)" if empty

		// Print detected profiles using utils helper
		tempCfgProfiles := &config.Config{Profiles: discoveredProfiles}
		printProfileList(tempCfgProfiles) // Will print header and "(None detected)" if empty

		fmt.Println("\nRun with --save to update the configuration file.")
		return
	}

	// --- Save Logic --- (--save flag is true)
	log.Info().Msg("Comparing detected browsers/profiles with current configuration (--save specified)...")

	// Build maps for easy lookup during comparison
	detectedBrowserMap := make(map[string]config.Browser)
	for _, b := range discoveredBrowsers {
		detectedBrowserMap[b.BrowserID] = b
	}
	detectedProfileMap := make(map[string]config.Profile)
	for _, p := range discoveredProfiles {
		detectedProfileMap[p.ID] = p
	}

	// Identify items in config not found by detection
	_, _, profileIDsToRemove := compareDetectedWithConfig(cfg, detectedBrowserMap, detectedProfileMap)

	// Prepare intermediate state (start with detected items)
	browsersToKeep := discoveredBrowsers
	profilesToKeep := discoveredProfiles
	newDefaultProfileID := cfg.DefaultProfileID // Start with current, may change
	rulesToUpdate := make(map[string]string)
	rulesToDelete := make(map[string]struct{})

	// Handle Default Profile Interactively if it's being removed
	newDefaultProfileID = handleOrphanedDefaultProfile(cfg.DefaultProfileID, newDefaultProfileID, profileIDsToRemove, profilesToKeep)

	// Handle Orphaned Rules Interactively
	rulesToUpdate, rulesToDelete = handleOrphanedRules(cfg.Rules, profileIDsToRemove, profilesToKeep)

	// --- Construct Final Proposed Config State ---
	finalRules := []config.Rule{}
	for _, rule := range cfg.Rules { // Iterate original rules
		if _, markedForDeletion := rulesToDelete[rule.Name]; markedForDeletion {
			continue // Skip deleted rules
		}
		if updatedProfileID, needsUpdate := rulesToUpdate[rule.Name]; needsUpdate {
			rule.ProfileID = updatedProfileID // Update profile ID
		}
		finalRules = append(finalRules, rule) // Add rule (updated or unchanged)
	}

	finalCfg := config.Config{
		Browsers:         browsersToKeep,
		Profiles:         profilesToKeep,
		Rules:            finalRules,
		DefaultProfileID: newDefaultProfileID,
		Shorteners:       cfg.Shorteners, // Preserve other sections like Shorteners
	}

	// --- Final Comparison and Confirmation ---
	browsersActuallyChanged := !reflect.DeepEqual(originalBrowsers, finalCfg.Browsers)
	profilesActuallyChanged := !reflect.DeepEqual(originalProfiles, finalCfg.Profiles)
	defaultActuallyChanged := originalDefaultProfileID != finalCfg.DefaultProfileID
	rulesActuallyChanged := !reflect.DeepEqual(originalRules, finalCfg.Rules)

	configActuallyChanged := browsersActuallyChanged || profilesActuallyChanged || defaultActuallyChanged || rulesActuallyChanged

	if !configActuallyChanged {
		log.Info().Msg("No effective changes detected between configuration and detected state.")
		fmt.Println("\nConfiguration matches detected state. No changes needed.")
		return
	}

	// --- Display Summary of Changes ---
	fmt.Println("\nConfiguration changes detected:")
	fmt.Printf("- Browsers Changed: %s\n", mapChangeToString(browsersActuallyChanged))
	fmt.Printf("- Profiles Changed: %s\n", mapChangeToString(profilesActuallyChanged))
	fmt.Printf("- Default Profile Changed: %s\n", mapChangeToString(defaultActuallyChanged))
	fmt.Printf("- Rules Updated/Deleted: %s\n", mapChangeToString(rulesActuallyChanged))

	// Optionally show detailed diff here later if needed

	// --- Confirm and Save Changes ---
	if confirmAndSaveChanges(&finalCfg, cfgFile) {
		log.Info().Msg("Configuration updated and saved successfully.")
		fmt.Println("Configuration updated and saved successfully.")
	} else {
		log.Info().Msg("Configuration changes discarded by user.")
		fmt.Println("Configuration changes discarded.")
	}
}

// mapChangeToString helper for summary
func mapChangeToString(changed bool) string {
	if changed {
		return "Yes"
	}
	return "No"
}

// runSetDefaultProfileCmd sets the default profile ID, prompting if none is provided
func runSetDefaultProfileCmd(cmd *cobra.Command, args []string) {
	if cfg == nil {
		log.Error().Msg("Configuration not loaded.")
		os.Exit(1)
	}

	var profileID string
	var err error

	if len(args) == 0 {
		// No profile ID provided, check number of profiles
		numProfiles := len(cfg.Profiles)
		if numProfiles == 0 {
			fmt.Fprintln(os.Stderr, "Error: No profiles configured. Cannot set a default.")
			os.Exit(1)
		} else if numProfiles == 1 {
			profileID = cfg.Profiles[0].ID
			fmt.Printf("Only one profile found ('%s'). Setting it as default.\n", profileID)
			// Proceed to set and save below
		} else {
			// Multiple profiles, prompt for selection
			// Pass current default as hint, empty string for currentRuleProfileID
			profileID, err = promptSelectProfile("Select the profile to set as default:", cfg.Profiles, cfg.DefaultProfileID, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error selecting profile: %v\n", err)
				os.Exit(1)
			}
			if profileID == "" { // User cancelled
				os.Exit(0)
			}
			log.Info().Str("profile_id", profileID).Msg("Default profile selected by user.")
		}
	} else {
		// Profile ID provided as argument
		profileID = args[0]
	}

	// Validate that the profile ID exists (whether provided or selected)
	profileExists := false
	for _, p := range cfg.Profiles {
		if p.ID == profileID {
			profileExists = true
			break
		}
	}

	if !profileExists {
		fmt.Fprintf(os.Stderr, "Error: Profile with ID '%s' not found.\n", profileID)
		fmt.Fprintln(os.Stderr, "Use 'rurl config list' or 'rurl config profile list' to see available profiles.")
		os.Exit(1)
	}

	// Set the default profile ID
	cfg.DefaultProfileID = profileID

	// Save the config
	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		log.Error().Err(err).Str("profile_id", profileID).Msg("Failed to save config after setting default profile")
		fmt.Fprintf(os.Stderr, "Error saving configuration after setting default profile to '%s': %v\n", profileID, err)
		os.Exit(1)
	}

	fmt.Printf("Default profile successfully set to '%s'.\n", profileID)
}
