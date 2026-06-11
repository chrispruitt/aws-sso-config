package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/chrispruitt/aws-sso/internal/auth"
	"github.com/chrispruitt/aws-sso/internal/awsconfig"
	"github.com/spf13/cobra"
)

var expiryCmd = &cobra.Command{
	Use:   "expiry",
	Short: "Show when the current SSO session expires",
	RunE:  runExpiry,
}

func init() {
	rootCmd.AddCommand(expiryCmd)
}

func runExpiry(_ *cobra.Command, _ []string) error {
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

	exp, err := auth.TokenExpiry(profile.StartURL)
	if err != nil {
		return fmt.Errorf("no cached session for profile %q: %w", profileName, err)
	}

	remaining := time.Until(exp)
	if remaining <= 0 {
		fmt.Fprintf(os.Stderr, "Session for %q expired at %s\n", profileName, exp.Local().Format("2006-01-02 15:04:05"))
		return nil
	}

	h := int(remaining.Hours())
	m := int(remaining.Minutes()) % 60
	fmt.Printf("Profile %q — session expires in %dh %dm (at %s)\n",
		profileName, h, m, exp.Local().Format("2006-01-02 15:04:05"))
	return nil
}
