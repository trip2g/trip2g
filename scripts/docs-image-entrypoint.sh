#!/bin/sh
# Boots trip2g, pushes /vault into it once, then keeps serving. The database
# lives inside the container: the image is the content, a restart rebuilds it.
set -eu

hex() {
    od -A n -t x1 -N "$1" /dev/urandom | tr -d ' \n'
}

: "${JWT_SECRET:=$(hex 32)}"
: "${DATA_ENCRYPTION_KEY:=$(hex 16)}"
: "${OWNER_PERSONAL_TOKEN_VALUE:=t2g_$(hex 32)}"
export JWT_SECRET DATA_ENCRYPTION_KEY OWNER_PERSONAL_TOKEN_VALUE

if [ -z "$JWT_SECRET" ] || [ -z "$DATA_ENCRYPTION_KEY" ] || [ "$OWNER_PERSONAL_TOKEN_VALUE" = "t2g_" ]; then
    echo "trip2g-docs: a secret came out empty" >&2
    exit 1
fi

mkdir -p "$(dirname "$DB_FILE")" "$STORAGE_LOCAL_DIR"

/trip2g &
server=$!

i=0
until wget -q -O /dev/null "http://${INTERNAL_LISTEN_ADDR}/readyz"; do
    i=$((i + 1))
    if [ "$i" -gt 150 ]; then
        echo "trip2g-docs: trip2g did not come up" >&2
        exit 1
    fi
    sleep 2
done

# The onboarding archive carries the sync client and a key for it; nothing
# else from it is kept.
tmp=/tmp/onboarding
rm -rf "$tmp"
mkdir -p "$tmp"
wget -q -O "$tmp/vault.zip" --header "Authorization: Bearer $OWNER_PERSONAL_TOKEN_VALUE" \
    "http://127.0.0.1:8080/_system/onboarding-vault?name=docs"
unzip -q "$tmp/vault.zip" -d "$tmp"

mkdir -p "$tmp/base"
mv "$tmp/docs/.obsidian" "$tmp/base/.obsidian"
cp -R /vault/. "$tmp/base/"
cd "$tmp/base"

# PUBLIC_URL is what the archive stamped as the sync target; the push goes to
# the local listener whatever the public address is.
node -e '
  const fs = require("fs");
  const p = ".obsidian/plugins/trip2g/data.json";
  const d = JSON.parse(fs.readFileSync(p, "utf8"));
  for (const dir of d.syncDirs || []) {
    dir.apiUrl = "http://127.0.0.1:8080";
  }
  fs.writeFileSync(p, JSON.stringify(d, null, 2) + "\n");
'

node .obsidian/plugins/trip2g/trip2g-sync.mjs .
cd /
rm -rf "$tmp"
echo "trip2g-docs: the vault is loaded"

wait "$server"
