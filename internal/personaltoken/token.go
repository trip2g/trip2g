package personaltoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"strings"
)

const Prefix = "t2g_"

const alphanum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func Generate() string {
	b := make([]byte, 64)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphanum))))
		if err != nil {
			panic(err)
		}
		b[i] = alphanum[n.Int64()]
	}
	return Prefix + string(b)
}

func Hash(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func DisplayPrefix(plaintext string) string {
	return plaintext[:8]
}

func IsPersonal(s string) bool {
	return strings.HasPrefix(s, Prefix)
}
