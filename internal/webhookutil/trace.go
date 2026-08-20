package webhookutil

import "strconv"

// Delivery kinds. A delivery id is only unique within its own table, so every
// cross-kind reference (trace ids, parent links, attribution rows) carries the
// kind alongside the id.
const (
	DeliveryKindChange = "change"
	DeliveryKindCron   = "cron"
)

// TraceID is the identifier of one delivery chain: the serialized root
// delivery, e.g. "cron:4242". A root carries its own id; every delivery caused
// by its writes inherits the same string, so a whole chain — across both
// delivery kinds and however many fleets handled its hops — is one equality
// check away.
func TraceID(kind string, deliveryID int64) string {
	return kind + ":" + strconv.FormatInt(deliveryID, 10)
}
