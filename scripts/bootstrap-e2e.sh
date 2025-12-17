#!/bin/bash
set -e

DB_PATH="${1:-data.sqlite3}"
INPUT_DIR="${2:-.}"

if [ ! -f "$DB_PATH" ]; then
    echo "Database not found: $DB_PATH"
    exit 1
fi

echo "Bootstrapping $DB_PATH from $INPUT_DIR"

# Create test_channel tag
sqlite3 "$DB_PATH" "insert or ignore into telegram_publish_tags (label) values ('test_channel')"
TAG_ID=$(sqlite3 "$DB_PATH" "select id from telegram_publish_tags where label = 'test_channel'")
echo "Created tag test_channel (id=$TAG_ID)"

# Import telegram accounts from .tgsession.* files
for f in "$INPUT_DIR"/.tgsession.*; do
    [ -f "$f" ] || continue
    source "$f"
    echo "Importing telegram account: $phone"
    sqlite3 "$DB_PATH" "insert or ignore into telegram_accounts (phone, session_data, display_name, is_premium, enabled, api_id, api_hash, created_by) values ('$phone', x'$session_data_hex', '$display_name', $is_premium, $enabled, $api_id, '$api_hash', $created_by)"
done

# Import tg bots from .tgbottoken.* files
# for f in "$INPUT_DIR"/.tgbottoken.*; do
#     [ -f "$f" ] || continue
#     source "$f"
#     echo "Importing tg bot: $name"
#     sqlite3 "$DB_PATH" "insert or ignore into tg_bots (token, enabled, name, description, created_by) values ('$token', $enabled, '$name', '$description', $created_by)"
# done

# Get imported account IDs
ACCOUNT_IDS=$(sqlite3 "$DB_PATH" "select id from telegram_accounts")

# Bot setup (disabled for now)
# BOT_ID=$(sqlite3 "$DB_PATH" "select id from tg_bots limit 1")
# if [ -n "$BOT_ID" ]; then
#     sqlite3 "$DB_PATH" "insert or ignore into tg_bot_chats (telegram_id, chat_type, chat_title, can_invite, bot_id) values (3359176498, 'channel', 'Test Channel', 0, $BOT_ID)"
#     CHAT_ID=$(sqlite3 "$DB_PATH" "select id from tg_bot_chats where telegram_id = 3359176498")
#     echo "Created tg_bot_chat (id=$CHAT_ID)"
#
#     # Link bot chat with test_channel tag
#     sqlite3 "$DB_PATH" "insert or ignore into telegram_publish_chats (chat_id, tag_id, created_by) values ($CHAT_ID, $TAG_ID, 1)"
#     sqlite3 "$DB_PATH" "insert or ignore into telegram_publish_instant_chats (chat_id, tag_id, created_by) values ($CHAT_ID, $TAG_ID, 1)"
#     echo "Linked bot chat with test_channel tag"
# fi

# Link accounts with test_channel tag (using same telegram_chat_id)
for account_id in $ACCOUNT_IDS; do
    sqlite3 "$DB_PATH" "insert or ignore into telegram_publish_account_chats (account_id, telegram_chat_id, tag_id, created_by) values ($account_id, 3359176498, $TAG_ID, 1)"
    sqlite3 "$DB_PATH" "insert or ignore into telegram_publish_account_instant_chats (account_id, telegram_chat_id, tag_id, created_by) values ($account_id, 3359176498, $TAG_ID, 1)"
    echo "Linked account $account_id with test_channel tag"
done

echo "Done!"
