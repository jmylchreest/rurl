//go:build darwin

package browser

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/rs/zerolog/log"
)

// darwinDetector implements browser detection for macOS.
type darwinDetector struct{}

// NewDetector creates a new macOS-specific detector.
func NewDetector() (Detector, error) {
	return &darwinDetector{}, nil
}

// knownBrowserInfo holds information about browsers we know how to detect on macOS.
type knownBrowserInfo struct {
	name         string // User-friendly name (e.g., "Google Chrome")
	browserID    string // Stable ID (chrome, firefox, edge)
	executable   string // URI-style executable (e.g., "file://Google Chrome.app" or "bundle://com.google.Chrome")
	profileDir   string // Path relative to ~/Library/Application Support
	profileArg   string // Command line arg for profile
	incognitoArg string // Command line arg for incognito
}

// knownBrowsers contains the list of supported browsers and their configurations
var knownBrowsers = []knownBrowserInfo{
	// Google Chrome
	{
		name:         "Google Chrome",
		browserID:    "chrome",
		executable:   "bundle://com.google.Chrome",
		profileDir:   "Google/Chrome",
		profileArg:   "--profile-directory=",
		incognitoArg: "--incognito",
	},
	{
		name:         "Google Chrome Beta",
		browserID:    "chrome-beta",
		executable:   "bundle://com.google.Chrome.beta",
		profileDir:   "Google/Chrome Beta",
		profileArg:   "--profile-directory=",
		incognitoArg: "--incognito",
	},
	{
		name:         "Google Chrome Dev",
		browserID:    "chrome-dev",
		executable:   "bundle://com.google.Chrome.dev",
		profileDir:   "Google/Chrome Dev",
		profileArg:   "--profile-directory=",
		incognitoArg: "--incognito",
	},
	{
		name:         "Google Chrome Canary",
		browserID:    "chrome-canary",
		executable:   "bundle://com.google.Chrome.canary",
		profileDir:   "Google/Chrome Canary",
		profileArg:   "--profile-directory=",
		incognitoArg: "--incognito",
	},
	// Microsoft Edge
	{
		name:         "Microsoft Edge",
		browserID:    "edge",
		executable:   "bundle://com.microsoft.edgemac",
		profileDir:   "Microsoft Edge",
		profileArg:   "--profile-directory=",
		incognitoArg: "--inprivate",
	},
	{
		name:         "Microsoft Edge Beta",
		browserID:    "edge-beta",
		executable:   "bundle://com.microsoft.edgemac.Beta",
		profileDir:   "Microsoft Edge Beta",
		profileArg:   "--profile-directory=",
		incognitoArg: "--inprivate",
	},
	{
		name:         "Microsoft Edge Dev",
		browserID:    "edge-dev",
		executable:   "bundle://com.microsoft.edgemac.Dev",
		profileDir:   "Microsoft Edge Dev",
		profileArg:   "--profile-directory=",
		incognitoArg: "--inprivate",
	},
	{
		name:         "Microsoft Edge Canary",
		browserID:    "edge-canary",
		executable:   "bundle://com.microsoft.edgemac.Canary",
		profileDir:   "Microsoft Edge Canary",
		profileArg:   "--profile-directory=",
		incognitoArg: "--inprivate",
	},
	// Firefox
	{
		name:         "Firefox",
		browserID:    "firefox",
		executable:   "bundle://org.mozilla.firefox",
		profileDir:   "Firefox",
		profileArg:   "-P",
		incognitoArg: "--private-window",
	},
	{
		name:         "Firefox Developer Edition",
		browserID:    "firefox-dev",
		executable:   "bundle://org.mozilla.firefoxdeveloperedition",
		profileDir:   "Firefox Developer Edition",
		profileArg:   "-P",
		incognitoArg: "--private-window",
	},
	{
		name:         "Firefox Nightly",
		browserID:    "firefox-nightly",
		executable:   "bundle://org.mozilla.nightly",
		profileDir:   "Firefox Nightly",
		profileArg:   "-P",
		incognitoArg: "--private-window",
	},
	// Brave
	{
		name:         "Brave Browser",
		browserID:    "brave",
		executable:   "bundle://com.brave.Browser",
		profileDir:   "BraveSoftware/Brave-Browser",
		profileArg:   "--profile-directory=",
		incognitoArg: "--incognito",
	},
	// Vivaldi
	{
		name:         "Vivaldi",
		browserID:    "vivaldi",
		executable:   "bundle://com.vivaldi.Vivaldi",
		profileDir:   "Vivaldi",
		profileArg:   "--profile-directory=",
		incognitoArg: "--incognito",
	},
	// Arc
	{
		name:         "Arc",
		browserID:    "arc",
		executable:   "bundle://company.thebrowser.Browser",
		profileDir:   "Arc",
		profileArg:   "",
		incognitoArg: "",
	},
	// Safari
	{
		name:         "Safari",
		browserID:    "safari",
		executable:   "bundle://com.apple.Safari",
		profileDir:   "Safari",
		profileArg:   "",
		incognitoArg: "--private",
	},
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
	case "file":
		// Search in common locations
		searchPaths := []string{
			"/Applications",
			filepath.Join(os.Getenv("HOME"), "Applications"),
		}

		for _, base := range searchPaths {
			appPath := filepath.Join(base, path)
			if _, err := os.Stat(appPath); err == nil {
				// Get the actual executable path within the .app bundle
				exePath := filepath.Join(appPath, "Contents", "MacOS", strings.TrimSuffix(path, ".app"))
				if _, err := os.Stat(exePath); err == nil {
					return exePath
				}
			}
		}

	case "bundle":
		// Check if the bundle is installed using mdfind
		cmd := exec.Command("mdfind", "kMDItemCFBundleIdentifier =="+path)
		if output, err := cmd.Output(); err == nil {
			appPath := strings.TrimSpace(string(output))
			if appPath != "" {
				// Get the actual executable path within the .app bundle
				exePath := filepath.Join(appPath, "Contents", "MacOS", filepath.Base(appPath))
				if _, err := os.Stat(exePath); err == nil {
					return exePath
				}
			}
		}

	default:
		log.Warn().Str("scheme", scheme).Msg("Unknown executable scheme")
	}

	return ""
}

