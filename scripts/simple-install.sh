#!/bin/sh
# trip2g installer for a fresh Ubuntu 24.04 box (run as root):
#
#   curl -fsSL https://raw.githubusercontent.com/trip2g/trip2g/main/scripts/simple-install.sh | sh
#
# Interactive: asks a short setup questionnaire (access mode, hostname,
# owner email, SMTP / bootstrap login) reading from /dev/tty — plain `read`
# can't prompt under curl|sh because stdin is the pipe. With no terminal
# attached (CI), it falls back to env-var answers and never blocks.
#
# HTTPS needs no reverse proxy: trip2g has built-in ACME (ACME_DOMAIN=...)
# and binds :80/:443 itself. The no-domain mode uses sslip.io magic DNS
# (<ip-with-dashes>.sslip.io resolves to that IP — a free real hostname,
# which is all Let's Encrypt needs).
#
# Non-interactive overrides (preset any of these to skip its question):
#   TRIP2G_MODE            ip | sslip | domain
#   TRIP2G_HOST            hostname for mode=domain (ignored otherwise)
#   TRIP2G_OWNER_EMAIL     owner account email
#   TRIP2G_SMTP_HOST / TRIP2G_SMTP_USER / TRIP2G_SMTP_PASS / TRIP2G_MAIL_FROM
#   TRIP2G_LOG_CODES       y|n — bootstrap first login by printing sign-in
#                          codes to the journal (LOG_SIGN_IN_CODES; needs a
#                          build with PR #197; insecure, disable after setup)
#   TRIP2G_YES=1           skip the final confirmation
#   TRIP2G_BINARY_URL      see the gap note below
#
# ---------------------------------------------------------------------------
# Binary source: by default the script fetches the LATEST GitHub release
# (v0.9.0+ ships trip2g_<tag>_linux_<arch>.tar.gz + .sha256; the tarball
# contains the `trip2g-server` binary). Overrides:
#   TRIP2G_BINARY_URL=<url>  fetch from a custom URL instead
#   pre-placed binary        an existing /usr/local/bin/trip2g is kept as-is
BINARY_URL="${TRIP2G_BINARY_URL:-}"
REPO=trip2g/trip2g
# ---------------------------------------------------------------------------

set -eu

BIN=/usr/local/bin/trip2g
ENVFILE=/etc/trip2g.env
UNIT=/etc/systemd/system/trip2g.service
DATADIR=/var/lib/trip2g

say() { printf '\n==> %s\n' "$*"; }
die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

# ask <var> <prompt> <default> — reads from /dev/tty when available;
# otherwise keeps the preset/default (curl|sh-safe, CI-safe).
ask() {
  _var=$1; _prompt=$2; _default=$3
  eval "_cur=\${$_var:-}"
  if [ -n "$_cur" ]; then return 0; fi           # env override wins
  if [ -e /dev/tty ]; then
    printf '%s [%s]: ' "$_prompt" "$_default" > /dev/tty 2>/dev/null || {
      eval "$_var=\$_default"; return 0; }
    IFS= read -r _ans < /dev/tty || _ans=""
    eval "$_var=\${_ans:-\$_default}"
  else
    eval "$_var=\$_default"
  fi
}

[ "$(id -u)" = 0 ] || die "run as root"

ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  GOARCH=amd64 ;;
  aarch64) GOARCH=arm64 ;;
  *) die "unsupported arch: $ARCH" ;;
esac

IP=$(hostname -I 2>/dev/null | awk '{print $1}')
[ -n "$IP" ] || IP=127.0.0.1
SSLIP_HOST=$(printf '%s' "$IP" | tr '.' '-').sslip.io

# --- questionnaire ----------------------------------------------------------
if [ -z "${TRIP2G_MODE:-}" ]; then
  cat 2>/dev/null > /dev/tty <<EOF || true

