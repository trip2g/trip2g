#!/usr/bin/env python3

import sys
import os
import json
import hashlib
import base64
import requests

GRAPHQL_URL = "http://localhost:8081/graphql"

def sha256_urlsafe_base64(content: bytes) -> str:
    """
    Считает SHA256 хэш и возвращает результат в формате URL-safe base64.
    """
    sha256_hash = hashlib.sha256(content).digest()
    b64_encoded = base64.urlsafe_b64encode(sha256_hash).decode('utf-8')
    return b64_encoded  # base64.urlsafe_b64encode может содержать '=', это допустимо

def graphql_request(query: str, variables: dict = None) -> dict:
    """
    Выполняет GraphQL-запрос и возвращает словарь ответа.
    """
    try:
        response = requests.post(GRAPHQL_URL, json={"query": query, "variables": variables or {}})
        response.raise_for_status()
        result = response.json()
        if 'errors' in result:
            raise Exception(result['errors'])
        return result['data']
    except Exception as e:
        raise RuntimeError(f"GraphQL запрос не удался: {e}")

def fetch_note_paths() -> dict:
    """
    Запрашивает список заметок с сервера через GraphQL.
    """
    query = """
    query {
      notePaths {
        value
        latestContentHash
      }
    }
    """
    data = graphql_request(query)
    result = {}
    for note in data.get("notePaths", []):
        result[note["value"]] = note["latestContentHash"]
    return result

def push_updates(updates: list[dict]):
    """
    Отправляет обновлённые заметки на сервер через GraphQL.
    """
    mutation = """
    mutation PushNotes($input: PushNotesInput!) {
      pushNotes(input: $input) {
        ... on PushNotesPayload {
          assets {
            path
            uploadUrl
          }
        }
        ... on ErrorPayload {
          message
        }
      }
    }
    """
    variables = {"input": {"updates": updates}}
    data = graphql_request(mutation, variables)

    if "message" in data["pushNotes"]:
        raise RuntimeError(f"Сервер вернул ошибку: {data['pushNotes']['message']}")
    print("Все обновления успешно загружены!")

def main():
    base_path = sys.argv[1] if len(sys.argv) > 1 else 'demo'

    try:
        server_hash_map = fetch_note_paths()
    except Exception as e:
        print(e)
        return

    updates = []

    for root, _, files in os.walk(base_path):
        for f in files:
            if f.lower().endswith(".md"):
                full_path = os.path.join(root, f)
                rel_path = os.path.relpath(full_path, base_path)

                try:
                    with open(full_path, 'rb') as fp:
                        content_bytes = fp.read()
                except Exception as e:
                    print(f"Не удалось прочитать файл {full_path}: {e}")
                    continue

                local_hash = sha256_urlsafe_base64(content_bytes)
                server_hash = server_hash_map.get(rel_path)

                if server_hash != local_hash:
                    print(f"Файл «{rel_path}» требует обновления.")
                    updates.append({
                        "path": rel_path,
                        "content": content_bytes.decode('utf-8', errors='replace')
                    })

    if updates:
        try:
            print("Отправляем изменения на сервер...")
            push_updates(updates)
        except Exception as e:
            print(f"Ошибка при загрузке изменений: {e}")
    else:
        print("Нет файлов для обновления. Всё в актуальном состоянии.")

if __name__ == "__main__":
    main()
