package cli

import (
	"fmt"
	"os"
	"strings"

	// Needed for printProfileList
	"github.com/jmylchreest/rurl/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	//"github.com/spf13/viper" // Not needed directly in profile funcs anymore
)

// addProfileCommands adds the profile subcommands to the parent command
func addProfileCommands(parentCmd *cobra.Command) {
	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage profile configurations",
		Long:  `Add, edit, delete, and list profile configurations.`,
	}

	profileListCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured profiles",
		Long:  `Display all configured profiles.`,
		Run:   runProfileListCmd,
	}
	profileAddCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new profile configuration",
		Long:  `Interactively add a new profile configuration.`,
		Run:   runProfileAddCmd,
	}
	profileEditCmd := &cobra.Command{
		Use:               "edit [profile-id]",
		Short:             "Edit a profile configuration",
		Long:              `Interactively edit an existing profile configuration. If only one profile exists, it will be selected automatically if no ID is provided.`,
		Args:              cobra.MaximumNArgs(1), // Allow 0 or 1 arg
		Run:               runProfileEditCmd,
		ValidArgsFunction: completeProfileIDs, // Register completer
	}
	profileDeleteCmd := &cobra.Command{
		Use:               "delete [profile-id]",
		Short:             "Delete a profile configuration",
		Long:              `Delete an existing profile configuration. If only one exists, it will be selected automatically if no ID is provided (confirmation still required).`,
		Args:              cobra.MaximumNArgs(1), // Allow 0 or 1 arg
		Run:               runProfileDeleteCmd,
		ValidArgsFunction: completeProfileIDs, // Register completer
	}

	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileAddCmd)
	profileCmd.AddCommand(profileEditCmd)
	profileCmd.AddCommand(profileDeleteCmd)
	parentCmd.AddCommand(profileCmd)
}

// runProfileListCmd displays all configured profiles
func runProfileListCmd(cmd *cobra.Command, args []string) {
	if cfg == nil {
		log.Error().Msg("Configuration not loaded.")
		os.Exit(1)
	}
	printProfileList(cfg) // Pass cfg explicitly
}

// printProfileList handles the actual printing of the profile list using tabwriter
// (Moved to utils.go)

// runProfileAddCmd adds a new profile configuration
func runProfileAddCmd(cmd *cobra.Command, args []string) {
	if cfg == nil {
		log.Error().Msg("Configuration not loaded.")
		os.Exit(1)
	}

	if len(cfg.Browsers) == 0 {
		fmt.Println("No browsers configured. Please add a browser first using 'rurl config browser add' or 'rurl config detect-browsers --save'.")
		os.Exit(1)
	}

	var profile config.Profile

	fmt.Println("\nEnter details for the new profile:")

	// List available browsers for user convenience
	// fmt.Println("\nAvailable Browsers:") // Header printed by printBrowserList now
	printBrowserList(cfg) // Call helper from utils.go

	// Prompt for browser ID and validate
	for {
		profile.BrowserID = promptString("Browser ID for this profile", "")
		found := false
		for _, b := range cfg.Browsers {
			if b.BrowserID == profile.BrowserID {
				found = true
				break
			}
		}
		if found {
			break
		}
		fmt.Println("Error: Invalid browser ID. Please choose from the list above.")
	}

	// Prompt for Profile Name first
	profile.Name = promptString("Profile Name", "")
	if profile.Name == "" {
		fmt.Fprintln(os.Stderr, "Error: Profile Name cannot be empty.")
		// Exiting for simplicity as originally implemented
		os.Exit(1)
	}

	// Prompt for Profile ID (loop until unique)
	for {
		defaultID := fmt.Sprintf("%s-%s", profile.BrowserID, strings.ToLower(strings.ReplaceAll(profile.Name, " ", "-")))
		profile.ID = strings.ToLower(strings.ReplaceAll(promptString("Profile ID", defaultID), " ", "-"))

		if profile.ID == "" {
			fmt.Fprintln(os.Stderr, "Error: Profile ID cannot be empty.")
			continue
		}

		// Validate Profile ID uniqueness
		idExists := false
		for _, p := range cfg.Profiles {
			if p.ID == profile.ID {
				fmt.Fprintf(os.Stderr, "Error: Profile with ID '%s' already exists.\n", profile.ID)
				idExists = true
				break
			}
		}
		if !idExists {
			break // Unique ID entered
		}
		// If ID exists, loop continues and prompts for ID again
	}

	profile.ProfileDir = promptString("Profile Directory Name/Path (relative to browser's user data)", "Default") // Often "Default", "Profile 1", etc.

	// Add the profile to config
	cfg.Profiles = append(cfg.Profiles, profile)

	// If this is the first profile ever added, offer to make it the default
	if cfg.DefaultProfileID == "" && len(cfg.Profiles) == 1 {
		if promptYesNo(fmt.Sprintf("This is the first profile. Make '%s' the default profile?", profile.ID), true) {
			cfg.DefaultProfileID = profile.ID
			fmt.Println("Profile set as default.")
		}
	} else if cfg.DefaultProfileID == "" {
		// Offer to set default if none exists yet
		if promptYesNo(fmt.Sprintf("No default profile set. Make '%s' the default profile?", profile.ID), false) {
			cfg.DefaultProfileID = profile.ID
			fmt.Println("Profile set as default.")
		}
	}

	// Save the config
	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nProfile '%s' (ID: %s) added successfully.\n", profile.Name, profile.ID)
}

