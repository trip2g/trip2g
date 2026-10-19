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
	require.Equal(t, link, got)
}

func TestLinkRefusals(t *testing.T) {
	link := federatedassetlink.Link{KBID: "peer", NotePath: "a.md", Name: "a.zip"}

	expired, err := federatedassetlink.Sign(link, secret, time.Now().Add(-time.Minute))
	require.NoError(t, err)

	otherInstance, err := federatedassetlink.Sign(link, "another-secret", time.Now().Add(time.Minute))
	require.NoError(t, err)

	sessionShaped, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Minute).Unix(),
		"kb":  "peer", "np": "a.md", "n": "a.zip",
	}).SignedString([]byte(secret))
	require.NoError(t, err)

	cases := map[string]string{
		"expired":                        expired,
		"signed by another instance":     otherInstance,
		"signed with the session secret": sessionShaped,
		"garbage":                        "not-a-token",
	}

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			_, parseErr := federatedassetlink.Parse(token, secret)
			require.Error(t, parseErr)
		})
	}
}
