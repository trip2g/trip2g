package model

type BackgroundQueue struct {
	Name    string
	Stopped bool
}

type BackgroundQueueID int

const (
	BackgroundDefaultQueue BackgroundQueueID = iota
	BackgroundTelegramJobQueue
	BackgroundTelegramAPICallQueue
)

func (id BackgroundQueueID) String() string {
	switch id {
	case BackgroundDefaultQueue:
		return "default"
	case BackgroundTelegramJobQueue:
		return "telegram_jobs"
	case BackgroundTelegramAPICallQueue:
		return "telegram_api_calls"
	}

	return "unknown"
}

type BackgroundTask struct {
	ID       string
	Queue    BackgroundQueueID
	Data     any
	Priority int
}
