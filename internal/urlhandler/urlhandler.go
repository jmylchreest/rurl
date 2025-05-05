package urlhandler

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/rs/zerolog/log"
)

// ProcessURL takes an input URL string, checks if the domain matches known or
// manually added shortener services, and resolves if necessary. It returns the final URL
// to be used for rule matching, the original input URL, a flag indicating if the
// original domain was marked as a safelink, and any fatal processing error.
func ProcessURL(cfg *config.Config, inputURL string) (urlForMatching string, originalURL string, isSafelink bool, err error) {
	originalURL = inputURL // Store the original input

	// 1. Parse the input URL
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		// If parsing fails, it cannot be a shortener domain we recognize.
		// Return original URL, flag false, and the parsing error if it's critical,
		// or maybe just return the inputURL as is if the error isn't format related?
		// For now, returning error.
		return inputURL, originalURL, false, fmt.Errorf("failed to parse input URL: %w", err)
	}

	hostname := parsedURL.Hostname()
	var matchedShortener *config.ShortenerService = nil

	// Only attempt shortener resolution for http/https URLs
	if parsedURL.Scheme == "http" || parsedURL.Scheme == "https" {
		// 2. Check if the hostname matches any known (built-in or manual) shortener domain
		// Check manual list first, then built-in
		for i := range cfg.ManualShorteners {
			if cfg.ManualShorteners[i].Domain == hostname {
				matchedShortener = &cfg.ManualShorteners[i]
				log.Debug().Str("domain", hostname).Msg("Matched manual shortener domain.")
				break
			}
		}

		if matchedShortener == nil { // If not found in manual list, check built-in
			for i := range cfg.Shorteners {
				if cfg.Shorteners[i].Domain == hostname {
					matchedShortener = &cfg.Shorteners[i]
					log.Debug().Str("domain", hostname).Msg("Matched built-in shortener domain.")
					break
				}
			}
		}

		// 3. If a shortener domain was matched, attempt resolution
		if matchedShortener != nil {
			log.Info().Str("domain", hostname).Msg("Detected shortener domain, resolving...")
			resolved, resolveErr := ResolveShortenedURL(inputURL)
			if resolveErr != nil {
				log.Warn().Err(resolveErr).Str("original_url", inputURL).Msg("Failed to resolve shortened URL, using original for matching.")
				// Return original URL for matching, original input, safelink=false, nil error (non-fatal for matching)
				return inputURL, originalURL, false, nil
			}
			log.Info().Str("original_url", inputURL).Str("resolved_url", resolved).Msg("Resolved shortener domain URL.")
			// Return resolved URL for matching, original input, the configured safelink flag, nil error
			return resolved, originalURL, matchedShortener.IsSafelink, nil
		}
	} else {
		log.Debug().Str("url", inputURL).Str("scheme", parsedURL.Scheme).Msg("URL scheme is not http/https, skipping shortener checks.")
	}

	// 4. If not a recognized shortener domain or not http/https, return the input URL as is.
	log.Debug().Str("url", inputURL).Msg("URL is not a recognized shortener domain.")
	return inputURL, originalURL, false, nil
}

// ResolveShortenedURL attempts to follow redirects for a given URL.
func ResolveShortenedURL(shortURL string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second, // Add a timeout
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	maxRedirects := 5
	currentURL := shortURL

	for i := 0; i < maxRedirects; i++ {
		req, err := http.NewRequest("HEAD", currentURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request for %s: %w", currentURL, err)
		}
		req.Header.Set("User-Agent", "rurl/1.0")

		resp, err := client.Do(req)
		if err != nil {
			// Fallback to GET if HEAD fails
			if i == 0 { // Only try GET on the first attempt
				log.Debug().Str("url", currentURL).Msg("HEAD request failed, falling back to GET")
				req, _ = http.NewRequest("GET", currentURL, nil)
				req.Header.Set("User-Agent", "rurl/1.0")
				resp, err = client.Do(req)
			}
			if err != nil {
				return "", fmt.Errorf("failed to perform request for %s: %w", currentURL, err)
			}
		}

		if resp.Body != nil {
			resp.Body.Close()
		}

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location := resp.Header.Get("Location")
			if location == "" {
				return "", fmt.Errorf("redirect response for %s had no Location header (status: %d)", currentURL, resp.StatusCode)
			}

			redirectURL, err := url.Parse(location)
			if err != nil {
				return "", fmt.Errorf("failed to parse redirect location '%s': %w", location, err)
			}
			baseReqURL, _ := url.Parse(currentURL)
			currentURL = baseReqURL.ResolveReference(redirectURL).String()
			log.Debug().Str("from", resp.Request.URL.String()).Str("to", currentURL).Int("status", resp.StatusCode).Msg("Following redirect")

			if i == maxRedirects-1 {
				return "", fmt.Errorf("too many redirects resolving %s", shortURL)
			}
		} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Debug().Str("url", currentURL).Int("status", resp.StatusCode).Msg("Resolved URL")
			return currentURL, nil
		} else {
			return "", fmt.Errorf("unexpected status code %d while resolving %s", resp.StatusCode, currentURL)
		}
	}

	return "", fmt.Errorf("unexpected state after resolving redirects for %s", shortURL) // Should not be reached
}
