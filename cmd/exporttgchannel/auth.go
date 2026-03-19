package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-faster/errors"
	tdsession "github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

func runAuth(ctx context.Context, sessionPath string, args []string) error {
	fs := flag.NewFlagSet("auth", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Println("usage: exporttgchannel auth\n\nReads TELEGRAM_API_ID and TELEGRAM_API_HASH from environment,\nthen runs an interactive login flow and saves the session.")
	}
	_ = fs.Parse(args)

	apiID, apiHash, err := getAPICredentials()
	if err != nil {
		return err
	}

	storage := &tdsession.StorageMemory{}
	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: storage,
	})

	var phone string

	err = client.Run(ctx, func(ctx context.Context) error {
		status, statusErr := client.Auth().Status(ctx)
		if statusErr == nil && status.Authorized {
			self, selfErr := client.Self(ctx)
			if selfErr == nil {
				phone = self.Phone
				fmt.Printf("already logged in as %s\n", displayName(self))
				return nil
			}
		}

		phone = readLine("Phone number (e.g. +79001234567): ")
		phone = strings.TrimSpace(phone)
		if phone == "" {
			return errors.New("phone number is required")
		}

		fmt.Println("Sending code...")
		sentCode, sendErr := client.Auth().SendCode(ctx, phone, auth.SendCodeOptions{})
		if sendErr != nil {
			return fmt.Errorf("send code: %w", sendErr)
		}

		var codeHash string
		switch sc := sentCode.(type) {
		case *tg.AuthSentCode:
			codeHash = sc.PhoneCodeHash
			switch sc.Type.(type) {
			case *tg.AuthSentCodeTypeApp:
				fmt.Println("Code sent to Telegram app.")
			case *tg.AuthSentCodeTypeSMS:
				fmt.Println("Code sent via SMS.")
			default:
				fmt.Println("Code sent.")
			}
		case *tg.AuthSentCodeSuccess:
			fmt.Println("Already authorized.")
			return nil
		default:
			return fmt.Errorf("unexpected code type: %T", sentCode)
		}

		code := readLine("Code: ")
		code = strings.TrimSpace(code)
		if code == "" {
			return errors.New("code is required")
		}

		_, signInErr := client.Auth().SignIn(ctx, phone, code, codeHash)
		if signInErr != nil {
			if errors.Is(signInErr, auth.ErrPasswordAuthNeeded) {
				fmt.Println("2FA required.")
				pwd := readLine("2FA password: ")
				if _, pwdErr := client.Auth().Password(ctx, strings.TrimSpace(pwd)); pwdErr != nil {
					return fmt.Errorf("2FA: %w", pwdErr)
				}
			} else {
				var signUpReq *auth.SignUpRequired
				if errors.As(signInErr, &signUpReq) {
					return errors.New("sign up required — please register in the Telegram app first")
				}
				return fmt.Errorf("sign in: %w", signInErr)
			}
		}

		self, selfErr := client.Self(ctx)
		if selfErr != nil {
			return fmt.Errorf("get self: %w", selfErr)
		}
		phone = self.Phone
		fmt.Printf("Logged in as %s\n", displayName(self))
		return nil
	})
	if err != nil {
		return err
	}

	sessionData, err := storage.Bytes(nil)
	if err != nil {
		return fmt.Errorf("export session: %w", err)
	}

	sf := &sessionFile{
		APIID:       apiID,
		APIHash:     apiHash,
		Phone:       phone,
		SessionData: base64.StdEncoding.EncodeToString(sessionData),
	}
	if err := saveSession(sessionPath, sf); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	fmt.Printf("Session saved to %s\n", sessionPath)
	return nil
}

// getAPICredentials reads TELEGRAM_API_ID and TELEGRAM_API_HASH from env,
// falling back to interactive prompts.
func getAPICredentials() (int, string, error) {
	apiIDStr := os.Getenv("TELEGRAM_API_ID")
	apiHash := os.Getenv("TELEGRAM_API_HASH")

	if apiIDStr == "" {
		apiIDStr = strings.TrimSpace(readLine("TELEGRAM_API_ID: "))
	}
	if apiHash == "" {
		apiHash = strings.TrimSpace(readLine("TELEGRAM_API_HASH: "))
	}

	apiID, err := strconv.Atoi(strings.TrimSpace(apiIDStr))
	if err != nil || apiID == 0 {
		return 0, "", fmt.Errorf("invalid TELEGRAM_API_ID: %q", apiIDStr)
	}
	if apiHash == "" {
		return 0, "", errors.New("TELEGRAM_API_HASH is required")
	}
	return apiID, apiHash, nil
}

func readLine(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func displayName(u *tg.User) string {
	var parts []string
	if u.FirstName != "" {
		parts = append(parts, u.FirstName)
	}
	if u.LastName != "" {
		parts = append(parts, u.LastName)
	}
	if u.Username != "" {
		parts = append(parts, "@"+u.Username)
	}
	return strings.Join(parts, " ")
}
