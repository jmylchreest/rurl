//go:build linux

package browser

import (
	"bufio"
	// "encoding/base64" // Removed icon import
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/rs/zerolog/log"
)

// knownBrowserInfo holds information about browsers we know how to detect on Linux.
type knownBrowserInfo struct {
	name         string // User-friendly name (e.g., "Google Chrome")
	browserID    string // Stable ID (chrome, firefox, edge)
	executable   string // URI-style executable (e.g., "file://google-chrome" or "flatpak://com.google.Chrome")
	profileDir   string // Path relative to user home directory
	profileArg   string // Command line arg for profile
	incognitoArg string // Command line arg for incognito
	// iconPath     string   // Path to browser icon - REMOVED
}

// knownBrowsers contains the list of supported browsers and their configurations
var knownBrowsers = []knownBrowserInfo{
	// Google Chrome
	{
		name:         "Google Chrome",
		browserID:    "chrome",
		executable:   "file://google-chrome-stable",
		profileDir:   ".config/google-chrome",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/google-chrome.png",
	},
	{
		name:         "Google Chrome Beta",
		browserID:    "chrome-beta",
		executable:   "file://google-chrome-beta",
		profileDir:   ".config/google-chrome-beta",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/google-chrome-beta.png",
	},
	{
		name:         "Google Chrome Dev",
		browserID:    "chrome-dev",
		executable:   "file://google-chrome-unstable",
		profileDir:   ".config/google-chrome-unstable",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/google-chrome-unstable.png",
	},
	{
		name:         "Google Chrome (Flatpak)",
		browserID:    "chrome-flatpak",
		executable:   "flatpak://com.google.Chrome",
		profileDir:   ".var/app/com.google.Chrome/config/google-chrome",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
	},
	// Microsoft Edge
	{
		name:         "Microsoft Edge",
		browserID:    "edge",
		executable:   "file://microsoft-edge-stable",
		profileDir:   ".config/microsoft-edge",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--inprivate",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/microsoft-edge.png",
	},
	{
		name:         "Microsoft Edge Beta",
		browserID:    "edge-beta",
		executable:   "file://microsoft-edge-beta",
		profileDir:   ".config/microsoft-edge-beta",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--inprivate",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/microsoft-edge-beta.png",
	},
	{
		name:         "Microsoft Edge Dev",
		browserID:    "edge-dev",
		executable:   "file://microsoft-edge-dev",
		profileDir:   ".config/microsoft-edge-dev",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--inprivate",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/microsoft-edge-dev.png",
	},
	{
		name:         "Microsoft Edge Canary",
		browserID:    "edge-canary",
		executable:   "file://microsoft-edge-canary",
		profileDir:   ".config/microsoft-edge-canary",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--inprivate",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/microsoft-edge-canary.png",
	},
	{
		name:         "Microsoft Edge (Flatpak)",
		browserID:    "edge-flatpak",
		executable:   "flatpak://com.microsoft.Edge",
		profileDir:   ".var/app/com.microsoft.Edge/config/microsoft-edge",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--inprivate",
	},
	// Brave Browser
	{
		name:         "Brave",
		browserID:    "brave",
		executable:   "file://brave-browser",
		profileDir:   ".config/BraveSoftware/Brave-Browser",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/brave-browser.png",
	},
	{
		name:         "Brave Beta",
		browserID:    "brave-beta",
		executable:   "file://brave-browser-beta",
		profileDir:   ".config/BraveSoftware/Brave-Browser-Beta",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/brave-browser-beta.png",
	},
	{
		name:         "Brave Dev",
		browserID:    "brave-dev",
		executable:   "file://brave-browser-dev",
		profileDir:   ".config/BraveSoftware/Brave-Browser-Dev",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/brave-browser-dev.png",
	},
	{
		name:         "Brave (Flatpak)",
		browserID:    "brave-flatpak",
		executable:   "flatpak://com.brave.Browser",
		profileDir:   ".var/app/com.brave.Browser/config/BraveSoftware/Brave-Browser",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
	},
	// Firefox
	{
		name:         "Firefox",
		browserID:    "firefox",
		executable:   "file://firefox",
		profileDir:   ".mozilla/firefox",
		profileArg:   "-P %s",
		incognitoArg: "--private-window",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/firefox.png",
	},
	{
		name:         "Firefox Developer Edition",
		browserID:    "firefox-dev",
		executable:   "file://firefox-developer-edition",
		profileDir:   ".mozilla/firefox",
		profileArg:   "-P %s",
		incognitoArg: "--private-window",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/firefox-developer-edition.png",
	},
	{
		name:         "Firefox Beta",
		browserID:    "firefox-beta",
		executable:   "file://firefox-beta",
		profileDir:   ".mozilla/firefox",
		profileArg:   "-P %s",
		incognitoArg: "--private-window",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/firefox-beta.png",
	},
	{
		name:         "Firefox (Flatpak)",
		browserID:    "firefox-flatpak",
		executable:   "flatpak://org.mozilla.firefox",
		profileDir:   ".var/app/org.mozilla.firefox/data/mozilla/firefox",
		profileArg:   "-P %s",
		incognitoArg: "--private-window",
	},
	// Chromium
	{
		name:         "Chromium",
		browserID:    "chromium",
		executable:   "file://chromium",
		profileDir:   ".config/chromium",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/chromium.png",
	},
	// Vivaldi
	{
		name:         "Vivaldi",
		browserID:    "vivaldi",
		executable:   "file://vivaldi-stable",
		profileDir:   ".config/vivaldi",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
	},
	{
		name:         "Vivaldi Snapshot",
		browserID:    "vivaldi-snapshot",
		executable:   "file://vivaldi-snapshot",
		profileDir:   ".config/vivaldi-snapshot",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
	},
	{
		name:         "Vivaldi (Flatpak)",
		browserID:    "vivaldi-flatpak",
		executable:   "flatpak://com.vivaldi.Vivaldi",
		profileDir:   ".var/app/com.vivaldi.Vivaldi/config/vivaldi",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
	},
	// Arc Browser
	{
		name:         "Arc",
		browserID:    "arc",
		executable:   "file://arc",
		profileDir:   ".config/arc",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--incognito",
	},
	// Opera
	{
		name:         "Opera",
		browserID:    "opera",
		executable:   "file://opera",
		profileDir:   ".config/opera",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--private",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/opera.png",
	},
	{
		name:         "Opera Beta",
		browserID:    "opera-beta",
		executable:   "file://opera-beta",
		profileDir:   ".config/opera-beta",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--private",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/opera-beta.png",
	},
	{
		name:         "Opera Developer",
		browserID:    "opera-dev",
		executable:   "file://opera-developer",
		profileDir:   ".config/opera-developer",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--private",
		// iconPath:     "/usr/share/icons/hicolor/256x256/apps/opera-developer.png",
	},
	{
		name:         "Opera (Flatpak)",
		browserID:    "opera-flatpak",
		executable:   "flatpak://com.opera.Opera",
		profileDir:   ".var/app/com.opera.Opera/config/opera",
		profileArg:   "--profile-directory=%s",
		incognitoArg: "--private",
	},
	// Epiphany (GNOME Web)
	{
		name:         "Epiphany (GNOME Web)",
		browserID:    "epiphany",
		executable:   "file://epiphany",
		profileDir:   "~/.local/share/epiphany", // Regular installation path with home directory
		profileArg:   "--profile=%s",            // Will be replaced with full path
		incognitoArg: "--incognito-mode",
	},
	{
		name:         "Epiphany (Flatpak)",
		browserID:    "epiphany-flatpak",
		executable:   "flatpak://org.gnome.Epiphany",
		profileDir:   "~/.var/app/org.gnome.Epiphany/data/epiphany", // Flatpak path with home directory
		profileArg:   "--profile=%s",                                // Will be replaced with full path
		incognitoArg: "--incognito-mode",
	},
	// Falkon
	{
		name:         "Falkon",
		browserID:    "falkon",
		executable:   "file://falkon",
		profileDir:   ".config/falkon",     // Common location
		profileArg:   "--profile=%s",       // Common profile flag pattern
		incognitoArg: "--private-browsing", // Common private browsing flag
	},
	// Konqueror
	{
		name:         "Konqueror",
		browserID:    "konqueror",
		executable:   "file://konqueror",
		profileDir:   ".config/konqueror", // Might vary, using common config location
		profileArg:   "--profile %s",      // Common pattern, space separated
		incognitoArg: "--private",         // Common private flag
	},
}

