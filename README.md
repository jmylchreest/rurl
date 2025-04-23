# rurl (Route URL)

`rurl` is a command-line tool that acts as a smart default browser handler. It intercepts URLs and routes them to the appropriate browser profile based on configurable rules. This allows you to automatically open different URLs in different browser profiles, making it easier to manage multiple accounts, separate work/personal browsing, or maintain different browser configurations for different purposes.

## Features

* **Rule-Based Routing:** Define rules using regular expressions to match URLs (full URL, domain, or path)
* **Browser Profile Support:** Automatically detects installed browsers and their profiles
* **Profile Management:** Configure and manage browser profiles for different contexts
* **URL Shortener Resolution:** Resolves shortened URLs before applying rules
* **Safelinks Handling:** Properly handles Office 365 safelinks
* **Cross-Platform:** Works on Windows, macOS, and Linux

## Installation

### From Source
```bash
# Clone the repository
git clone https://github.com/jmylchreest/rurl.git
cd rurl

# Build the project
go build -o rurl .

# Optional: Install to your PATH
sudo mv rurl /usr/local/bin/  # Linux/macOS
# or add to your PATH on Windows
```

### From Releases
Download the latest release for your platform from the [releases page](https://github.com/jmylchreest/rurl/releases).

## Usage

### Basic Commands
```bash
# Process a URL
rurl https://example.com

# Show version information
rurl version

# Show help
rurl --help
```

### Configuration Commands
```bash
# Detect installed browsers
rurl config detect-browsers

# List configured browsers
rurl config browser list

# Add a browser
rurl config browser add

# List profiles
rurl config profile list

# Add a profile
rurl config profile add

# List rules
rurl config rule list

# Add a rule
rurl config rule add

# Show all configuration
rurl config show
```

### Setting as Default Browser

#### Linux
Add to your `.desktop` file:
```ini
[Desktop Entry]
Type=Application
Name=rurl
Exec=rurl %u
Terminal=false
Categories=Network;WebBrowser;
MimeType=x-scheme-handler/http;x-scheme-handler/https;
```

Then run:
```bash
xdg-mime default rurl.desktop x-scheme-handler/http
xdg-mime default rurl.desktop x-scheme-handler/https
```

#### macOS
```bash
# Set rurl as default browser for http and https
defaults write com.apple.LaunchServices/com.apple.launchservices.secure LSHandlers -array-add '{LSHandlerRoleAll=com.yourcompany.rurl;LSHandlerURLScheme=http;}'
defaults write com.apple.LaunchServices/com.apple.launchservices.secure LSHandlers -array-add '{LSHandlerRoleAll=com.yourcompany.rurl;LSHandlerURLScheme=https;}'
```

#### Windows
Use Windows Settings > Apps > Default Apps > Web Browser and select rurl.

## Configuration

`rurl` uses a TOML configuration file located at:
* Linux: `~/.config/rurl/config.toml`
* macOS: `~/Library/Application Support/rurl/config.toml`
* Windows: `%APPDATA%\rurl\config.toml`

The configuration file is automatically created with default values when you first run the application.

### Configuration Structure
```toml
# Default profile to use when no rules match
default_profile_id = "chrome-default"

# Browser definitions
[[browsers]]
name = "Google Chrome"
browser_id = "chrome"
executable = "/usr/bin/google-chrome-stable"
profile_arg = "--profile-directory=%s"
incognito_arg = "--incognito"

# Profile definitions
[[profiles]]
id = "chrome-default"
name = "Chrome (Default)"
browser_id = "chrome"
profile_dir = "Default"

# URL routing rules
[[rules]]
name = "Work Email"
pattern = "^https://outlook\\.office\\.com"
scope = "domain"
profile_id = "chrome-work"
incognito = false
```

## Development

### Prerequisites
* Go 1.21 or later
* Git

### Building
```bash
# Build with version information
go build -ldflags="-X github.com/jmylchreest/rurl/internal/config.Version=$(git describe --tags) -X github.com/jmylchreest/rurl/internal/config.Commit=$(git rev-parse HEAD) -X github.com/jmylchreest/rurl/internal/config.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o rurl .

# Build with stripped debug information (smaller binary)
go build -ldflags="-s -w" -o rurl .
```

### Testing
```bash
go test ./...
```

## Status

[![Test](https://github.com/jmylchreest/rurl/actions/workflows/test.yml/badge.svg)](https://github.com/jmylchreest/rurl/actions/workflows/test.yml)
[![Release](https://github.com/jmylchreest/rurl/actions/workflows/release.yml/badge.svg)](https://github.com/jmylchreest/rurl/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/jmylchreest/rurl/branch/main/graph/badge.svg)](https://codecov.io/gh/jmylchreest/rurl)
[![Go Report Card](https://goreportcard.com/badge/github.com/jmylchreest/rurl)](https://goreportcard.com/report/github.com/jmylchreest/rurl)
