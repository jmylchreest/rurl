package cli

import (
	"fmt"

	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/choose"
	"github.com/jmylchreest/rurl/internal/config"
	"github.com/spf13/cobra"
)

// Define a constant for the special default rule name
const defaultRuleName = "Default"

// AddRuleCommands defines and adds the rule management commands to the parent config command.
func AddRuleCommands(configCmd *cobra.Command) {
	ruleCmd := &cobra.Command{
		Use:   "rule",
		Short: "Manage URL routing rules",
		Long:  `Add, edit, delete, and list rules for routing URLs to specific browser profiles.`,
	}

	// --- Rule Subcommands ---
	ruleListCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured rules",
		Long:  `Display all configured rules.`,
		RunE:  runRuleListCmd,
	}

	ruleAddCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new rule",
		Long:  `Interactively add a new URL routing rule.`,
		RunE:  runRuleAddCmd,
	}

	ruleEditCmd := &cobra.Command{
		Use:               "edit [rule-name]",
		Short:             "Edit an existing rule",
		Long:              `Interactively edit an existing rule. If only one rule exists, it will be selected automatically if no name is provided.`,
		Args:              cobra.MaximumNArgs(1),
		RunE:              runRuleEditCmd,
		ValidArgsFunction: completeRuleNames,
	}

	ruleDeleteCmd := &cobra.Command{
		Use:               "delete [rule-name]",
		Short:             "Delete an existing rule",
		Long:              `Delete an existing rule. If only one exists, it will be selected automatically if no name is provided (confirmation still required).`,
		Args:              cobra.MaximumNArgs(1),
		RunE:              runRuleDeleteCmd,
		ValidArgsFunction: completeRuleNames,
	}

	ruleCmd.AddCommand(ruleListCmd)
	ruleCmd.AddCommand(ruleAddCmd)
	ruleCmd.AddCommand(ruleEditCmd)
	ruleCmd.AddCommand(ruleDeleteCmd)

	// Add the main rule command to the config command
	configCmd.AddCommand(ruleCmd)
}

// --- Run Functions for Rules ---

func runRuleListCmd(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Rules) == 0 {
		fmt.Println("No rules configured. Run 'rurl config rule add' to add a rule.")
		return nil
	}

	printRuleList(cfg)
	return nil
}

func getProfileNote(profile config.Profile, cfg *config.Config, isDefault bool) string {
	browser, err := cfg.FindBrowserByID(profile.BrowserID)
	browserName := profile.BrowserID
	if err == nil {
		browserName = browser.Name
	}
	note := fmt.Sprintf("Browser: %s, Profile Dir: %s", browserName, profile.ProfileDir)
	if isDefault {
		note += " [DEFAULT]"
	}
	return note
}

func getRuleNote(rule config.Rule, cfg *config.Config) string {
	profile, err := cfg.FindProfileByID(rule.ProfileID)
	profileDesc := rule.ProfileID
	if err == nil {
		profileDesc = profile.Name
	}
	return fmt.Sprintf("Pattern: %s (%s), Profile: %s, Scope: %s",
		rule.Pattern,
		rule.Scope,
		profileDesc,
		rule.Scope)
}

