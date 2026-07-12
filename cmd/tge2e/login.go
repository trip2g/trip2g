package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
	"trip2g/internal/dataencryption"
)

func runLogin() error {
	apiID, apiHash, err := LoadAPICredentials()
	if err != nil {
		return err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var exists int
	err = db.QueryRow(`select 1 from telegram_accounts where id = 1`).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("telegram_accounts: not found (id=1) - seed the DB first: sqlite3 %s < testdata/e2e_seed.sql", dbPath)
		}
		return fmt.Errorf("failed to query telegram_accounts: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, sessionData, err := AuthenticateAccount(ctx, apiID, apiHash)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	encryptor, err := dataencryption.NewManager(dataencryption.DefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	encryptedSession, err := encryptor.EncryptData(sessionData)
	if err != nil {
		return fmt.Errorf("failed to encrypt session: %w", err)
	}

	res, err := db.Exec(`
		update telegram_accounts
		set phone = ?, session_data = ?, display_name = ?
		where id = 1
	`, result.Phone, encryptedSession, result.DisplayName)
	if err != nil {
		return fmt.Errorf("failed to update telegram_accounts: %w", err)
	}
	rows, _ := res.RowsAffected()
	fmt.Printf("telegram_accounts: updated %d row(s)\n", rows)
	fmt.Printf("Logged in as: %s (%s)\n", result.DisplayName, result.Phone)

	fmt.Println("\nNext step: run ./tge2e -db " + dbPath + " extract to discover channels and write .tg_e2e_session")
	fmt.Println("(the account must be a member/admin of the 4 test channels for discovery to succeed)")

	return nil
}
