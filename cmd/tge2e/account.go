package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-faster/errors"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// AuthResult contains successful auth result.
type AuthResult struct {
	SessionData []byte
	DisplayName string
	Phone       string
	IsPremium   bool
}

// AuthenticateAccount authenticates with Telegram using interactive prompts.
func AuthenticateAccount(ctx context.Context, apiID int, apiHash string) (*AuthResult, []byte, error) {
	fmt.Println("Authenticating with Telegram...")
	fmt.Printf("  Using API ID: %d\n", apiID)

	// Use memory session storage to capture session data
	sessionStorage := &MemorySessionStorage{}

	// Create client with default production DC
	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: sessionStorage,
	})

	var result *AuthResult

	err := client.Run(ctx, func(ctx context.Context) error {
		fmt.Println("  Connected to Telegram")

		// Check if already authorized
		status, statusErr := client.Auth().Status(ctx)
		if statusErr != nil {
			fmt.Printf("  Status check error: %v\n", statusErr)
		} else if status.Authorized {
			fmt.Println("  Already authorized!")
			self, selfErr := client.Self(ctx)
			if selfErr != nil {
				return fmt.Errorf("failed to get self: %w", selfErr)
			}
			result = &AuthResult{
				DisplayName: buildDisplayName(self),
				Phone:       self.Phone,
				IsPremium:   self.Premium,
			}
			fmt.Printf("  Logged in as: %s\n", result.DisplayName)
			return nil
		}

		fmt.Println("  Not authorized, starting auth flow...")

		// Get phone number from user
		phone := readLine("Enter phone number (with country code, e.g. +79001234567): ")
		phone = strings.TrimSpace(phone)
		if phone == "" {
			return fmt.Errorf("phone number is required")
		}

		// Step 1: Send code
		fmt.Println("  Sending auth code...")
		sentCode, sendErr := client.Auth().SendCode(ctx, phone, auth.SendCodeOptions{})
		if sendErr != nil {
			return fmt.Errorf("send code failed: %w", sendErr)
		}

		var codeHash string
		switch sc := sentCode.(type) {
		case *tg.AuthSentCode:
			codeHash = sc.PhoneCodeHash
			fmt.Println("  Code sent!")
			switch sc.Type.(type) {
			case *tg.AuthSentCodeTypeSMS:
				fmt.Println("  Code sent via SMS")
			case *tg.AuthSentCodeTypeApp:
				fmt.Println("  Code sent via Telegram app")
			case *tg.AuthSentCodeTypeCall:
				fmt.Println("  Code will be delivered via call")
			}
		case *tg.AuthSentCodeSuccess:
			fmt.Println("  Already authorized via sent code success")
			self, selfErr := client.Self(ctx)
			if selfErr != nil {
				return fmt.Errorf("failed to get self: %w", selfErr)
			}
			result = &AuthResult{
				DisplayName: buildDisplayName(self),
				Phone:       self.Phone,
				IsPremium:   self.Premium,
			}
			return nil
		default:
			return fmt.Errorf("unexpected sent code type: %T", sentCode)
		}

		// Get code from user
		code := readLine("Enter confirmation code: ")
		code = strings.TrimSpace(code)
		if code == "" {
			return fmt.Errorf("confirmation code is required")
		}

		// Step 2: Sign in
		fmt.Println("  Signing in...")
		_, signInErr := client.Auth().SignIn(ctx, phone, code, codeHash)
		if signInErr != nil {
			// Check if 2FA is required
			if errors.Is(signInErr, auth.ErrPasswordAuthNeeded) {
				fmt.Println("  2FA required")
				password := readLine("Enter 2FA password: ")
				password = strings.TrimSpace(password)
				if password == "" {
					return fmt.Errorf("2FA password is required")
				}
				_, twoFAErr := client.Auth().Password(ctx, password)
				if twoFAErr != nil {
					return fmt.Errorf("2FA auth failed: %w", twoFAErr)
				}
			} else {
				// Check if sign up is required
				var signUpRequired *auth.SignUpRequired
				if errors.As(signInErr, &signUpRequired) {
					return fmt.Errorf("sign up required - please register in Telegram app first")
				}
				return fmt.Errorf("sign in failed: %w", signInErr)
			}
		}

		// Get self info
		self, selfErr := client.Self(ctx)
		if selfErr != nil {
			return fmt.Errorf("failed to get self: %w", selfErr)
		}

		result = &AuthResult{
			DisplayName: buildDisplayName(self),
			Phone:       self.Phone,
			IsPremium:   self.Premium,
		}

		fmt.Printf("  Authenticated as: %s (Premium: %v)\n", result.DisplayName, self.Premium)
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return result, sessionStorage.Data, nil
}

// MemorySessionStorage stores session data in memory.
type MemorySessionStorage struct {
	Data []byte
}

// LoadSession loads session from memory.
func (s *MemorySessionStorage) LoadSession(_ context.Context) ([]byte, error) {
	if s.Data == nil {
		return nil, nil
	}
	return s.Data, nil
}

// StoreSession stores session to memory.
func (s *MemorySessionStorage) StoreSession(_ context.Context, data []byte) error {
	s.Data = make([]byte, len(data))
	copy(s.Data, data)
	return nil
}

// RunWithClient runs a function with an authenticated client.
func RunWithClient(ctx context.Context, creds *Credentials, fn func(ctx context.Context, client *telegram.Client, api *tg.Client) error) error {
	sessionData, err := creds.AccountSession()
	if err != nil {
		return fmt.Errorf("failed to decode session: %w", err)
	}

	// Create storage from session data
	storage := &MemorySessionStorage{Data: sessionData}

	// Use credentials from saved config (production DC)
	client := telegram.NewClient(creds.APIID, creds.APIHash, telegram.Options{
		SessionStorage: storage,
	})

	// Add timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	return client.Run(ctx, func(ctx context.Context) error {
		return fn(ctx, client, client.API())
	})
}

func buildDisplayName(user *tg.User) string {
	var parts []string
	if user.FirstName != "" {
		parts = append(parts, user.FirstName)
	}
	if user.LastName != "" {
		parts = append(parts, user.LastName)
	}
	if user.Username != "" {
		parts = append(parts, "@"+user.Username)
	}
	return strings.Join(parts, " ")
}

func readLine(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
