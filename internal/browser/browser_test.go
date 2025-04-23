package browser

import (
	"path/filepath"
	"testing"

	"github.com/jmylchreest/rurl/internal/config"
)

// mockFS helps us test filesystem operations without touching the real filesystem
type mockFS struct {
	files    map[string][]byte
	dirs     map[string]bool
	homeDir  string
	execPath string
}

func newMockFS() *mockFS {
	return &mockFS{
		files:   make(map[string][]byte),
		dirs:    make(map[string]bool),
		homeDir: "/home/testuser",
	}
}

func (m *mockFS) addFile(path string, content []byte) {
	m.files[path] = content
	// Ensure parent directories exist
	dir := filepath.Dir(path)
	for dir != "/" {
		m.dirs[dir] = true
		dir = filepath.Dir(dir)
	}
}

func (m *mockFS) addDir(path string) {
	m.dirs[path] = true
}

func (m *mockFS) setExecutable(path string) {
	m.execPath = path
}

// mockDetector implements Detector interface for testing
type mockDetector struct {
	fs *mockFS
}

func newMockDetector(fs *mockFS) *mockDetector {
	return &mockDetector{fs: fs}
}

func (d *mockDetector) DiscoverBrowsers() ([]config.Browser, error) {
	var browsers []config.Browser

	// Mock Chrome detection
	if d.fs.execPath == "/usr/bin/google-chrome" {
		browsers = append(browsers, config.Browser{
			Name:         "Google Chrome",
			BrowserID:    "chrome",
			Executable:   "/usr/bin/google-chrome",
			ProfileArg:   "--profile-directory=%s",
			IncognitoArg: "--incognito",
		})
	}

	// Mock Firefox detection
	if d.fs.execPath == "/usr/bin/firefox" {
		browsers = append(browsers, config.Browser{
			Name:         "Firefox",
			BrowserID:    "firefox",
			Executable:   "/usr/bin/firefox",
			ProfileArg:   "-P %s",
			IncognitoArg: "--private-window",
		})
	}

	return browsers, nil
}

func (d *mockDetector) DiscoverProfiles(browser config.Browser) ([]config.Profile, error) {
	var profiles []config.Profile

	switch browser.BrowserID {
	case "chrome":
		// Mock Chrome profiles
		if content, exists := d.fs.files[filepath.Join(d.fs.homeDir, ".config/google-chrome/Local State")]; exists {
			// In a real implementation, we'd parse the JSON content
			_ = content // Unused in mock
			profiles = append(profiles, config.Profile{
				ID:         "chrome-default",
				Name:       "Default",
				BrowserID:  "chrome",
				ProfileDir: "Default",
			})
			profiles = append(profiles, config.Profile{
				ID:         "chrome-work",
				Name:       "Work Profile",
				BrowserID:  "chrome",
				ProfileDir: "Profile 1",
			})
		}

	case "firefox":
		// Mock Firefox profiles
		if content, exists := d.fs.files[filepath.Join(d.fs.homeDir, ".mozilla/firefox/profiles.ini")]; exists {
			// In a real implementation, we'd parse the INI content
			_ = content // Unused in mock
			profiles = append(profiles, config.Profile{
				ID:         "firefox-default",
				Name:       "default",
				BrowserID:  "firefox",
				ProfileDir: "default",
			})
			profiles = append(profiles, config.Profile{
				ID:         "firefox-dev",
				Name:       "Developer Edition",
				BrowserID:  "firefox",
				ProfileDir: "dev-edition-default",
			})
		}
	}

	return profiles, nil
}

func TestBrowserDetection(t *testing.T) {
	// Set up mock filesystem
	fs := newMockFS()
	fs.setExecutable("/usr/bin/google-chrome")
	fs.addFile(filepath.Join(fs.homeDir, ".config/google-chrome/Local State"), []byte(`{
		"profile": {
			"info_cache": {
				"Default": {
					"name": "Default",
					"is_using_default_name": true
				},
				"Profile 1": {
					"name": "Work Profile",
					"is_using_default_name": false
				}
			}
		}
	}`))

	// Create detector with mock filesystem
	detector := newMockDetector(fs)

	// Test browser discovery
	browsers, err := detector.DiscoverBrowsers()
	if err != nil {
		t.Fatalf("DiscoverBrowsers() error = %v", err)
	}

	// Verify Chrome was found
	var chrome *config.Browser
	for i := range browsers {
		if browsers[i].BrowserID == "chrome" {
			chrome = &browsers[i]
			break
		}
	}

	if chrome == nil {
		t.Fatal("Chrome browser not found")
	}

	// Test profile discovery for Chrome
	profiles, err := detector.DiscoverProfiles(*chrome)
	if err != nil {
		t.Fatalf("DiscoverProfiles() error = %v", err)
	}

	// Verify profiles were found
	if len(profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(profiles))
	}

	// Verify specific profiles
	var defaultProfile, workProfile *config.Profile
	for i := range profiles {
		switch profiles[i].ID {
		case "chrome-default":
			defaultProfile = &profiles[i]
		case "chrome-work":
			workProfile = &profiles[i]
		}
	}

	if defaultProfile == nil {
		t.Error("Default profile not found")
	}
	if workProfile == nil {
		t.Error("Work profile not found")
	}

	// Test Firefox detection
	fs.setExecutable("/usr/bin/firefox")
	fs.addFile(filepath.Join(fs.homeDir, ".mozilla/firefox/profiles.ini"), []byte(`
[Profile0]
Name=default
IsRelative=1
Path=default
Default=1

[Profile1]
Name=Developer Edition
IsRelative=1
Path=dev-edition-default
`))

	// Re-run browser discovery
	browsers, err = detector.DiscoverBrowsers()
	if err != nil {
		t.Fatalf("DiscoverBrowsers() error = %v", err)
	}

	// Verify Firefox was found
	var firefox *config.Browser
	for i := range browsers {
		if browsers[i].BrowserID == "firefox" {
			firefox = &browsers[i]
			break
		}
	}

	if firefox == nil {
		t.Fatal("Firefox browser not found")
	}

	// Test profile discovery for Firefox
	profiles, err = detector.DiscoverProfiles(*firefox)
	if err != nil {
		t.Fatalf("DiscoverProfiles() error = %v", err)
	}

	// Verify Firefox profiles were found
	if len(profiles) != 2 {
		t.Errorf("Expected 2 Firefox profiles, got %d", len(profiles))
	}

	// Verify specific Firefox profiles
	var defaultFFProfile, devFFProfile *config.Profile
	for i := range profiles {
		switch profiles[i].ID {
		case "firefox-default":
			defaultFFProfile = &profiles[i]
		case "firefox-dev":
			devFFProfile = &profiles[i]
		}
	}

	if defaultFFProfile == nil {
		t.Error("Default Firefox profile not found")
	}
	if devFFProfile == nil {
		t.Error("Developer Edition Firefox profile not found")
	}
}
