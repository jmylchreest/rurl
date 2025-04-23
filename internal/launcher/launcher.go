package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/rs/zerolog/log" // Added for structured logging
)

// Launch opens the given URL in the specified browser profile with appropriate flags.
func Launch(cfg *config.Config, profileID string, targetURL string, incognito bool) error {
	profile, err := cfg.FindProfileByID(profileID)
	if err != nil {
		return fmt.Errorf("cannot launch profile: %w", err)
	}

	browser, err := cfg.GetProfileBrowser(profile)
	if err != nil {
		return fmt.Errorf("cannot find browser '%s' for profile '%s': %w", profile.BrowserID, profile.Name, err)
	}

	// Start with empty args
	args := []string{}

	// For Flatpak apps, we need to split the command into executable and arguments
	var cmd *exec.Cmd
	if strings.HasPrefix(browser.Executable, "flatpak run ") {
		// Split the command into parts
		parts := strings.Split(browser.Executable, " ")
		cmd = exec.Command(parts[0], parts[1:]...)
	} else {
		cmd = exec.Command(browser.Executable)
	}

	// 1. Add profile argument first (as a single combined argument if possible)
	if browser.ProfileArg != "" && profile.ProfileDir != "" {
		// Check if the ProfileArg contains "%s" to replace
		if strings.Contains(browser.ProfileArg, "%s") {
			// Replace %s with the actual profile directory identifier
			args = append(args, strings.Replace(browser.ProfileArg, "%s", profile.ProfileDir, 1))
		} else {
			// If ProfileArg doesn't contain %s, assume it's a simple flag like --use-this-profile
			args = append(args, browser.ProfileArg)
		}
	}

	// 2. Add incognito argument
	if incognito && browser.IncognitoArg != "" {
		args = append(args, browser.IncognitoArg)
	}

	// 3. Add Wayland specific flags for Chromium-based browsers only
	if runtime.GOOS == "linux" && os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		// Check if this is a Chromium-based browser by looking at the profile argument format
		if strings.Contains(browser.ProfileArg, "--profile-directory") {
			log.Debug().Str("XDG_SESSION_TYPE", os.Getenv("XDG_SESSION_TYPE")).Msg("Wayland session detected, adding Wayland flags for Chromium-based browser")
			args = append(args, "--enable-features=UseOzonePlatform")
			args = append(args, "--ozone-platform=wayland")
		} else {
			log.Debug().Str("browser", browser.Name).Msg("Wayland session detected, but skipping Wayland flags for non-Chromium browser")
		}
	} else if runtime.GOOS == "linux" {
		log.Debug().Str("XDG_SESSION_TYPE", os.Getenv("XDG_SESSION_TYPE")).Msg("Linux detected, but not Wayland session, skipping Wayland flags")
	}

	// 4. Add the target URL LAST
	args = append(args, targetURL)

	// Set the command arguments
	cmd.Args = append(cmd.Args, args...)

	// Debug logging for the exact command and arguments
	log.Debug().
		Str("browser", browser.Name).
		Str("browser_id", browser.BrowserID).
		Str("executable", cmd.Path).
		Interface("args", cmd.Args).
		Str("profile_dir", profile.ProfileDir).
		Str("profile_arg", browser.ProfileArg).
		Msg("Preparing to launch browser")

	// Run the command asynchronously
	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Str("command", cmd.Path).Interface("args", cmd.Args).Msg("Failed to start browser process")
		return fmt.Errorf("failed to start browser process %s with args %v: %w", cmd.Path, cmd.Args, err)
	}

	// Release the process. We don't wait for the browser to close.
	if err := cmd.Process.Release(); err != nil {
		log.Warn().Err(err).Msg("Failed to release browser process")
	}

	return nil
}
