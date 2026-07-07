package tgbots

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"
)

// fakeTgServer is a minimal Telegram Bot API stub. It records setWebhook
// params and serves getUpdates from the updates channel.
type fakeTgServer struct {
	srv *httptest.Server

	mu        sync.Mutex
	setParams []url.Values

	updates chan string // JSON of a single Update to deliver via getUpdates
}

func newFakeTgServer(t *testing.T) *fakeTgServer {
	t.Helper()

	f := &fakeTgServer{updates: make(chan string, 4)}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		_ = r.ParseForm()

		w.Header().Set("Content-Type", "application/json")

		switch method {
		case "getMe":
			fmt.Fprint(w, `{"ok":true,"result":{"id":100,"is_bot":true,"first_name":"Test","username":"testbot"}}`)
		case "setWebhook", "deleteWebhook":
			f.mu.Lock()
			f.setParams = append(f.setParams, r.PostForm)
			f.mu.Unlock()
			fmt.Fprint(w, `{"ok":true,"result":true}`)
		case "getUpdates":
			select {
			case u := <-f.updates:
				fmt.Fprintf(w, `{"ok":true,"result":[%s]}`, u)
			case <-time.After(100 * time.Millisecond):
				fmt.Fprint(w, `{"ok":true,"result":[]}`)
			}
		default:
			fmt.Fprint(w, `{"ok":true,"result":true}`)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeTgServer) endpoint() string {
	return f.srv.URL + "/bot%s/%s"
}

func (f *fakeTgServer) setWebhookParams() []url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]url.Values, 0, len(f.setParams))
	for _, p := range f.setParams {
		if p.Get("url") != "" { // real setWebhook calls, not removals
			out = append(out, p)
		}
	}
	return out
}

func (f *fakeTgServer) newBot(t *testing.T, token string) *tgbotapi.BotAPI {
	t.Helper()
	bot, err := tgbotapi.NewBotAPIWithAPIEndpoint(token, f.endpoint())
	require.NoError(t, err)
	return bot
}

func TestWebhookTokenDerivation(t *testing.T) {
	const botToken1 = "123456:AAErealBotTokenNumberOne_abcdefghijklmno"
	const botToken2 = "654321:AAErealBotTokenNumberTwo_abcdefghijklmno"

	path1 := webhookPathToken(botToken1)
	path2 := webhookPathToken(botToken2)
	secret1 := webhookSecretToken(botToken1)

	require.NotEmpty(t, path1)
	require.Equal(t, path1, webhookPathToken(botToken1), "derivation must be stable")
	require.NotEqual(t, path1, path2, "different bots must get different path tokens")
	require.NotEqual(t, path1, secret1, "path token must differ from secret token")
	require.NotContains(t, path1, botToken1, "raw bot token must not leak into the path")
	require.NotContains(t, secret1, botToken1, "raw bot token must not leak into the secret")
	// Telegram secret_token allows only A-Z, a-z, 0-9, _ and -.
	for _, r := range secret1 {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
		require.True(t, ok, "secret token contains invalid char %q", r)
	}
}

func TestRegisterWebhookDistinctPathsAndSecret(t *testing.T) {
	fake := newFakeTgServer(t)

	const botToken1 = "111:AAEtokenOne_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const botToken2 = "222:AAEtokenTwo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	webhookURL, err := url.Parse("https://example.com")
	require.NoError(t, err)
	webhookURL.Path = DefaultConfig().WebhookPathPrefix

	bots := &TgBots{
		cancelMap:    make(map[int64]context.CancelFunc),
		webhookMap:   make(map[string]*HandlerIO),
		handlerIOMap: make(map[int64]*HandlerIO),
		logger:       &logger.DummyLogger{},
		webhookURL:   webhookURL,
	}

	io1 := newHandlerIO(fake.newBot(t, botToken1), 1, botToken1, &logger.DummyLogger{})
	io2 := newHandlerIO(fake.newBot(t, botToken2), 2, botToken2, &logger.DummyLogger{})

	bots.registerWebhook(1, io1)
	bots.registerWebhook(2, io2)

	require.Len(t, bots.webhookMap, 2, "two bots must register two distinct webhook paths")

	paths := make(map[string]int64)
	for path, hio := range bots.webhookMap {
		require.True(t, strings.HasPrefix(path, DefaultConfig().WebhookPathPrefix+"/"))
		require.NotEqual(t, DefaultConfig().WebhookPathPrefix+"/", path, "path token must not be empty")
		require.NotContains(t, path, botToken1)
		require.NotContains(t, path, botToken2)
		paths[path] = hio.BotID()
	}
	require.Len(t, paths, 2)

	params := fake.setWebhookParams()
	require.Len(t, params, 2)
	secrets := map[string]bool{}
	for _, p := range params {
		require.NotEmpty(t, p.Get("secret_token"), "setWebhook must carry secret_token")
		require.NotContains(t, p.Get("url"), botToken1)
		require.NotContains(t, p.Get("url"), botToken2)
		secrets[p.Get("secret_token")] = true
	}
	require.Len(t, secrets, 2, "secrets must be distinct per bot")
}

