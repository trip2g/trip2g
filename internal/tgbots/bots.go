package tgbots

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerFunc func(ctx context.Context, io *HandlerIO, update tgbotapi.Update) error

type Config struct {
	WebhookPathPrefix string
}

func DefaultConfig() Config {
	return Config{
		WebhookPathPrefix: "/graphql/tg/webhook",
	}
}

type botInfo struct {
	ID  int64
	Bot *tgbotapi.BotAPI
}

type TgBots struct {
	mu sync.Mutex

	cancelMap  map[int64]context.CancelFunc
	webhookMap map[string]*HandlerIO // Map webhook path to bot info

	handlerIOMap map[int64]*HandlerIO

	env    Env
	config Config
	logger logger.Logger

	webhookURL *url.URL
	handler    HandlerFunc
}

type Env interface {
	ListEnabledTgBots(ctx context.Context) ([]db.TgBot, error)
	TgBot(ctx context.Context, id int64) (db.TgBot, error)
	Logger() logger.Logger
	PublicURL() string
}

func New(ctx context.Context, env Env, config Config) (*TgBots, error) {
	bots := TgBots{
		cancelMap:    make(map[int64]context.CancelFunc),
		webhookMap:   make(map[string]*HandlerIO),
		handlerIOMap: make(map[int64]*HandlerIO),

		env:    env,
		config: config,
		logger: logger.WithPrefix(env.Logger(), "tgbots:"),
	}

	publicURL := env.PublicURL()
	if publicURL != "" {
		webhookURL, err := url.Parse(publicURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public URL: %w", err)
		}

		webhookURL.Path = config.WebhookPathPrefix
		bots.webhookURL = webhookURL
	}

	bots.handler = func(ctx context.Context, io *HandlerIO, update tgbotapi.Update) error {
		bots.logger.Debug("received update", "update", update)
		return nil
	}

	activeBots, err := env.ListEnabledTgBots(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled tg bots: %w", err)
	}

	for _, bot := range activeBots {
		bots.StartTgBot(ctx, bot.ID)
	}

	return &bots, nil
}

func (io *TgBots) SetHandler(handler HandlerFunc) {
	io.handler = handler
}

func (io *TgBots) StartTgBot(ctx context.Context, id int64) {
	env := io.env

	// extract the env with a sql transaction
	req, err := appreq.FromCtx(ctx)
	if err == nil {
		reqEnv, ok := req.Env.(Env)
		if ok {
			env = reqEnv
		}
	}

	botConfig, err := env.TgBot(ctx, id)
	if err != nil {
		io.logger.Error("failed to get tg bot data", "id", id, "error", err)
		return
	}

	bot, err := tgbotapi.NewBotAPI(botConfig.Token)
	if err != nil {
		io.logger.Error("failed to create tg bot", "id", id, "error", err)
		return
	}

	handlerIO := HandlerIO{bot: bot}

	// Store bot instance in botMap
	io.mu.Lock()
	io.handlerIOMap[id] = &handlerIO
	io.mu.Unlock()

	if io.webhookURL != nil {
		io.registerWebhook(id, &handlerIO)
		return
	}

	// Remove webhook if it was previously set
	io.removeWebhook(id, bot)

	io.mu.Lock()
	defer io.mu.Unlock()

	cancel, exists := io.cancelMap[id]
	if exists {
		io.logger.Info("stop existing tg bot", "id", id)
		cancel()
	}

	ctx, cancel = context.WithCancel(ctx)
	io.cancelMap[id] = cancel

	go func() {
		io.logger.Info("starting tg bot loop", "id", id)

		updateConfig := tgbotapi.NewUpdate(0)
		updateConfig.Timeout = 60

		updates := bot.GetUpdatesChan(updateConfig)

		for {
			select {
			case <-ctx.Done():
				io.logger.Info("stopping tg bot loop", "id", id)
				bot.StopReceivingUpdates()
				return

			case update := <-updates:
				if io.handler != nil {
					handlerErr := io.handler(ctx, &handlerIO, update)
					if handlerErr != nil {
						io.logger.Error("failed to handle update", "botID", id, "error", handlerErr)
					}
				} else {
					io.logger.Warn("no handler set for update", "botID", id, "update", update)
				}
			}
		}
	}()
}

