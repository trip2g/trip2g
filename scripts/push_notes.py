#!/usr/bin/env python3

import sys
import os
import json
import hashlib
import base64
import requests

# get first arg or http://localhost:8081

GRAPHQL_URL = "http://localhost:8081/graphql"
API_KEY = os.getenv("API_KEY", None)

if not API_KEY:
    print("❌ Не указан API_KEY в переменных окружения. Установите переменную API_KEY.")
    sys.exit(1)

headers = {
    "Content-Type": "application/json",
    "X-API-Key": API_KEY,
}

def sha256_urlsafe_base64(content: bytes) -> str:
    h = hashlib.sha256(content).digest()
    return base64.urlsafe_b64encode(h).decode("utf-8")

def fetch_server_hashes():
    query = """
    query {
      notePaths {
        path: value
        hash: latestContentHash
      }
    }
    """
    try:
        response = requests.post(GRAPHQL_URL, json={"query": query}, headers=headers)
        response.raise_for_status()
        data = response.json()

        if "errors" in data:
            print(f"❌ GraphQL ошибка: {data['errors']}")
            return {}

        result = data.get("data", {}).get("notePaths", [])
        return {item["path"]: item["hash"] for item in result if "path" in item and "hash" in item}

    except Exception as e:
        print(f"❌ Ошибка при запросе хэшей через GraphQL: {e}")
        return {}

def push_updates_graphql(updates):
    query = """
    mutation PushNotes($input: PushNotesInput!) {
      pushNotes(input: $input) {
        ... on ErrorPayload {
          message
        }
        ... on PushNotesPayload {
          notes {
            id
            path
            assets {
              path
              sha256Hash
            }
          }
        }
      }
    }
    """
    variables = {
        "input": {
            "updates": updates
        }
    }

    try:
        response = requests.post(GRAPHQL_URL, headers=headers, json={
            "query": query,
            "variables": variables
        })
        response.raise_for_status()
        result = response.json()
        if 'errors' in result:
            print(f"❌ GraphQL ошибка: {result['errors']}")
        else:
            print("✅ Обновления успешно отправлены через GraphQL.")
    except Exception as e:
        print(f"❌ Ошибка при отправке GraphQL: {e}")

def hide_notes_graphql(paths):
    query = """
    mutation HideNotes($input: HideNotesInput!) {
      hideNotes(input: $input) {
        ... on HideNotesPayload {
          success
        }
        ... on ErrorPayload {
          message
        }
      }
    }
    """
    variables = {
        "input": {
            "paths": paths
        }
    }

    try:
        response = requests.post(GRAPHQL_URL, headers=headers, json={
            "query": query,
            "variables": variables
        })
        response.raise_for_status()
        result = response.json()
        if 'errors' in result:
            print(f"❌ GraphQL ошибка при скрытии заметок: {result['errors']}")
            return False
        else:
            payload = result.get("data", {}).get("hideNotes", {})
            if "message" in payload:
                print(f"❌ Ошибка при скрытии заметок: {payload['message']}")
                return False
            else:
                print(f"✅ Успешно скрыто {len(paths)} заметок.")
                return True
    except Exception as e:
        print(f"❌ Ошибка при скрытии заметок через GraphQL: {e}")
        return False

def main():
    global GRAPHQL_URL
    base_path = sys.argv[1] if len(sys.argv) > 1 else "demo"

    if len(sys.argv) > 2:
        GRAPHQL_URL = sys.argv[2]

    server_hashes = fetch_server_hashes()
    server_empty = len(server_hashes) == 0
    updates = []
    local_paths = set()

    print("GraphQL URL:", GRAPHQL_URL)

    print("📦 Сравнение файлов:")
    print("-" * 80)

    for root, dirs, files in os.walk(base_path):
        # Filter out directories that start with dot
        dirs[:] = [d for d in dirs if not d.startswith('.')]
        
        for fname in files:
            # Skip files that start with dot
            if fname.startswith('.'):
                continue
                
            if not fname.lower().endswith(".md"):
                continue

            full_path = os.path.join(root, fname)
            rel_path = os.path.relpath(full_path, base_path)
            local_paths.add(rel_path)

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

            if server_empty or remote_hash != local_hash:
                print(f"{log_prefix} | {log_local} | {log_remote} | ⏩ SEND")
                updates.append({
                    "path": rel_path,
                    "content": content.decode("utf-8", errors="replace")
                })
            else:
                print(f"{log_prefix} | {log_local} | {log_remote} | ✅ SKIP")

    # Find server notes that don't exist locally (should be hidden)
    server_only_paths = []
    for server_path in server_hashes.keys():
        if server_path not in local_paths:
            server_only_paths.append(server_path)
            print(f"{server_path:<30} | local=— | remote={server_hashes[server_path]} | 🙈 HIDE")

    print("-" * 80)
    if updates:
        print(f"📤 Отправка {len(updates)} файлов через GraphQL...")
        push_updates_graphql(updates)
    else:
        print("✅ Все файлы актуальны. Обновлений нет.")

    if server_only_paths:
        print(f"🙈 Скрытие {len(server_only_paths)} заметок, отсутствующих локально...")
        hide_notes_graphql(server_only_paths)

if __name__ == "__main__":
    main()
