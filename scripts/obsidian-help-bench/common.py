"""Shared paths for the Obsidian Help retrieval benchmark.

Committed inputs live in testdata/eval/obsidian-help/. Everything generated
(the downloaded vaults, tool installs, indexes, run files) goes to the work dir,
$OHB_WORK or tmp/obsidian-help-bench under the repo root.
"""
import json
import os

REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
DATA = os.path.join(REPO, "testdata", "eval", "obsidian-help")
WORK = os.path.abspath(os.environ.get("OHB_WORK", os.path.join(REPO, "tmp", "obsidian-help-bench")))
RUNS = os.path.join(WORK, "runs")


def vault(lang):
    return os.path.join(WORK, f"vault-{lang}")


def queries(lang):
    """Base + tricky queries for one vault language, in a stable order."""
    out = []
    for name in (f"golden-{lang}.json", f"golden-tricky-{lang}.json"):
        out += json.load(open(os.path.join(DATA, name), encoding="utf-8"))
    return out


def write_run(label, lang, results, quiet=False):
    os.makedirs(RUNS, exist_ok=True)
    path = os.path.join(RUNS, f"{label}-{lang}.json")
    json.dump({"system": label, "results": results}, open(path, "w", encoding="utf-8"), ensure_ascii=False)
    if not quiet:
        print(f"{label} {lang}: {len(results)} queries, "
              f"{sum(1 for v in results.values() if not v)} empty -> {path}")
