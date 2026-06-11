package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aws-sso-config",
	Short: "Populate AWS config file from SSO accounts and roles",
	Long: `aws-sso-config discovers all AWS accounts and roles available via SSO
and writes them as named profiles to your AWS config file.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
