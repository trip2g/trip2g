package configregistry

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

func validateSiteTitleTemplate(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return errors.New("value must be a string")
	}

	if !strings.Contains(s, "%s") {
		return errors.New("site title template must contain %s placeholder")
	}

	return nil
}

func validateTimezone(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return errors.New("value must be a string")
	}

	_, err := time.LoadLocation(s)
	if err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}

	return nil
}
