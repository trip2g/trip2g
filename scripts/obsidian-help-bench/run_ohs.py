#!/usr/bin/env python3
"""Run every query through obsidian-hybrid-search (its CLI, local e5-small model).

usage: run_ohs.py <lang> [label] [extra ohs search flags, e.g. --rerank]
Install once:  (cd $OHB_WORK && npm i obsidian-hybrid-search@0.15.2)
"""
import os
import subprocess
import sys

from common import WORK, queries, vault, write_run

lang = sys.argv[1]
label = sys.argv[2] if len(sys.argv) > 2 else "ohs"
extra = sys.argv[3:]
ohs = os.path.join(WORK, "node_modules", ".bin", "ohs")
db = os.path.join(WORK, f"ohs-{lang}.db")
env = {k: v for k, v in os.environ.items() if not k.startswith("OPENAI_")}  # force the local model
env.update(OBSIDIAN_VAULT_PATH=vault(lang), OBSIDIAN_RESPECT_GITIGNORE="false")
if not os.path.exists(db):
    subprocess.run([ohs, "--db", db, "reindex"], env=env, check=True, cwd=vault(lang))
results = {}
for q in queries(lang):
    out = subprocess.run([ohs, "--db", db, "search", q["query"], "--limit", "20", "--only-paths", *extra],
                         env=env, cwd=vault(lang), capture_output=True, text=True)
    results[q["id"]] = [l.strip() for l in out.stdout.splitlines() if l.strip().endswith(".md")]
write_run(label, lang, results)
