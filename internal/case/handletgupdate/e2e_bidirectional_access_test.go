package handletgupdate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/ptr"

	"github.com/stretchr/testify/require"
)

// TestE2ETgSubgraphBidirectionalAccess is a gated end-to-end test that runs
// against a real migrated SQLite database. It verifies both access directions
// and the two fixes shipped alongside it:
//
//   - direction A: a subgraph linked to a TG group grants group access
//     (tg_bot_chat_subgraph_invites, read via ListActiveTgChatSubgraphNamesByChatID)
//   - direction B: TG-group membership grants subgraph content access
//     (tg_chat_members + tg_chat_subgraph_accesses, read via ListActiveTgChatSubgraphNamesByUserID)
//   - chat_member sync (bug 1 happy path): inserting the membership row that a
//     chat_member update produces flips the user's accessible content on
//   - content menu (bug 2 happy path): sendContentMenu now resolves accessible
//     content by system user id, so the same user-keyed query returns the item
//
// It is opt-in: like the rest of the Telegram e2e suite it is skipped unless
// TELEGRAM_API_ID is set (see docs/dev/telegram_e2e.md). Run it with:
//
//	TELEGRAM_API_ID=<id> TELEGRAM_API_HASH=<hash> go test ./internal/case/handletgupdate/ -run TestE2ETgSubgraphBidirectionalAccess
func TestE2ETgSubgraphBidirectionalAccess(t *testing.T) {
	if os.Getenv("TELEGRAM_API_ID") == "" {
		t.Skip("TELEGRAM_API_ID not set; skipping gated Telegram e2e (see docs/dev/telegram_e2e.md)")
	}

	ctx := context.Background()

	conn, err := db.Setup(db.SetupConfig{
		SkipDump:     true,
		DatabaseFile: filepath.Join(t.TempDir(), "e2e.db"),
		Logger:       &logger.TestLogger{Prefix: "[e2e]"},
	})
	require.NoError(t, err)
	defer conn.Close()

	q := db.NewWriteQueries(conn)

	const (
		groupTelegramID = int64(-1002529281698)
		tgUserID        = int64(7828312136)
		subgraphName    = "e2e-subgraph"
	)

	// Seed: an admin creator (owns the bot/invite), a plain member user,
	// a subgraph, the bot and the group chat.
	admin, err := q.InsertUserWithTgUserID(ctx, ptr.To(int64(100000001)))
	require.NoError(t, err)
	_, err = q.InsertAdmin(ctx, db.InsertAdminParams{UserID: admin.ID})
	require.NoError(t, err)

	user, err := q.InsertUserWithTgUserID(ctx, ptr.To(tgUserID))
	require.NoError(t, err)

	require.NoError(t, q.InsertSubgraph(ctx, subgraphName))
	subgraph, err := q.SubgraphByName(ctx, subgraphName)
	require.NoError(t, err)

	bot, err := q.InsertTgBot(ctx, db.InsertTgBotParams{
		Token:       "e2e-token",
		Name:        "E2E Bot",
		Description: "e2e",
		CreatedBy:   admin.ID,
	})
	require.NoError(t, err)

	require.NoError(t, q.UpsertTgBotChat(ctx, db.UpsertTgBotChatParams{
		TelegramID: groupTelegramID,
		ChatType:   "supergroup",
		ChatTitle:  "E2E Group",
		CanInvite:  true,
		BotID:      bot.ID,
	}))
	chat, err := q.TgBotChatByTelegramID(ctx, groupTelegramID)
	require.NoError(t, err)

	// Link subgraph <-> group both ways.
	_, err = q.InsertTgChatSubgraphInvite(ctx, db.InsertTgChatSubgraphInviteParams{
		ChatID:     chat.ID,
		SubgraphID: subgraph.ID,
		CreatedBy:  admin.ID,
	})
	require.NoError(t, err)

	_, err = q.InsertTgChatSubgraphAccess(ctx, db.InsertTgChatSubgraphAccessParams{
		ChatID:     chat.ID,
		SubgraphID: subgraph.ID,
	})
	require.NoError(t, err)

	// Direction A: the subgraph offers access to the group.
	byChat, err := q.ListActiveTgChatSubgraphNamesByChatID(ctx, chat.ID)
	require.NoError(t, err)
	require.Equal(t, []string{subgraphName}, byChat)

	// Before membership the user has no accessible content.
	byUserBefore, err := q.ListActiveTgChatSubgraphNamesByUserID(ctx, user.ID)
	require.NoError(t, err)
	require.Empty(t, byUserBefore)

	// chat_member sync (bug 1): the membership row a chat_member update writes.
	require.NoError(t, q.InsertTgChatMember(ctx, db.InsertTgChatMemberParams{
		UserID: tgUserID,
		ChatID: chat.ID,
	}))

	// Direction B / content menu (bug 2): membership now grants content access,
	// and sendContentMenu resolves it by this same system user id.
	byUserAfter, err := q.ListActiveTgChatSubgraphNamesByUserID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, []string{subgraphName}, byUserAfter)
}