func TestProcessWebhookRequestSecretCheck(t *testing.T) {
	const path = "/_system/tg/webhook/abc123"
	const secret = "s3cr3t"

	newBots := func(handled *bool) *TgBots {
		hio := &HandlerIO{dbBotID: 7, webhookSecret: secret, logger: &logger.DummyLogger{}}
		return &TgBots{
			webhookMap: map[string]*HandlerIO{path: hio},
			logger:     &logger.DummyLogger{},
			handler: func(ctx context.Context, io *HandlerIO, update tgbotapi.Update) error {
				*handled = true
				return nil
			},
		}
	}

	body := func() []byte { return []byte(`{"update_id":1}`) }

	t.Run("secret mismatch rejected with 403 before handler", func(t *testing.T) {
		var handlerCalled bool
		bots := newBots(&handlerCalled)

		handled, status := bots.ProcessWebhookRequest(path, "wrong", body)
		require.True(t, handled)
		require.Equal(t, http.StatusForbidden, status)
		require.False(t, handlerCalled, "handler must not run on secret mismatch")
	})

	t.Run("missing secret rejected", func(t *testing.T) {
		var handlerCalled bool
		bots := newBots(&handlerCalled)

		handled, status := bots.ProcessWebhookRequest(path, "", body)
		require.True(t, handled)
		require.Equal(t, http.StatusForbidden, status)
		require.False(t, handlerCalled)
	})

	t.Run("secret match processed", func(t *testing.T) {
		var handlerCalled bool
		bots := newBots(&handlerCalled)

		handled, status := bots.ProcessWebhookRequest(path, secret, body)
		require.True(t, handled)
		require.Equal(t, http.StatusOK, status)
		require.True(t, handlerCalled)
	})

	t.Run("unknown path not handled", func(t *testing.T) {
		var handlerCalled bool
		bots := newBots(&handlerCalled)

		handled, _ := bots.ProcessWebhookRequest("/_system/tg/webhook/other", secret, body)
		require.False(t, handled)
		require.False(t, handlerCalled)
	})
}

func TestStartTgBotGoroutinesSurviveRequestCtxCancel(t *testing.T) {
	fake := newFakeTgServer(t)

	const botToken = "333:AAEtokenThree_cccccccccccccccccccccccccccc"
	botRow := db.TgBot{ID: 1, Token: botToken, Enabled: true}

	permCtxCh := make(chan context.Context, 1)
	permGate := make(chan struct{})

	appEnv := &EnvMock{
		LoggerFunc:    func() logger.Logger { return &logger.DummyLogger{} },
		PublicURLFunc: func() string { return "" },
		TgBotFunc: func(ctx context.Context, id int64) (db.TgBot, error) {
			t.Error("sync bot lookup must use the request env when present")
			return botRow, nil
		},
		TgBotChatsByBotIDFunc: func(ctx context.Context, arg db.TgBotChatsByBotIDParams) ([]db.TgBotChat, error) {
			permCtxCh <- ctx
			<-permGate
			return nil, nil
		},
	}

	txEnv := &EnvMock{
		LoggerFunc:    func() logger.Logger { return &logger.DummyLogger{} },
		PublicURLFunc: func() string { return "" },
		TgBotFunc: func(ctx context.Context, id int64) (db.TgBot, error) {
			return botRow, nil
		},
		TgBotChatsByBotIDFunc: func(ctx context.Context, arg db.TgBotChatsByBotIDParams) ([]db.TgBotChat, error) {
			t.Error("background permissions check must use the app env, not the tx env")
			return nil, nil
		},
	}

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	bots := &TgBots{
		ctx:          serverCtx,
		cancelMap:    make(map[int64]context.CancelFunc),
		webhookMap:   make(map[string]*HandlerIO),
		handlerIOMap: make(map[int64]*HandlerIO),
		env:          appEnv,
		config:       DefaultConfig(),
		logger:       &logger.DummyLogger{},
		newBotAPI: func(token string) (*tgbotapi.BotAPI, error) {
			return tgbotapi.NewBotAPIWithAPIEndpoint(token, fake.endpoint())
		},
	}

	handlerCtxCh := make(chan context.Context, 1)
	bots.SetHandler(func(ctx context.Context, io *HandlerIO, update tgbotapi.Update) error {
		select {
		case handlerCtxCh <- ctx:
		default:
		}
		return nil
	})

	reqCtx, reqCancel := context.WithCancel(context.Background())
	reqCtx = appreq.NewContext(reqCtx, &appreq.Request{Env: txEnv})

	bots.StartTgBot(reqCtx, 1)

	// The permissions goroutine has started and holds its ctx.
	var permCtx context.Context
	select {
	case permCtx = <-permCtxCh:
	case <-time.After(5 * time.Second):
		t.Fatal("permissions check goroutine did not start")
	}

	// Cancel the request context; long-lived goroutines must not be affected.
	reqCancel()
	close(permGate)

	require.NoError(t, permCtx.Err(), "permissions goroutine ctx must survive request ctx cancellation")

	// The polling loop must also survive: deliver an update after cancellation
	// and expect the handler to run with a live context.
	update, err := json.Marshal(tgbotapi.Update{UpdateID: 1})
	require.NoError(t, err)
	fake.updates <- string(update)

	select {
	case handlerCtx := <-handlerCtxCh:
		require.NoError(t, handlerCtx.Err(), "polling loop ctx must survive request ctx cancellation")
	case <-time.After(5 * time.Second):
		t.Fatal("polling loop did not deliver update after request ctx cancellation")
	}

	bots.StopTgBot(context.Background(), 1)
}
