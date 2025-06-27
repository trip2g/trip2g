package model

type TgAuthToken struct {
	ChatID int64 `json:"c"`
	BotID  int64 `json:"b"`
}
