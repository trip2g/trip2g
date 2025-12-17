#!/bin/bash
set -e

DB_PATH="${1:-data.sqlite3}"
OUTPUT_DIR="${2:-.}"

if [ ! -f "$DB_PATH" ]; then
    echo "Database not found: $DB_PATH"
    exit 1
fi

echo "Exporting from $DB_PATH to $OUTPUT_DIR"

# Export telegram account session data (encrypted blob as hex)
sqlite3 "$DB_PATH" "select id, phone, hex(session_data), display_name, is_premium, enabled, api_id, api_hash, created_by from telegram_accounts" | while IFS='|' read -r id phone session_hex display_name is_premium enabled api_id api_hash created_by; do
    echo "Exporting telegram account: $phone (id=$id)"
    cat > "$OUTPUT_DIR/.tgsession.$id" <<EOF
id=$id
phone=$phone
session_data_hex=$session_hex
display_name=$display_name
is_premium=$is_premium
enabled=$enabled
api_id=$api_id
api_hash=$api_hash
created_by=$created_by
EOF
done

# Export tg bot tokens
sqlite3 "$DB_PATH" "select id, token, enabled, name, description, created_by from tg_bots" | while IFS='|' read -r id token enabled name description created_by; do
    echo "Exporting tg bot: $name (id=$id)"
    cat > "$OUTPUT_DIR/.tgbottoken.$id" <<EOF
id=$id
token=$token
enabled=$enabled
name=$name
description=$description
created_by=$created_by
EOF
done

echo "Done!"
