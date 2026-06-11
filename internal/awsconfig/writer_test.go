package awsconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chrispruitt/aws-sso-config/internal/auth"
)

func TestWriteProfiles_NewFile(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ".aws", "config")

	profiles := []auth.Profile{
		{
			ProfileName: "new-dev.AWSAdministratorAccess",
			StartURL:    "https://d-9067a1ef3f.awsapps.com/start",
			SSORegion:   "us-east-1",
			AccountName: "new-dev",
			AccountID:   "905418254301",
			RoleName:    "AWSAdministratorAccess",
			Region:      "us-east-1",
			Output:      "json",
		},
		{
			ProfileName: "new-qa.AWSAdministratorAccess",
			StartURL:    "https://d-9067a1ef3f.awsapps.com/start",
			SSORegion:   "us-east-1",
			AccountName: "new-qa",
			AccountID:   "774305613089",
			RoleName:    "AWSAdministratorAccess",
			Region:      "us-east-1",
			Output:      "json",
		},
	}

	added, updated, err := WriteProfiles(cfg, profiles)
	if err != nil {
		t.Fatalf("WriteProfiles: %v", err)
	}
	if added != 2 || updated != 0 {
		t.Fatalf("expected 2 added 0 updated, got %d added %d updated", added, updated)
	}

	data, _ := os.ReadFile(cfg)
	content := string(data)

	for _, want := range []string{
		"[profile new-dev.AWSAdministratorAccess]",
		"sso_account_id = 905418254301",
		"sso_auto_populated = true",
		"[profile new-qa.AWSAdministratorAccess]",
		"sso_account_id = 774305613089",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("config missing %q\n\nGot:\n%s", want, content)
		}
	}
}

func TestWriteProfiles_PreservesManual(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ".aws", "config")

	initial := `[default]
region = us-west-2
output = text

[profile my-manual-profile]
region = eu-west-1
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
`
	_ = os.MkdirAll(filepath.Dir(cfg), 0700)
	_ = os.WriteFile(cfg, []byte(initial), 0600)

	profiles := []auth.Profile{
		{
			ProfileName: "sso-dev.ReadOnly",
			StartURL:    "https://example.awsapps.com/start",
			SSORegion:   "us-east-1",
			AccountName: "sso-dev",
			AccountID:   "123456789012",
			RoleName:    "ReadOnly",
			Region:      "us-east-1",
			Output:      "json",
		},
	}

	_, _, err := WriteProfiles(cfg, profiles)
	if err != nil {
		t.Fatalf("WriteProfiles: %v", err)
	}

	data, _ := os.ReadFile(cfg)
	content := string(data)

	if !strings.Contains(content, "[default]") {
		t.Error("default section was removed")
	}
	if !strings.Contains(content, "[profile my-manual-profile]") {
		t.Error("manual profile was removed")
	}
	if !strings.Contains(content, "aws_access_key_id = AKIAIOSFODNN7EXAMPLE") {
		t.Error("manual profile content was altered")
	}
	if !strings.Contains(content, "[profile sso-dev.ReadOnly]") {
		t.Error("new SSO profile was not added")
	}
}

func TestWriteProfiles_UpdatesAutoPopulated(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ".aws", "config")

	initial := `[profile old.AdminAccess]
sso_start_url = https://old.awsapps.com/start
sso_region = us-east-1
sso_account_name = old
sso_account_id = 000000000000
sso_role_name = AdminAccess
region = us-east-1
output = json
sso_auto_populated = true
`
	_ = os.MkdirAll(filepath.Dir(cfg), 0700)
	_ = os.WriteFile(cfg, []byte(initial), 0600)

	// Same profile name, new account ID (simulating an update).
	profiles := []auth.Profile{
		{
			ProfileName: "old.AdminAccess",
			StartURL:    "https://new.awsapps.com/start",
			SSORegion:   "us-east-1",
			AccountName: "old",
			AccountID:   "111111111111",
			RoleName:    "AdminAccess",
			Region:      "us-east-1",
			Output:      "json",
		},
	}

	added, updated, err := WriteProfiles(cfg, profiles)
	if err != nil {
		t.Fatalf("WriteProfiles: %v", err)
	}
	if added != 0 || updated != 1 {
		t.Fatalf("expected 0 added 1 updated, got %d added %d updated", added, updated)
	}

	data, _ := os.ReadFile(cfg)
	if !strings.Contains(string(data), "sso_account_id = 111111111111") {
		t.Errorf("account ID was not updated\n\nGot:\n%s", string(data))
	}
	if strings.Contains(string(data), "000000000000") {
		t.Errorf("old account ID still present\n\nGot:\n%s", string(data))
	}
}

func TestFormatProfile_ExtraConfig(t *testing.T) {
	p := auth.Profile{
		ProfileName: "dev.Admin",
		StartURL:    "https://x.awsapps.com/start",
		SSORegion:   "us-east-1",
		AccountName: "dev",
		AccountID:   "123",
		RoleName:    "Admin",
		Region:      "us-east-1",
		Output:      "json",
		ExtraConfig: map[string]string{"credential_process": "/usr/local/bin/aws-cred-helper credential-process"},
	}
	out := FormatProfile(p)
	if !strings.Contains(out, "credential_process = /usr/local/bin/aws-cred-helper credential-process") {
		t.Errorf("extra config key missing from output:\n%s", out)
	}
}
