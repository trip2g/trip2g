package telegram

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// HandleRateLimit checks if error is "Too Many Requests" and returns retry delay.
func HandleRateLimit(err error) (bool, time.Duration) {
	if err == nil {
		return false, 0
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Too Many Requests") {
		return false, 0
	}

	// Try to parse "retry after X" from error message
	re := regexp.MustCompile(`retry after (\d+)`)
	matches := re.FindStringSubmatch(errMsg)

	seconds := 10 // default delay
	if len(matches) > 1 {
		parsed, parseErr := strconv.Atoi(matches[1])
		if parseErr == nil {
			seconds = parsed
		}
	}

	// Add +1 second to the delay
	return true, time.Duration(seconds+1) * time.Second
}
