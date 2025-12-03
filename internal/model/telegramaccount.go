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
