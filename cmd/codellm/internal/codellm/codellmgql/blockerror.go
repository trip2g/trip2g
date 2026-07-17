package codellmgql

import (
	"regexp"
	"strconv"

	"trip2g/cmd/codellm/internal/codellm/codellmgql/model"
)

var blockErrorPattern = regexp.MustCompile(`block ([0-9]+)(?:/[0-9]+)?:`)

func blockErrorPayload(err error) model.BlockErrorPayload {
	index := 0
	match := blockErrorPattern.FindStringSubmatch(err.Error())
	if len(match) > 1 {
		if parsed, parseErr := strconv.Atoi(match[1]); parseErr == nil {
			index = parsed - 1
		}
	}
	return model.BlockErrorPayload{Index: index, Message: err.Error()}
}
