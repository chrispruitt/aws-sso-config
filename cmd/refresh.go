package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/chrispruitt/aws-sso-config/internal/auth"
	"github.com/chrispruitt/aws-sso-config/internal/awsconfig"
	"github.com/spf13/cobra"
)

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Force a new SSO login for the current profile",
	RunE:  runRefresh,
}

func init() {
	rootCmd.AddCommand(refreshCmd)
}

func runRefresh(_ *cobra.Command, _ []string) error {
	profileName := os.Getenv("AWS_PROFILE")
	if profileName == "" {
		return fmt.Errorf("AWS_PROFILE is not set")
	}

	cfgFile, err := resolveConfigFile()
	if err != nil {
		return err
	}

	profile, err := awsconfig.ReadProfile(cfgFile, profileName)
	if err != nil {
		return err
	}

	_ = auth.DeleteToken(profile.StartURL)

	ctx := context.Background()
	fmt.Fprintln(os.Stderr, "Authenticating with AWS SSO...")
	if _, err = auth.GetToken(ctx, profile.StartURL, profile.SSORegion); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Printf("export AWS_PROFILE=%s\n", profileName)
	fmt.Printf("export AWS_REGION=%s\n", profile.Region)
	fmt.Printf("export AWS_ACCOUNT=%s\n", profile.AccountID)
	return nil
}
