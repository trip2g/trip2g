package convertnoteviewtotgpost

import (
	"context"
	"fmt"
	"trip2g/internal/markdownv2"
	"trip2g/internal/model"
)

// applyRich runs the rich converter when the note asked for it, and leaves the
// post classic otherwise. It runs after the classic pass rather than instead of
// it: the sent-message row stores classic content, and the update path still
// reads it.
//
// Only an explicit `telegram_rich: on` selects rich. `auto` stays classic in
// V1 — the predicate it will eventually use is the rich converter's typed loss
// set, and shipping `auto` before that predicate exists would flip a note's
// format for reasons nobody chose.
func applyRich(
	ctx context.Context,
	env Env,
	post *model.TelegramPost,
	source model.TelegramPostSource,
	resolveLink markdownv2.LinkResolver,
	publicURL string,
) {
	mode, err := source.NoteView.ExtractTelegramRichMode()
	if err != nil {
		post.Warnings = append(post.Warnings, err.Error())
		return
	}

	if !mode.UseRich() {
		return
	}

	// An account destination reaches rich through MTProto, where it is gated on
	// the sending account holding Premium — enforced server-side by the call
	// itself. Checking here means the note never gets scheduled against an
	// account that cannot post it, and the admin reads the precondition instead
	// of a RICH_MESSAGE_UNSUPPORTED that names nothing.
	//
	// A warning and not an error on purpose: writing
	// telegram_publish_notes.last_error here would block every other
	// destination of the same note, because that column is note-wide.
	if source.ChatID == 0 {
		capability := env.TelegramAccountRichCapability(ctx, source.AccountID)
		if !capability.Allowed {
			post.Warnings = append(post.Warnings,
				fmt.Sprintf("telegram_rich: on cannot be sent from this account: %s, sending classic instead",
					capability.Reason))
			return
		}
	}

	// The classic pass already counted this note's links. Counting them again
	// would double every figure the admin sees.
	links, unresolved, external := post.LinkCount, post.UnresolvedLinkCount, post.ExternalLinkCount

	converter := markdownv2.RichConverter{}
	converter.SetLinkResolver(resolveLink)
	converter.SetAssetResolver(func(path string) (string, bool) {
		replace, ok := source.NoteView.AssetReplaces[path]
		if !ok {
			return "", false
		}

		return model.AbsoluteURL(publicURL, replace.URL), true
	})

	res := converter.Process(source.NoteView)

	post.LinkCount, post.UnresolvedLinkCount, post.ExternalLinkCount = links, unresolved, external

	if len(res.Blocks) == 0 {
		post.Warnings = append(post.Warnings,
			"telegram_rich: on produced no blocks, sending classic instead")
		return
	}

	for _, loss := range res.Losses {
		post.Warnings = append(post.Warnings,
			fmt.Sprintf("telegram rich conversion dropped %s (%s): %s", loss.Node, loss.Kind, loss.Detail))
	}

	post.RichBlocks = res.Blocks

	// Media rides inside the block tree, in document order. The classic list is
	// built from a map and has no order at all, so keeping both would send the
	// same assets twice in two different sequences.
	post.Media = nil
}
