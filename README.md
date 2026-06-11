# aws-sso-config

A Go CLI that discovers every AWS account and role available to you via AWS SSO and writes them as named profiles to your `~/.aws/config` file.

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
go install github.com/chrispruitt/aws-sso-config@latest
```

### Build from source

Requires Go 1.24+.

```bash
git clone https://github.com/chrispruitt/aws-sso-config.git
cd aws-sso-config
go build -o aws-sso-config .
sudo mv aws-sso-config /usr/local/bin/
```

### Docker

```bash
docker build -t aws-sso-config .
```

---

## Usage

### Basic

```bash
aws-sso-config populate \
  --start-url https://<your-org>.awsapps.com/start \
  --sso-region us-east-1 \
  --region us-east-1
```

On first run (or after token expiry) a browser window opens for SSO login. If the browser does not open automatically, the URL and user code are printed to stderr.

### Dry run

Preview what would be written without modifying the config file:

```bash
aws-sso-config populate --dry-run
```

### Docker

Mount your `~/.aws` directory so the tool can read the token cache and write the config file:

```bash
docker run --rm \
  -v "$HOME/.aws:/root/.aws" \
  -e AWS_DEFAULT_SSO_START_URL \
  -e AWS_DEFAULT_SSO_REGION \
  -e AWS_CONFIG_FILE \
  aws-sso-config populate
```

> **Note:** The Docker container cannot open a browser. On first run the URL and confirmation code are printed; visit the URL manually to complete login.

---

## Environment variables

All flags have environment variable equivalents. Set these to avoid passing flags on every run.

| Variable | Flag equivalent | Description |
|---|---|---|
| `AWS_DEFAULT_SSO_START_URL` | `--start-url` | SSO portal URL |
| `AWS_DEFAULT_SSO_REGION` | `--sso-region` | SSO OIDC region |
| `AWS_DEFAULT_REGION` / `AWS_REGION` | `--region` | Default region written into each profile |
| `AWS_CONFIG_FILE` | `--config-file` | Path to the config file to write (defaults to `~/.aws/config`) |

---

## Flags

```
aws-sso-config populate [flags]

Flags:
  -u, --start-url string      AWS SSO start URL (env: AWS_DEFAULT_SSO_START_URL)
      --sso-region string     AWS SSO OIDC region (env: AWS_DEFAULT_SSO_REGION) (default "us-east-1")
      --region string         Default region written into each profile (env: AWS_DEFAULT_REGION / AWS_REGION) (default "us-east-1")
      --output string         Default output format written into each profile (default "json")
  -f, --config-file string    Path to AWS config file (env: AWS_CONFIG_FILE, default: ~/.aws/config)
      --separator string      Separator between account name and role name (default ".")
      --dry-run               Print profiles that would be written without modifying the file
  -c, --config stringArray    Extra key=value pairs added to every profile (repeatable)
  -h, --help                  help for populate
```

---

## Token caching

After a successful login the access token is cached at `~/.aws/sso/cache/<sha1-of-start-url>.json`, compatible with the AWS CLI cache. Subsequent runs reuse the cached token until it expires (typically 8 hours).

---

## Platform support

| Platform | Browser open |
|---|---|
| macOS | `open` |
| Linux | `xdg-open` |
| WSL | `wslview` (wslu), falls back to `cmd.exe /c start` |
| Docker | URL printed to stderr — visit manually |
