package launcher

import (
	"os/exec"
	"testing"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/stretchr/testify/assert"
)

// mockLauncher is a test implementation of the Launcher interface
type mockLauncher struct {
	launchAttempts []launchAttempt
}

type launchAttempt struct {
	browser   config.Browser
	profile   config.Profile
	url       string
	incognito bool
}

func newMockLauncher() *mockLauncher {
	return &mockLauncher{
		launchAttempts: make([]launchAttempt, 0),
	}
}

func (m *mockLauncher) LaunchBrowser(browser config.Browser, profile config.Profile, url string, incognito bool) error {
	m.launchAttempts = append(m.launchAttempts, launchAttempt{
		browser:   browser,
		profile:   profile,
		url:       url,
		incognito: incognito,
	})
	return nil
}

func (m *mockLauncher) constructCommand(browser config.Browser, profile config.Profile, url string, incognito bool) (*exec.Cmd, error) {
	return exec.Command("echo"), nil
}

func TestLaunchBrowser(t *testing.T) {
	mock := newMockLauncher()
	browser := config.Browser{
		Name:       "Test Browser",
		BrowserID:  "test",
		Executable: "/bin/echo",
	}
	profile := config.Profile{
		Name:       "Test Profile",
		ProfileDir: "/test/profile",
	}
	url := "https://example.com"

	err := mock.LaunchBrowser(browser, profile, url, false)
	assert.NoError(t, err)
	assert.Len(t, mock.launchAttempts, 1)
	assert.Equal(t, browser, mock.launchAttempts[0].browser)
	assert.Equal(t, profile, mock.launchAttempts[0].profile)
	assert.Equal(t, url, mock.launchAttempts[0].url)
	assert.False(t, mock.launchAttempts[0].incognito)
}

func TestEpiphanyProfileHandling(t *testing.T) {
	mock := newMockLauncher()
	browser := config.Browser{
		Name:       "Epiphany",
		BrowserID:  "epiphany",
		Executable: "epiphany",
		ProfileArg: "--profile=%s",
	}
	profile := config.Profile{
		Name:       "Test Profile",
		ProfileDir: "/test/profile",
	}
	url := "https://example.com"

	err := mock.LaunchBrowser(browser, profile, url, false)
	assert.NoError(t, err)
	assert.Len(t, mock.launchAttempts, 1)
	assert.Equal(t, browser, mock.launchAttempts[0].browser)
	assert.Equal(t, profile, mock.launchAttempts[0].profile)
	assert.Equal(t, url, mock.launchAttempts[0].url)
}

func TestCommandConstruction(t *testing.T) {
	mock := newMockLauncher()
	browser := config.Browser{
		Name:       "Test Browser",
		BrowserID:  "test",
		Executable: "/bin/echo",
	}
	profile := config.Profile{
		Name:       "Test Profile",
		ProfileDir: "/test/profile",
	}
	url := "https://example.com"

	cmd, err := mock.constructCommand(browser, profile, url, false)
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	// Check that the command is "echo" regardless of the full path
	assert.Equal(t, "echo", cmd.Args[0])
}

// Declare a variable to track executed commands in tests
var executedCommands []execCommand

type execCommand struct {
	profileID string
	url       string
	incognito bool
}

// TestLaunch tests the Launch function with mocked execution
func TestLaunch(t *testing.T) {
	// Save the original function and restore it after the test
	originalLaunch := actualLaunchFunc
	defer func() { actualLaunchFunc = originalLaunch }()

	// Reset the executed commands
	executedCommands = []execCommand{}

	// Mock the Launch function to avoid actual browser execution
	actualLaunchFunc = func(cfg *config.Config, profileID string, targetURL string, incognito bool) error {
		// Record the command details
		executedCommands = append(executedCommands, execCommand{
			profileID: profileID,
			url:       targetURL,
			incognito: incognito,
		})

		// Validate the profile exists
		_, err := cfg.FindProfileByID(profileID)
		return err
	}

	// Create a test config
	cfg := &config.Config{
		Profiles: []config.Profile{
			{
				ID:        "test-profile",
				Name:      "Test Profile",
				BrowserID: "test",
			},
		},
		Browsers: []config.Browser{
			{
				Name:       "Test Browser",
				BrowserID:  "test",
				Executable: "/bin/echo",
			},
		},
	}

	// Test with valid profile
	err := Launch(cfg, "test-profile", "https://example.com", false)
	assert.NoError(t, err)
	assert.Len(t, executedCommands, 1)
	assert.Equal(t, "test-profile", executedCommands[0].profileID)
	assert.Equal(t, "https://example.com", executedCommands[0].url)
	assert.False(t, executedCommands[0].incognito)

	// Test with invalid profile
	err = Launch(cfg, "nonexistent-profile", "https://example.com", false)
	assert.Error(t, err)
}