How should the site be reachable?
  1) HTTP by IP        http://$IP:8081        (quick test, no TLS)
  2) HTTPS, no domain  https://$SSLIP_HOST    (built-in Let's Encrypt + sslip.io magic DNS)
  3) HTTPS, own domain https://<your host>    (built-in Let's Encrypt; A-record must point here)
EOF
  CHOICE=""
  ask CHOICE "Choose 1/2/3" "2"
  case "$CHOICE" in
    1) TRIP2G_MODE=ip ;;
    3) TRIP2G_MODE=domain ;;
    *) TRIP2G_MODE=sslip ;;
  esac
fi

USE_ACME=0
case "$TRIP2G_MODE" in
  ip)     PUBLIC_URL="http://$IP:8081"; HOST="" ;;
  sslip)  HOST=$SSLIP_HOST; PUBLIC_URL="https://$HOST"; USE_ACME=1 ;;
  domain) ask TRIP2G_HOST "Your domain (A-record → $IP)" "docs.example.com"
          HOST=$TRIP2G_HOST; PUBLIC_URL="https://$HOST"; USE_ACME=1 ;;
  *) die "TRIP2G_MODE must be ip|sslip|domain" ;;
esac

ask TRIP2G_OWNER_EMAIL "Owner email (your login)" "owner@example.com"

# First login. Sign-in codes go out by email; with no SMTP they are not
# delivered. Login options, in the order the questionnaire offers them:
#   A) OAuth (GitHub/Google) — recommended SMTP-free login. Creds are DB-stored
#      and configured in the ADMIN PANEL, not env, so the script can't write
#      them; it prints the exact recipe + callback URL and you still bootstrap
#      once to reach the panel.
#   B) Dex — self-hosted OIDC username/password, fully autonomous (no external
#      service). OIDC *is* env-configurable, so the script fully wires it:
#      installs Dex, writes a staticPasswords config, sets OIDC_* env.
#   C) SMTP — the designed email-code path (use a relay provider, not your own
#      MTA; fresh VPSes block outbound port 25).
#   D) Bootstrap — LOG_SIGN_IN_CODES=true (merged, PR #197): print the sign-in
#      code to journalctl. Insecure, first-login only; turn off after.
# All four still need one bootstrap login first for A/B/C, so D is auto-enabled
# unless SMTP is configured.
ask TRIP2G_LOGIN "Login method: A=OAuth recipe  B=Dex(self-hosted)  C=SMTP  D=bootstrap-only" "A"
TRIP2G_SMTP_HOST="${TRIP2G_SMTP_HOST:-}"
USE_DEX=0
case "$TRIP2G_LOGIN" in
  b|B)
    USE_DEX=1
    ask TRIP2G_DEX_PASSWORD "Password for the Dex login (user = owner email)" "changeme-$(head -c4 /dev/urandom | od -An -tx1 | tr -d ' ')"
    TRIP2G_LOG_CODES=n   # Dex is the login; no code needed
    ;;
  c|C)
    ask TRIP2G_SMTP_HOST "SMTP host" "smtp.resend.com"
    ask TRIP2G_SMTP_USER "SMTP user" "resend"
    ask TRIP2G_SMTP_PASS "SMTP password / API key" ""
    ask TRIP2G_MAIL_FROM "Mail From (verified sender)" "no-reply@example.com"
    TRIP2G_LOG_CODES=n
    ;;
  *)
    # A (OAuth, printed recipe) and D (bootstrap) both need the code bootstrap.
    ask TRIP2G_LOG_CODES "Print sign-in codes to the server log for bootstrap? (insecure; disable after first login) (Y/n)" "y"
    ;;
esac

cat <<EOF

