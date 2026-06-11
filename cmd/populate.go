package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chrispruitt/aws-sso-config/internal/auth"
	"github.com/chrispruitt/aws-sso-config/internal/awsconfig"
	"github.com/spf13/cobra"
)

var (
	ssoStartURL       string
	ssoRegion         string
	defaultRegion     string
	outputFormat      string
	configFile        string
	profileSeparator  string
	dryRun            bool
	extraConfig       []string
)

var populateCmd = &cobra.Command{
	Use:   "populate",
	Short: "Populate AWS config with all SSO accounts and roles",
	Long: `Queries AWS SSO for all accessible accounts and roles, then writes
a named profile for each one to your AWS config file.

Profiles use the format: [profile <account-name><sep><role-name>]

Example:
  aws-sso-config populate \
    --start-url https://myorg.awsapps.com/start \
    --sso-region us-east-1 \
    --region us-east-1`,
	RunE: runPopulate,
}

func init() {
	// Seed flag defaults from environment so flags are optional when env is set.
	envURL := os.Getenv("AWS_DEFAULT_SSO_START_URL")
	envSSORegion := firstNonEmpty(os.Getenv("AWS_DEFAULT_SSO_REGION"), "us-east-1")
	envRegion := firstNonEmpty(os.Getenv("AWS_DEFAULT_REGION"), os.Getenv("AWS_REGION"), "us-east-1")

	populateCmd.Flags().StringVarP(&ssoStartURL, "start-url", "u", envURL, "AWS SSO start URL (env: AWS_DEFAULT_SSO_START_URL)")
	populateCmd.Flags().StringVar(&ssoRegion, "sso-region", envSSORegion, "AWS SSO OIDC region (env: AWS_DEFAULT_SSO_REGION)")
	populateCmd.Flags().StringVar(&defaultRegion, "region", envRegion, "Default region written into each profile (env: AWS_DEFAULT_REGION / AWS_REGION)")
	populateCmd.Flags().StringVar(&outputFormat, "output", "json", "Default output format written into each profile")
	populateCmd.Flags().StringVarP(&configFile, "config-file", "f", "", "Path to AWS config file (env: AWS_CONFIG_FILE, default: ~/.aws/config)")
	populateCmd.Flags().StringVar(&profileSeparator, "separator", ".", "Separator between account name and role name in profile names")
	populateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print profiles that would be written without modifying the file")
	populateCmd.Flags().StringArrayVarP(&extraConfig, "config", "c", nil, "Extra key=value pairs to add to every profile (e.g. -c credential_process=aws)")

	if envURL == "" {
		_ = populateCmd.MarkFlagRequired("start-url")
	}
	rootCmd.AddCommand(populateCmd)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func runPopulate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if configFile == "" {
		if env := os.Getenv("AWS_CONFIG_FILE"); env != "" {
			configFile = env
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolving home directory: %w", err)
			}
			configFile = filepath.Join(home, ".aws", "config")
		}
	}

	extra, err := parseExtraConfig(extraConfig)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Authenticating with AWS SSO...")
	token, err := auth.GetToken(ctx, ssoStartURL, ssoRegion)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Fetching accounts and roles...")
	profiles, err := auth.ListProfiles(ctx, auth.ListProfilesInput{
		AccessToken:      token,
		StartURL:         ssoStartURL,
		SSORegion:        ssoRegion,
		DefaultRegion:    defaultRegion,
		DefaultOutput:    outputFormat,
		ProfileSeparator: profileSeparator,
		ExtraConfig:      extra,
	})
	if err != nil {
		return fmt.Errorf("listing profiles: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d profile(s)\n", len(profiles))

	if dryRun {
		for _, p := range profiles {
			fmt.Println(awsconfig.FormatProfile(p))
		}
		return nil
	}

	added, updated, err := awsconfig.WriteProfiles(configFile, profiles)
	if err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Config updated: %d added, %d updated → %s\n", added, updated, configFile)
	return nil
}

func parseExtraConfig(pairs []string) (map[string]string, error) {
	out := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		idx := strings.Index(pair, "=")
		if idx < 0 {
			return nil, fmt.Errorf("invalid --config value %q: expected key=value", pair)
		}
		out[strings.TrimSpace(pair[:idx])] = strings.TrimSpace(pair[idx+1:])
	}
	return out, nil
}
