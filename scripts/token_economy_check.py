#!/usr/bin/env python3
"""
token_economy_check.py — verify trip2g's "Token Economy" claim against a LIVE site.

No dependencies. Python 3 standard library only (urllib + json). Run it anywhere:

    python3 scripts/token_economy_check.py
    python3 scripts/token_economy_check.py --endpoint https://trip2g.com/_system/mcp
    python3 scripts/token_economy_check.py "how do i set up a custom domain" "telegram limits"

What it does
------------
For each query it talks to the trip2g MCP server (JSON-RPC 2.0 over HTTP, anonymous
access to public notes) and compares three ways an agent can read the answer:

  full    = note_html(pid)                  -> the whole note (the naive dump)
  section = note_html(pid, toc_path=[...])  -> just the answer-bearing section
  window  = note_html(pid, match_id=...)    -> the focused chunk around the match

It reports, per query: token cost of each arm, the savings ratio (full/section,
full/window), and a fidelity check — does the cheap arm actually still contain the
matched terms? A cheap arm that drops the answer is a loss, not a win.

Tokens are approximated with a transparent, dependency-free tokenizer (word + punct
split). Absolute counts are approximate; the SAVINGS RATIOS are what matter and they
are robust to the exact tokenizer (both arms share it). Bytes are printed too.
"""

import argparse
import json
import re
import statistics
import sys
import urllib.request

DEFAULT_ENDPOINT = "https://trip2g.com/_system/mcp"

# A handful of queries spanning different public docs. Override on the command line.
DEFAULT_QUERIES = [
    "how do i publish a post to telegram",
    "telegram post types and limits",
    "set up a custom domain for my site",
    "how does the 14kb single page work",
    "accept paid subscriptions and monetization",
    "two way sync between obsidian and the site",
]

_TOKEN_RE = re.compile(r"\w+|[^\w\s]", re.UNICODE)
_MARK_RE = re.compile(r"</?mark>")


def approx_tokens(text: str) -> int:
    """Dependency-free token approximation: words + standalone punctuation.

    Not Claude's tokenizer. Consistent across arms, so the ratios it produces are
    sound; absolute numbers are indicative only.
    """
    return len(_TOKEN_RE.findall(text or ""))


