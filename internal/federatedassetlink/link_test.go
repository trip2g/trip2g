package federatedassetlink_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"trip2g/internal/federatedassetlink"
)

const secret = "instance-secret"

func TestLinkRoundTrips(t *testing.T) {
	link := federatedassetlink.Link{KBID: "peer", NotePath: "skills/reel.md", Name: "files/attach.zip"}

	token, err := federatedassetlink.Sign(link, secret, time.Now().Add(time.Minute))
	require.NoError(t, err)

	got, err := federatedassetlink.Parse(token, secret)
	require.NoError(t, err)
	require.Equal(t, link, got.Link)
	require.NotEmpty(t, got.ID)
	require.WithinDuration(t, time.Now().Add(time.Minute), got.ExpiresAt, 2*time.Second)
}

func TestEachLinkGetsItsOwnID(t *testing.T) {
	link := federatedassetlink.Link{KBID: "peer", NotePath: "a.md", Name: "a.zip"}
	expiresAt := time.Now().Add(time.Minute)

	first, err := federatedassetlink.Sign(link, secret, expiresAt)
	require.NoError(t, err)
	second, err := federatedassetlink.Sign(link, secret, expiresAt)
	require.NoError(t, err)

	firstClaim, err := federatedassetlink.Parse(first, secret)
	require.NoError(t, err)
	secondClaim, err := federatedassetlink.Parse(second, secret)
	require.NoError(t, err)
	require.NotEqual(t, firstClaim.ID, secondClaim.ID)
}

func TestALinkIsSpentOnce(t *testing.T) {
	var spent federatedassetlink.Spent
	expiresAt := time.Now().Add(time.Minute)

	require.True(t, spent.SpendFederatedAssetLink("a", expiresAt))
	require.False(t, spent.SpendFederatedAssetLink("a", expiresAt), "a second use is refused")
	require.True(t, spent.SpendFederatedAssetLink("b", expiresAt), "another link is unaffected")
}

func TestSpentForgetsExpiredLinks(t *testing.T) {
	var spent federatedassetlink.Spent

	require.True(t, spent.SpendFederatedAssetLink("old", time.Now().Add(-time.Second)))
	require.True(t, spent.SpendFederatedAssetLink("new", time.Now().Add(time.Minute)))
	require.Equal(t, 1, spent.Len(), "an id past its expiry is no longer held")
}

func TestLinkRefusals(t *testing.T) {
	link := federatedassetlink.Link{KBID: "peer", NotePath: "a.md", Name: "a.zip"}

	expired, err := federatedassetlink.Sign(link, secret, time.Now().Add(-time.Minute))
	require.NoError(t, err)

	otherInstance, err := federatedassetlink.Sign(link, "another-secret", time.Now().Add(time.Minute))
	require.NoError(t, err)

	sessionShaped, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Minute).Unix(),
		"jti": "id",
		"kb":  "peer", "np": "a.md", "n": "a.zip",
	}).SignedString([]byte(secret))
	require.NoError(t, err)

	withoutID, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Minute).Unix(),
		"kb":  "peer", "np": "a.md", "n": "a.zip",
	}).SignedString(federatedassetlink.SigningKey(secret))
	require.NoError(t, err)

	cases := map[string]string{
		"expired":                        expired,
		"signed by another instance":     otherInstance,
		"signed with the session secret": sessionShaped,
		"garbage":                        "not-a-token",
		"without an id":                  withoutID,
	}

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			_, parseErr := federatedassetlink.Parse(token, secret)
			require.Error(t, parseErr)
		})
	}
}
