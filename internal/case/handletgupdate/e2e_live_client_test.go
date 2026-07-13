package handletgupdate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestE2ELiveTgSubgraphAccess is a REAL end-to-end test: it drives an actual,
// authenticated Telegram user account against a running trip2g bot and asserts
// the two fixes shipped in this PR from the client side.
//
// It does NOT poke the database directly. It shells out to testdata/tg_e2e/tg_driver.py,
// which reuses the telegram-mcp server's own tool functions (send_message,
// get_messages, list_inline_buttons, join_chat_by_link, leave_chat) to act as a
// live MCP-equivalent client, authenticated non-interactively from telegram-mcp's
// pre-generated Telethon StringSession.
//
// Flow exercised end-to-end (real client -> real Telegram -> trip2g bot -> back):
//
//  1. connectivity: the driver connects the real account and returns its id.
//  2. bug 1 (chat_member -> access): the account joins the throwaway test group
//     (a real `chat_member` update the bot now receives thanks to the
//     AllowedUpdates fix); the bot grants the account access to the linked subgraph.
//  3. bug 2 (content menu id-space): the account DMs the bot `/content`; the bot
//     replies with the accessible subgraph as an inline button instead of
//     "Ничего не найдено" (sendContentMenu now resolves content by system user id).
//  4. revocation: the account leaves the group; `/content` now returns
//     "Ничего не найдено".
//
// It is opt-in and skips cleanly with no config. Required env (all must be set):
//
//	TG_E2E_BOT           @username or numeric id of the trip2g bot to DM
//	TG_E2E_GROUP_INVITE  invite link (t.me/+hash) of the dedicated throwaway test
//	                     group that is linked to TG_E2E_SUBGRAPH on the instance
//	TG_E2E_GROUP_ID      numeric telegram id of that group (used to leave it)
//	TG_E2E_SUBGRAPH      subgraph name expected as an inline-button label
//
// Optional env:
//
//	TG_E2E_PYTHON        interpreter that has telegram-mcp's deps (default: python3)
//	TG_E2E_MCP_DIR       telegram-mcp checkout dir (default: ~/projects/telegram-mcp)
//
// See docs/dev/telegram_e2e.md ("Live TG<->subgraph access e2e") for the full
// manual setup (running instance, bot token, subgraph<->group link, session).
//
// Run it with, e.g.:
//
//	TG_E2E_BOT=@my_test_bot TG_E2E_GROUP_INVITE='https://t.me/+abcdEFGH' \
//	TG_E2E_GROUP_ID=-1002529281698 TG_E2E_SUBGRAPH=e2e-subgraph \
//	TG_E2E_PYTHON=/path/to/telegram-mcp/.venv/bin/python \
//	go test ./internal/case/handletgupdate/ -run TestE2ELiveTgSubgraphAccess -v
func TestE2ELiveTgSubgraphAccess(t *testing.T) {
	cfg := liveE2EConfig(t)

	bot := cfg["TG_E2E_BOT"]
	invite := cfg["TG_E2E_GROUP_INVITE"]
	groupID := cfg["TG_E2E_GROUP_ID"]
	subgraph := cfg["TG_E2E_SUBGRAPH"]

	// Connectivity: prove the real account is authenticated and reachable.
	accountID := strings.TrimSpace(runDriver(t, "whoami"))
	require.NotEmpty(t, accountID, "driver whoami returned no account id")
	t.Logf("connected real Telegram account id=%s", accountID)

	// Bug 1: join the group -> real chat_member update -> bot grants access.
	joinOut := runDriver(t, "join", "--link", invite)
	t.Logf("join: %s", strings.TrimSpace(joinOut))

	// Bug 2: after membership, /content must list the subgraph (not "Ничего не найдено").
	requireContentMenuGranted(t, bot, subgraph)

	// Revocation: leave the group -> access removed -> /content is empty again.
	leaveOut := runDriver(t, "leave", "--chat", groupID)
	t.Logf("leave: %s", strings.TrimSpace(leaveOut))
	requireContentMenuEmpty(t, bot)
}

// requireContentMenuGranted sends /content to the bot and waits until the reply
// exposes the subgraph as an inline button (bug 2 fixed + membership access granted).
func requireContentMenuGranted(t *testing.T, bot, subgraph string) {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		runDriver(t, "send", "--chat", bot, "--text", "/content")
		time.Sleep(4 * time.Second)
		last = runDriver(t, "buttons", "--chat", bot, "--limit", "5")
		if strings.Contains(last, subgraph) {
			t.Logf("content menu granted access to subgraph %q", subgraph)
			return
		}
	}
	t.Fatalf("content menu never listed subgraph %q; last buttons output:\n%s", subgraph, last)
}

// requireContentMenuEmpty sends /content and waits until the bot reports no
// accessible content, confirming access was revoked after leaving the group.
func requireContentMenuEmpty(t *testing.T, bot string) {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		runDriver(t, "send", "--chat", bot, "--text", "/content")
		time.Sleep(4 * time.Second)
		last = runDriver(t, "messages", "--chat", bot, "--limit", "3")
		if strings.Contains(last, "Ничего не найдено") {
			t.Log("content menu empty after leaving group (access revoked)")
			return
		}
	}
	t.Fatalf("content menu did not become empty after leaving; last messages:\n%s", last)
}

// liveE2EConfig returns the required env vars, skipping the test if any is unset.
func liveE2EConfig(t *testing.T) map[string]string {
	t.Helper()
	required := []string{"TG_E2E_BOT", "TG_E2E_GROUP_INVITE", "TG_E2E_GROUP_ID", "TG_E2E_SUBGRAPH"}
	cfg := make(map[string]string, len(required))
	for _, k := range required {
		v := os.Getenv(k)
		if v == "" {
			t.Skipf("%s not set; skipping live Telegram e2e (see docs/dev/telegram_e2e.md)", k)
		}
		cfg[k] = v
	}
	return cfg
}

// runDriver invokes the Python driver subcommand and returns its stdout (the
// driver prints results to stdout and Telethon connection logs to stderr). A
// nonzero exit fails the test with both streams for diagnosis.
func runDriver(t *testing.T, args ...string) string {
	t.Helper()

	python := os.Getenv("TG_E2E_PYTHON")
	if python == "" {
		python = "python3"
	}

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "cannot resolve test file path")
	driver := filepath.Join(filepath.Dir(thisFile), "testdata", "tg_e2e", "tg_driver.py")

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, python, append([]string{driver}, args...)...)
	cmd.Env = os.Environ()

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.NoErrorf(t, err, "driver %v failed\nstdout:\n%s\nstderr:\n%s", args, stdout.String(), stderr.String())
	return stdout.String()
}
