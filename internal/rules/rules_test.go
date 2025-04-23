package rules

import (
	"testing"

	"github.com/jmylchreest/rurl/internal/config"
)

func TestApplyRules(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		url     string
		want    MatchResult
		wantErr bool
	}{
		{
			name:    "nil config",
			cfg:     nil,
			url:     "https://example.com",
			want:    MatchResult{},
			wantErr: true,
		},
		{
			name: "no rules with default profile",
			cfg: &config.Config{
				DefaultProfileID: "default-profile",
				Profiles: []config.Profile{
					{ID: "default-profile", Name: "Default"},
				},
			},
			url: "https://example.com",
			want: MatchResult{
				Rule:      nil,
				ProfileID: "default-profile",
				Incognito: false,
			},
			wantErr: false,
		},
		{
			name: "no rules and no default profile",
			cfg: &config.Config{
				Profiles: []config.Profile{
					{ID: "some-profile", Name: "Some Profile"},
				},
			},
			url:     "https://example.com",
			want:    MatchResult{},
			wantErr: true,
		},
		{
			name: "invalid regex pattern",
			cfg: &config.Config{
				DefaultProfileID: "default-profile",
				Profiles: []config.Profile{
					{ID: "default-profile", Name: "Default"},
				},
				Rules: []config.Rule{
					{
						Name:      "Invalid Rule",
						Pattern:   "[invalid(regex",
						ProfileID: "default-profile",
					},
				},
			},
			url: "https://example.com",
			want: MatchResult{
				Rule:      nil,
				ProfileID: "default-profile",
				Incognito: false,
			},
			wantErr: false,
		},
		{
			name: "matching rule with non-existent profile",
			cfg: &config.Config{
				DefaultProfileID: "default-profile",
				Profiles: []config.Profile{
					{ID: "default-profile", Name: "Default"},
				},
				Rules: []config.Rule{
					{
						Name:      "Test Rule",
						Pattern:   "^https://example\\.com",
						ProfileID: "non-existent-profile",
					},
				},
			},
			url:     "https://example.com",
			want:    MatchResult{},
			wantErr: true,
		},
		{
			name: "multiple rules with different specificity",
			cfg: &config.Config{
				DefaultProfileID: "default-profile",
				Profiles: []config.Profile{
					{ID: "default-profile", Name: "Default"},
					{ID: "work-profile", Name: "Work"},
					{ID: "personal-profile", Name: "Personal"},
				},
				Rules: []config.Rule{
					{
						Name:      "Generic Domain",
						Pattern:   "^https://example\\.com",
						ProfileID: "default-profile",
					},
					{
						Name:      "Specific Path",
						Pattern:   "^https://example\\.com/work",
						ProfileID: "work-profile",
						Incognito: true,
					},
				},
			},
			url: "https://example.com/work/dashboard",
			want: MatchResult{
				Rule: &config.Rule{
					Name:      "Specific Path",
					Pattern:   "^https://example\\.com/work",
					ProfileID: "work-profile",
					Incognito: true,
				},
				ProfileID: "work-profile",
				Incognito: true,
			},
			wantErr: false,
		},
		{
			name: "domain scope rule",
			cfg: &config.Config{
				DefaultProfileID: "default-profile",
				Profiles: []config.Profile{
					{ID: "default-profile", Name: "Default"},
					{ID: "work-profile", Name: "Work"},
				},
				Rules: []config.Rule{
					{
						Name:      "Work Domain",
						Pattern:   "^https://work\\.example\\.com",
						Scope:     config.ScopeDomain,
						ProfileID: "work-profile",
					},
				},
			},
			url: "https://work.example.com/path",
			want: MatchResult{
				Rule: &config.Rule{
					Name:      "Work Domain",
					Pattern:   "^https://work\\.example\\.com",
					Scope:     config.ScopeDomain,
					ProfileID: "work-profile",
				},
				ProfileID: "work-profile",
				Incognito: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyRules(tt.cfg, tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyRules() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Compare Rule fields individually since we can't do direct struct comparison with pointers
			if (got.Rule == nil) != (tt.want.Rule == nil) {
				t.Errorf("ApplyRules() Rule = %v, want %v", got.Rule, tt.want.Rule)
			} else if got.Rule != nil && tt.want.Rule != nil {
				if got.Rule.Name != tt.want.Rule.Name ||
					got.Rule.Pattern != tt.want.Rule.Pattern ||
					got.Rule.ProfileID != tt.want.Rule.ProfileID ||
					got.Rule.Incognito != tt.want.Rule.Incognito ||
					got.Rule.Scope != tt.want.Rule.Scope {
					t.Errorf("ApplyRules() Rule = %v, want %v", got.Rule, tt.want.Rule)
				}
			}

			if got.ProfileID != tt.want.ProfileID {
				t.Errorf("ApplyRules() ProfileID = %v, want %v", got.ProfileID, tt.want.ProfileID)
			}
			if got.Incognito != tt.want.Incognito {
				t.Errorf("ApplyRules() Incognito = %v, want %v", got.Incognito, tt.want.Incognito)
			}
		})
	}
}