// linuxDetector implements browser detection for Linux.
type linuxDetector struct{}

// NewDetector creates a new Linux-specific detector.
func NewDetector() (Detector, error) {
	return &linuxDetector{}, nil
}

// findExecutable tries to find the executable for a browser
func findExecutable(executable string) string {
	// Split the URI into scheme and path
	parts := strings.SplitN(executable, "://", 2)
	if len(parts) != 2 {
		return ""
	}
	scheme, path := parts[0], parts[1]

	switch scheme {
	case "flatpak":
		// Check if flatpak is installed
		if _, err := exec.LookPath("flatpak"); err != nil {
			return ""
		}
		// Check if the flatpak app is installed using flatpak info
		cmd := exec.Command("flatpak", "info", path)
		if err := cmd.Run(); err == nil {
			return "flatpak"
		}
		return ""

	case "file":
		// Regular executable search
		path, err := exec.LookPath(path)
		if err == nil {
			return path
		}
		return ""

	default:
		log.Warn().Str("scheme", scheme).Msg("Unknown executable scheme")
		return ""
	}
}

// DiscoverBrowsers finds installed browsers on Linux.
func (d *linuxDetector) DiscoverBrowsers() ([]config.Browser, error) {
	found := make(map[string]config.Browser) // Key: Executable Path
	for _, browserInfo := range knownBrowsers {
		// Find executable path (implementation detail)
		exePath := findExecutable(browserInfo.executable)
		if exePath == "" {
			continue // Skip if not found
		}

		// Split the executable URI to get the scheme and path
		parts := strings.SplitN(browserInfo.executable, "://", 2)
		if len(parts) != 2 {
			continue
		}
		scheme, path := parts[0], parts[1]

		// For Flatpak apps, we need to add the run command and app ID as arguments
		var fullExePath string
		if scheme == "flatpak" {
			fullExePath = fmt.Sprintf("%s run %s", exePath, path)
		} else {
			fullExePath = exePath
		}

		if _, exists := found[fullExePath]; !exists {
			// Construct browser object
			found[fullExePath] = config.Browser{
				Name:         browserInfo.name,
				BrowserID:    browserInfo.browserID,
				Executable:   fullExePath,
				ProfileArg:   browserInfo.profileArg,
				IncognitoArg: browserInfo.incognitoArg,
			}
			log.Debug().Str("name", browserInfo.name).Str("path", fullExePath).Msg("Discovered browser")
		}
	}
	// Convert map to slice
	result := make([]config.Browser, 0, len(found))
	for _, browser := range found {
		result = append(result, browser)
	}
	return result, nil
}

