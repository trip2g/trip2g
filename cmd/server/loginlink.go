package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"trip2g/internal/appconfig"
	"trip2g/internal/hotauthtoken"
	"trip2g/internal/model"
)

// loginLinkExpiry is how long the minted HAT stays valid; mirrors the fleet's
// admin-HAT lifetime in internal/fleet/hatauth.go.
const loginLinkExpiry = 5 * time.Minute

// runLoginLink is the entry-point for the "login-link" subcommand. It mints a
// one-time admin sign-in URL so a self-host operator can log in without SMTP
// or journalctl access. It is called before the full server boots so it never
// opens SQLite.
//
// Usage:
//
//	trip2g-server login-link
func runLoginLink() {
	config, err := appconfig.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "login-link: failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	secret := config.UserToken.Secret
	if secret == "" {
		fmt.Fprintln(os.Stderr, "login-link: JWT_SECRET is not set")
		os.Exit(1)
	}

	if config.OwnerEmail == "" {
		fmt.Fprintln(os.Stderr, "login-link: OWNER_EMAIL is not set")
		os.Exit(1)
	}

	// AdminEnter mirrors internal/fleet/hatauth.go: the owner must land as admin,
	// not just as a signed-in user, since this is the bootstrap login on a fresh box.
	manager := hotauthtoken.NewManager(hotauthtoken.Config{Secret: secret, ExpiresIn: loginLinkExpiry})
	token, err := manager.NewToken(model.HotAuthToken{Email: config.OwnerEmail, AdminEnter: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "login-link: failed to mint token: %v\n", err)
		os.Exit(1)
	}

	url := strings.TrimRight(config.PublicURL, "/") + "/_system/hat?token=" + token
	fmt.Fprintln(os.Stdout, url)
	fmt.Fprintf(os.Stdout, "Open this within 5 minutes to sign in as %s (one-time).\n", config.OwnerEmail)
}
