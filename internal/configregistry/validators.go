package configregistry

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"trip2g/internal/model"
)

func validateSiteTitleTemplate(value string) error {
	if !strings.Contains(value, "%s") {
		return errors.New("site title template must contain %s placeholder")
	}

	return nil
}

func validateTimezone(value string) error {
	_, err := time.LoadLocation(value)
	if err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}

	return nil
}

func validateURLNormalizationMethod(value string) error {
	m := model.URLNormalizationMethod(value)
	if !m.Valid() {
		return fmt.Errorf("unknown URL normalization method: %s", value)
	}

	return nil
}