// discoverFirefoxProfiles reads Firefox profiles from profiles.ini
func (d *linuxDetector) discoverFirefoxProfiles(profilesPath, browserID string) ([]config.Profile, error) {
	var profiles []config.Profile

	// Check if profiles directory exists
	if _, err := os.Stat(profilesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("firefox profiles directory not found: %s", profilesPath)
	}

	// Read profiles.ini
	iniPath := filepath.Join(profilesPath, "profiles.ini")
	file, err := os.Open(iniPath)
	if err != nil {
		log.Warn().Err(err).Str("path", iniPath).Msg("Could not open profiles.ini")
		// Fall back to default profile
		return []config.Profile{{
			ID:         fmt.Sprintf("%s-default", browserID),
			Name:       "Default",
			BrowserID:  browserID,
			ProfileDir: profilesPath,
		}}, nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentProfile string
	var isRelative bool
	var profilePath string
	var profileName string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// Save previous profile if we have one
			if currentProfile != "" && profilePath != "" {
				if profileName == "" {
					profileName = profilePath
				}
				if isRelative {
					profilePath = filepath.Join(profilesPath, profilePath)
				}
				profiles = append(profiles, config.Profile{
					ID:         fmt.Sprintf("%s-%s", browserID, strings.ReplaceAll(profileName, " ", "-")),
					Name:       profileName,
					BrowserID:  browserID,
					ProfileDir: profilePath,
				})
			}

			// Start new profile section
			currentProfile = line[1 : len(line)-1]
			isRelative = false
			profilePath = ""
			profileName = ""
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Name":
			profileName = value
		case "IsRelative":
			isRelative = value == "1"
		case "Path":
			profilePath = value
		}
	}

	// Add the last profile if we have one
	if currentProfile != "" && profilePath != "" {
		if profileName == "" {
			profileName = profilePath
		}
		if isRelative {
			profilePath = filepath.Join(profilesPath, profilePath)
		}
		profiles = append(profiles, config.Profile{
			ID:         fmt.Sprintf("%s-%s", browserID, strings.ReplaceAll(profileName, " ", "-")),
			Name:       profileName,
			BrowserID:  browserID,
			ProfileDir: profilePath,
		})
	}

	if len(profiles) == 0 {
		// Fall back to default profile if no profiles found
		return []config.Profile{{
			ID:         fmt.Sprintf("%s-default", browserID),
			Name:       "Default",
			BrowserID:  browserID,
			ProfileDir: profilesPath,
		}}, nil
	}

	return profiles, nil
}

