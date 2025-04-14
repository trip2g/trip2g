package validator

import (
	"errors"
	"strings"
)

var ErrInvalidEmail = errors.New("invalid email format")

func CheckEmail(v string) error {
	if v == "" {
		return ErrInvalidEmail
	}

	if !strings.Contains(v, "@") {
		return ErrInvalidEmail
	}

	return nil
}
