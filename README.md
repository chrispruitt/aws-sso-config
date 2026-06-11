# aws-sso

A Go CLI that discovers every AWS account and role available to you via AWS SSO, writes them as named profiles to your `~/.aws/config` file, and provides shell commands for switching between profiles with tab completion.

Profiles are written in the format:

```ini
[profile dev.Admin]
sso_start_url = https://myorg.awsapps.com/start
sso_region = us-east-1
sso_account_name = dev
sso_account_id = 123456789111
sso_role_name = Admin
region = us-east-1
output = json
sso_auto_populated = true
```

Auto-populated profiles are replaced on subsequent runs. Profiles you create manually are never touched.

---

## Installation

### go install

Requires Go 1.24+.

```bash
go install github.com/chrispruitt/aws-sso@latest
```

### Build from source

Requires Go 1.24+.

```bash
git clone https://github.com/chrispruitt/aws-sso.git
cd aws-sso
go build -o aws-sso .
sudo mv aws-sso /usr/local/bin/
```

### Docker

```bash
docker build -t aws-sso .
```

---

## Setup

### 1. Populate your AWS config

Run `populate` once (and again whenever your SSO account/role assignments change) to write all available profiles to `~/.aws/config`:

```bash
aws-sso populate \
  --start-url https://<your-org>.awsapps.com/start \
  --sso-region us-east-1 \
  --region us-east-1
```

On first run (or after token expiry) a browser window opens for SSO login. If it doesn't open automatically, the URL and user code are printed to stderr.

### 2. Add shell integration

Add the following line to your `~/.bashrc` or `~/.zshrc`, then restart your shell or run `source ~/.bashrc`:

```bash
eval "$(aws-sso shell-init)"
```

This defines the `awssso` function and its companions, plus tab completion for profile names.

---

## Shell commands

After adding shell integration, the following functions are available:

| Command | Description |
|---|---|
| `awssso <profile>` | Switch to a profile, authenticating via SSO if needed. Sets `AWS_PROFILE`, `AWS_REGION`, and `AWS_ACCOUNT`. |
| `awssso-logout` | Revoke the current SSO session and unset profile env vars. |
| `awssso-refresh` | Force a new SSO login for the current profile (useful when a token has expired mid-session). |
| `awssso-export` | Fetch temporary IAM credentials and export `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_SESSION_TOKEN`. |
| `awssso-unset` | Unset the temporary IAM credential env vars set by `awssso-export`. |
| `awssso-expiry` | Show how long until the current SSO session expires. |

Profile names tab-complete for `awssso`.

### Example workflow

```bash
# Switch to a profile (triggers SSO login in browser if needed)
awssso dev.Admin

# Check when the session expires
awssso-expiry

# Export temporary creds for tools that don't support SSO profiles
awssso-export

# Clean up temporary creds
awssso-unset

# Log out
awssso-logout
```

---

## CLI reference

### populate

Discovers all SSO accounts and roles and writes them as named profiles to your AWS config file.

```bash
aws-sso populate [flags]

Flags:
  -u, --start-url string      AWS SSO start URL (env: AWS_DEFAULT_SSO_START_URL)
      --sso-region string     AWS SSO OIDC region (env: AWS_DEFAULT_SSO_REGION) (default "us-east-1")
      --region string         Default region written into each profile (env: AWS_DEFAULT_REGION / AWS_REGION) (default "us-east-1")
      --output string         Default output format written into each profile (default "json")
  -f, --config-file string    Path to AWS config file (env: AWS_CONFIG_FILE, default: ~/.aws/config)
      --separator string      Separator between account name and role name (default ".")
      --dry-run               Print profiles that would be written without modifying the file
  -c, --config stringArray    Extra key=value pairs added to every profile (repeatable)
```

Use `--dry-run` to preview changes without writing:

```bash
aws-sso populate --dry-run
```

### login / logout / refresh

These are used internally by the shell functions but can also be called directly. They print eval-able shell statements to stdout and diagnostic messages to stderr.

```bash
# Prints: export AWS_PROFILE=... AWS_REGION=... AWS_ACCOUNT=...
aws-sso login <profile>

# Prints: unset AWS_PROFILE AWS_REGION AWS_ACCOUNT
aws-sso logout

# Deletes cached token, re-authenticates, prints export statements
aws-sso refresh
```

### credentials

```bash
# Prints: export AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_SESSION_TOKEN=...
aws-sso credentials

# Prints: unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN
aws-sso credentials unset
```

### expiry

```bash
aws-sso expiry
# Profile "dev.Admin" — session expires in 6h 42m (at 2024-01-15 23:30:00)
```

### shell-init

```bash
# Auto-detects shell from $SHELL
aws-sso shell-init

# Explicitly specify the shell
aws-sso shell-init --shell bash
aws-sso shell-init --shell zsh
```

---

## Environment variables

| Variable | Description |
|---|---|
| `AWS_DEFAULT_SSO_START_URL` | SSO portal URL (used by `populate`) |
| `AWS_DEFAULT_SSO_REGION` | SSO OIDC region (used by `populate`) |
| `AWS_DEFAULT_REGION` / `AWS_REGION` | Default region written into each profile |
| `AWS_CONFIG_FILE` | Path to the AWS config file (defaults to `~/.aws/config`) |

---

## Token caching

After a successful login the access token is cached at `~/.aws/sso/cache/<sha1-of-start-url>.json`, compatible with the AWS CLI token cache. Subsequent logins reuse the cached token until it expires (typically 8 hours).

---

## Docker

Mount your `~/.aws` directory so the tool can read the token cache and write the config file:

```bash
docker run --rm \
  -v "$HOME/.aws:/root/.aws" \
  -e AWS_DEFAULT_SSO_START_URL \
  -e AWS_DEFAULT_SSO_REGION \
  -e AWS_CONFIG_FILE \
  aws-sso populate
```

> **Note:** The Docker container cannot open a browser. On first run the URL and confirmation code are printed to stderr; visit the URL manually to complete login.

---

## Platform support

| Platform | Browser open |
|---|---|
| macOS | `open` |
| Linux | `xdg-open` |
| WSL | `wslview` (wslu), falls back to `cmd.exe /c start` |
| Docker | URL printed to stderr — visit manually |
