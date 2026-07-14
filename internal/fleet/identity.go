package fleet

import (
	"crypto/sha256"
	"encoding/hex"
)

// fleetHash derives the stable, namespaced identity marker folded into the
// fleet's webhook delivery URL: h = sha256("fleet:" + fleetID) in hex. The
// "fleet:" prefix namespaces it against other sha256 uses and keeps the
// plaintext fleet_id out of the URL. It is NOT a secret and is never used for
// auth — just an opaque, stable id the monolith routes deliveries on and the
// reconcile loop de-dups by (see docs/dev/fleet_graphql.md).
func fleetHash(fleetID string) string {
	sum := sha256.Sum256([]byte("fleet:" + fleetID))
	return hex.EncodeToString(sum[:])
}

// webhookPathPrefix is the delivery path segment identifying this fleet:
// "/_fleet/<h>/webhook/". Both the change and cron delivery URLs extend it, and
// the receive handler strips it. The trailing slash makes it a valid
// http.ServeMux subtree prefix.
func webhookPathPrefix(fleetID string) string {
	return "/_fleet/" + fleetHash(fleetID) + "/webhook/"
}

// changeDeliveryURL builds the monolith->fleet change-delivery URL:
// <callbackURL>/_fleet/<h>/webhook/<urlKey(notePath)>. The <h> segment is the
// fleet identity (takeover/dedup key); the trailing urlKey identifies the role.
func changeDeliveryURL(callbackURL, fleetID, notePath string) string {
	return callbackURL + webhookPathPrefix(fleetID) + urlKey(notePath)
}

// cronDeliveryURL builds the monolith->fleet cron-delivery URL:
// <callbackURL>/_fleet/<h>/webhook/cron/<urlKey(notePath)>.
func cronDeliveryURL(callbackURL, fleetID, notePath string) string {
	return callbackURL + webhookPathPrefix(fleetID) + "cron/" + urlKey(notePath)
}
