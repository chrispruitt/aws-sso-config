package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var shellInitShell string

var shellInitCmd = &cobra.Command{
	Use:   "shell-init",
	Short: "Print shell integration functions and completions",
	Long: `Print shell functions and tab-completions for awssso.

Add this line to your ~/.bashrc or ~/.zshrc:

  eval "$(aws-sso shell-init)"`,
	RunE: runShellInit,
}

func init() {
	shellInitCmd.Flags().StringVar(&shellInitShell, "shell", "", "Shell type: bash or zsh (auto-detected from $SHELL if omitted)")
	rootCmd.AddCommand(shellInitCmd)
}

func runShellInit(_ *cobra.Command, _ []string) error {
	shell := shellInitShell
	if shell == "" {
		shell = filepath.Base(os.Getenv("SHELL"))
	}
	switch strings.ToLower(shell) {
	case "bash":
		fmt.Print(bashInit)
	case "zsh":
		fmt.Print(zshInit)
	default:
		return fmt.Errorf("unsupported shell %q — use --shell bash or --shell zsh", shell)
	}
	return nil
}

const bashInit = `
awssso() {
  if [[ -z "$1" ]]; then
    echo "Usage: awssso <profile>" >&2
    return 1
  fi
  eval "$(aws-sso login "$1")"
}

awssso-logout() {
  eval "$(aws-sso logout)"
}

awssso-refresh() {
  eval "$(aws-sso refresh)"
}

awssso-export() {
  eval "$(aws-sso credentials)"
}

awssso-unset() {
  eval "$(aws-sso credentials unset)"
}

awssso-expiry() {
  aws-sso expiry
}

_awssso_completions() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local config_file="${AWS_CONFIG_FILE:-$HOME/.aws/config}"
  local profiles
  profiles=$(grep "^\[profile " "$config_file" 2>/dev/null | sed 's/\[profile //;s/\]//')
  COMPREPLY=($(compgen -W "$profiles" -- "$cur"))
}

complete -F _awssso_completions awssso
`

const zshInit = `
awssso() {
  if [[ -z "$1" ]]; then
    echo "Usage: awssso <profile>" >&2
    return 1
  fi
  eval "$(aws-sso login "$1")"
}

awssso-logout() {
  eval "$(aws-sso logout)"
}

awssso-refresh() {
  eval "$(aws-sso refresh)"
}

awssso-export() {
  eval "$(aws-sso credentials)"
}

awssso-unset() {
  eval "$(aws-sso credentials unset)"
}

awssso-expiry() {
  aws-sso expiry
}

_awssso() {
  local config_file="${AWS_CONFIG_FILE:-$HOME/.aws/config}"
  compadd $(grep "^\[profile " "$config_file" 2>/dev/null | sed 's/\[profile //;s/\]//')
}

compdef _awssso awssso
`
