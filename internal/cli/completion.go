package cli

import (
	"strings"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// Function type for loading config that can be mocked in tests
type loadConfigFunc func() *config.Config

// Default implementation of loadConfigForCompletion
var loadConfigForCompletion loadConfigFunc = func() *config.Config {
	// Ensure logging is initialized minimally if not already done
	// logging.InitLogging(false) // Or determine debug status if possible

	// Use the cfgFile variable from root.go if set, otherwise defaults
	loadedCfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		// Log the error but don't exit, completion might still work partially or
		// the user might be completing a command before config exists.
		log.Debug().Err(err).Msg("Failed to load config during completion")
		return nil // Return nil, completers should handle this
	}
	return loadedCfg
}

// completeRuleNames provides completion for rule names.
func completeRuleNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg := loadConfigForCompletion()
	if cfg == nil {
		return nil, cobra.ShellCompDirectiveError // Indicate error if config failed
	}

	var names []string
	for _, rule := range cfg.Rules {
		if strings.HasPrefix(rule.Name, toComplete) {
			names = append(names, rule.Name)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeProfileIDs provides completion for profile IDs.
func completeProfileIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg := loadConfigForCompletion()
	if cfg == nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var ids []string
	for _, profile := range cfg.Profiles {
		if strings.HasPrefix(profile.ID, toComplete) {
			ids = append(ids, profile.ID)
		}
	}
	return ids, cobra.ShellCompDirectiveNoFileComp
}

// completeBrowserIDs provides completion for browser IDs.
func completeBrowserIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg := loadConfigForCompletion()
	if cfg == nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var ids []string
	for _, browser := range cfg.Browsers {
		if strings.HasPrefix(browser.BrowserID, toComplete) {
			// Ensure uniqueness if multiple browsers share an ID (shouldn't happen with current detection)
			found := false
			for _, existing := range ids {
				if existing == browser.BrowserID {
					found = true
					break
				}
			}
			if !found {
				ids = append(ids, browser.BrowserID)
			}
		}
	}
	return ids, cobra.ShellCompDirectiveNoFileComp
}
