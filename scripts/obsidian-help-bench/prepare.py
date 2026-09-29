#!/usr/bin/env python3
"""Download Obsidian's help docs at a pinned commit and lay out vault-en / vault-ru.

The help texts carry no licence, so they are fetched here rather than committed.
en and ru pages share a `permalink`; testdata/eval/obsidian-help/golden-ru.json
was mapped from the English judgements through it.
"""
import os
import re
import shutil
import subprocess

from common import WORK, vault

REPO_URL = "https://github.com/obsidianmd/obsidian-help.git"
COMMIT = "bc5b4f2"  # 2026-09-15, the snapshot the golden sets were judged against

src = os.path.join(WORK, "obsidian-help")
if not os.path.isdir(src):
    subprocess.run(["git", "clone", "-q", REPO_URL, src], check=True)
subprocess.run(["git", "-C", src, "checkout", "-q", COMMIT], check=True)

for lang in ("en", "ru"):
    dst = vault(lang)
    shutil.rmtree(dst, ignore_errors=True)
    n = 0
    for root, _, files in os.walk(os.path.join(src, lang)):
        for f in files:
            if not f.endswith(".md"):
                continue
            full = os.path.join(root, f)
            rel = os.path.relpath(full, os.path.join(src, lang))
            text = open(full, encoding="utf-8").read()
            # trip2g rejects a null frontmatter scalar ("description:" empty or null)
            # and fails the whole push batch; the field carries no text.
            text = re.sub(r"(?m)^description:[ \t]*(null|~)?[ \t]*\n", "", text)
            os.makedirs(os.path.dirname(os.path.join(dst, rel)), exist_ok=True)
            open(os.path.join(dst, rel), "w", encoding="utf-8").write(text)
            n += 1
    print(f"vault-{lang}: {n} notes -> {dst}")