func (io *TgBots) ProcessWebhookRequest(path string, getBody func() []byte) bool {
	io.mu.Lock()
	handleIO, webhookExists := io.webhookMap[path]
	io.mu.Unlock()

	if !webhookExists {
		return false
	}

	var update tgbotapi.Update

	err := json.Unmarshal(getBody(), &update)
	if err != nil {
		io.logger.Error("failed to unmarshal update", "error", err)
		return true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if io.handler != nil {
		err = io.handler(ctx, handleIO, update)
		if err != nil {
			io.logger.Error("failed to handle webhook update", "botID", handleIO.BotID(), "error", err)
		}
	}

	return true
}

func (io *TgBots) StopTgBot(ctx context.Context, id int64) {
	io.mu.Lock()
	defer io.mu.Unlock()

	cancel, exists := io.cancelMap[id]
	if exists {
		io.logger.Info("stopping tg bot", "id", id)
		cancel()
		delete(io.cancelMap, id)
	}

	// Remove from botMap
	delete(io.handlerIOMap, id)

	// Remove webhook mapping if exists
	for path, info := range io.webhookMap {
		if info.BotID() == id {
			io.removeWebhookDuringStop(id, info)
			delete(io.webhookMap, path)
			break // There should only be one webhook per bot
		}
	}
}

func (io *TgBots) Stop(ctx context.Context) {
	io.mu.Lock()
	defer io.mu.Unlock()

	// Remove all webhooks if webhooks are enabled
	if io.webhookURL != nil {
		io.removeAllWebhooks()
	}

	// Cancel all running bot goroutines
	for id, cancel := range io.cancelMap {
		io.logger.Info("stopping tg bot", "id", id)
		cancel()
	}

	// Clear all mappings
	io.cancelMap = make(map[int64]context.CancelFunc)
	io.webhookMap = make(map[string]*HandlerIO)
	io.handlerIOMap = make(map[int64]*HandlerIO)
}

// registerWebhook registers a webhook for the given bot.
func (io *TgBots) registerWebhook(id int64, handlerIO *HandlerIO) {
	webhookPath := fmt.Sprintf("%s/%s", io.webhookURL.Path, handlerIO.token)
	fullWebhookURL := *io.webhookURL
	fullWebhookURL.Path = webhookPath

	webhookConfig, webhookErr := tgbotapi.NewWebhook(fullWebhookURL.String())
	if webhookErr != nil {
		io.logger.Error("failed to create webhook config", "id", id, "error", webhookErr)
		return
	}

	_, webhookErr = handlerIO.bot.Request(webhookConfig)
	if webhookErr != nil {
		io.logger.Error("failed to set webhook", "id", id, "error", webhookErr)
		return
	}

	io.mu.Lock()
	io.webhookMap[webhookPath] = handlerIO
	io.mu.Unlock()

	io.logger.Info("webhook registered", "id", id)
}

// removeWebhook removes a webhook for the given bot.
func (io *TgBots) removeWebhook(id int64, bot *tgbotapi.BotAPI) {
	webhookConfig, removeErr := tgbotapi.NewWebhookWithCert("", nil)
	if removeErr != nil {
		io.logger.Error("failed to create webhook removal config", "id", id, "error", removeErr)
		return
	}

	_, removeErr = bot.Request(webhookConfig)
	if removeErr != nil {
		io.logger.Error("failed to remove webhook", "id", id, "error", removeErr)
	} else {
		io.logger.Info("webhook removed", "id", id)
	}
}

// removeWebhookDuringStop removes a webhook during bot stop.
func (io *TgBots) removeWebhookDuringStop(id int64, handlerIO *HandlerIO) {
	webhookConfig, removeErr := tgbotapi.NewWebhookWithCert("", nil)
	if removeErr != nil {
		io.logger.Error("failed to create webhook removal config during bot stop", "id", id, "error", removeErr)
		return
	}

	_, removeErr = handlerIO.bot.Request(webhookConfig)
	if removeErr != nil {
		io.logger.Error("failed to remove webhook during bot stop", "id", id, "error", removeErr)
	} else {
		io.logger.Info("webhook removed during bot stop", "id", id)
	}
}

// removeAllWebhooks removes all registered webhooks during shutdown.
func (io *TgBots) removeAllWebhooks() {
	for _, info := range io.webhookMap {
		webhookConfig, removeErr := tgbotapi.NewWebhookWithCert("", nil)
		if removeErr != nil {
			io.logger.Error("failed to create webhook removal config during shutdown", "botID", info.BotID(), "error", removeErr)
			continue
		}

		_, removeErr = info.bot.Request(webhookConfig)
		if removeErr != nil {
			io.logger.Error("failed to remove webhook during shutdown", "botID", info.BotID(), "error", removeErr)
		} else {
			io.logger.Info("webhook removed during shutdown", "botID", info.BotID())
		}
	}
}