Planned configuration:
  Mode:          $TRIP2G_MODE
  Public URL:    $PUBLIC_URL
  Owner:         $TRIP2G_OWNER_EMAIL
  Login:         $( case "$TRIP2G_LOGIN" in b|B) echo "Dex self-hosted OIDC (installed + wired)";; c|C) echo "SMTP email codes";; d|D) echo "bootstrap code-in-log only";; *) echo "OAuth (recipe printed; configure in admin)";; esac )
  SMTP:          ${TRIP2G_SMTP_HOST:-none}
  Bootstrap:     $( case "$TRIP2G_LOG_CODES" in y|Y|yes) echo "LOG_SIGN_IN_CODES=true (codes in journalctl; disable after setup)";; *) echo "off";; esac )
  Storage:       local FS ($DATADIR/storage) — no S3/MinIO
  TLS:           $( [ "$USE_ACME" = 1 ] && echo "built-in Let's Encrypt for $HOST (needs ports 80+443)" || echo "none (plain HTTP)" )
  Existing $ENVFILE: $( [ -f "$ENVFILE" ] && echo "KEPT AS-IS (idempotent re-run)" || echo "will be created" )
EOF
PROCEED="${TRIP2G_YES:-}"
ask PROCEED "Proceed? (Y/n)" "y"
case "$PROCEED" in n|N|no) die "aborted" ;; esac

# --- deps -------------------------------------------------------------------
say "Installing runtime dependencies"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq git ca-certificates curl sqlite3 >/dev/null

# --- binary -----------------------------------------------------------------
if [ -x "$BIN" ]; then
  say "Binary already present at $BIN — keeping it"
elif [ -n "$BINARY_URL" ]; then
  say "Fetching binary from $BINARY_URL"
  case "$BINARY_URL" in
    *.tar.gz|*.tgz)
      TMPD=$(mktemp -d)
      curl -fsSL -o "$TMPD/trip2g.tar.gz" "$BINARY_URL"
      tar xz -C "$TMPD" -f "$TMPD/trip2g.tar.gz"
      # accept either binary name inside the tarball
      if [ -f "$TMPD/trip2g-server" ]; then mv "$TMPD/trip2g-server" "$BIN"
      elif [ -f "$TMPD/trip2g" ]; then mv "$TMPD/trip2g" "$BIN"
      else die "no trip2g-server binary inside $BINARY_URL"; fi
      rm -rf "$TMPD"
      ;;
    *) curl -fsSL -o "$BIN" "$BINARY_URL" ;;
  esac
  chmod +x "$BIN"
else
  say "Fetching the latest trip2g release"
  TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
  [ -n "$TAG" ] || die "could not resolve the latest release tag from api.github.com.
Pass TRIP2G_BINARY_URL=<url> or pre-place a binary at $BIN and re-run."
  ASSET="trip2g_${TAG}_linux_${GOARCH}.tar.gz"
  URL="https://github.com/$REPO/releases/download/$TAG/$ASSET"
  say "Downloading $ASSET"
  TMPD=$(mktemp -d)
  curl -fL --progress-bar -o "$TMPD/$ASSET" "$URL" \
    || die "download failed: $URL
Pass TRIP2G_BINARY_URL=<url> or pre-place a binary at $BIN and re-run."
  # checksum: the release ships <asset>.sha256 — verify when possible
  if curl -fsSL -o "$TMPD/$ASSET.sha256" "$URL.sha256" 2>/dev/null; then
    ( cd "$TMPD" && grep -o '^[0-9a-f]\{64\}' "$ASSET.sha256" | sed "s|\$|  $ASSET|" | sha256sum -c - >/dev/null ) \
      || die "sha256 mismatch for $ASSET — refusing to install"
    say "sha256 verified"
  else
    echo "WARN: no .sha256 asset found; skipping checksum verification"
  fi
  tar xz -C "$TMPD" -f "$TMPD/$ASSET" trip2g-server
  mv "$TMPD/trip2g-server" "$BIN"
  chmod +x "$BIN"
  rm -rf "$TMPD"
fi

# --- data + env -------------------------------------------------------------
mkdir -p "$DATADIR/storage"

if [ -f "$ENVFILE" ]; then
  say "$ENVFILE exists — keeping it (idempotent re-run; edit it manually to change settings)"
