package main

import (
	"os"

	"trip2g/internal/appconfig"
	"trip2g/internal/logger"
)

// applyLegacyEmailConfig maps the pre-SMTP-migration RESEND_API_KEY onto the
// Resend SMTP gateway when SMTP_PASS is unset, logging a deprecation warning
// when the fallback fires.
//
// Deprecated: instances provisioned before the SMTP migration receive only
// RESEND_API_KEY (the instance jsonnet renders that name, not SMTP_*); without
// this mapping SendMail silently skips on SMTPHost=="" — the prod regression we
// hit. Remove once every instance is re-provisioned with
// SMTP_HOST/SMTP_USER/SMTP_PASS.
func applyLegacyEmailConfig(cfg *appconfig.Config, log logger.Logger) {
	if cfg.SMTPPass != "" {
		return
	}

	resend := os.Getenv("TRIP2G_RESEND_API_KEY")
	if resend == "" {
		resend = os.Getenv("RESEND_API_KEY")
	}
	if resend == "" {
		return
	}

	cfg.SMTPPass = resend
	if cfg.SMTPHost == "" {
		cfg.SMTPHost = "smtp.resend.com"
	}
	if cfg.SMTPUser == "" {
		cfg.SMTPUser = "resend"
	}

	log.Warn("DEPRECATED: email configured via RESEND_API_KEY fallback (Resend SMTP gateway); provision SMTP_HOST/SMTP_USER/SMTP_PASS to remove this")
}
