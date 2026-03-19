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
	"time"

	"github.com/go-faster/errors"
	tdclock "github.com/gotd/td/clock"
	tdsession "github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// stdin is a single shared reader so bufio doesn't lose buffered bytes between calls.
var stdin = bufio.NewReader(os.Stdin)

func runAuth(ctx context.Context, sessionPath string, timeOffset time.Duration, args []string) error {
	fs := flag.NewFlagSet("auth", flag.ExitOnError)
	debug := fs.Bool("debug", false, "enable gotd debug logging")
	fs.Usage = func() {
		fmt.Println("usage: exporttgchannel [--time-offset +1h] auth [--debug]\n\nReads TELEGRAM_API_ID and TELEGRAM_API_HASH from environment,\nthen runs an interactive login flow and saves the session.\n\nUse --time-offset if auth hangs with 'created too far in future' errors.")
	}
	_ = fs.Parse(args)

	apiID, apiHash, err := getAPICredentials()
	if err != nil {
		return err
	}

	var clk tdclock.Clock
	if timeOffset != 0 {
		fmt.Printf("Applying time offset: %+v\n", timeOffset)
		clk = offsetClock{offset: timeOffset}
	}

	storage := &tdsession.StorageMemory{}
	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: storage,
		Logger:         buildLogger(*debug),
		Clock:          clk,
	})

	var phone string

	fmt.Println("Connecting to Telegram...")
	err = client.Run(ctx, func(ctx context.Context) error {
		fmt.Println("Connected.")

		status, statusErr := client.Auth().Status(ctx)
		if statusErr == nil && status.Authorized {
			self, selfErr := client.Self(ctx)
			if selfErr == nil {
				phone = self.Phone
				fmt.Printf("Already logged in as %s\n", displayName(self))
				return nil
			}
		}

		phone = readLine("Phone number (e.g. +79001234567): ")
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
		if code == "" {
			return errors.New("code is required")
		}

		_, signInErr := client.Auth().SignIn(ctx, phone, code, codeHash)
		if signInErr != nil {
			if errors.Is(signInErr, auth.ErrPasswordAuthNeeded) {
				fmt.Println("2FA required.")
				pwd := readLine("2FA password: ")
				if _, pwdErr := client.Auth().Password(ctx, pwd); pwdErr != nil {
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
		apiIDStr = readLine("TELEGRAM_API_ID: ")
	}
	if apiHash == "" {
		apiHash = readLine("TELEGRAM_API_HASH: ")
	}

	apiID, err := strconv.Atoi(strings.TrimSpace(apiIDStr))
	if err != nil || apiID == 0 {
		return 0, "", fmt.Errorf("invalid TELEGRAM_API_ID: %q", apiIDStr)
	}
	if apiHash == "" {
		return 0, "", errors.New("TELEGRAM_API_HASH is required")
	}
	return apiID, strings.TrimSpace(apiHash), nil
}

func readLine(prompt string) string {
	fmt.Print(prompt)
	line, _ := stdin.ReadString('\n')
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