// discoverChromiumProfiles finds profiles for Chromium-based browsers
func (d *linuxDetector) discoverChromiumProfiles(profilesPath, browserID string) ([]config.Profile, error) {
	var profiles []config.Profile

	// Check if profiles directory exists
	entries, err := os.ReadDir(profilesPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If the directory doesn't exist, return a default profile
			return d.createSingleDefaultProfile(browserID, "Default"), nil
		}
		return nil, fmt.Errorf("failed to read profiles directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip known non-profile directories
		name := entry.Name()
		if name == "Crash Reports" || name == "System Profile" || name == "GrShaderCache" || name == "ShaderCache" {
			continue
		}

		// Check if it's a valid profile by looking for Preferences file
		prefsPath := filepath.Join(profilesPath, name, "Preferences")
		if _, err := os.Stat(prefsPath); err == nil {
			profile := config.Profile{
				ID:         fmt.Sprintf("%s-%s", browserID, strings.ToLower(strings.ReplaceAll(name, " ", "-"))),
				Name:       name,
				BrowserID:  browserID,
				ProfileDir: name, // Chrome-based browsers use relative profile paths
			}
			profiles = append(profiles, profile)
			log.Debug().Str("browser", browserID).Str("profile", name).Msg("Found profile")
		}
	}

	// If no profiles were found, return a default profile
	if len(profiles) == 0 {
		return d.createSingleDefaultProfile(browserID, "Default"), nil
	}

	return profiles, nil
}

