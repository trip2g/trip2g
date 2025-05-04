package model

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Lifetime string

var reLifetime = regexp.MustCompile(`^\s*([+-]?)\s*(\d+)\s+(day|days|hour|hours|minute|minutes|second|seconds)\s*$`)

var ErrUnknownLifetimeUnit = errors.New("unknown lifetime unit")

func (lt Lifetime) Validate() error {
	if reLifetime.MatchString(strings.ToLower(string(lt))) {
		return nil
	}

	return fmt.Errorf("invalid lifetime format: %q", lt)
}

func (lt Lifetime) Duration() (time.Duration, error) {
	s := strings.ToLower(string(lt))

	matches := reLifetime.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid lifetime format: %q", lt)
	}

	signStr := matches[1]
	valueStr := matches[2]
	unit := matches[3]

	n, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, err
	}

	if signStr == "-" {
		n = -n
	}

	switch unit {
	case "day", "days":
		return time.Duration(n) * 24 * time.Hour, nil
	case "hour", "hours":
		return time.Duration(n) * time.Hour, nil
	case "minute", "minutes":
		return time.Duration(n) * time.Minute, nil
	case "second", "seconds":
		return time.Duration(n) * time.Second, nil
	default:
		return 0, ErrUnknownLifetimeUnit
	}
}