// runProfileEditCmd edits an existing profile configuration
func runProfileEditCmd(cmd *cobra.Command, args []string) {
	if cfg == nil {
		log.Error().Msg("Configuration not loaded.")
		os.Exit(1)
	}

	var profileID string
	var err error // For error handling from prompts

	if len(args) == 0 {
		// No ID provided
		if len(cfg.Profiles) == 1 {
			profileID = cfg.Profiles[0].ID
			log.Info().Str("profile_id", profileID).Msg("Only one profile found, selecting it automatically for editing.")
		} else if len(cfg.Profiles) == 0 {
			fmt.Fprintln(os.Stderr, "Error: No profiles configured to edit.")
			fmt.Fprintln(os.Stderr, "Use 'rurl config profile add'.")
			os.Exit(1)
		} else {
			// Multiple profiles exist, prompt user
			fmt.Println("Multiple profiles configured.")
			profileID, err = promptSelectProfile("Select the profile to edit:", cfg.Profiles, cfg.DefaultProfileID, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error selecting profile: %v\n", err)
				os.Exit(1)
			}
			if profileID == "" { // User cancelled selection
				os.Exit(0)
			}
			log.Info().Str("profile_id", profileID).Msg("Profile selected by user for editing.")
		}
	} else {
		// ID provided
		profileID = args[0]
	}

	var profile *config.Profile
	var index int = -1

	// Find the profile
	for i := range cfg.Profiles {
		if cfg.Profiles[i].ID == profileID {
			profile = &cfg.Profiles[i] // Get pointer
			index = i
			break
		}
	}

	if profile == nil {
		fmt.Fprintf(os.Stderr, "Error: Profile with ID '%s' not found.\n", profileID)
		os.Exit(1)
	}

	fmt.Printf("\nEditing Profile '%s' (ID: %s):\n", profile.Name, profile.ID)

	// Prompt for updated values
	originalName := profile.Name // Store original name
	newName := promptString("Profile Name", originalName)
	if newName == "" {
		fmt.Fprintln(os.Stderr, "Error: Profile Name cannot be empty. Keeping original.")
		newName = originalName // Revert if empty
	}
	profile.Name = newName

	originalID := profile.ID // Store original ID
	var newProfileID string
	for {
		newProfileID = strings.ToLower(strings.ReplaceAll(promptString("Profile ID", originalID), " ", "-"))
		if newProfileID == "" {
			fmt.Fprintln(os.Stderr, "Error: Profile ID cannot be empty.")
			continue
		}
		// Check if new profile ID already exists (if changed)
		idExists := false
		if newProfileID != originalID { // Check against original ID
			for i, p := range cfg.Profiles {
				if i != index && p.ID == newProfileID {
					idExists = true
					break
				}
			}
		}
		if idExists {
			fmt.Fprintf(os.Stderr, "Error: Profile with ID '%s' already exists.\n", newProfileID)
			// Loop continues, prompts again
		} else {
			break // Unique ID entered
		}
	}

	// Apply ID change if it occurred
	if newProfileID != originalID {
		profile.ID = newProfileID // Update the ID in the profile object

		// Update default profile ID if necessary
		if cfg.DefaultProfileID == originalID {
			cfg.DefaultProfileID = newProfileID
			log.Debug().Str("old_id", originalID).Str("new_id", newProfileID).Msg("Updated default profile ID reference")
		}

		// Update any Rules that reference this profile ID
		rulesUpdatedCount := 0
		for i := range cfg.Rules {
			if cfg.Rules[i].ProfileID == originalID {
				cfg.Rules[i].ProfileID = newProfileID
				rulesUpdatedCount++
			}
		}
		if rulesUpdatedCount > 0 {
			log.Info().Int("count", rulesUpdatedCount).Str("old_id", originalID).Str("new_id", newProfileID).Msg("Updated rules referencing changed profile ID")
		}
	}

	profile.ProfileDir = promptString("Profile Directory Name/Path", profile.ProfileDir)

	// Offer to make this the default profile
	if cfg.DefaultProfileID != profile.ID { // Use potentially updated profile.ID
		if promptYesNo(fmt.Sprintf("Make '%s' the default profile?", profile.ID), false) {
			cfg.DefaultProfileID = profile.ID
			fmt.Println("Profile set as default.")
		}
	} else {
		fmt.Println("This is already the default profile.")
	}

	// Save the config
	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nProfile '%s' (ID: %s) updated successfully.\n", profile.Name, profile.ID)
}

