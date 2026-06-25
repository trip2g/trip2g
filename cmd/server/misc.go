package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/notiontypes"
	"trip2g/internal/openai"

	"github.com/oklog/ulid/v2"
)

func (a *app) NotionClientByIntegrationID(integrationID int64) notiontypes.Client {
	client, err := a.notionClientManager.Get(a.ctx, a, integrationID)
	if err != nil {
		a.log.Error("failed to get notion client by integration ID", "integrationID", integrationID, "error", err)
		return nil
	}

	return client
}

func (a *app) CalculateSha256(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

func (a *app) IDHash(entity string, id int64) string {
	sha256 := sha256.New()
	fmt.Fprintf(sha256, "%s:%d", entity, id)
	return hex.EncodeToString(sha256.Sum(nil))
}

func (a *app) SearchLiveNotes(query string) ([]model.SearchResult, error) {
	return a.liveNoteLoader.Search(query)
}

func (a *app) SearchLatestNotes(query string) ([]model.SearchResult, error) {
	return a.latestNoteLoader.Search(query)
}

func (a *app) EnqueueGenerateNoteVersionEmbedding(ctx context.Context, versionID int64) error {
	return a.GenerateNoteVersionEmbeddingJob.Enqueue(ctx, versionID)
}

func (a *app) OpenAI() *openai.Client {
	return a.openaiClient
}

func (a *app) VacuumDB(ctx context.Context) error {
	// 1. Checkpoint WAL file before vacuum
	_, err := a.conn.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return fmt.Errorf("failed to checkpoint WAL: %w", err)
	}

	// 2. Reclaim unused space
	_, err = a.conn.ExecContext(ctx, "VACUUM")
	if err != nil {
		return fmt.Errorf("failed to vacuum: %w", err)
	}

	// 3. Update query planner statistics
	_, err = a.conn.ExecContext(ctx, "ANALYZE")
	if err != nil {
		return fmt.Errorf("failed to analyze: %w", err)
	}

	return nil
}

func (a *app) RecordUserNoteView(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
	err := db.WithRetry(ctx, 3, func() error {
		return a.doRecordUserNoteView(ctx, userID, note, referrerVersionID)
	})

	if err != nil {
		a.log.Error(
			"failed to record user note view",
			"error", err,
			"user_id", userID,
			"note_id", note.ID,
		)

		return
	}
}

func (a *app) doRecordUserNoteView(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) error {
	return a.WithTransaction(ctx, func(txCtx context.Context, env *app) (bool, error) {
		err := a.recordUserNoteViewTx(txCtx, env.WriteQueries, userID, note, referrerVersionID)
		return err == nil, err
	})
}

func (a *app) recordUserNoteViewTx(
	ctx context.Context,
	queries *db.WriteQueries,
	userID int64,
	note *model.NoteView,
	referrerVersionID *int64,
) error {
	const maxCount = int64(100)

	dailyParams := db.UpsertUserNoteDailyViewParams{
		UserID: userID,
		PathID: note.PathID,
	}

	dailyCount, err := queries.UpsertUserNoteDailyView(ctx, dailyParams)
	if err != nil {
		return fmt.Errorf("failed to upsert user note daily view: %w", err)
	}

	// TODO: read from the app config
	if dailyCount < maxCount {
		viewParams := db.InsertUserNoteViewParams{
			UserID:           userID,
			VersionID:        note.VersionID,
			RefererVersionID: referrerVersionID,
		}

		err = queries.InsertUserNoteView(ctx, viewParams)
		if err != nil {
			return fmt.Errorf("failed to insert user note view: %w", err)
		}

		err = queries.IncreaseUserNoteViewCount(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to increase user note view count: %w", err)
		}
	}

	return nil
}

func (a *app) GenerateUniqID() string {
	return ulid.Make().String()
}

func (a *app) GenerateAPIKey() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 64

	result := make([]byte, length)

	for i := range length {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			panic(err)
		}

		result[i] = alphabet[n.Int64()]
	}

	return string(result)
}

func (a *app) GenerateGitToken() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 64

	result := make([]byte, length)

	for i := range length {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			panic(err)
		}

		result[i] = alphabet[n.Int64()]
	}

	return string(result)
}

func (a *app) GenerateTgAttachCode() string {
	code, err := generateEightCharCode()
	if err != nil {
		// Log error and generate a fallback code
		a.Logger().Error("failed to generate attach code", "error", err)
		// Fallback to timestamp-based code if random generation fails
		return fmt.Sprintf("%08x", time.Now().Unix()%0xFFFFFFFF)
	}
	return code
}

func (a *app) GetAdminEmails(ctx context.Context) ([]string, error) {
	admins, err := a.Queries.ListAllAdmins(ctx)
	if err != nil {
		return nil, err
	}
	emails := make([]string, 0, len(admins))
	for _, ad := range admins {
		u, uErr := a.Queries.UserByID(ctx, ad.UserID)
		if uErr != nil {
			continue
		}
		if u.Email != nil && *u.Email != "" {
			emails = append(emails, *u.Email)
		}
	}
	return emails, nil
}
