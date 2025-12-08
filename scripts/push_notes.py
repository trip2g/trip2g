#!/usr/bin/env python3

import sys
import os
import json
import hashlib
import base64
import requests
import time
from collections import OrderedDict

# get first arg or http://localhost:8081

GRAPHQL_URL = os.getenv("ENDPOINT", "http://localhost:8081/graphql")
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
    query NotePaths {
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


def push_updates_graphql(updates, base_path):
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
            return []
        else:
            print("✅ Обновления успешно отправлены через GraphQL.")
            push_result = result.get("data", {}).get("pushNotes", {})
            return push_result.get("notes", [])
    except Exception as e:
        print(f"❌ Ошибка при отправке GraphQL: {e}")
        return []

def sha256_hash_file(file_path: str) -> str:
    """Calculate SHA256 hash of a file (hex format)"""
    with open(file_path, 'rb') as f:
        content = f.read()
    return hashlib.sha256(content).hexdigest()

def should_retry_multipart_error(error_obj, attempt: int, max_retries: int) -> bool:
    """Check if error is a multipart order issue and we should retry"""
    if attempt >= max_retries - 1:
        return False

    error_str = str(error_obj).lower()
    return 'first part must be operations' in error_str

def build_file_index(base_path: str) -> dict:
    """Build index of all files in vault for global resolution"""
    file_index = {}

    for root, dirs, files in os.walk(base_path):
        # Filter out directories that start with dot and node_modules
        dirs[:] = [d for d in dirs if not d.startswith('.') and d != 'node_modules']

        for fname in files:
            # Skip files that start with dot
            if fname.startswith('.'):
                continue

            full_path = os.path.join(root, fname)
            rel_path = os.path.relpath(full_path, base_path)

            # Index by filename (case-insensitive like Obsidian)
            fname_lower = fname.lower()
            if fname_lower not in file_index:
                file_index[fname_lower] = []
            file_index[fname_lower].append(rel_path)

    # Sort by path depth (shortest first = root priority)
    for fname in file_index:
        file_index[fname].sort(key=lambda p: p.count('/'))

    return file_index

def resolve_asset_path(relative_path: str, note_path: str, base_path: str, file_index: dict = None) -> str:
    """Resolve relative asset path to absolute path using Obsidian's algorithm"""
    # Handle absolute paths from root
    if relative_path.startswith('/'):
        return os.path.join(base_path, relative_path[1:])

    # Handle explicit relative paths (./file or ../file)
    if relative_path.startswith('./'):
        note_dir = os.path.dirname(note_path)
        return os.path.join(base_path, note_dir, relative_path[2:]) if note_dir else os.path.join(base_path, relative_path[2:])

    if relative_path.startswith('../'):
        note_path_parts = note_path.split('/')[:-1]  # Remove filename
        relative_path_parts = relative_path.split('/')

        i = 0
        while i < len(relative_path_parts) and relative_path_parts[i] == '..':
            if note_path_parts:
                note_path_parts.pop()
            i += 1

        resolved_parts = note_path_parts + relative_path_parts[i:]
        return os.path.join(base_path, *resolved_parts) if resolved_parts else base_path

    # Handle explicit paths with slashes (folder/file.png)
    if '/' in relative_path:
        explicit_path = os.path.join(base_path, relative_path)
        if os.path.exists(explicit_path):
            return explicit_path
        return explicit_path

    # Global search using file index (Obsidian behavior)
    if file_index:
        fname_lower = relative_path.lower()
        if fname_lower in file_index:
            # Return first match (shortest path due to sorting)
            return os.path.join(base_path, file_index[fname_lower][0])

    # Fallback: try root first, then note directory
    note_dir = os.path.dirname(note_path)
    candidate_paths = []

    # 1. First check root (Obsidian global resolution priority)
    candidate_paths.append(os.path.join(base_path, relative_path))

    # 2. Then check relative to note directory
    if note_dir:
        candidate_paths.append(os.path.join(base_path, note_dir, relative_path))

    for candidate_path in candidate_paths:
        if os.path.exists(candidate_path):
            return candidate_path

    return candidate_paths[0] if candidate_paths else os.path.join(base_path, relative_path)

def upload_asset(note_id: str, asset_path: str, relative_path: str, sha256_hash: str, base_path: str, max_retries: int = 3) -> bool:
    """Upload asset file to server with retry logic"""
    if not os.path.exists(asset_path):
        print(f"⚠️ Asset file not found: {asset_path}")
        return False

    query = """
    mutation UploadAsset($input: UploadNoteAssetInput!) {
        uploadNoteAsset(input: $input) {
            ... on ErrorPayload {
                __typename
                message
            }
            ... on UploadNoteAssetPayload {
                __typename
                uploadSkipped
            }
        }
    }
    """

    operations = json.dumps({
        "variables": {
            "input": {
                "file": None,
                "noteId": note_id,
                "sha256Hash": sha256_hash,
                "path": relative_path,
                "absolutePath": os.path.relpath(asset_path, base_path)
            }
        },
        "query": query
    })

    files_map = json.dumps({"0": ["variables.input.file"]})

    for attempt in range(max_retries):
        try:
            with open(asset_path, 'rb') as f:
                # Create multipart data with explicit ordering
                # Using OrderedDict-like list to preserve order
                files_ordered = [
                    ('operations', (None, operations, 'application/json')),
                    ('map', (None, files_map, 'application/json')),
                    ('0', (os.path.basename(asset_path), f.read(), 'application/octet-stream'))
                ]

                upload_headers = {
                    "X-API-Key": API_KEY,
                }

                response = requests.post(GRAPHQL_URL, headers=upload_headers, files=files_ordered)
                response.raise_for_status()

                result = response.json()

                if result.get('errors'):
                    if should_retry_multipart_error(result.get('errors'), attempt, max_retries):
                        print(f"⚠️ Retry {attempt + 1}/{max_retries} for {relative_path}: multipart order issue")
                        time.sleep(0.5 * (attempt + 1))  # Exponential backoff
                        continue

                    print(f"❌ Asset upload error for {relative_path}: {result['errors']}")
                    return False

                payload = result.get('data', {}).get('uploadNoteAsset', {})
                if payload.get('__typename') == 'ErrorPayload':
                    print(f"❌ Asset upload failed: {payload.get('message')}")
                    return False
                elif payload.get('__typename') == 'UploadNoteAssetPayload' and not payload.get('uploadSkipped'):
                    if attempt > 0:
                        print(f"✅ Asset uploaded: {relative_path} (after {attempt + 1} attempts)")
                    else:
                        print(f"✅ Asset uploaded: {relative_path}")
                    return True
                else:
                    print(f"⏩ Asset upload skipped (already exists): {relative_path}")
                    return True

        except requests.exceptions.HTTPError as e:
            if e.response.status_code == 422 and should_retry_multipart_error(e.response.text, attempt, max_retries):
                print(f"⚠️ Retry {attempt + 1}/{max_retries} for {relative_path}: 422 multipart order issue")
                time.sleep(0.5 * (attempt + 1))  # Exponential backoff
                continue

            # Log error and don't retry
            if e.response.status_code == 422:
                print(f"❌ 422 Unprocessable Entity for {relative_path}: {e.response.text}")
            else:
                print(f"❌ HTTP error uploading asset {relative_path}: {e}")
            return False
        except Exception as e:
            print(f"❌ Failed to upload asset {relative_path}: {e}")
            return False

    # If we exhausted all retries
    print(f"❌ Failed to upload {relative_path} after {max_retries} attempts")
    return False

def process_note_assets(notes, base_path):
    """Process assets for all notes after successful push"""
    if not notes:
        return {
            'total_notes_with_assets': 0,
            'total_assets': 0,
            'assets_uploaded': 0,
            'assets_up_to_date': 0,
            'assets_not_found': 0,
            'assets_failed': 0
        }

    stats = {
        'total_notes_with_assets': 0,
        'total_assets': 0,
        'assets_uploaded': 0,
        'assets_up_to_date': 0,
        'assets_not_found': 0,
        'assets_failed': 0
    }

    # Build file index once for global resolution
    file_index = build_file_index(base_path)

    for note in notes:
        note_assets = note.get('assets', [])
        if not note_assets:
            continue

        stats['total_notes_with_assets'] += 1
        note_id = note['id']
        note_path = note['path']

        print(f"📎 Processing assets for {note_path}:")

        for asset in note_assets:
            relative_path = asset['path']
            server_hash = asset.get('sha256Hash', '')
            stats['total_assets'] += 1

            absolute_path = resolve_asset_path(relative_path, note_path, base_path, file_index)

            if not os.path.exists(absolute_path):
                print(f"   ⚠️ Asset not found: {relative_path} -> {absolute_path}")
                stats['assets_not_found'] += 1
                continue

            local_hash = sha256_hash_file(absolute_path)

            if not server_hash or server_hash != local_hash:
                print(f"   📤 Uploading: {relative_path}")
                if upload_asset(note_id, absolute_path, relative_path, local_hash, base_path):
                    stats['assets_uploaded'] += 1
                else:
                    stats['assets_failed'] += 1
            else:
                print(f"   ✅ Up to date: {relative_path}")
                stats['assets_up_to_date'] += 1

    return stats

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
        # Filter out directories that start with dot and node_modules
        dirs[:] = [d for d in dirs if not d.startswith('.') and d != 'node_modules']
        
        for fname in files:
            # Skip files that start with dot
            if fname.startswith('.'):
                continue

            ext = os.path.splitext(fname)[1].lower()

            if ext not in ['.md', '.html']:
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
    asset_stats = None
    
    # Always call pushNotes to get current state of notes and assets
    if updates:
        print(f"📤 Отправка {len(updates)} файлов через GraphQL...")
    else:
        print("✅ Все файлы актуальны. Проверяем ассеты...")
    
    notes = push_updates_graphql(updates, base_path)
    
    # Process assets for all notes (whether updated or not)
    if notes:
        print(f"📎 Обработка ассетов для {len(notes)} заметок...")
        asset_stats = process_note_assets(notes, base_path)

    if server_only_paths:
        print(f"🙈 Скрытие {len(server_only_paths)} заметок, отсутствующих локально...")
        hide_notes_graphql(server_only_paths)
    
    # Print final statistics
    print("-" * 80)
    print("📊 ИТОГОВАЯ СТАТИСТИКА:")
    print(f"   📝 Заметки: {len(updates)} обновлено, {len(server_only_paths)} скрыто")
    
    if asset_stats:
        print(f"   📎 Ассеты:")
        print(f"      • Заметок с ассетами: {asset_stats['total_notes_with_assets']}")
        print(f"      • Всего ассетов: {asset_stats['total_assets']}")
        print(f"      • Загружено: {asset_stats['assets_uploaded']}")
        print(f"      • Актуальные: {asset_stats['assets_up_to_date']}")
        if asset_stats['assets_not_found'] > 0:
            print(f"      • Не найдено: {asset_stats['assets_not_found']}")
        if asset_stats['assets_failed'] > 0:
            print(f"      • Ошибки загрузки: {asset_stats['assets_failed']}")
        
        # Asset success rate
        total_processed = asset_stats['assets_uploaded'] + asset_stats['assets_up_to_date']
        if asset_stats['total_assets'] > 0:
            success_rate = (total_processed / asset_stats['total_assets']) * 100
            print(f"      • Успешность: {success_rate:.1f}%")
    else:
        print(f"   📎 Ассеты: не обрабатывались")
    
    print("-" * 80)

if __name__ == "__main__":
    main()
