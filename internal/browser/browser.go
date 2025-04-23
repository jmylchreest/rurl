package browser

import (
	"fmt"
	// "os" // No longer needed here
	// "text/tabwriter" // No longer needed here

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/rs/zerolog/log"
)

// Detector defines the interface for OS-specific browser detection.
type Detector interface {
	DiscoverBrowsers() ([]config.Browser, error)
	DiscoverProfiles(browser config.Browser) ([]config.Profile, error)
}

// DetectAll orchestrates the detection across all browsers found.
// It returns the combined list of discovered browsers and profiles.
func DetectAll() ([]config.Browser, []config.Profile, error) {
	log.Debug().Msg("Starting browser and profile detection...")
	detector, err := NewDetector() // Gets OS-specific implementation
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create browser detector: %w", err)
	}

	discoveredBrowsers, err := detector.DiscoverBrowsers()
	if err != nil {
		log.Warn().Err(err).Msg("Failed during browser discovery (results may be incomplete)")
		// Continue to profile discovery even if browser discovery fails partially
	}

	var allDiscoveredProfiles []config.Profile
	for _, b := range discoveredBrowsers {
		discoveredProfiles, err := detector.DiscoverProfiles(b)
		if err != nil {
			log.Warn().Err(err).Str("browser_id", b.BrowserID).Msg("Failed to discover profiles for browser")
			continue // Skip profiles for this browser on error
		}
		allDiscoveredProfiles = append(allDiscoveredProfiles, discoveredProfiles...)
	}

	log.Debug().Int("browser_count", len(discoveredBrowsers)).Int("profile_count", len(allDiscoveredProfiles)).Msg("Detection finished")
	return discoveredBrowsers, allDiscoveredProfiles, nil // Return nil error even if some discoveries failed partially
}

/*
// DetectAndSaveBrowsers orchestrates the detection and optional saving.
// THIS FUNCTION IS DEPRECATED - Logic moved to cli.runDetectBrowsersCmd
func DetectAndSaveBrowsers(cfg *config.Config, cfgFile string, save bool) error {
	log.Info().Msg("Detecting browsers and profiles...")
	detector, err := NewDetector() // Gets OS-specific implementation
	if err != nil {
		return fmt.Errorf("failed to create browser detector: %w", err)
	}

	discoveredBrowsers, err := detector.DiscoverBrowsers()
	if err != nil {
		log.Warn().Err(err).Msg("Failed during browser discovery")
		// Continue to profile discovery even if browser discovery fails partially?
		// For now, let's proceed but the browser list might be incomplete.
	}

	var allDiscoveredProfiles []config.Profile
	for _, b := range discoveredBrowsers {
		discoveredProfiles, err := detector.DiscoverProfiles(b)
		if err != nil {
			log.Warn().Err(err).Str("browser_id", b.BrowserID).Msg("Failed to discover profiles for browser")
			continue
		}
		allDiscoveredProfiles = append(allDiscoveredProfiles, discoveredProfiles...)
	}

	log.Info().Int("count", len(discoveredBrowsers)).Msg("Discovered browsers")
	log.Info().Int("count", len(allDiscoveredProfiles)).Msg("Discovered profiles")

	if save {
		// Overwrite existing browsers and profiles in the config
		cfg.Browsers = discoveredBrowsers
		cfg.Profiles = allDiscoveredProfiles

		// Assign a default profile if none is set and profiles were found
		if cfg.DefaultProfileID == "" && len(cfg.Profiles) > 0 {
			cfg.DefaultProfileID = cfg.Profiles[0].ID
			log.Info().Str("profile_id", cfg.DefaultProfileID).Msg("Setting first discovered profile as default")
		}

		if err := config.SaveConfig(cfg, cfgFile); err != nil {
			return fmt.Errorf("failed to save updated configuration: %w", err)
		}
		fmt.Println("Detected browsers and profiles saved to configuration file.")
	} else {
		// Print detected browsers
		fmt.Println("\n--- Detected Browsers ---")
		wBrowsers := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(wBrowsers, "BrowserID\tName\tExecutable Path\tProfile Arg\tIncognito Arg")
		fmt.Fprintln(wBrowsers, "---------\t----\t--------------- ----\t------------\t--------------")
		if len(discoveredBrowsers) == 0 {
			fmt.Fprintln(wBrowsers, "(No browsers detected)")
		} else {
			for _, b := range discoveredBrowsers {
				fmt.Fprintf(wBrowsers, "%s\t%s\t%s\t%s\t%s\n",
					b.BrowserID,
					b.Name,
					b.Executable,
					b.ProfileArg,
					b.IncognitoArg,
				)
			}
		}
		wBrowsers.Flush()

		// Print detected profiles
		fmt.Println("\n--- Detected Profiles ---")
		wProfiles := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(wProfiles, "ID\tName\tBrowser ID\tProfile Dir")
		fmt.Fprintln(wProfiles, "--\t----	----------\t------------")
		if len(allDiscoveredProfiles) == 0 {
			fmt.Fprintln(wProfiles, "(No profiles detected)")
		} else {
			for _, p := range allDiscoveredProfiles {
				fmt.Fprintf(wProfiles, "%s\t%s\t%s\t%s\n",
					p.ID,
					p.Name,
					p.BrowserID,
					p.ProfileDir,
				)
			}
		}
		wProfiles.Flush()

		fmt.Println("\nRun with --save to update the configuration file.")
	}

	return nil
}
*/
