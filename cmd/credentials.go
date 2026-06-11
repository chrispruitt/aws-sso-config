package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/chrispruitt/aws-sso/internal/auth"
	"github.com/chrispruitt/aws-sso/internal/awsconfig"
	"github.com/spf13/cobra"
)

var credentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "Export temporary IAM credentials for the active profile",
	RunE:  runCredentials,
}

var credentialsUnsetCmd = &cobra.Command{
	Use:   "unset",
	Short: "Unset temporary IAM credential env vars",
	RunE:  runCredentialsUnset,
}

func init() {
	credentialsCmd.AddCommand(credentialsUnsetCmd)
	rootCmd.AddCommand(credentialsCmd)
}

func runCredentials(_ *cobra.Command, _ []string) error {
	profileName := os.Getenv("AWS_PROFILE")
	if profileName == "" {
		return fmt.Errorf("AWS_PROFILE is not set — run 'awssso <profile>' first")
	}

	cfgFile, err := resolveConfigFile()
	if err != nil {
		return err
	}

	profile, err := awsconfig.ReadProfile(cfgFile, profileName)
	if err != nil {
		return err
	}

	ctx := context.Background()
	token, err := auth.GetToken(ctx, profile.StartURL, profile.SSORegion)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(profile.SSORegion))
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}

	out, err := sso.NewFromConfig(cfg).GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{
		AccessToken: aws.String(token),
		AccountId:   aws.String(profile.AccountID),
		RoleName:    aws.String(profile.RoleName),
	})
	if err != nil {
		return fmt.Errorf("getting credentials: %w", err)
	}

	rc := out.RoleCredentials
	fmt.Printf("export AWS_DEFAULT_REGION=%s\n", profile.Region)
	fmt.Printf("export AWS_ACCESS_KEY_ID=%s\n", aws.ToString(rc.AccessKeyId))
	fmt.Printf("export AWS_SECRET_ACCESS_KEY=%s\n", aws.ToString(rc.SecretAccessKey))
	fmt.Printf("export AWS_SESSION_TOKEN=%s\n", aws.ToString(rc.SessionToken))
	return nil
}

func runCredentialsUnset(_ *cobra.Command, _ []string) error {
	fmt.Println("unset AWS_ACCESS_KEY_ID")
	fmt.Println("unset AWS_SECRET_ACCESS_KEY")
	fmt.Println("unset AWS_SESSION_TOKEN")
	return nil
}
