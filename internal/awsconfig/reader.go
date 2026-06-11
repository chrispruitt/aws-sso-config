package awsconfig

import (
	"fmt"
	"os"
	"strings"
)

// ProfileConfig holds the SSO-related fields parsed from a named AWS config profile.
type ProfileConfig struct {
	StartURL  string
	SSORegion string
	AccountID string
	RoleName  string
	Region    string
}

// ReadProfile returns the config for the named profile from the given config file.
func ReadProfile(configPath, profileName string) (*ProfileConfig, error) {
	sections, err := parseConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	target := "profile " + profileName
	for _, sec := range sections {
		if sec.name != target {
			continue
		}
		cfg := &ProfileConfig{}
		for _, line := range strings.Split(sec.text, "\n") {
			key, val, ok := splitKV(strings.TrimSpace(line))
			if !ok {
				continue
			}
			switch key {
			case "sso_start_url":
				cfg.StartURL = val
			case "sso_region":
				cfg.SSORegion = val
			case "sso_account_id":
				cfg.AccountID = val
			case "sso_role_name":
				cfg.RoleName = val
			case "region":
				cfg.Region = val
			}
		}
		if cfg.StartURL == "" {
			return nil, fmt.Errorf("profile %q is missing sso_start_url — run 'aws-sso populate' first", profileName)
		}
		return cfg, nil
	}
	return nil, fmt.Errorf("profile %q not found in %s", profileName, configPath)
}

// ListProfileNames returns all named profiles from the config file.
func ListProfileNames(configPath string) ([]string, error) {
	sections, err := parseConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var names []string
	for _, sec := range sections {
		if strings.HasPrefix(sec.name, "profile ") {
			names = append(names, strings.TrimPrefix(sec.name, "profile "))
		}
	}
	return names, nil
}
