package awsconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chrispruitt/aws-sso/internal/auth"
)

// FormatProfile renders a single profile as an AWS config stanza.
func FormatProfile(p auth.Profile) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[profile %s]\n", p.ProfileName)
	fmt.Fprintf(&sb, "sso_start_url = %s\n", p.StartURL)
	fmt.Fprintf(&sb, "sso_region = %s\n", p.SSORegion)
	fmt.Fprintf(&sb, "sso_account_name = %s\n", p.AccountName)
	fmt.Fprintf(&sb, "sso_account_id = %s\n", p.AccountID)
	fmt.Fprintf(&sb, "sso_role_name = %s\n", p.RoleName)
	fmt.Fprintf(&sb, "region = %s\n", p.Region)
	if p.Output != "" {
		fmt.Fprintf(&sb, "output = %s\n", p.Output)
	}

	// Write extra config keys in deterministic order.
	if len(p.ExtraConfig) > 0 {
		keys := make([]string, 0, len(p.ExtraConfig))
		for k := range p.ExtraConfig {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&sb, "%s = %s\n", k, p.ExtraConfig[k])
		}
	}

	fmt.Fprintf(&sb, "sso_auto_populated = true\n")
	return sb.String()
}

// WriteProfiles merges the supplied profiles into the AWS config file.
// Existing auto-populated profiles are replaced; manually created profiles are untouched.
// Returns the number of profiles added and updated.
func WriteProfiles(configPath string, profiles []auth.Profile) (added, updated int, err error) {
	if err = os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return 0, 0, fmt.Errorf("creating config directory: %w", err)
	}

	sections, err := parseConfig(configPath)
	if err != nil && !os.IsNotExist(err) {
		return 0, 0, fmt.Errorf("reading config file: %w", err)
	}

	// Index existing sections by name.
	existing := make(map[string]*configSection, len(sections))
	for i := range sections {
		existing[sections[i].name] = &sections[i]
	}

	// Decide what to add/update.
	for _, p := range profiles {
		sectionName := "profile " + p.ProfileName
		if sec, ok := existing[sectionName]; ok {
			if sec.autoPopulated {
				sec.text = FormatProfile(p)
				updated++
			}
			// Manually created profile — leave untouched.
		} else {
			sections = append(sections, configSection{
				name:          sectionName,
				text:          FormatProfile(p),
				autoPopulated: true,
			})
			added++
		}
	}

	return added, updated, writeConfig(configPath, sections)
}

// ---- INI parser ----

type configSection struct {
	name          string // text between [ and ] e.g. "profile foo"
	text          string // full text of the section including header line
	autoPopulated bool
}

func parseConfig(path string) ([]configSection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var sections []configSection
	var cur *configSection

	flush := func() {
		if cur != nil {
			// Trim a single trailing blank line that belongs to the gap
			// between sections rather than the section itself, then re-add
			// it during write so sections are separated by one blank line.
			sections = append(sections, *cur)
			cur = nil
		}
	}

	for _, line := range lines {
		if len(line) > 0 && line[0] == '[' {
			end := strings.LastIndex(line, "]")
			if end > 0 {
				flush()
				name := line[1:end]
				cur = &configSection{name: name, text: line + "\n"}
				continue
			}
		}
		if cur == nil {
			// Preamble (comments / blank lines before the first section).
			cur = &configSection{name: "", text: line + "\n"}
			continue
		}
		cur.text += line + "\n"
		trimmed := strings.TrimSpace(line)
		if key, val, ok := splitKV(trimmed); ok && key == "sso_auto_populated" && val == "true" {
			cur.autoPopulated = true
		}
	}
	flush()

	return sections, nil
}

func writeConfig(path string, sections []configSection) error {
	var sb strings.Builder
	for i, sec := range sections {
		if i > 0 && sec.name != "" {
			// Ensure sections are separated by exactly one blank line.
			prev := sections[i-1].text
			if !strings.HasSuffix(prev, "\n\n") {
				sb.WriteString("\n")
			}
		}
		sb.WriteString(sec.text)
	}

	content := sb.String()
	// Ensure file ends with exactly one newline.
	content = strings.TrimRight(content, "\n") + "\n"

	return os.WriteFile(path, []byte(content), 0600)
}

func splitKV(line string) (key, value string, ok bool) {
	idx := strings.Index(line, "=")
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}
