package onboardingvault

import (
	_ "embed"
)

//go:generate bash generate.sh

//go:embed vault.zip
var ZipData []byte
