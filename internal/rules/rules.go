package rules

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/rs/zerolog/log"
)

// MatchResult holds the outcome of applying rules.
// If a rule matched, Rule will be non-nil.
// If no rule matched, ProfileID will be the DefaultProfileID.
type MatchResult struct {
	Rule      *config.Rule // Pointer to the matched rule (nil if no match)
	ProfileID string       // The ID of the profile to use
	Incognito bool         // Whether to launch in incognito mode
}

// getMatchString returns the appropriate part of the URL to match against based on the rule's scope
func getMatchString(parsedURL *url.URL, scope config.RuleScope) string {
	var matchStr string
	switch scope {
	case config.ScopeDomain:
		matchStr = parsedURL.Hostname() // Just the hostname part (e.g., "images.google.com")
	case config.ScopePath:
		matchStr = parsedURL.Path // Just the path part (e.g., "/search/images")
	default: // config.ScopeURL
		// For URL scope, include host, path, and query, but only include scheme if it exists
		if parsedURL.Scheme != "" {
			matchStr = fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path)
		} else {
			matchStr = fmt.Sprintf("%s%s", parsedURL.Host, parsedURL.Path)
		}
		if parsedURL.RawQuery != "" {
			matchStr = fmt.Sprintf("%s?%s", matchStr, parsedURL.RawQuery)
		}
	}
	log.Debug().
		Str("scope", string(scope)).
		Str("match_string", matchStr).
		Str("hostname", parsedURL.Hostname()).
		Str("host", parsedURL.Host).
		Str("scheme", parsedURL.Scheme).
		Str("path", parsedURL.Path).
		Msg("Generated match string")
	return matchStr
}

// ApplyRules iterates through the configured rules and returns the first match.
// Rules are checked in order of pattern length (descending) to prioritize specificity.
// If no rules match, it returns the default profile.
func ApplyRules(cfg *config.Config, inputURL string) (MatchResult, error) {
	if cfg == nil {
		return MatchResult{}, fmt.Errorf("configuration is nil")
	}

	// Parse the URL once for all rules
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return MatchResult{}, fmt.Errorf("failed to parse URL '%s': %w", inputURL, err)
	}

	// If there's no scheme and the path contains a domain-like string, treat it as the host
	if parsedURL.Scheme == "" && parsedURL.Host == "" && parsedURL.Path != "" {
		// Try parsing with a dummy scheme
		if tmpURL, err := url.Parse("http://" + inputURL); err == nil {
			parsedURL.Host = tmpURL.Host
			parsedURL.Path = tmpURL.Path
		}
	}

	log.Debug().
		Str("input_url", inputURL).
		Str("parsed_scheme", parsedURL.Scheme).
		Str("parsed_host", parsedURL.Host).
		Str("parsed_hostname", parsedURL.Hostname()).
		Str("parsed_path", parsedURL.Path).
		Msg("URL parsing results")

	// Create a copy of the rules to avoid modifying the original config order
	rulesToSort := make([]config.Rule, len(cfg.Rules))
	copy(rulesToSort, cfg.Rules)

	// Sort rules by pattern length descending (longer patterns first)
	sort.Slice(rulesToSort, func(i, j int) bool {
		return len(rulesToSort[i].Pattern) > len(rulesToSort[j].Pattern)
	})

	log.Debug().Str("url", inputURL).Int("rule_count", len(rulesToSort)).Msg("Applying rules (sorted by pattern length desc)")

	for i := range rulesToSort {
		rule := &rulesToSort[i] // Use pointer to the rule in the sorted slice
		log.Debug().
			Str("rule_name", rule.Name).
			Str("pattern", rule.Pattern).
			Int("pattern_len", len(rule.Pattern)).
			Str("scope", string(rule.Scope)).
			Msg("Checking rule")

		// Compile the regex pattern for the rule
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			log.Error().Err(err).Str("rule_name", rule.Name).Str("pattern", rule.Pattern).Msg("Invalid regex pattern in rule")
			// Skip this rule, but don't stop processing others
			continue
		}

		// Get the appropriate part of the URL to match against based on the rule's scope
		matchString := getMatchString(parsedURL, rule.Scope)

		// Check if the URL matches the pattern
		matches := re.MatchString(matchString)
		log.Debug().
			Str("rule_name", rule.Name).
			Str("pattern", rule.Pattern).
			Str("match_string", matchString).
			Bool("matches", matches).
			Msg("Rule match attempt")

		if matches {
			log.Info().
				Str("url", inputURL).
				Str("rule_name", rule.Name).
				Str("profile_id", rule.ProfileID).
				Bool("incognito", rule.Incognito).
				Str("scope", string(rule.Scope)).
				Str("matched_part", matchString).
				Msg("Rule matched")

			// Ensure the profile ID specified by the rule exists
			_, profileErr := cfg.FindProfileByID(rule.ProfileID)
			if profileErr != nil {
				log.Error().Err(profileErr).Str("rule_name", rule.Name).Str("profile_id", rule.ProfileID).Msg("Profile specified in matched rule not found")
				// Fallback to default? Or return error? Returning error seems safer.
				return MatchResult{}, fmt.Errorf("profile '%s' specified in rule '%s' not found", rule.ProfileID, rule.Name)
			}

			// Return the match result
			return MatchResult{
				Rule:      rule,
				ProfileID: rule.ProfileID,
				Incognito: rule.Incognito,
			}, nil
		}
	}

	// No rules matched, use the default profile
	log.Debug().Str("url", inputURL).Msg("No rules matched")
	if cfg.DefaultProfileID == "" {
		log.Error().Msg("No rules matched and no default profile set.")
		return MatchResult{}, fmt.Errorf("no matching rule found and no default profile is configured")
	}

	// Ensure the default profile ID actually exists
	_, err = cfg.FindProfileByID(cfg.DefaultProfileID)
	if err != nil {
		log.Error().Err(err).Str("default_profile_id", cfg.DefaultProfileID).Msg("Default profile specified in config not found")
		return MatchResult{}, fmt.Errorf("default profile '%s' not found", cfg.DefaultProfileID)
	}

	log.Info().Str("url", inputURL).Str("profile_id", cfg.DefaultProfileID).Msg("Using default profile")
	return MatchResult{
		Rule:      nil, // No specific rule matched
		ProfileID: cfg.DefaultProfileID,
		Incognito: false, // Default is not incognito
	}, nil
}
