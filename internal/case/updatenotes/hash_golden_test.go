package updatenotes

import "testing"

// goldenHashOfHello pins the wire format of expectedHash. fleet computes this
// value independently (cmd/fleet/internal/agentruntime, contentHash) so it can
// make a patch conditional on the exact bytes it verified. The two
// implementations never meet in code, so they are tied together here: changing
// the algorithm or the encoding on either side breaks this test rather than
// silently turning every conditional patch into a hash mismatch.
const goldenHashOfHello = "LPJNul-wow4m6DsqxbninhsWHlwfp0JecwQzYpOLmCQ="

func TestHashContentGolden(t *testing.T) {
	if got := hashContent([]byte("hello")); got != goldenHashOfHello {
		t.Fatalf("hashContent drifted from the value fleet computes:\n got %q\nwant %q", got, goldenHashOfHello)
	}
}
