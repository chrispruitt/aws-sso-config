package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/chrispruitt/aws-sso/internal/auth"
	"github.com/chrispruitt/aws-sso/internal/awsconfig"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login <profile>",
	Short: "Switch to an AWS SSO profile, authenticating if needed",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(_ *cobra.Command, args []string) error {
	profileName := args[0]

	cfgFile, err := resolveConfigFile()
	if err != nil {
		return err
	}

	profile, err := awsconfig.ReadProfile(cfgFile, profileName)
	if err != nil {
		return err
	}

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
