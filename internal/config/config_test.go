package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir, err := os.MkdirTemp("", "rurl-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test config file with matching field names from the structs
	configPath := filepath.Join(tmpDir, "config.toml")
	configContent := `
default_profile_id = "chrome-default"

[[browsers]]
name = "Google Chrome"
BrowserID = "chrome"
executable = "/usr/bin/google-chrome-stable"
ProfileArg = "--profile-directory=%s"
IncognitoArg = "--incognito"

[[profiles]]
id = "chrome-default"
name = "Default"
BrowserID = "chrome"
ProfileDir = "Default"

[[rules]]
name = "Work Sites"
pattern = "^https://work\\."
scope = "domain"
ProfileID = "chrome-default"
incognito = false
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify config contents
	assert.Equal(t, "chrome-default", cfg.DefaultProfileID)
	assert.Len(t, cfg.Browsers, 1)
	assert.Len(t, cfg.Profiles, 1)
	assert.Len(t, cfg.Rules, 1)

	// Verify browser details
	browser := cfg.Browsers[0]
	assert.Equal(t, "Google Chrome", browser.Name)
	assert.Equal(t, "chrome", browser.BrowserID)
	assert.Equal(t, "/usr/bin/google-chrome-stable", browser.Executable)
	assert.Equal(t, "--profile-directory=%s", browser.ProfileArg)
	assert.Equal(t, "--incognito", browser.IncognitoArg)

	// Verify profile details
	profile := cfg.Profiles[0]
	assert.Equal(t, "chrome-default", profile.ID)
	assert.Equal(t, "Default", profile.Name)
	assert.Equal(t, "chrome", profile.BrowserID)
	assert.Equal(t, "Default", profile.ProfileDir)

	// Verify rule details
	rule := cfg.Rules[0]
	assert.Equal(t, "Work Sites", rule.Name)
	assert.Equal(t, "^https://work\\.", rule.Pattern)
	assert.Equal(t, RuleScope("domain"), rule.Scope)
	assert.Equal(t, "chrome-default", rule.ProfileID)
	assert.False(t, rule.Incognito)
}

func TestFindProfileByID(t *testing.T) {
	cfg := &Config{
		Profiles: []Profile{
			{
				ID:         "profile1",
				Name:       "Profile 1",
				BrowserID:  "browser1",
				ProfileDir: "dir1",
			},
			{
				ID:         "profile2",
				Name:       "Profile 2",
				BrowserID:  "browser2",
				ProfileDir: "dir2",
			},
		},
	}

	// Test finding existing profile
	profile, err := cfg.FindProfileByID("profile1")
	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, "Profile 1", profile.Name)

	// Test finding non-existent profile
	profile, err = cfg.FindProfileByID("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, profile)
}

func TestFindBrowserByID(t *testing.T) {
	cfg := &Config{
		Browsers: []Browser{
			{
				Name:         "Browser 1",
				BrowserID:    "browser1",
				Executable:   "/path/to/browser1",
				ProfileArg:   "--profile=%s",
				IncognitoArg: "--incognito",
			},
			{
				Name:         "Browser 2",
				BrowserID:    "browser2",
				Executable:   "/path/to/browser2",
				ProfileArg:   "--profile=%s",
				IncognitoArg: "--private",
			},
		},
	}

	// Test finding existing browser
	browser, err := cfg.FindBrowserByID("browser1")
	assert.NoError(t, err)
	assert.NotNil(t, browser)
	assert.Equal(t, "Browser 1", browser.Name)

	// Test finding non-existent browser
	browser, err = cfg.FindBrowserByID("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, browser)
}