// runProfileDeleteCmd deletes a profile configuration
func runProfileDeleteCmd(cmd *cobra.Command, args []string) {
	if cfg == nil {
		log.Error().Msg("Configuration not loaded.")
		os.Exit(1)
	}

	var profileID string

	if len(args) == 0 {
		// No ID provided, check if there's only one profile
		if len(cfg.Profiles) == 1 {
			profileID = cfg.Profiles[0].ID
			log.Info().Str("profile_id", profileID).Msg("Only one profile found, selecting it automatically for deletion attempt.")
		} else if len(cfg.Profiles) == 0 {
			fmt.Fprintln(os.Stderr, "Error: No profiles configured to delete.")
			os.Exit(1)
		} else {
			// Multiple profiles exist, prompt user
			fmt.Println("Multiple profiles configured.")
			var err error
			profileID, err = promptSelectProfile("Select the profile to delete:", cfg.Profiles, cfg.DefaultProfileID, "") // Use the existing helper
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error selecting profile: %v\n", err)
				os.Exit(1)
			}
			if profileID == "" { // User cancelled selection
				fmt.Println("Deletion cancelled.")
				os.Exit(0) // Exit gracefully
			}
			log.Info().Str("profile_id", profileID).Msg("Profile selected by user for deletion.")
			// The rest of the function will proceed with the selected profileID
		}
	} else {
		// ID provided
		profileID = args[0]
	}

	var index = -1
	var profileName string

	// Find the profile
	for i, p := range cfg.Profiles {
		if p.ID == profileID {
			index = i
			profileName = p.Name
			break
		}
	}

	if index == -1 {
		fmt.Fprintf(os.Stderr, "Error: Profile with ID '%s' not found.\n", profileID)
		os.Exit(1)
	}

	// Prevent deleting the default profile
	if cfg.DefaultProfileID == profileID {
		fmt.Fprintf(os.Stderr, "Error: Cannot delete the default profile (ID: %s).\n", profileID)
		fmt.Fprintln(os.Stderr, "Please set a different default profile first using 'rurl config set-default <other-profile-id>'.")
		os.Exit(1)
	}

	// Check if any rules reference this profile ID
	var referencingRules []string
	for _, r := range cfg.Rules {
		if r.ProfileID == profileID {
			referencingRules = append(referencingRules, r.Name)
		}
	}
	if len(referencingRules) > 0 {
		fmt.Fprintf(os.Stderr, "Error: Cannot delete profile '%s' (ID: %s) because it is referenced by the following rule(s):\n", profileName, profileID)
		for _, ruleName := range referencingRules {
			fmt.Fprintf(os.Stderr, "  - %s\n", ruleName)
		}
		fmt.Fprintln(os.Stderr, "Please edit or delete the rule(s) first.")
		os.Exit(1)
	}

	// Confirm deletion
	confirm := promptString(fmt.Sprintf("Are you sure you want to delete profile '%s' (ID: %s)? (yes/no)", profileName, profileID), "no")
	if !strings.EqualFold(confirm, "yes") {
		fmt.Println("Deletion cancelled.")
		return
	}

	// Remove the profile
	cfg.Profiles = append(cfg.Profiles[:index], cfg.Profiles[index+1:]...)

	// Save the config
	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nProfile '%s' (ID: %s) deleted successfully.\n", profileName, profileID)
}
