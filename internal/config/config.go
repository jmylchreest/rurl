package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/mitchellh/mapstructure" // Need this for decoding struct to map
	"github.com/spf13/viper"
)

// RuleScope defines where a rule\'s pattern should be matched.
type RuleScope string

const (
	ScopeURL    RuleScope = "url"    // Match against the entire URL
	ScopeDomain RuleScope = "domain" // Match against the domain part only
	ScopePath   RuleScope = "path"   // Match against the path part only
)

// Browser represents a detected browser application.
type Browser struct {
	Name         string `mapstructure:"name"`         // User-friendly name (e.g., "Google Chrome")
	BrowserID    string `mapstructure:"BrowserID"`    // Stable identifier (e.g., "chrome", "firefox")
	Executable   string `mapstructure:"executable"`   // Path to the browser executable or .app bundle (macOS)
	BundleID     string `mapstructure:"bundle_id"`    // macOS Bundle Identifier (optional)
	ProfileArg   string `mapstructure:"ProfileArg"`   // Argument template for specifying profile (e.g., "--profile-directory=%s")
	IncognitoArg string `mapstructure:"IncognitoArg"` // Argument for incognito/private mode (e.g., "--incognito")
	// FramelessArg string `mapstructure:"frameless_arg"` // Argument for frameless/app mode (e.g., "--app=%s") - Future?
}

// Profile represents a specific browser profile.
type Profile struct {
	ID         string `mapstructure:"id"`         // Unique identifier (e.g., "chrome-default", "firefox-dev")
	Name       string `mapstructure:"name"`       // User-friendly name (e.g., "Chrome (Default)", "Firefox Developer")
	BrowserID  string `mapstructure:"BrowserID"`  // ID of the Browser this profile belongs to
	ProfileDir string `mapstructure:"ProfileDir"` // Profile directory identifier used by the browser (e.g., "Default", "profile.dev")
}

// Rule defines how to match a URL and which profile to use.
type Rule struct {
	ID        string    `mapstructure:"id"`        // Unique identifier for the rule
	Name      string    `mapstructure:"name"`      // User-friendly name (e.g., "Work Links", "Dev Server")
	Pattern   string    `mapstructure:"pattern"`   // Regex pattern to match
	Scope     RuleScope `mapstructure:"scope"`     // Where to apply the pattern (url, domain, path)
	ProfileID string    `mapstructure:"ProfileID"` // ID of the Profile to use if matched (Changed tag to PascalCase)
	Incognito bool      `mapstructure:"incognito"` // Open in incognito/private mode?
	// Frameless bool      `mapstructure:"frameless"` // Open in frameless/app mode? - Future?
}

// ShortenerService defines configuration for a URL shortener domain.
// Used for both built-in defaults and manually added domains.
type ShortenerService struct {
	Domain     string `mapstructure:"domain"`      // Domain of the shortener (e.g., "t.co", "bit.ly")
	IsSafelink bool   `mapstructure:"is_safelink"` // If true, pass original short URL to browser after rule matching (Default: false)
}

// Config holds the entire application configuration.
type Config struct {
	DefaultProfileID string             `mapstructure:"default_profile_id"`
	Browsers         []Browser          `mapstructure:"browsers"`
	Profiles         []Profile          `mapstructure:"profiles"`
	Rules            []Rule             `mapstructure:"rules"`
	Shorteners       []ShortenerService `mapstructure:"shorteners"`        // List of built-in known shortener domains
	ManualShorteners []ShortenerService `mapstructure:"manual_shorteners"` // List of user-added shortener domains
}

// Default values for configuration
func DefaultConfig() *Config {
	return &Config{
		Browsers: []Browser{},
		Profiles: []Profile{},
		Rules:    []Rule{},
		Shorteners: []ShortenerService{ // Built-in common shorteners
			{Domain: "t.co", IsSafelink: false},
			{Domain: "bit.ly", IsSafelink: false},
			{Domain: "goo.gl", IsSafelink: false},
			{Domain: "tinyurl.com", IsSafelink: false},
			{Domain: "73.nu", IsSafelink: false},
			{Domain: "bitly.kr", IsSafelink: false},
			{Domain: "bl.ink", IsSafelink: false},
			{Domain: "buff.ly", IsSafelink: false},
			{Domain: "clicky.me", IsSafelink: false},
			{Domain: "cutt.ly", IsSafelink: false},
			{Domain: "dub.co", IsSafelink: false},
			{Domain: "fox.ly", IsSafelink: false},
			{Domain: "gg.gg", IsSafelink: false},
			{Domain: "han.gl", IsSafelink: false},
			{Domain: "is.gd", IsSafelink: false},
			{Domain: "kurzelinks.de", IsSafelink: false},
			{Domain: "kutt.it", IsSafelink: false},
			{Domain: "lstu.fr", IsSafelink: false},
			{Domain: "lyn.bz", IsSafelink: false},
			{Domain: "oe.cd", IsSafelink: false},
			{Domain: "ow.ly", IsSafelink: false},
			{Domain: "rebrandly.com", IsSafelink: false},
			{Domain: "reduced.to", IsSafelink: false},
			{Domain: "rip.to", IsSafelink: false},
			{Domain: "san.aq", IsSafelink: false},
			{Domain: "short.io", IsSafelink: false},
			{Domain: "shorten-url.com", IsSafelink: false},
			{Domain: "shorturl.at", IsSafelink: false},
			{Domain: "sor.bz", IsSafelink: false},
			{Domain: "spoo.me", IsSafelink: false},
			{Domain: "switchy.io", IsSafelink: false},
			{Domain: "t.ly", IsSafelink: false},
			{Domain: "tinu.be", IsSafelink: false},
			{Domain: "urlr.me", IsSafelink: false},
			{Domain: "v.gd", IsSafelink: false},
			{Domain: "vo.la", IsSafelink: false},
			{Domain: "yaso.su", IsSafelink: false},
			{Domain: "zlnk.com", IsSafelink: false},
			{Domain: "safelinks.protection.outlook.com", IsSafelink: true},
		},
		ManualShorteners: []ShortenerService{}, // Initialize manual shorteners as empty
	}
}

