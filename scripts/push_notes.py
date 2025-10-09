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

def resolve_asset_path(relative_path: str, note_path: str, base_path: str) -> str:
    """Resolve relative asset path to absolute path"""
    if relative_path.startswith('/'):
        return os.path.join(base_path, relative_path[1:])
    
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
    
    # Try multiple candidate paths
    note_dir = os.path.dirname(note_path)
    candidate_paths = []
    
    if note_dir:
        candidate_paths.append(os.path.join(base_path, note_dir, relative_path))
    
    candidate_paths.append(os.path.join(base_path, relative_path))
    
    for candidate_path in candidate_paths:
        if os.path.exists(candidate_path):
            return candidate_path
    
    return candidate_paths[0] if candidate_paths else os.path.join(base_path, relative_path)

def upload_asset(note_id: str, asset_path: str, relative_path: str, sha256_hash: str) -> bool:
    """Upload asset file to server"""
    if not os.path.exists(asset_path):
        print(f"⚠️ Asset file not found: {asset_path}")
        return False
    
    query = """
    mutation($input: UploadNoteAssetInput!) { 
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
                "absolutePath": asset_path
            }
        },
        "query": query
    })
    
    files_map = json.dumps({"0": ["variables.input.file"]})
    
    try:
        with open(asset_path, 'rb') as f:
            files = {
                'operations': (None, operations),
                'map': (None, files_map),
                '0': (os.path.basename(asset_path), f, 'application/octet-stream')
            }
            
            upload_headers = {
                "X-API-Key": API_KEY,
            }
            
            response = requests.post(GRAPHQL_URL, headers=upload_headers, files=files)
            response.raise_for_status()
            
            result = response.json()
            if result.get('errors'):
                print(f"❌ Asset upload error for {relative_path}: {result['errors']}")
                return False
            
            payload = result.get('data', {}).get('uploadNoteAsset', {})
            if payload.get('__typename') == 'ErrorPayload':
                print(f"❌ Asset upload failed: {payload.get('message')}")
                return False
            elif payload.get('__typename') == 'UploadNoteAssetPayload' and not payload.get('uploadSkipped'):
                print(f"✅ Asset uploaded: {relative_path}")
                return True
            else:
                print(f"⏩ Asset upload skipped (already exists): {relative_path}")
                return True
                
    except Exception as e:
        print(f"❌ Failed to upload asset {relative_path}: {e}")
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
            
            absolute_path = resolve_asset_path(relative_path, note_path, base_path)
            
            if not os.path.exists(absolute_path):
                print(f"   ⚠️ Asset not found: {relative_path} -> {absolute_path}")
                stats['assets_not_found'] += 1
                continue
            
            local_hash = sha256_hash_file(absolute_path)
            
            if not server_hash or server_hash != local_hash:
                print(f"   📤 Uploading: {relative_path}")
                if upload_asset(note_id, absolute_path, relative_path, local_hash):
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
        # Filter out directories that start with dot
        dirs[:] = [d for d in dirs if not d.startswith('.')]
        
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
    
    if updates:
        print(f"📤 Отправка {len(updates)} файлов через GraphQL...")
        notes = push_updates_graphql(updates, base_path)
        
        # Step 2: Process assets for uploaded notes
        if notes:
            print(f"📎 Обработка ассетов для {len(notes)} заметок...")
            asset_stats = process_note_assets(notes, base_path)
    else:
        print("✅ Все файлы актуальны. Обновлений нет.")

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
