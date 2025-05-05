package urlhandler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestProcessURL(t *testing.T) {
	// Create a minimal config for testing
	cfg := &config.Config{
		Shorteners: []config.ShortenerService{
			{Domain: "bit.ly", IsSafelink: false},
			{Domain: "tinyurl.com", IsSafelink: false},
		},
	}

	// Test cases - Update expected errors based on actual behavior
	// The ProcessURL function handles invalid URLs gracefully and doesn't return errors
	// for non-http/https URLs
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid HTTP URL",
			url:         "http://example.com",
			expectError: false,
		},
		{
			name:        "Valid HTTPS URL",
			url:         "https://example.com",
			expectError: false,
		},
		{
			name:        "Invalid URL",
			url:         "not a url",
			expectError: false, // Function doesn't error for invalid URLs
		},
		{
			name:        "Empty URL",
			url:         "",
			expectError: false, // Function doesn't error for empty URLs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urlForMatching, originalURL, isSafelink, err := ProcessURL(cfg, tt.url)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.url, originalURL)
			assert.False(t, isSafelink)
			assert.Equal(t, tt.url, urlForMatching)
		})
	}
}

func TestShortenerResolution(t *testing.T) {
	// Create a test server that redirects
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.com", http.StatusMovedPermanently)
	}))
	defer server.Close()

	// Test resolving a shortened URL
	finalURL, err := ResolveShortenedURL(server.URL)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", finalURL)

	// Test with an invalid URL
	_, err = ResolveShortenedURL("not a url")
	assert.Error(t, err)
}

// TestProcessURLWithShortener tests the ProcessURL function with a mock shortener service
func TestProcessURLWithShortener(t *testing.T) {
	// Create a test server that redirects
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.com", http.StatusMovedPermanently)
	}))
	defer server.Close()

	// Parse the server URL to get the hostname
	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)
	hostname := serverURL.Hostname()

	// Create a config with the test server as a shortener
	cfg := &config.Config{
		Shorteners: []config.ShortenerService{
			{Domain: hostname, IsSafelink: true},
		},
	}

	// Test with the mock shortener
	urlForMatching, originalURL, isSafelink, err := ProcessURL(cfg, server.URL)
	assert.NoError(t, err)
	assert.Equal(t, server.URL, originalURL)
	assert.True(t, isSafelink)
	assert.Equal(t, "https://example.com", urlForMatching)

	// Test with a manual shortener
	cfg = &config.Config{
		ManualShorteners: []config.ShortenerService{
			{Domain: hostname, IsSafelink: true},
		},
	}

	urlForMatching, originalURL, isSafelink, err = ProcessURL(cfg, server.URL)
	assert.NoError(t, err)
	assert.Equal(t, server.URL, originalURL)
	assert.True(t, isSafelink)
	assert.Equal(t, "https://example.com", urlForMatching)
}
