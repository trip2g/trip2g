#!/usr/bin/env bash
# Deploy trip2g to fly.io in one shot: app, volume, secrets, deploy.
#
#   APP=my-notes REGION=fra OWNER_EMAIL=me@example.com ./scripts/deploy-fly.sh
#
# Re-running is safe: existing app/volume/secrets are reused, not recreated.
# Requires flyctl (https://fly.io/docs/flyctl/install) and `fly auth login`.
set -euo pipefail

APP="${APP:-trip2g}"
REGION="${REGION:-fra}"
VOLUME_SIZE="${VOLUME_SIZE:-1}"
OWNER_EMAIL="${OWNER_EMAIL:-}"
PUBLIC_URL="${PUBLIC_URL:-https://${APP}.fly.dev}"

if [[ -z "$OWNER_EMAIL" ]]; then
  echo "Set OWNER_EMAIL=you@example.com — this email becomes the site owner." >&2
  exit 1
fi

cd "$(dirname "$0")/.."

# 1. App. Create it if it does not exist yet.
if ! fly apps list 2>/dev/null | awk '{print $1}' | grep -qx "$APP"; then
  echo "==> Creating app $APP"
  fly apps create "$APP" --org personal
fi

# 2. Persistent volume for SQLite, the git mirror and uploaded files.
if ! fly volumes list -a "$APP" 2>/dev/null | grep -q "trip2g_data"; then
  echo "==> Creating volume trip2g_data ($VOLUME_SIZE GB) in $REGION"
  fly volumes create trip2g_data --region "$REGION" --size "$VOLUME_SIZE" -a "$APP" --yes
fi

# 3. Secrets. Generated once and kept across deploys.
echo "==> Setting secrets"
SECRETS=(
  "PUBLIC_URL=$PUBLIC_URL"
  "OWNER_EMAIL=$OWNER_EMAIL"
)
# Only generate keys that are not set yet, so re-runs keep existing sessions.
existing="$(fly secrets list -a "$APP" 2>/dev/null || true)"
grep -q "JWT_SECRET" <<<"$existing" || SECRETS+=("JWT_SECRET=$(openssl rand -hex 32)")
grep -q "DATA_ENCRYPTION_KEY" <<<"$existing" || SECRETS+=("DATA_ENCRYPTION_KEY=$(openssl rand -hex 16)")
fly secrets set "${SECRETS[@]}" -a "$APP" --stage

# 4. Deploy (remote build on fly builders).
echo "==> Deploying"
fly deploy --remote-only -a "$APP" --region "$REGION"

echo "==> Done. Open: $PUBLIC_URL"
echo "    Sign in with $OWNER_EMAIL."
echo "    No email provider configured yet? Read the login code from: fly logs -a $APP"
