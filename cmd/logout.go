package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/chrispruitt/aws-sso-config/internal/auth"
	"github.com/chrispruitt/aws-sso-config/internal/awsconfig"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Revoke the current SSO session and unset profile env vars",
	RunE:  runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(_ *cobra.Command, _ []string) error {
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

	// Revoke server-side if we have a cached token.
	if tok, err := auth.GetCachedToken(profile.StartURL); err == nil {
		ctx := context.Background()
		if cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(profile.SSORegion)); err == nil {
			ssoClient := sso.NewFromConfig(cfg)
			_, _ = ssoClient.Logout(ctx, &sso.LogoutInput{AccessToken: aws.String(tok)})
		}
	}
	_ = auth.DeleteToken(profile.StartURL)

	fmt.Println("unset AWS_PROFILE")
	fmt.Println("unset AWS_REGION")
	fmt.Println("unset AWS_ACCOUNT")
	return nil
}
