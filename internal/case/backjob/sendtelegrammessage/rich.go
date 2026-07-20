package sendtelegrammessage

import (
	"context"
	"fmt"
	"trip2g/internal/model"
	"trip2g/internal/tgrich"
)

// sendRich validates the block tree and sends it through sendRichMessage.
//
// Validation is not belt-and-braces here. Several server-side violations answer
// ok:true and discard content without saying so, so a payload that is only
// checked by the server is a payload whose failures are invisible. Rejecting
// locally turns them into a normal send error the admin can see.
//
// Limits are the documented defaults rather than the ones help.getAppConfig
// advertises: nothing on the bot path reads that config today. Substituting
// LimitsFromAppConfig is the upgrade when it does.
func sendRich(ctx context.Context, env Env, params model.TelegramSendPostParams) (int64, error) {
	req := tgrich.Request{
		ChatID: params.TelegramChatID,
		RichMessage: tgrich.InputRichMessage{
			Blocks: params.Post.RichBlocks,
			// Auto-detection is on by default and broader than the classic parse
			// modes: `$USD` becomes a cashtag and any bare 16-digit run becomes a
			// bank card number.
			SkipEntityDetection: true,
		},
		DisableNotification: params.DisableNotification,
	}

	if err := req.RichMessage.Validate(tgrich.DefaultLimits()); err != nil {
		return 0, fmt.Errorf("rich message rejected before sending: %w", err)
	}

	res, err := env.SendTelegramRichMessage(ctx, params.DBChatID, req)
	if err != nil {
		return 0, fmt.Errorf("failed to send rich message: %w", err)
	}

	return res.MessageID, nil
}
