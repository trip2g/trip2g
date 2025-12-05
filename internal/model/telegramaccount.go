package model

// TelegramAuthState represents the current state of authentication.
type TelegramAuthState string

const (
	TelegramAuthStateWaitingForCode     TelegramAuthState = "WAITING_FOR_CODE"
	TelegramAuthStateWaitingForPassword TelegramAuthState = "WAITING_FOR_PASSWORD"
	TelegramAuthStateAuthorized         TelegramAuthState = "AUTHORIZED"
	TelegramAuthStateError              TelegramAuthState = "ERROR"
)

// TelegramStartAuthResult contains the result of starting authentication.
type TelegramStartAuthResult struct {
	Phone        string
	State        TelegramAuthState
	PasswordHint string
}

// TelegramCompleteAuthResult contains the result of completing authentication.
type TelegramCompleteAuthResult struct {
	SessionData []byte
	DisplayName string
	IsPremium   bool
	APIID       int
	APIHash     string
}

// TelegramAccountDialogType represents the type of a Telegram dialog.
type TelegramAccountDialogType string

const (
	TelegramAccountDialogTypeUser    TelegramAccountDialogType = "user"
	TelegramAccountDialogTypeChannel TelegramAccountDialogType = "channel"
	TelegramAccountDialogTypeChat    TelegramAccountDialogType = "chat"
)

// TelegramAccountDialog represents a dialog (user, channel, or group) in Telegram.
// PublishTags and PublishInstantTags are resolved via GraphQL forceResolver.
type TelegramAccountDialog struct {
	AccountID int64
	ID        int64
	Username  string
	Title     string
	Type      TelegramAccountDialogType
}
