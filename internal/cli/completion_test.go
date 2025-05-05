package cli

import (
	"testing"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// mockLoadConfigFunc is a variable to hold the mock function implementation
var mockLoadConfigFunc func() *config.Config

func TestCompleteRuleNames(t *testing.T) {
	// Set up test config
	cfgFile = "" // Reset global config file path
	testConfig := &config.Config{
		Rules: []config.Rule{
			{Name: "Work Sites"},
			{Name: "Work Email"},
			{Name: "Personal Sites"},
		},
	}

	// Save original function and restore after tests
	originalLoadConfig := loadConfigForCompletion
	loadConfigForCompletion = func() *config.Config {
		return testConfig
	}
	defer func() { loadConfigForCompletion = originalLoadConfig }()

	tests := []struct {
		name       string
		toComplete string
		want       []string
		wantDir    cobra.ShellCompDirective
	}{
		{
			name:       "empty prefix",
			toComplete: "",
			want:       []string{"Work Sites", "Work Email", "Personal Sites"},
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "work prefix",
			toComplete: "Work",
			want:       []string{"Work Sites", "Work Email"},
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "personal prefix",
			toComplete: "Personal",
			want:       []string{"Personal Sites"},
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "no match",
			toComplete: "Unknown",
			want:       nil,
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, dir := completeRuleNames(nil, nil, tt.toComplete)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantDir, dir)
		})
	}

	// Test with nil config
	nullConfig := loadConfigForCompletion
	loadConfigForCompletion = func() *config.Config {
		return nil
	}
	got, dir := completeRuleNames(nil, nil, "")
	assert.Nil(t, got)
	assert.Equal(t, cobra.ShellCompDirectiveError, dir)
	loadConfigForCompletion = nullConfig
}

func TestCompleteProfileIDs(t *testing.T) {
	// Set up test config
	cfgFile = "" // Reset global config file path
	testConfig := &config.Config{
		Profiles: []config.Profile{
			{ID: "work-chrome"},
			{ID: "work-firefox"},
			{ID: "personal"},
		},
	}

	// Save original function and restore after tests
	originalLoadConfig := loadConfigForCompletion
	loadConfigForCompletion = func() *config.Config {
		return testConfig
	}
	defer func() { loadConfigForCompletion = originalLoadConfig }()

	tests := []struct {
		name       string
		toComplete string
		want       []string
		wantDir    cobra.ShellCompDirective
	}{
		{
			name:       "empty prefix",
			toComplete: "",
			want:       []string{"work-chrome", "work-firefox", "personal"},
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "work prefix",
			toComplete: "work-",
			want:       []string{"work-chrome", "work-firefox"},
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "personal prefix",
			toComplete: "personal",
			want:       []string{"personal"},
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "no match",
			toComplete: "unknown",
			want:       nil,
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, dir := completeProfileIDs(nil, nil, tt.toComplete)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantDir, dir)
		})
	}

	// Test with nil config
	nullConfig := loadConfigForCompletion
	loadConfigForCompletion = func() *config.Config {
		return nil
	}
	got, dir := completeProfileIDs(nil, nil, "")
	assert.Nil(t, got)
	assert.Equal(t, cobra.ShellCompDirectiveError, dir)
	loadConfigForCompletion = nullConfig
}

func TestCompleteBrowserIDs(t *testing.T) {
	// Set up test config
	cfgFile = "" // Reset global config file path
	testConfig := &config.Config{
		Browsers: []config.Browser{
			{BrowserID: "chrome"},
			{BrowserID: "firefox"},
			{BrowserID: "chrome-beta"}, // Duplicate prefix with chrome
		},
	}

	// Save original function and restore after tests
	originalLoadConfig := loadConfigForCompletion
	loadConfigForCompletion = func() *config.Config {
		return testConfig
	}
	defer func() { loadConfigForCompletion = originalLoadConfig }()

	tests := []struct {
		name       string
		toComplete string
		want       []string
		wantDir    cobra.ShellCompDirective
	}{
		{
			name:       "empty prefix",
			toComplete: "",
			want:       []string{"chrome", "firefox", "chrome-beta"},
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "chrome prefix",
			toComplete: "chrome",
			want:       []string{"chrome", "chrome-beta"},
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "firefox prefix",
			toComplete: "firefox",
			want:       []string{"firefox"},
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "no match",
			toComplete: "unknown",
			want:       nil,
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, dir := completeBrowserIDs(nil, nil, tt.toComplete)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantDir, dir)
		})
	}

	// Test with nil config
	nullConfig := loadConfigForCompletion
	loadConfigForCompletion = func() *config.Config {
		return nil
	}
	got, dir := completeBrowserIDs(nil, nil, "")
	assert.Nil(t, got)
	assert.Equal(t, cobra.ShellCompDirectiveError, dir)
	loadConfigForCompletion = nullConfig
}