// DiscoverBrowsers finds installed browsers on macOS.
func (d *darwinDetector) DiscoverBrowsers() ([]config.Browser, error) {
	found := make(map[string]config.Browser) // Key: Executable Path

	for _, browserInfo := range knownBrowsers {
		// Find executable path
		exePath := findExecutable(browserInfo.executable)
		if exePath == "" {
			continue // Skip if not found
		}

		if _, exists := found[exePath]; !exists {
			// Construct browser object
			found[exePath] = config.Browser{
				Name:         browserInfo.name,
				BrowserID:    browserInfo.browserID,
				Executable:   exePath,
				ProfileArg:   browserInfo.profileArg,
				IncognitoArg: browserInfo.incognitoArg,
			}
			log.Debug().Str("name", browserInfo.name).Str("path", exePath).Msg("Discovered browser")
		}
	}

	// Convert map to slice
	result := make([]config.Browser, 0, len(found))
	for _, browser := range found {
		result = append(result, browser)
	}

	return result, nil
}

// DiscoverProfiles finds profiles for a given browser on macOS.
func (d *darwinDetector) DiscoverProfiles(browser config.Browser) ([]config.Profile, error) {
	log.Debug().Str("browser_id", browser.BrowserID).Str("browser_name", browser.Name).Msg("Discovering macOS profiles...")
	profiles := []config.Profile{}

	var info *knownBrowserInfo
	for i := range knownBrowsers {
		// Match using BrowserID and Name for robustness
		if knownBrowsers[i].browserID == browser.BrowserID && knownBrowsers[i].name == browser.Name {
			info = &knownBrowsers[i]
			break
		}
	}

	if info == nil || info.profileDir == "" {
		log.Warn().Str("browser_id", browser.BrowserID).Str("browser_name", browser.Name).Msg("No profile discovery info known for this browser")
		// Return a single default profile representation
		profiles = append(profiles, createSingleDefaultProfile(browser.BrowserID, "default"))
		return profiles, nil
	}

	appSupportPath, err := getAppSupportPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get Application Support path: %w", err)
	}
	profileBaseDir := filepath.Join(appSupportPath, info.profileDir)

	if info.profileArg == "--profile-directory=" {
		log.Debug().Str("path", profileBaseDir).Msg("Discovering Chromium profiles")
		// --- Chromium-based Profile Discovery (User Data directory) ---
		foundProfiles, err := discoverChromiumProfiles(profileBaseDir, browser.BrowserID)
		if err != nil {
			log.Warn().Err(err).Str("path", profileBaseDir).Msg("Failed to discover Chromium profiles")
			// Fallback to default profile
			profiles = append(profiles, createSingleDefaultProfile(browser.BrowserID, "Default"))
		} else {
			profiles = foundProfiles
		}
	} else {
		// Browsers with no known profile method (Safari, Arc)
		log.Debug().Msg("Browser uses unknown or no profile discovery method, creating default profile")
		profiles = append(profiles, createSingleDefaultProfile(browser.BrowserID, "default"))
	}

	log.Debug().Int("count", len(profiles)).Str("browser_id", browser.BrowserID).Msg("Finished macOS profile discovery")
	return profiles, nil
}

