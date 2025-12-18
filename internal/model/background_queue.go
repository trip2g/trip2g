package model

type BackgroundQueue struct {
	Name    string
	Stopped bool
}

type BackgroundQueueID int

const (
	BackgroundDefaultQueue BackgroundQueueID = iota
	BackgroundTelegramJobQueue
	BackgroundTelegramBotAPIQueue     // Bot API calls (telegram-bot-api)
	BackgroundTelegramAccountAPIQueue // Account API calls (MTProto/tgtd)
	BackgroundTelegramLongRunningQueue
)

func (id BackgroundQueueID) String() string {
	switch id {
	case BackgroundDefaultQueue:
		return "default"
	case BackgroundTelegramJobQueue:
		return "telegram_jobs"
	case BackgroundTelegramBotAPIQueue:
		return "telegram_bot_api"
	case BackgroundTelegramAccountAPIQueue:
		return "telegram_account_api"
	case BackgroundTelegramLongRunningQueue:
		return "telegram_long_running"
	}

	return "unknown"
}

type BackgroundTask struct {
	ID       string
	Queue    BackgroundQueueID
	Data     any
	Priority int
}
