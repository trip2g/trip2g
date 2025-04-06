#!/usr/bin/env python3

import sys
import os
import json
import hashlib
import base64
import requests

def sha1_urlsafe_base64(content: bytes) -> str:
    """
    Считает SHA1 хэш и возвращает результат в формате URL base64.
    """
    sha1_hash = hashlib.sha1(content).digest()
    b64_encoded = base64.urlsafe_b64encode(sha1_hash).decode('utf-8')
    return b64_encoded  # оставляем, как есть, включая '=' на конце, если она будет

def main():
    # 1) Путь к локальной директории: либо из аргумента, либо 'demo' по умолчанию
    if len(sys.argv) > 1:
        base_path = sys.argv[1]
    else:
        base_path = 'demo'

    # 2) Запрашиваем список заметок с сервера
    url = "http://localhost:8080/api/note_paths"
    try:
        response = requests.get(url)
        response.raise_for_status()
        note_paths = response.json().get("paths", [])
    except Exception as e:
        print(f"Ошибка при запросе {url}: {e}")
        return

    # Превратим массив в словарь: { "file.md": "latest_content_hash", ... }
    server_hash_map = {}
    for item in note_paths:
        server_file = item.get("value")                 # например "test.md"
        server_hash = item.get("latest_content_hash")   # SHA1 в URL base64
        if server_file:
            server_hash_map[server_file] = server_hash

    # Массив для накопления всех изменений, которые будем отправлять одним POST
    updates = []

    # 3) Рекурсивно обходим локальные .md-файлы
    for root, dirs, files in os.walk(base_path):
        for f in files:
            if f.lower().endswith(".md"):
                full_path = os.path.join(root, f)
                # Относительный путь к файлу без префикса base_path
                rel_path = os.path.relpath(full_path, base_path)

                # Считаем локальный хэш
                try:
                    with open(full_path, 'rb') as fp:
                        content_bytes = fp.read()
                except Exception as e:
                    print(f"Не удалось прочитать файл {full_path}: {e}")
                    continue

                local_hash = sha1_urlsafe_base64(content_bytes)
                server_file_hash = server_hash_map.get(rel_path)

                # Если файла нет на сервере или хэши не совпадают
                if server_file_hash != local_hash:
                    print(f"Файл «{rel_path}» требует обновления.")
                    # Готовим структуру под обновление
                    updates.append({
                        "path": rel_path,
                        "content": content_bytes.decode('utf-8', errors='replace')
                    })

    # 4) Если есть обновлённые файлы, отправляем одним POST-запросом
    if updates:
        post_url = "http://localhost:8080/api/notes"
        payload = {"updates": updates}

        try:
            print("Отправляем изменения на сервер...")
            r = requests.post(post_url, json=payload)
            r.raise_for_status()
            print("Все обновления успешно загружены!")
        except Exception as e:
            print(f"Ошибка при загрузке изменений: {e}")
    else:
        print("Нет файлов для обновления. Всё в актуальном состоянии.")

if __name__ == "__main__":
    main()