// DiscoverProfiles finds profiles for a given browser on Linux.
func (d *linuxDetector) DiscoverProfiles(browser config.Browser) ([]config.Profile, error) {
	// Find the browser configuration from knownBrowsers to get the base profile directory
	var browserConfig *knownBrowserInfo
	for i := range knownBrowsers {
		if knownBrowsers[i].browserID == browser.BrowserID {
			browserConfig = &knownBrowsers[i]
			break
		}
	}

	if browserConfig == nil {
		// This shouldn't happen if DiscoverBrowsers found it, but handle defensively
		log.Warn().Str("browser_id", browser.BrowserID).Msg("Browser config not found in knownBrowsers during profile discovery")
		// Create a single default profile anyway
		return d.createSingleDefaultProfile(browser.BrowserID, "Default"), nil
	}

	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Construct the absolute path to the base profile directory for this browser
	// Replace ~ with the actual home directory and expand the path
	profileDir := strings.Replace(browserConfig.profileDir, "~", homeDir, 1)
	baseProfilesPath := filepath.Clean(profileDir) // Clean the path to remove any duplicates

	// Special handling for Epiphany
	if browser.BrowserID == "epiphany" || browser.BrowserID == "epiphany-flatpak" {
		// Ensure the profile directory exists
		if err := os.MkdirAll(baseProfilesPath, 0755); err != nil {
			log.Warn().Err(err).Str("path", baseProfilesPath).Msg("Failed to create Epiphany profile directory")
		}
		// Epiphany uses a different profile structure
		profiles := []config.Profile{{
			ID:         fmt.Sprintf("%s-default", browser.BrowserID),
			Name:       "Default",
			BrowserID:  browser.BrowserID,
			ProfileDir: baseProfilesPath, // Use the full path for Epiphany
		}}
		log.Debug().Str("profile_dir", baseProfilesPath).Msg("Created Epiphany profile with full path")
		return profiles, nil
	}

	// Special handling for Firefox-based browsers (based on profileArg format?)
	// Using BrowserID prefix is fragile if IDs change, maybe check ProfileArg?
	if strings.Contains(browser.ProfileArg, "-P") { // Heuristic for Firefox style
		profiles, err := d.discoverFirefoxProfiles(baseProfilesPath, browser.BrowserID)
		if err != nil || len(profiles) == 0 {
			// If Firefox profile discovery fails or finds no profiles, fall back to default
			return d.createSingleDefaultProfile(browser.BrowserID, "Default"), nil
		}
		return profiles, nil
	}

	// Handle Chromium-based browsers (Chrome, Edge, Brave, Vivaldi, Opera, etc.)
	// Use ProfileArg format as a heuristic
	if strings.Contains(browser.ProfileArg, "--profile-directory") {
		profiles, err := d.discoverChromiumProfiles(baseProfilesPath, browser.BrowserID)
		if err != nil || len(profiles) == 0 {
			// If Chromium profile discovery fails or finds no profiles, fall back to default
			return d.createSingleDefaultProfile(browser.BrowserID, "Default"), nil
		}
		return profiles, nil
	}

	// --- Fallback: Assume single default profile ---
	log.Debug().Str("browser_id", browser.BrowserID).Msg("No specific profile discovery logic found, creating single default profile.")
	// Use the base name of the profileDir from knownBrowsers as the ProfileDir value
	// e.g., for .config/epiphany, use "epiphany"
	// If profileDir is empty in knownBrowsers, use "Default"
	defaultProfileDir := filepath.Base(browserConfig.profileDir)
	if defaultProfileDir == "." || defaultProfileDir == "/" || defaultProfileDir == "" {
		defaultProfileDir = "Default" // Fallback if base name isn't useful
	}
	return d.createSingleDefaultProfile(browser.BrowserID, defaultProfileDir), nil
}

// createSingleDefaultProfile is a helper to generate a default profile entry.
func (d *linuxDetector) createSingleDefaultProfile(browserID, profileDirName string) []config.Profile {
	return []config.Profile{{
		ID:         browserID, // Use browser ID as the profile ID
		Name:       "Default", // User-friendly name
		BrowserID:  browserID,
		ProfileDir: profileDirName, // Use provided name (often base of config dir)
	}}
}