else
  say "Writing $ENVFILE"
  JWT=$(head -c32 /dev/urandom | od -An -tx1 | tr -d ' \n')
  DEK=$(head -c16 /dev/urandom | od -An -tx1 | tr -d ' \n')
  umask 077
  {
    if [ "$USE_ACME" = 1 ]; then
      # Built-in ACME: trip2g binds :443 (TLS) and :80 (redirect) itself.
      echo "ACME_DOMAIN=$HOST"
      echo "PUBLIC_URL=$PUBLIC_URL"
    else
      echo "PUBLIC_URL=$PUBLIC_URL"
      echo "LISTEN_ADDR=0.0.0.0:8081"
      # Session cookies over plain HTTP need this; remove when you add TLS.
      echo "USER_TOKEN_INSECURE=true"
    fi
    cat <<INNER
INTERNAL_LISTEN_ADDR=127.0.0.1:8082
DB_FILE=$DATADIR/data.sqlite3
GIT_API_REPO_PATH=$DATADIR/git
STORAGE_BACKEND=local
STORAGE_LOCAL_DIR=$DATADIR/storage
LOG_LEVEL=info
DEV=false
OWNER_EMAIL=$TRIP2G_OWNER_EMAIL
JWT_SECRET=$JWT
DATA_ENCRYPTION_KEY=$DEK
# Cloud-plan defaults are 100MB DB / 1GB assets; 0 = unlimited on your own disk.
STORAGE_DB_LIMIT=0
STORAGE_ASSETS_LIMIT=0
INNER
    if [ -n "$TRIP2G_SMTP_HOST" ]; then
      echo "SMTP_HOST=$TRIP2G_SMTP_HOST"
      echo "SMTP_USER=$TRIP2G_SMTP_USER"
      echo "SMTP_PASS=$TRIP2G_SMTP_PASS"
      echo "MAIL_FROM=$TRIP2G_MAIL_FROM"
    fi
    if [ "$USE_DEX" = 1 ]; then
      # OIDC is env-configurable (unlike OAuth), so we can fully wire Dex here.
      echo "OIDC_ISSUER=http://$IP:5556"
      echo "OIDC_CLIENT_ID=trip2g-sso-client"
      echo "OIDC_CLIENT_SECRET=trip2g-secret"
    fi
    case "$TRIP2G_LOG_CODES" in
      y|Y|yes)
        echo "# Bootstrap only — prints sign-in codes to the journal. Remove after"
        echo "# you set up SMTP or OAuth (admin -> Integrations -> GitHub/Google OAuth)."
        echo "LOG_SIGN_IN_CODES=true"
        ;;
    esac
  } > "$ENVFILE"
fi

# --- Dex (self-hosted OIDC login) -------------------------------------------
# DEX_BINARY_URL is the same kind of gap as BINARY_URL: no official prebuilt
# URL, so pass it or pre-place /usr/local/bin/dex (build: go build ./cmd/dex).
if [ "$USE_DEX" = 1 ]; then
  DEX_BIN=/usr/local/bin/dex
  if [ ! -x "$DEX_BIN" ]; then
    if [ -n "${DEX_BINARY_URL:-}" ]; then
      say "Fetching Dex from DEX_BINARY_URL"
      curl -fsSL -o "$DEX_BIN" "$DEX_BINARY_URL" && chmod +x "$DEX_BIN"
    else
      die "Dex login selected but no dex binary: set DEX_BINARY_URL=<url> or
place a built dex at $DEX_BIN (GOOS=linux GOARCH=$GOARCH go build -o dex ./cmd/dex
from github.com/dexidp/dex), then re-run."
    fi
  fi
  apt-get install -y -qq apache2-utils >/dev/null   # htpasswd, for the bcrypt hash
  DEX_HASH=$(htpasswd -bnBC 10 "" "$TRIP2G_DEX_PASSWORD" | tr -d ':\n' | sed 's/^[^$]*//')
  mkdir -p /etc/dex
  cat > /etc/dex/config.yaml <<DEXEOF
