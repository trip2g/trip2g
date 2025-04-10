#!/usr/bin/env python3

import sys
import os
import json
import hashlib
import base64
import requests

GET_HASHES_URL = "http://localhost:8081/api/getnotehashes"
POST_NOTES_URL = "http://localhost:8081/api/pushnotes"

def sha256_urlsafe_base64(content: bytes) -> str:
    h = hashlib.sha256(content).digest()
    return base64.urlsafe_b64encode(h).decode("utf-8")

def fetch_server_hashes():
    try:
        response = requests.get(GET_HASHES_URL)
        response.raise_for_status()
        return response.json().get("map", {})
    except Exception as e:
        print(f"Ошибка при получении хэшей: {e}")
        return {}

def push_updates(updates):
    payload = {"updates": updates}
    try:
        response = requests.post(POST_NOTES_URL, json=payload)
        response.raise_for_status()
        print("✅ Все обновления успешно отправлены.")
    except Exception as e:
        print(f"❌ Ошибка при отправке обновлений: {e}")

def main():
    base_path = sys.argv[1] if len(sys.argv) > 1 else "demo"
    server_hashes = fetch_server_hashes()
    updates = []

    print("📦 Сравнение файлов:")
    print("-" * 80)

    for root, _, files in os.walk(base_path):
        for fname in files:
            if not fname.lower().endswith(".md"):
                continue

            full_path = os.path.join(root, fname)
            rel_path = os.path.relpath(full_path, base_path)

            try:
                with open(full_path, 'rb') as f:
                    content = f.read()
            except Exception as e:
                print(f"⚠️ Не удалось прочитать {rel_path}: {e}")
                continue

            local_hash = sha256_urlsafe_base64(content)
            remote_hash = server_hashes.get(rel_path)

            log_prefix = f"{rel_path:<30}"
            log_local = f"local={local_hash}"
            log_remote = f"remote={remote_hash or '—'}"

            if remote_hash != local_hash:
                print(f"{log_prefix} | {log_local} | {log_remote} | ⏩ SEND")
                updates.append({
                    "path": rel_path,
                    "content": content.decode("utf-8", errors="replace")
                })
            else:
                print(f"{log_prefix} | {log_local} | {log_remote} | ✅ SKIP")

    print("-" * 80)
    if updates:
        print(f"📤 Отправка {len(updates)} файлов...")
        push_updates(updates)
    else:
        print("✅ Все файлы актуальны. Обновлений нет.")

if __name__ == "__main__":
    main()
