package tgrich

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// Method is the Bot API method name. It is an ordinary JSON-over-HTTPS method,
// which is why no library upgrade is needed: go-telegram-bot-api cannot express
// it as a Chattable (params() and method() are unexported), but the same
// MakeRequest that Send and Request already call underneath takes it directly.
const Method = "sendRichMessage"

// ErrContentDiscarded reports that the server kept less than it was given.
var ErrContentDiscarded = errors.New("telegram discarded part of the rich message")

// Params renders the request as the flat form parameters MakeRequest takes.
// Nested objects ride as JSON strings because the transport form-encodes a
// map[string]string; that is how the library's own helpers send reply markup.
func (r Request) Params() (map[string]string, error) {
	message, err := json.Marshal(r.RichMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rich message: %w", err)
	}

	params := map[string]string{
		"chat_id":      strconv.FormatInt(r.ChatID, 10),
		"rich_message": string(message),
	}

	if r.DisableNotification {
		params["disable_notification"] = "true"
	}

	return params, nil
}

// SendResult is what the server returned. Blocks is the echoed block tree when
// the server sent one back; it is the only way to detect the silent truncation
// past the run-cost ceiling, which otherwise returns ok:true and says nothing.
type SendResult struct {
	MessageID int64
	Blocks    []Block
}

// sendResponse is the Message object sendRichMessage returns. Message.text is
// empty for a rich message, so nothing but rich_message describes the content.
type sendResponse struct {
	MessageID   int64 `json:"message_id"`
	RichMessage *struct {
		Blocks []Block `json:"blocks"`
	} `json:"rich_message"`
}

func DecodeSendResult(raw json.RawMessage) (SendResult, error) {
	var resp sendResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return SendResult{}, fmt.Errorf("failed to decode %s response: %w", Method, err)
	}

	result := SendResult{MessageID: resp.MessageID}
	if resp.RichMessage != nil {
		result.Blocks = resp.RichMessage.Blocks
	}

	return result, nil
}

// Stats summarises a block tree. Blocks counts every block in the tree, not
// only the top-level ones the block limit applies to: the point here is
// detecting loss anywhere, not enforcing a limit.
type Stats struct {
	Blocks    int
	Runs      int
	TextUnits int
}

func Measure(blocks []Block) Stats {
	var stats Stats
	stats.add(blocks)

	return stats
}

func (s *Stats) add(blocks []Block) {
	for _, block := range blocks {
		s.Blocks++

		s.addText(block.Text)
		s.addText(block.Title)
		s.TextUnits += utf16Len(block.Code)

		if block.Caption != nil {
			s.addText(&block.Caption.Text)
		}

		for _, row := range block.Rows {
			for _, cell := range row.Cells {
				s.addText(&cell.Text)
			}
		}

		s.add(block.Blocks)
		for _, item := range block.Items {
			s.add(item.Blocks)
		}
	}
}

func (s *Stats) addText(text *RichText) {
	if text == nil {
		return
	}

	if text.Text != "" {
		s.Runs++
		s.TextUnits += utf16Len(text.Text)
	}

	for i := range text.Children {
		s.addText(&text.Children[i])
	}
}

// VerifyEcho compares what the server echoed against what was submitted.
//
// This is not defensive coding for its own sake: past a run-cost ceiling the
// server silently drops content and still answers ok:true, with no error and no
// flag. There is no other way to notice. An absent echo is not evidence of loss
// — the server simply did not return one — so it passes.
func VerifyEcho(sent, echoed []Block) error {
	if len(echoed) == 0 {
		return nil
	}

	want, got := Measure(sent), Measure(echoed)

	if got.Blocks < want.Blocks || got.Runs < want.Runs || got.TextUnits < want.TextUnits {
		return fmt.Errorf("%w: sent %d blocks/%d runs/%d text units, got back %d/%d/%d",
			ErrContentDiscarded,
			want.Blocks, want.Runs, want.TextUnits,
			got.Blocks, got.Runs, got.TextUnits)
	}

	return nil
}