// GetConfigDir returns the default configuration directory for the OS.
func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not get user config directory: %w", err)
	}
	return filepath.Join(configDir, "rurl"), nil
}

// LoadConfig loads the configuration from the specified file or default locations.
func LoadConfig(cfgFile string) (*Config, error) {
	v := viper.New()

	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.AddConfigPath(configDir)
		v.SetConfigName("config")
		v.SetConfigType("toml")
	}

	v.AutomaticEnv()

	// Set default values
	defaults := DefaultConfig()
	v.SetDefault("default_profile_id", defaults.DefaultProfileID)
	v.SetDefault("browsers", defaults.Browsers)
	v.SetDefault("profiles", defaults.Profiles)
	v.SetDefault("rules", defaults.Rules)
	v.SetDefault("shorteners", defaults.Shorteners)
	v.SetDefault("manual_shorteners", defaults.ManualShorteners) // Use new key

	// Ensure config directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create config directory '%s': %w", configDir, err)
		}
	}

	configFilePath := filepath.Join(configDir, "config.toml")
	if cfgFile != "" {
		configFilePath = cfgFile
	}

	// Attempt to read the config file
	err = v.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		fmt.Printf("Config file not found. Creating default config at: %s\n", configFilePath)
		// Use MergeConfigMap with defaults before writing
		defaultMap := make(map[string]interface{})
		decoderConfig := &mapstructure.DecoderConfig{Result: &defaultMap, TagName: "mapstructure"}
		decoder, _ := mapstructure.NewDecoder(decoderConfig)
		_ = decoder.Decode(defaults)
		_ = v.MergeConfigMap(defaultMap)

		if err := v.WriteConfigAs(configFilePath); err != nil {
			return nil, fmt.Errorf("failed to write default config file '%s': %w", configFilePath, err)
		}
		// Re-read after writing defaults
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read newly created config file '%s': %w", configFilePath, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", configFilePath, err)
	}

	var cfg Config
	// Custom decode hook for RuleScope
	decodeHook := func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String || t != reflect.TypeOf(ScopeURL) {
			return data, nil
		}
		str := data.(string)
		switch RuleScope(str) {
		case ScopeURL, ScopeDomain, ScopePath:
			return RuleScope(str), nil
		default:
			return ScopeURL, nil // Default to ScopeURL if invalid
		}
	}
	if err := v.Unmarshal(&cfg, viper.DecodeHook(decodeHook)); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	cfg.Shorteners = defaults.Shorteners
	return &cfg, nil
}

// SaveConfig saves the current configuration back to the file.
func SaveConfig(cfg *Config, cfgFile string) error {
	v := viper.New()

	if cfgFile == "" {
		configDir, err := GetConfigDir()
		if err != nil {
			return fmt.Errorf("failed to get config directory for saving: %w", err)
		}
		cfgFile = filepath.Join(configDir, "config.toml")
	}
	v.SetConfigFile(cfgFile)

	// Convert the config struct to a map[string]interface{}
	cfgMap := make(map[string]interface{})
	decoderConfig := &mapstructure.DecoderConfig{
		Result:  &cfgMap,
		TagName: "mapstructure",
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return fmt.Errorf("failed to create mapstructure decoder: %w", err)
	}
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("failed to decode config struct to map: %w", err)
	}

	// Set all values in the fresh viper instance from the map
	for key, value := range cfgMap {
		v.Set(key, value)
	}

	// Ensure the directory exists before writing
	configDir := filepath.Dir(cfgFile)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0750); err != nil {
			return fmt.Errorf("failed to create config directory '%s' for saving: %w", configDir, err)
		}
	}

	// Write the configuration file
	if err := v.WriteConfigAs(cfgFile); err != nil {
		return fmt.Errorf("failed to write config file '%s': %w", cfgFile, err)
	}
	return nil
}

// FindProfileByID looks up a profile by its unique ID.
func (c *Config) FindProfileByID(id string) (*Profile, error) {
	for i := range c.Profiles {
		if c.Profiles[i].ID == id {
			return &c.Profiles[i], nil
		}
	}
	return nil, fmt.Errorf("profile with ID '%s' not found", id)
}

// FindBrowserByID looks up a browser by its unique ID.
func (c *Config) FindBrowserByID(id string) (*Browser, error) {
	for i := range c.Browsers {
		b := &c.Browsers[i]
		if b.BrowserID == id {
			return b, nil
		}
	}
	return nil, fmt.Errorf("browser with ID '%s' not found", id)
}

// GetProfileBrowser returns the Browser associated with a given Profile.
func (c *Config) GetProfileBrowser(profile *Profile) (*Browser, error) {
	return c.FindBrowserByID(profile.BrowserID)
}

// FindManualShortenerByDomain finds a manually added shortener by its domain.
func (c *Config) FindManualShortenerByDomain(domain string) (*ShortenerService, int, error) {
	for i := range c.ManualShorteners {
		if c.ManualShorteners[i].Domain == domain {
			return &c.ManualShorteners[i], i, nil // Return pointer, index, and nil error
		}
	}
	return nil, -1, fmt.Errorf("manual shortener service for domain '%s' not found", domain)
}
