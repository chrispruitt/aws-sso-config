package auth

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	oidctypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/chrispruitt/aws-sso/internal/browser"
)

const oidcClientName = "aws-sso"

// Profile is a single AWS named profile derived from an SSO account + role.
type Profile struct {
	ProfileName string
	StartURL    string
	SSORegion   string
	AccountName string
	AccountID   string
	RoleName    string
	Region      string
	Output      string
	ExtraConfig map[string]string
}

// ListProfilesInput holds parameters for ListProfiles.
type ListProfilesInput struct {
	AccessToken      string
	StartURL         string
	SSORegion        string
	DefaultRegion    string
	DefaultOutput    string
	ProfileSeparator string
	ExtraConfig      map[string]string
}

// GetToken returns a valid SSO access token, using a cached one if available.
func GetToken(ctx context.Context, startURL, region string) (string, error) {
	if tok, err := loadCachedToken(startURL); err == nil {
		if exp, err := tok.expiresAtTime(); err == nil && time.Now().Before(exp.Add(-5*time.Minute)) {
			return tok.AccessToken, nil
		}
	}
	return doOIDCFlow(ctx, startURL, region)
}

func doOIDCFlow(ctx context.Context, startURL, region string) (string, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("loading AWS config: %w", err)
	}
	client := ssooidc.NewFromConfig(cfg)

	// Register a public OIDC client.
	reg, err := client.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String(oidcClientName),
		ClientType: aws.String("public"),
	})
	if err != nil {
		return "", fmt.Errorf("registering OIDC client: %w", err)
	}

	// Start device authorization.
	auth, err := client.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     reg.ClientId,
		ClientSecret: reg.ClientSecret,
		StartUrl:     aws.String(startURL),
	})
	if err != nil {
		return "", fmt.Errorf("starting device authorization: %w", err)
	}

	verifyURL := aws.ToString(auth.VerificationUriComplete)
	if verifyURL == "" {
		verifyURL = aws.ToString(auth.VerificationUri)
	}

	fmt.Fprintln(os.Stderr, "\nOpening browser for SSO login...")
	fmt.Fprintf(os.Stderr, "If the browser does not open, visit:\n  %s\n", verifyURL)
	if code := aws.ToString(auth.UserCode); code != "" {
		fmt.Fprintf(os.Stderr, "Confirm this code in the browser: %s\n", code)
	}
	fmt.Fprintln(os.Stderr, "\nWaiting for authorization (Ctrl+C to cancel)...")

	_ = browser.Open(verifyURL)

	interval := auth.Interval
	if interval == 0 {
		interval = 5
	}

	tok, err := pollForToken(ctx, client, &ssooidc.CreateTokenInput{
		ClientId:     reg.ClientId,
		ClientSecret: reg.ClientSecret,
		GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		DeviceCode:   auth.DeviceCode,
	}, interval)
	if err != nil {
		return "", err
	}

	accessToken := aws.ToString(tok.AccessToken)
	expiresAt := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	if err := saveToken(startURL, region, accessToken, expiresAt); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not cache token: %v\n", err)
	}

	return accessToken, nil
}

func pollForToken(ctx context.Context, client *ssooidc.Client, input *ssooidc.CreateTokenInput, interval int32) (*ssooidc.CreateTokenOutput, error) {
	delay := time.Duration(interval) * time.Second
	deadline := time.NewTimer(10 * time.Minute)
	defer deadline.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, fmt.Errorf("timed out waiting for browser authorization")
		case <-time.After(delay):
		}

		out, err := client.CreateToken(ctx, input)
		if err == nil {
			return out, nil
		}

		var pending *oidctypes.AuthorizationPendingException
		if errors.As(err, &pending) {
			continue
		}

		var slow *oidctypes.SlowDownException
		if errors.As(err, &slow) {
			delay += 5 * time.Second
			continue
		}

		return nil, fmt.Errorf("creating token: %w", err)
	}
}

// ListProfiles queries SSO for all accounts and roles and returns them as profiles.
func ListProfiles(ctx context.Context, in ListProfilesInput) ([]Profile, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(in.SSORegion))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	ssoClient := sso.NewFromConfig(cfg)
	token := aws.String(in.AccessToken)
	sep := in.ProfileSeparator
	if sep == "" {
		sep = "."
	}

	var profiles []Profile

	accountPages := sso.NewListAccountsPaginator(ssoClient, &sso.ListAccountsInput{
		AccessToken: token,
	})
	for accountPages.HasMorePages() {
		page, err := accountPages.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing accounts: %w", err)
		}
		for _, acct := range page.AccountList {
			accountID := aws.ToString(acct.AccountId)
			accountName := aws.ToString(acct.AccountName)

			rolePages := sso.NewListAccountRolesPaginator(ssoClient, &sso.ListAccountRolesInput{
				AccessToken: token,
				AccountId:   acct.AccountId,
			})
			for rolePages.HasMorePages() {
				rPage, err := rolePages.NextPage(ctx)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: skipping roles for %s (%s): %v\n", accountName, accountID, err)
					break
				}
				for _, role := range rPage.RoleList {
					profiles = append(profiles, Profile{
						ProfileName: sanitizeName(accountName) + sep + aws.ToString(role.RoleName),
						StartURL:    in.StartURL,
						SSORegion:   in.SSORegion,
						AccountName: accountName,
						AccountID:   accountID,
						RoleName:    aws.ToString(role.RoleName),
						Region:      in.DefaultRegion,
						Output:      in.DefaultOutput,
						ExtraConfig: in.ExtraConfig,
					})
				}
			}
		}
	}

	return profiles, nil
}

func sanitizeName(name string) string {
	return strings.ReplaceAll(name, " ", "-")
}

// ---- token cache ----

type cachedToken struct {
	StartURL    string `json:"startUrl"`
	Region      string `json:"region"`
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"`
}

var timeFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05UTC",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05.999999999Z07:00",
}

func (t *cachedToken) expiresAtTime() (time.Time, error) {
	for _, f := range timeFormats {
		if ts, err := time.Parse(f, t.ExpiresAt); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse expiry %q", t.ExpiresAt)
}

func tokenCachePath(startURL string) (string, error) {
	h := sha1.New()
	h.Write([]byte(startURL))
	name := hex.EncodeToString(h.Sum(nil)) + ".json"

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".aws", "sso", "cache", name), nil
}

func loadCachedToken(startURL string) (*cachedToken, error) {
	path, err := tokenCachePath(startURL)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tok cachedToken
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, err
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("empty access token in cache")
	}
	return &tok, nil
}

// DeleteToken removes the cached SSO token for the given start URL.
func DeleteToken(startURL string) error {
	path, err := tokenCachePath(startURL)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// TokenExpiry returns the expiry time of the cached SSO token for the given start URL.
func TokenExpiry(startURL string) (time.Time, error) {
	tok, err := loadCachedToken(startURL)
	if err != nil {
		return time.Time{}, err
	}
	return tok.expiresAtTime()
}

// GetCachedToken returns the raw access token from cache without triggering re-auth.
func GetCachedToken(startURL string) (string, error) {
	tok, err := loadCachedToken(startURL)
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

func saveToken(startURL, region, accessToken string, expiresAt time.Time) error {
	path, err := tokenCachePath(startURL)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tok := cachedToken{
		StartURL:    startURL,
		Region:      region,
		AccessToken: accessToken,
		ExpiresAt:   expiresAt.UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
