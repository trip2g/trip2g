package updatetelegrammessage

import (
	"context"
	"fmt"
	"trip2g/internal/model"
	"trip2g/internal/tgrich"
)

// editRich replaces the block tree of a posted rich message.
//
// Validation runs before the request leaves for the same reason it does on the
// send path: several server-side violations answer ok:true and discard content
// silently, so a payload only the server checks is a payload whose failures are
// invisible.
//
// The edit replaces the whole block list — there is no per-block edit — which is
// why media inside a rich post cannot be updated in place.
func editRich(ctx context.Context, env Env, params model.TelegramUpdatePostParams) error {
	req := tgrich.EditRequest{
		ChatID:    params.TelegramChatID,
		MessageID: params.MessageID,
		RichMessage: tgrich.InputRichMessage{
			Blocks: params.Post.RichBlocks,
			// Same reason as the send path: auto-detection turns `$USD` into a
			// cashtag and any bare 16-digit run into a bank card number.
			SkipEntityDetection: true,
		},
	}

	if err := req.RichMessage.Validate(tgrich.DefaultLimits()); err != nil {
		return fmt.Errorf("rich message rejected before editing: %w", err)
	}

	if _, err := env.EditTelegramRichMessage(ctx, params.DBChatID, req); err != nil {
		return fmt.Errorf("failed to edit rich message: %w", err)
	}

	return nil
}