issuer: http://$IP:5556
storage:
  type: memory
web:
  http: 0.0.0.0:5556
staticClients:
  - id: trip2g-sso-client
    secret: trip2g-secret
    name: trip2g
    redirectURIs:
      - $PUBLIC_URL/_system/auth/oidc/callback
enablePasswordDB: true
staticPasswords:
  - email: "$TRIP2G_OWNER_EMAIL"
    hash: "$DEX_HASH"
    username: "owner"
    userID: "trip2g-owner-0000"
DEXEOF
  cat > /etc/systemd/system/dex.service <<'DEXUNIT'
[Unit]
Description=Dex OIDC IdP
After=network.target

[Service]
ExecStart=/usr/local/bin/dex serve /etc/dex/config.yaml
Restart=on-failure

[Install]
WantedBy=multi-user.target
DEXUNIT
  systemctl daemon-reload && systemctl enable dex >/dev/null 2>&1 && systemctl restart dex
  if [ "$USE_ACME" = 1 ]; then
    say "NOTE: Dex issuer is http://$IP:5556 (works, browser reaches it directly).
For a fully-HTTPS Dex issuer you must put Caddy IN FRONT OF DEX (trip2g's
built-in ACME covers only trip2g, not Dex): a dex.<host> sslip.io name,
reverse_proxy localhost:5556, and OIDC_ISSUER=https://dex.<host>. Not automated here."
  fi
fi

# --- systemd ----------------------------------------------------------------
say "Installing systemd unit"
cat > "$UNIT" <<'EOF'
[Unit]
Description=trip2g publishing server
After=network.target

[Service]
Type=simple
EnvironmentFile=/etc/trip2g.env
ExecStart=/usr/local/bin/trip2g
WorkingDirectory=/var/lib/trip2g
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable trip2g >/dev/null 2>&1
systemctl restart trip2g

say "Waiting for /healthz"
i=0
until curl -fsS -o /dev/null http://127.0.0.1:8082/healthz 2>/dev/null; do
  i=$((i+1)); [ $i -le 30 ] || die "server not healthy in 30s — check: journalctl -u trip2g -n 50"
  sleep 1
done
if [ "$USE_ACME" = 1 ]; then
  say "Waiting for HTTPS ($HOST) — first certificate can take ~15-30s"
  i=0
  until curl -fsS -o /dev/null "https://$HOST/" 2>/dev/null; do
    i=$((i+1)); [ $i -le 45 ] || { echo "WARN: HTTPS not up yet; check: journalctl -u trip2g -n 30 (ports 80/443 open? DNS resolves?)"; break; }
    sleep 2
  done
fi

# --- login-link helper (requires trip2g v0.9.0+) -----------------------------
# v0.9.0's `login-link` subcommand prints a one-time, 5-minute admin sign-in
# URL (<PUBLIC_URL>/_system/hat?token=...) signed with the box's own secret —
# the simplest first login. The helper re-runs it with the service's env so
# the operator can regenerate a fresh link anytime over SSH. On older
# binaries the subcommand doesn't exist; the script falls back to the
# LOG_SIGN_IN_CODES / sqlite instructions below.
# Guarded with timeout + URL-shape grep: pre-v0.9.0 binaries don't know the
# subcommand and try to BOOT THE SERVER instead (tested) — the timeout kills
# that, and the grep only lets an actual /_system/hat?token= URL through.
cat > /usr/local/bin/trip2g-login-link <<HELPER
#!/bin/sh
# Print a one-time (5-minute) admin sign-in link for this trip2g instance.
# Safe to re-run whenever the previous link expired or was lost.
# Requires trip2g v0.9.0+ (the login-link subcommand).
set -a; . $ENVFILE; set +a
LINK=\$(timeout 5 $BIN login-link 2>/dev/null | grep -o 'http[s]*://[^[:space:]]*_system/hat?token=[^[:space:]]*' | head -1)
[ -n "\$LINK" ] || { echo "no link — this trip2g binary predates v0.9.0 (login-link)" >&2; exit 1; }
echo "\$LINK"
HELPER
chmod +x /usr/local/bin/trip2g-login-link

