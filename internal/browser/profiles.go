package browser

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// FirefoxProfileInfo holds temporary parsed data from profiles.ini
type FirefoxProfileInfo struct {
	Name       string
	IsRelative int // 0 or 1
	Path       string
	Default    int // 1 if default
}

// ParseProfilesIni parses the Firefox profiles.ini file.
// Returns a slice of profile information or an error if parsing fails.
func ParseProfilesIni(iniPath string) ([]FirefoxProfileInfo, error) {
	file, err := os.Open(iniPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []FirefoxProfileInfo{}, nil // Not an error if file doesn't exist
		}
		return nil, fmt.Errorf("could not open %s: %w", iniPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	profiles := make(map[string]*FirefoxProfileInfo)
	var currentProfileKey string // e.g., "Profile0", "Profile1"

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := line[1 : len(line)-1]
			if strings.HasPrefix(section, "Profile") {
				currentProfileKey = section
				if _, ok := profiles[currentProfileKey]; !ok {
					profiles[currentProfileKey] = &FirefoxProfileInfo{}
				}
			} else {
				currentProfileKey = "" // Not in a profile section anymore
			}
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if currentProfileKey != "" {
			p := profiles[currentProfileKey]
			switch key {
			case "Name":
				p.Name = value
			case "IsRelative":
				if value == "1" {
					p.IsRelative = 1
				} else {
					p.IsRelative = 0
				}
			case "Path":
				p.Path = value
			case "Default":
				if value == "1" {
					p.Default = 1
				} else {
					p.Default = 0
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", iniPath, err)
	}

	// Convert map to slice
	result := make([]FirefoxProfileInfo, 0, len(profiles))
	for _, p := range profiles {
		// Basic validation
		if p.Name != "" && p.Path != "" {
			result = append(result, *p)
		}
	}

	return result, nil
}
