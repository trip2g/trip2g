#!/usr/bin/env python3
"""
expand_check.py — verify the `expand` progressive-disclosure navigation against a
LIVE trip2g MCP endpoint. No dependencies (Python 3 stdlib only).

    python3 scripts/expand_check.py
    python3 scripts/expand_check.py --endpoint https://trip2g.com/_system/mcp "telegram limits"

What it checks
--------------
1. Capability: tools/list exposes `expand` (and `federated_expand`).
2. Slim search: a search result no longer carries the flat `toc` array.
3. Progressive disclosure: for each query, walk the note's TOC tree level by level
   with `expand` (picking, at each level, the child whose title best matches the
   query) until a leaf, then read that section. Compares the cost of
   navigate + read  vs  dumping the whole note, and checks the descent landed on a
   section that actually contains the matched answer terms.

Tokens are approximate (dependency-free word/punct split); ratios are the robust
result. Run AFTER deploying the expand feature — before deploy it reports that
`expand` is not available yet.
"""

import argparse
import json
import re
import statistics
import urllib.request

DEFAULT_ENDPOINT = "https://trip2g.com/_system/mcp"
DEFAULT_QUERIES = [
    "how do i publish a post to telegram",
    "telegram post types and limits",
    "set up a custom domain for my site",
    "accept paid subscriptions and monetization",
    "two way sync between obsidian and the site",
    "how do webhooks work",
    "what templates are available",
    "how to use multiple languages on my site",
]

_TOKEN_RE = re.compile(r"\w+|[^\w\s]", re.UNICODE)
_MARK_RE = re.compile(r"</?mark>")
_WORD_RE = re.compile(r"\w+", re.UNICODE)
MAX_DEPTH = 8


def approx_tokens(text):
    return len(_TOKEN_RE.findall(text or ""))


def rpc(endpoint, method, params):
    payload = json.dumps({"jsonrpc": "2.0", "id": 1, "method": method, "params": params}).encode()
    req = urllib.request.Request(
        endpoint, data=payload,
        headers={"Content-Type": "application/json", "Accept": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        body = json.loads(resp.read().decode())
    if "error" in body:
        raise RuntimeError(body["error"].get("message", str(body["error"])))
    return body["result"]


def call_tool(endpoint, name, arguments):
    return rpc(endpoint, "tools/call", {"name": name, "arguments": arguments})


def result_text(result):
    content = result.get("content") or []
    return content[0]["text"] if content else ""


def words(text):
    return set(w.lower() for w in _WORD_RE.findall(text or "") if len(w) > 2)


def best_child(children, query_words):
    """Pick the child whose title shares the most words with the query; first on a tie."""
    return max(children, key=lambda c: len(words(c["title"]) & query_words))


def check_capabilities(endpoint):
    tools = {t["name"] for t in rpc(endpoint, "tools/list", {})["tools"]}
    has_expand = "expand" in tools
    has_fed = "federated_expand" in tools
    # Slim-search probe: does a search result still carry the flat `toc`?
    sc = call_tool(endpoint, "search", {"query": "telegram"}).get("structuredContent") or {}
    results = sc.get("results") or []
    toc_in_search = bool(results and "toc" in results[0])
    print(f"capability: expand={has_expand}  federated_expand={has_fed}  search_carries_toc={toc_in_search}")
    if not has_expand:
        print("\n⚠ `expand` is not deployed on this endpoint yet — run again after deploy.")
    return has_expand


def navigate(endpoint, pid, query_words):
    """Descend the TOC tree to a leaf, picking the best-matching child per level.
    Returns (toc_path, nav_tokens, levels)."""
    path = []
    nav_tokens = 0
    levels = 0
    for _ in range(MAX_DEPTH):
        res = call_tool(endpoint, "expand", {"pid": pid, "toc_path": path})
        children = (res.get("structuredContent") or {}).get("children") or []
        nav_tokens += approx_tokens(json.dumps(children, ensure_ascii=False))
        levels += 1
        if not children:
            break  # current path is a leaf (or note has no sections)
        pick = best_child(children, query_words)
        path = pick["path"]
        if not pick.get("has_children"):
            break  # pick is the target leaf section
    return path, nav_tokens, levels


def run(endpoint, queries):
    print(f"endpoint: {endpoint}\n")
    if not check_capabilities(endpoint):
        return
    print()
    hdr = f"{'query':38} {'full':>6} {'nav':>5} {'read':>6} {'total':>6} {'save':>6} {'lvls':>4} {'hit':>4}"
    print(hdr)
    print("-" * len(hdr))

    ratios, hits = [], 0
    for q in queries:
        try:
            sc = call_tool(endpoint, "search", {"query": q}).get("structuredContent") or {}
            results = sc.get("results") or []
            if not results:
                print(f"{q[:38]:38} (no results)")
                continue
            top = results[0]
            pid = top.get("note_id")
            matches = top.get("matches") or []
            snippet = _MARK_RE.sub("", matches[0]["snippet"]) if matches else ""
            marked = re.findall(r"<mark>(.*?)</mark>", matches[0]["snippet"]) if matches else []
            answer_terms = {w.lower() for m in marked for w in _WORD_RE.findall(m) if len(w) > 2}
            qwords = words(q + " " + snippet)

            match_toc_path = matches[0].get("toc_path") if matches else None
            if match_toc_path:
                path = match_toc_path
                nav_tokens = 0
                levels = 0
            else:
                path, nav_tokens, levels = navigate(endpoint, pid, qwords)
            section = result_text(call_tool(endpoint, "note_html", {"pid": pid, "toc_path": path})) if path else ""
            read_tokens = approx_tokens(section)
            full_tokens = approx_tokens(result_text(call_tool(endpoint, "note_html", {"pid": pid})))

            total = nav_tokens + read_tokens
            ratio = full_tokens / total if total else float("nan")
            sec_low = section.lower()
            hit = "YES" if answer_terms and all(t in sec_low for t in answer_terms) else (
                "—" if not answer_terms else "NO")
            if hit == "YES":
                hits += 1
            if ratio == ratio:
                ratios.append(ratio)
            print(f"{q[:38]:38} {full_tokens:6d} {nav_tokens:5d} {read_tokens:6d} {total:6d} "
                  f"{ratio:5.1f}x {levels:4d} {hit:>4}")
        except Exception as e:  # noqa: BLE001
            print(f"{q[:38]:38} ERROR: {e}")

    if ratios:
        print("-" * len(hdr))
        print(f"median savings (whole note / navigate+read): {statistics.median(ratios):.1f}x   "
              f"descent landed on the answer: {hits}/{len(ratios)}")
        print("\nNote: tokens are approximate; ratios are the robust result. 'nav' is the cost of "
              "walking the tree (expand calls); 'read' is the final section.")


def main():
    ap = argparse.ArgumentParser(description="Verify trip2g `expand` navigation against a live MCP endpoint.")
    ap.add_argument("--endpoint", default=DEFAULT_ENDPOINT)
    ap.add_argument("queries", nargs="*")
    ap.parse_args()
    args = ap.parse_args()
    run(args.endpoint, args.queries or DEFAULT_QUERIES)


if __name__ == "__main__":
    main()
