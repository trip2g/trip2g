package sendtelegramaccountmessage

import (
	"context"
	"encoding/json"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/tgrich"
	"trip2g/internal/tgtd"
)

// accountAppConfig unmarshals the stored help.getAppConfig document. A config
// that never parsed is an empty one, which lands every reader on its documented
// fallback rather than on a zero value — the same shape as
// (*app).TelegramCaptionLengthLimit.
func accountAppConfig(account db.TelegramAccount) map[string]interface{} {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(account.AppConfig), &config); err != nil {
		return nil
	}

	return config
}

// richCapability reports whether this account may post rich messages at all.
//
// The check is made from the stored account row, before the session is opened:
// Premium is a server-side precondition, so an unchecked send comes back as
// RICH_MESSAGE_UNSUPPORTED with nothing in it that names Premium.
func richCapability(account db.TelegramAccount) tgrich.Capability {
	return tgrich.AccountCapability(accountAppConfig(account), account.IsPremium == 1)
}

// sendRich sends the block tree through the account's MTProto session.
//
// There is no MTProto sendRichMessage: rich is an optional field on the
// ordinary messages.sendMessage, so this is the same call the classic path
// makes with a different payload. The blocks come from the same converter the
// bot path uses, mapped to Instant View's page-block tree by tgtd.ToPageBlocks.
//
// Validation is local and deliberate. Several server-side violations answer
// success and discard content without saying so, so a payload only the server
// checks is a payload whose failures are invisible.
func sendRich(
	ctx context.Context,
	env Env,
	client *tgtd.Client,
	sessionData []byte,
	account db.TelegramAccount,
	params model.TelegramAccountSendPostParams,
) (int64, error) {
	blocks := params.Post.RichBlocks

	message := tgrich.InputRichMessage{
		Blocks:              blocks,
		SkipEntityDetection: true,
	}

	// Unlike the bot path, the limits come from this account's own app_config
	// rather than the documented defaults: the account already stores that
	// document, so there is no reason to guess.
	if err := message.Validate(tgrich.LimitsFromAppConfig(accountAppConfig(account))); err != nil {
		return 0, fmt.Errorf("rich message rejected before sending: %w", err)
	}

	result, err := client.SendRichMessage(ctx, sessionData, tgtd.SendRichMessageParams{
		ChatID:    params.TelegramChatID,
		Blocks:    blocks,
		NoWebpage: params.Post.DisableWebPagePreview,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to send rich message: %w", err)
	}

	// Content can be discarded silently, so a send that returns fewer blocks
	// than it was given is reported rather than trusted. The message is already
	// posted at this point: the log is the only place the discrepancy can go.
	if result.EchoedBlocks >= 0 && result.EchoedBlocks < len(blocks) {
		env.Logger().Warn("telegram rich message came back with fewer blocks than sent",
			"note_path_id", params.NotePathID,
			"account_id", params.AccountID,
			"message_id", result.MessageID,
			"sent_blocks", len(blocks),
			"echoed_blocks", result.EchoedBlocks,
		)
	}

	return result.MessageID, nil
}
