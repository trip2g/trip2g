package onboardingvault

import (
	_ "embed"
)

//go:generate sh -c "cd .. && zip -r onboarding-vault/vault.zip onboarding-vault -x 'onboarding-vault/embed.go' -x 'onboarding-vault/vault.zip' -x 'onboarding-vault/.obsidian/workspace.json'"

//go:embed vault.zip
var ZipData []byte
