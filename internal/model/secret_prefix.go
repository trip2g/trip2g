package model

import (
	"errors"
	"fmt"
)

// SecretPrefix is the namespace prefix for secrets belonging to a specific entity.
// Format: "<entity_type>:<id>:" — e.g. "change_webhooks:42:".
type SecretPrefix string

var ErrInvalidSecretPrefixID = errors.New("secret prefix: id must be positive")

func ChangeWebhookSecretPrefix(id int64) (SecretPrefix, error) {
	if id <= 0 {
		return "", ErrInvalidSecretPrefixID
	}
	return SecretPrefix(fmt.Sprintf("change_webhooks:%d:", id)), nil
}

func CronWebhookSecretPrefix(id int64) (SecretPrefix, error) {
	if id <= 0 {
		return "", ErrInvalidSecretPrefixID
	}
	return SecretPrefix(fmt.Sprintf("cron_webhooks:%d:", id)), nil
}

// String returns the prefix as a plain string (includes trailing colon).
func (p SecretPrefix) String() string { return string(p) }

// Like returns a SQL LIKE pattern matching all keys under this prefix.
func (p SecretPrefix) Like() string { return string(p) + "%" }
