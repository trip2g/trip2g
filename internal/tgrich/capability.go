package tgrich

// KeyRichMessagePosting is the help.getAppConfig key gating rich-message
// posting from a user account. Measured July 2026: the value moved from
// "disabled" to "premium", and a non-Premium account is refused server-side
// with 400 RICH_MESSAGE_UNSUPPORTED — the call itself, not a UI gate. The
// entitlement follows the sending user, not the posting identity: a Premium
// account succeeds even when posting under a channel's identity.
//
// Bots are not subject to this at all, which is why the bot path never reads it.
const KeyRichMessagePosting = "rich_message_posting"

// Posting modes the key is known to carry.
const (
	PostingDisabled = "disabled"
	PostingPremium  = "premium"
	PostingEnabled  = "enabled"
)

// Reasons a user account may not post rich messages. They reach the admin in
// place of a send failure, so each names the precondition rather than the wire
// error it would otherwise surface as.
const (
	ReasonNeedsPremium = "the Telegram account needs Premium to post rich messages"
	ReasonDisabled     = "Telegram has rich message posting disabled for user accounts"
)

// Capability reports whether a user account may post rich messages. Reason is
// empty when Allowed, and otherwise explains the refusal in the admin's terms.
type Capability struct {
	Allowed bool
	Reason  string
}

// AccountCapability decides the capability from an unmarshalled
// help.getAppConfig document and the account's stored premium flag.
//
// Everything other than the two known-open spellings falls back to the premium
// gate: an unrecognised value must not open the gate, because the send would
// then fail server-side and lose the reason on the way back.
func AccountCapability(config map[string]interface{}, isPremium bool) Capability {
	mode, _ := config[KeyRichMessagePosting].(string)

	if mode == PostingDisabled {
		return Capability{Reason: ReasonDisabled}
	}

	if mode == PostingEnabled {
		return Capability{Allowed: true}
	}

	if !isPremium {
		return Capability{Reason: ReasonNeedsPremium}
	}

	return Capability{Allowed: true}
}