LOGIN_LINK=$(/usr/local/bin/trip2g-login-link 2>/dev/null) || LOGIN_LINK=""

# --- next steps -------------------------------------------------------------
cat <<EOF

trip2g is running and enabled on boot.

  Site:      $PUBLIC_URL
  Owner:     $TRIP2G_OWNER_EMAIL
  Logs:      journalctl -u trip2g -f
  Config:    $ENVFILE  (edit, then: systemctl restart trip2g)

NEXT STEPS
EOF
if [ -n "$LOGIN_LINK" ]; then
  cat <<EOF
 1. SIGN IN NOW — open this link within 5 minutes (one-time, signs you in
    as $TRIP2G_OWNER_EMAIL):

      $LOGIN_LINK

    Lost it or expired? SSH in and run:  trip2g-login-link
    (Don't paste the link in shared channels — it carries a sign-in token.)
EOF
  if [ "$TRIP2G_MODE" = ip ]; then
    cat <<EOF
    NOTE (plain HTTP): the link sets a session cookie that browsers drop
    over http unless USER_TOKEN_INSECURE=true is set — this install sets it.
    If you remove that flag while still on http, link login silently fails.
EOF
  fi
  if [ "$USE_DEX" = 1 ]; then
    cat <<EOF
    Day-to-day login: "Sign in with SSO" -> Dex ($TRIP2G_OWNER_EMAIL / your
    password). Add users in /etc/dex/config.yaml (staticPasswords) + restart dex.
EOF
  fi
elif [ "$USE_DEX" = 1 ]; then
  cat <<EOF
 1. Log in via Dex: open $PUBLIC_URL/admin -> "Sign in with SSO"
    -> Dex login: $TRIP2G_OWNER_EMAIL / (the password you set). No email needed.
    Add more users in /etc/dex/config.yaml (staticPasswords) + restart dex.
EOF
else
  cat <<EOF
 1. First login: open $PUBLIC_URL/admin, enter the owner email.
    (This binary predates the v0.9.0 'login-link' subcommand — otherwise a
    one-time sign-in URL would be printed here.)
EOF
  case "$TRIP2G_LOG_CODES" in
    y|Y|yes) cat <<EOF
    The sign-in code is printed to the server log (LOG_SIGN_IN_CODES bootstrap):
      journalctl -u trip2g | grep "sign-in code" | tail -1
    On older builds without that flag, read it from the database instead:
      sqlite3 "file:$DATADIR/data.sqlite3?mode=ro" \\
        "select code from sign_in_codes order by created_at desc limit 1"
EOF
    ;;
  esac
fi
if [ "$USE_DEX" != 1 ]; then
  cat <<EOF
 2. Set up permanent login (recommended): admin -> Integrations -> GitHub OAuth
    (or Google OAuth). Register an OAuth app at
    https://github.com/settings/applications/new with callback URL:
      $PUBLIC_URL/_system/auth/github/callback
    and paste the client id + secret into the admin form — no SMTP needed.
    Then remove LOG_SIGN_IN_CODES from $ENVFILE and restart.
EOF
fi
cat <<EOF
 3. Create an API key: admin -> Integrations -> API Keys.
 4. Push markdown:
      curl -L -o trip2g-sync.mjs \\
        https://github.com/trip2g/obsidian-sync/releases/download/0.3.5/trip2g-sync.mjs
      node trip2g-sync.mjs --folder ./vault --api-key KEY --api-url $PUBLIC_URL/graphql
 5. IMPORTANT: notes are subscriber-only by default — visitors see only the
    title and a paywall. Add 'free: true' to the frontmatter of every note
    that should be publicly readable.
EOF