// --- Helper Functions for Profile Discovery (Similar to Linux, adapted for macOS paths) ---

// discoverChromiumProfiles scans for profile directories like Default, Profile 1, etc.
func discoverChromiumProfiles(profileBaseDir, browserID string) ([]config.Profile, error) {
	profiles := []config.Profile{}
	entries, err := os.ReadDir(profileBaseDir)
	if err != nil {
		// Don't treat as fatal, maybe the browser exists but has no profiles yet
		return nil, fmt.Errorf("could not read profile directory '%s': %w", profileBaseDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirName := entry.Name()
			if dirName == "Default" || strings.HasPrefix(dirName, "Profile ") {
				// Basic check for a common file to ensure it's likely a valid profile
				if _, err := os.Stat(filepath.Join(profileBaseDir, dirName, "Preferences")); err == nil {
					profileID := fmt.Sprintf("%s-%s", browserID, strings.ToLower(strings.ReplaceAll(dirName, " ", "")))
					profileName := fmt.Sprintf("%s (%s)", browserID, dirName)
					profiles = append(profiles, config.Profile{
						ID:         profileID,
						Name:       profileName,
						BrowserID:  browserID,
						ProfileDir: dirName, // Use the directory name for --profile-directory flag
					})
				}
			}
		}
	}

	if len(profiles) == 0 {
		log.Warn().Str("path", profileBaseDir).Msg("No Chromium profiles found, creating default")
		profiles = append(profiles, createSingleDefaultProfile(browserID, "Default"))
	}

	return profiles, nil
}

// createSingleDefaultProfile creates a default profile entry when detection fails or isn't applicable.
func createSingleDefaultProfile(browserID, profileDirName string) config.Profile {
	profileID := fmt.Sprintf("%s-%s", browserID, strings.ToLower(profileDirName))
	profileName := fmt.Sprintf("%s (%s)", browserID, profileDirName)
	return config.Profile{
		ID:         profileID,
		Name:       profileName,
		BrowserID:  browserID,
		ProfileDir: profileDirName,
	}
}

// getAppSupportPath returns the user's Application Support directory path.
func getAppSupportPath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("cannot get current user: %w", err)
	}
	if usr.HomeDir == "" {
		return "", fmt.Errorf("cannot get home directory")
	}
	return filepath.Join(usr.HomeDir, "Library", "Application Support"), nil
}
