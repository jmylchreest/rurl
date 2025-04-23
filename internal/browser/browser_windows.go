//go:build windows

package browser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows/registry"
)

// windowsDetector implements browser detection for Windows.
type windowsDetector struct{}

// NewDetector creates a new Windows-specific detector.
func NewDetector() (Detector, error) {
	return &windowsDetector{}, nil
}

// knownBrowserInfo holds information about browsers we know how to detect on Windows.
type knownBrowserInfo struct {
	name         string // User-friendly name (e.g., "Google Chrome")
	browserID    string // Stable ID (chrome, firefox, edge)
	executable   string // URI-style executable (e.g., "file://chrome.exe")
	appDataPath  string // Path relative to %LOCALAPPDATA% or %APPDATA%
	profileArg   string // Command line arg for profile
	incognitoArg string // Command line arg for incognito
	firefoxIni   bool   // true if it uses Firefox profiles.ini
}

// knownBrowsers contains the list of supported browsers and their configurations
var knownBrowsers = []knownBrowserInfo{
	// Google Chrome
	{
		name:         "Google Chrome",
		browserID:    "chrome",
		executable:   "file://chrome.exe",
		appDataPath:  `Google\Chrome\User Data`,
		profileArg:   "--profile-directory=",
		incognitoArg: "--incognito",
	},
	{
		name:         "Google Chrome Beta",
		browserID:    "chrome-beta",
		executable:   "file://chrome.exe",
		appDataPath:  `Google\Chrome Beta\User Data`,
		profileArg:   "--profile-directory=",
		incognitoArg: "--incognito",
	},
	{
		name:         "Google Chrome Dev",
		browserID:    "chrome-dev",
		executable:   "file://chrome.exe",
		appDataPath:  `Google\Chrome Dev\User Data`,
		profileArg:   "--profile-directory=",
		incognitoArg: "--incognito",
	},
	{
		name:         "Google Chrome Canary",
		browserID:    "chrome-canary",
		executable:   "file://chrome.exe",
		appDataPath:  `Google\Chrome SxS\User Data`,
		profileArg:   "--profile-directory=",
		incognitoArg: "--incognito",
	},
	// Microsoft Edge
	{
		name:         "Microsoft Edge",
		browserID:    "edge",
		executable:   "file://msedge.exe",
		appDataPath:  `Microsoft\Edge\User Data`,
		profileArg:   "--profile-directory=",
		incognitoArg: "--inprivate",
	},
	{
		name:         "Microsoft Edge Beta",
		browserID:    "edge-beta",
		executable:   "file://msedge.exe",
		appDataPath:  `Microsoft\Edge Beta\User Data`,
		profileArg:   "--profile-directory=",
		incognitoArg: "--inprivate",
	},
	{
		name:         "Microsoft Edge Dev",
		browserID:    "edge-dev",
		executable:   "file://msedge.exe",
		appDataPath:  `Microsoft\Edge Dev\User Data`,
		profileArg:   "--profile-directory=",
		incognitoArg: "--inprivate",
	},
	{
		name:         "Microsoft Edge Canary",
		browserID:    "edge-canary",
		executable:   "file://msedge.exe",
		appDataPath:  `Microsoft\Edge SxS\User Data`,
		profileArg:   "--profile-directory=",
		incognitoArg: "--inprivate",
	},
	// Firefox
	{
		name:         "Firefox",
		browserID:    "firefox",
		executable:   "file://firefox.exe",
		appDataPath:  `Mozilla\Firefox`,
		firefoxIni:   true,
		profileArg:   "-P",
		incognitoArg: "--private-window",
	},
	{
		name:         "Firefox Developer Edition",
		browserID:    "firefox-dev",
		executable:   "file://firefox.exe",
		appDataPath:  `Mozilla\Firefox Developer Edition`,
		firefoxIni:   true,
		profileArg:   "-P",
		incognitoArg: "--private-window",
	},
	{
		name:         "Firefox Nightly",
		browserID:    "firefox-nightly",
		executable:   "file://firefox.exe",
		appDataPath:  `Mozilla\Firefox Nightly`,
		firefoxIni:   true,
		profileArg:   "-P",
		incognitoArg: "--private-window",
	},
	// Brave
	{
		name:         "Brave Browser",
		browserID:    "brave",
		executable:   "file://brave.exe",
		appDataPath:  `BraveSoftware\Brave-Browser\User Data`,
		profileArg:   "--profile-directory=",
		incognitoArg: "--incognito",
	},
	// Vivaldi
	{
		name:         "Vivaldi",
		browserID:    "vivaldi",
		executable:   "file://vivaldi.exe",
		appDataPath:  `Vivaldi\User Data`,
		profileArg:   "--profile-directory=",
		incognitoArg: "--incognito",
	},
	// Arc
	{
		name:         "Arc",
		browserID:    "arc",
		executable:   "file://arc.exe",
		appDataPath:  `Arc\User Data`,
		profileArg:   "",
		incognitoArg: "",
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
		programFiles := os.Getenv("ProgramFiles")
		programFilesX86 := os.Getenv("ProgramFiles(x86)")
		localAppData := os.Getenv("LOCALAPPDATA")

		searchPaths := []string{}
		if programFiles != "" {
			searchPaths = append(searchPaths, programFiles)
		}
		if programFilesX86 != "" && programFilesX86 != programFiles {
			searchPaths = append(searchPaths, programFilesX86)
		}
		if localAppData != "" {
			searchPaths = append(searchPaths, localAppData)
		}

		// Construct potential paths
		potentialDirs := []string{
			filepath.Join("Google", "Chrome", "Application"),
			filepath.Join("Microsoft", "Edge", "Application"),
			filepath.Join("Mozilla Firefox"),
			filepath.Join("BraveSoftware", "Brave-Browser", "Application"),
			filepath.Join("Vivaldi", "Application"),
			filepath.Join("Arc", "Application"),
		}

		for _, base := range searchPaths {
			for _, potentialDir := range potentialDirs {
				exePath := filepath.Join(base, potentialDir, path)
				if _, err := os.Stat(exePath); err == nil {
					return exePath
				}
			}
		}

		// Check PATH if not found in common locations
		if exePath, err := exec.LookPath(path); err == nil {
			return exePath
		}

		// Check Windows Registry (App Paths)
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\`+path, registry.QUERY_VALUE)
		if err == nil {
			defer key.Close()
			if exePath, _, err := key.GetStringValue(""); err == nil {
				if _, statErr := os.Stat(exePath); statErr == nil {
					return exePath
				}
			}
		}

	default:
		log.Warn().Str("scheme", scheme).Msg("Unknown executable scheme")
	}

	return ""
}

// DiscoverBrowsers finds installed browsers on Windows.
func (d *windowsDetector) DiscoverBrowsers() ([]config.Browser, error) {
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

// DiscoverProfiles finds profiles for a given browser on Windows.
func (d *windowsDetector) DiscoverProfiles(browser config.Browser) ([]config.Profile, error) {
	profiles := []config.Profile{}

	var info *knownBrowserInfo
	for i := range knownBrowsers {
		// Match using BrowserID now for consistency
		if knownBrowsers[i].browserID == browser.BrowserID && knownBrowsers[i].name == browser.Name {
			info = &knownBrowsers[i]
			break
		}
	}

	if info == nil || info.appDataPath == "" {
		// We don't know how to find profiles for this browser
		return profiles, fmt.Errorf("profile discovery not supported for browser ID %s, Name %s", browser.BrowserID, browser.Name)
	}

	// Try both APPDATA and LOCALAPPDATA locations
	baseDirs := []string{
		os.Getenv("LOCALAPPDATA"),
		os.Getenv("APPDATA"),
	}

	var profileBaseDir string
	for _, baseDir := range baseDirs {
		if baseDir == "" {
			continue
		}
		potentialPath := filepath.Join(baseDir, info.appDataPath)
		if _, err := os.Stat(potentialPath); err == nil {
			profileBaseDir = potentialPath
			break
		}
	}

	if profileBaseDir == "" {
		return profiles, fmt.Errorf("could not find profile directory in either APPDATA or LOCALAPPDATA")
	}

	if info.firefoxIni {
		// --- Firefox Profile Discovery (profiles.ini) ---
		iniPath := filepath.Join(profileBaseDir, "profiles.ini")
		parsedProfiles, err := ParseProfilesIni(iniPath)
		if err != nil {
			log.Warn().Err(err).Str("ini_path", iniPath).Msg("Failed to parse Firefox profiles.ini")
			// Fallback to default profile
			profiles = append(profiles, config.Profile{
				ID:         fmt.Sprintf("%s-default", info.browserID),
				Name:       fmt.Sprintf("%s (Default)", browser.Name),
				BrowserID:  browser.BrowserID,
				ProfileDir: "default",
			})
			return profiles, nil
		}

		for _, p := range parsedProfiles {
			profileDirResolved := p.Path
			if p.IsRelative == 1 {
				profileDirResolved = filepath.Join(profileBaseDir, p.Path)
			}
			// Check if the resolved directory actually exists
			if _, err := os.Stat(profileDirResolved); os.IsNotExist(err) {
				log.Warn().Str("profile_name", p.Name).Str("path", profileDirResolved).Msg("Firefox profile directory not found, skipping")
				continue
			}

			profileID := fmt.Sprintf("%s-%s", info.browserID, strings.ToLower(p.Name))
			profiles = append(profiles, config.Profile{
				ID:         profileID,
				Name:       fmt.Sprintf("%s (%s)", browser.Name, p.Name),
				BrowserID:  browser.BrowserID,
				ProfileDir: p.Name, // Use the actual profile name for -P flag
			})
		}

		if len(profiles) == 0 {
			log.Warn().Str("ini_path", iniPath).Msg("No valid Firefox profiles found, creating default")
			profiles = append(profiles, config.Profile{
				ID:         fmt.Sprintf("%s-default", info.browserID),
				Name:       fmt.Sprintf("%s (Default)", browser.Name),
				BrowserID:  browser.BrowserID,
				ProfileDir: "default",
			})
		}
	} else {
		// --- Chromium-based Profile Discovery (User Data directory) ---
		entries, err := os.ReadDir(profileBaseDir)
		if err != nil {
			// Don't treat as fatal, maybe the browser exists but has no profiles yet or path is wrong
			log.Warn().Err(err).Str("path", profileBaseDir).Str("browser_name", browser.Name).Msg("Could not read profile directory")
			return profiles, nil
		}

		for _, entry := range entries {
			if entry.IsDir() {
				dirName := entry.Name()
				// Check if it looks like a profile directory ("Default" or "Profile <N>")
				if dirName == "Default" || strings.HasPrefix(dirName, "Profile ") {
					// Basic check for a common file to ensure it's likely a valid profile
					if _, err := os.Stat(filepath.Join(profileBaseDir, dirName, "Preferences")); err == nil {
						profileID := fmt.Sprintf("%s-%s", info.browserID, strings.ToLower(strings.ReplaceAll(dirName, " ", "")))
						profileName := fmt.Sprintf("%s (%s)", browser.Name, dirName)
						profiles = append(profiles, config.Profile{
							ID:         profileID,
							Name:       profileName,
							BrowserID:  browser.Name,
							ProfileDir: dirName,
						})
					}
				}
			}
		}
	}

	return profiles, nil
}