def mcp_call(endpoint: str, name: str, arguments: dict) -> dict:
    """Call one MCP tool via JSON-RPC 2.0. Returns the JSON-RPC `result` object."""
    payload = json.dumps({
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {"name": name, "arguments": arguments},
    }).encode()
    req = urllib.request.Request(
        endpoint, data=payload,
        headers={"Content-Type": "application/json", "Accept": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        body = json.loads(resp.read().decode())
    if "error" in body:
        raise RuntimeError(f"MCP error for {name}: {body['error']}")
    return body["result"]


def result_text(result: dict) -> str:
    """note_html returns its HTML in result.content[0].text."""
    content = result.get("content") or []
    return content[0]["text"] if content else ""


def pick_answer_section(top: dict, query: str):
    """Choose the answer section's toc_path from the top search result.

    Strategy: the match snippet usually contains the section heading line. Prefer the
    toc entry whose title literally appears in the snippet; fall back to the toc entry
    whose title shares the most words with (query + snippet). Returns (toc_path, title)
    or (None, None) if the note has no sections.
    """
    toc = top.get("toc") or []
    if not toc:
        return None, None
    matches = top.get("matches") or []
    snippet = _MARK_RE.sub("", matches[0]["snippet"]) if matches else ""
    snip_low = snippet.lower()

    literal = [t for t in toc if t["title"].lower() in snip_low]
    if literal:
        best = max(literal, key=lambda t: len(t["title"]))
        return best["path"], best["title"]

    bag = set(re.findall(r"\w+", (query + " " + snippet).lower()))
    scored = max(
        toc,
        key=lambda t: len(set(re.findall(r"\w+", t["title"].lower())) & bag),
    )
    return scored["path"], scored["title"]


def fidelity(section_text: str, top: dict) -> str:
    """Does the cheap arm still contain the matched terms? YES / PARTIAL / NO."""
    matches = top.get("matches") or []
    if not matches:
        return "n/a"
    marked = re.findall(r"<mark>(.*?)</mark>", matches[0]["snippet"])
    terms = {w.lower() for m in marked for w in re.findall(r"\w+", m) if len(w) > 2}
    if not terms:
        return "n/a"
    sec = section_text.lower()
    hit = sum(1 for t in terms if t in sec)
    if hit == len(terms):
        return "YES"
    if hit == 0:
        return "NO"
    return "PARTIAL"


def run(endpoint: str, queries: list):
    rows = []
    print(f"endpoint: {endpoint}\n")
    hdr = f"{'query':38} {'full':>7} {'section':>8} {'window':>7} {'sec×':>6} {'win×':>6} {'sec✓':>6} {'win✓':>6}"
    print(hdr)
    print("-" * len(hdr))

    for q in queries:
        try:
            search = mcp_call(endpoint, "search", {"query": q})
            sc = search.get("structuredContent") or {}
            results = sc.get("results") or []
            if not results:
                print(f"{q[:38]:38} (no results)")
                continue
            top = results[0]
            pid = top.get("note_id")
            toc_path, _ = pick_answer_section(top, q)
            matches = top.get("matches") or []
            match_id = matches[0]["match_id"] if matches else None

            full_t = approx_tokens(result_text(mcp_call(endpoint, "note_html", {"pid": pid})))
            sec_txt = result_text(mcp_call(endpoint, "note_html", {"pid": pid, "toc_path": toc_path})) if toc_path else ""
            win_txt = result_text(mcp_call(endpoint, "note_html", {"pid": pid, "match_id": match_id})) if match_id else ""
            sec_t = approx_tokens(sec_txt)
            win_t = approx_tokens(win_txt)

            sec_ratio = full_t / sec_t if sec_t else float("nan")
            win_ratio = full_t / win_t if win_t else float("nan")
            sec_fid = fidelity(sec_txt, top) if sec_txt else "—"
            win_fid = fidelity(win_txt, top) if win_txt else "—"

            rows.append((sec_ratio, win_ratio, sec_fid, win_fid))
            print(f"{q[:38]:38} {full_t:7d} {sec_t:8d} {win_t:7d} "
                  f"{sec_ratio:5.1f}x {win_ratio:5.1f}x {sec_fid:>6} {win_fid:>6}")
        except Exception as e:  # noqa: BLE001 — a CLI tool should not crash on one bad query
            print(f"{q[:38]:38} ERROR: {e}")

    if rows:
        sec_ratios = [r[0] for r in rows if r[0] == r[0]]
        win_ratios = [r[1] for r in rows if r[1] == r[1]]
        sec_ok = sum(1 for r in rows if r[2] == "YES")
        win_ok = sum(1 for r in rows if r[3] == "YES")
        print("-" * len(hdr))
        print(f"median section savings: {statistics.median(sec_ratios):.1f}x   "
              f"median window savings: {statistics.median(win_ratios):.1f}x")
        print(f"answer fully retained — section: {sec_ok}/{len(rows)}   "
              f"window: {win_ok}/{len(rows)}")
        print("\nNote: tokens are approximate (dependency-free word/punct split); "
              "the ratios are the robust result.")


def main():
    ap = argparse.ArgumentParser(description="Verify trip2g Token Economy against a live MCP endpoint.")
    ap.add_argument("--endpoint", default=DEFAULT_ENDPOINT, help="MCP endpoint URL")
    ap.add_argument("queries", nargs="*", help="queries to test (default: a built-in set)")
    args = ap.parse_args()
    run(args.endpoint, args.queries or DEFAULT_QUERIES)


if __name__ == "__main__":
    sys.exit(main())