func runRuleAddCmd(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	p := prompt.New()
	name, err := p.Ask("Rule name:").Input("")
	if err != nil {
		return fmt.Errorf("failed to get rule name: %w", err)
	}

	pattern, err := p.Ask("URL pattern:").Input("")
	if err != nil {
		return fmt.Errorf("failed to get URL pattern: %w", err)
	}

	// Create scope choices
	scopeChoices := []choose.Choice{
		{Text: string(config.ScopeURL), Note: "Match against the entire URL"},
		{Text: string(config.ScopeDomain), Note: "Match against the domain part only"},
		{Text: string(config.ScopePath), Note: "Match against the path part only"},
	}

	scope, err := p.Ask("Select scope:").AdvancedChoose(scopeChoices)
	if err != nil {
		return fmt.Errorf("failed to select scope: %w", err)
	}

	// Create choices for profiles
	profileChoices := make([]choose.Choice, 0, len(cfg.Profiles))
	for _, profile := range cfg.Profiles {
		isDefault := profile.ID == cfg.DefaultProfileID
		browser, _ := cfg.FindBrowserByID(profile.BrowserID)
		browserName := profile.BrowserID
		if browser != nil {
			browserName = browser.Name
		}
		note := fmt.Sprintf("Name: %s, Browser: %s", profile.Name, browserName)
		if isDefault {
			note += " [DEFAULT]"
		}
		profileChoices = append(profileChoices, choose.Choice{
			Text: profile.ID,
			Note: note,
		})
	}

	profileID, err := p.Ask("Select profile:").AdvancedChoose(profileChoices)
	if err != nil {
		return fmt.Errorf("failed to select profile: %w", err)
	}

	rule := config.Rule{
		Name:      name,
		Pattern:   pattern,
		ProfileID: profileID,
		Scope:     config.RuleScope(scope),
	}

	cfg.Rules = append(cfg.Rules, rule)
	if err := config.SaveConfig(cfg, ""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

func runRuleEditCmd(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	p := prompt.New()

	// Create choices for rules
	ruleChoices := make([]choose.Choice, 0, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		ruleChoices = append(ruleChoices, choose.Choice{
			Text: rule.Name,
			Note: getRuleNote(rule, cfg),
		})
	}

	ruleName, err := p.Ask("Select rule to edit:").AdvancedChoose(ruleChoices)
	if err != nil {
		return fmt.Errorf("failed to select rule: %w", err)
	}

	var ruleIndex int
	var currentRule config.Rule
	for i, rule := range cfg.Rules {
		if rule.Name == ruleName {
			ruleIndex = i
			currentRule = rule
			break
		}
	}

	pattern, err := p.Ask("URL pattern:").Input(currentRule.Pattern)
	if err != nil {
		return fmt.Errorf("failed to get URL pattern: %w", err)
	}

	// Create scope choices
	scopeChoices := []choose.Choice{
		{Text: string(config.ScopeURL), Note: "Match against the entire URL"},
		{Text: string(config.ScopeDomain), Note: "Match against the domain part only"},
		{Text: string(config.ScopePath), Note: "Match against the path part only"},
	}

	// Find the current scope index for default selection
	currentScopeIndex := 0
	for i, choice := range scopeChoices {
		if choice.Text == string(currentRule.Scope) {
			currentScopeIndex = i
			break
		}
	}

	scope, err := p.Ask("Select scope:").AdvancedChoose(scopeChoices, choose.WithDefaultIndex(currentScopeIndex))
	if err != nil {
		return fmt.Errorf("failed to select scope: %w", err)
	}

	// Create choices for profiles
	profileChoices := make([]choose.Choice, 0, len(cfg.Profiles))
	for _, profile := range cfg.Profiles {
		isDefault := profile.ID == cfg.DefaultProfileID
		browser, _ := cfg.FindBrowserByID(profile.BrowserID)
		browserName := profile.BrowserID
		if browser != nil {
			browserName = browser.Name
		}
		note := fmt.Sprintf("Name: %s, Browser: %s", profile.Name, browserName)
		if isDefault {
			note += " [DEFAULT]"
		}
		profileChoices = append(profileChoices, choose.Choice{
			Text: profile.ID,
			Note: note,
		})
	}

	// Find the current profile index for default selection
	currentProfileIndex := 0
	for i, profile := range cfg.Profiles {
		if profile.ID == currentRule.ProfileID {
			currentProfileIndex = i
			break
		}
	}

	profileID, err := p.Ask("Select profile:").AdvancedChoose(profileChoices, choose.WithDefaultIndex(currentProfileIndex))
	if err != nil {
		return fmt.Errorf("failed to select profile: %w", err)
	}

	cfg.Rules[ruleIndex].Pattern = pattern
	cfg.Rules[ruleIndex].ProfileID = profileID
	cfg.Rules[ruleIndex].Scope = config.RuleScope(scope)

	if err := config.SaveConfig(cfg, ""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

func runRuleDeleteCmd(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	p := prompt.New()

	// Create choices for rules
	ruleChoices := make([]choose.Choice, 0, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		ruleChoices = append(ruleChoices, choose.Choice{
			Text: rule.Name,
			Note: getRuleNote(rule, cfg),
		})
	}

	ruleName, err := p.Ask("Select rule to delete:").AdvancedChoose(ruleChoices)
	if err != nil {
		return fmt.Errorf("failed to select rule: %w", err)
	}

	var ruleIndex int
	for i, rule := range cfg.Rules {
		if rule.Name == ruleName {
			ruleIndex = i
			break
		}
	}

	cfg.Rules = append(cfg.Rules[:ruleIndex], cfg.Rules[ruleIndex+1:]...)
	if err := config.SaveConfig(cfg, ""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// Helper function to filter out the implicit default rule concept
func getUserDefinedRules(cfg *config.Config) []config.Rule {
	var userRules []config.Rule
	for _, r := range cfg.Rules {
		// This assumes the default rule is *not* stored explicitly in the Rules slice
		// If it were, we'd need: if !strings.EqualFold(r.Name, defaultRuleName) { ... }
		userRules = append(userRules, r)
	}
	return userRules
}
