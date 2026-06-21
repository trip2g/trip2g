#!/usr/bin/env python3
"""
trip2g stdio MCP adapter — ONE composite tool over the trip2g knowledge base.

Zero dependencies (Python 3 stdlib only), so an agent can launch it directly with
no pip install. It speaks the MCP stdio transport (newline-delimited JSON-RPC 2.0
on stdin/stdout; logs go to stderr) and exposes a single, self-describing tool that
wraps several upstream operations — search, expand (TOC navigation), note_html —
into one call, returning only the most relevant section.

Configure it in an MCP client (Claude Desktop / Cursor / Code / any agent):

    {
      "mcpServers": {
        "trip2g": {
          "command": "python3",
          "args": ["/abs/path/to/scripts/trip2g_mcp_stdio_adapter.py"],
          "env": { "TRIP2G_MCP_URL": "https://trip2g.com/_system/mcp" }
        }
      }
    }

Env:
  TRIP2G_MCP_URL  upstream HTTP MCP endpoint (default https://trip2g.com/_system/mcp)
  TRIP2G_TOKEN    optional Bearer token (t2g_...) for private/subscriber content
"""

import json
import os
import re
import sys
import urllib.request

UPSTREAM = os.environ.get("TRIP2G_MCP_URL", "https://trip2g.com/_system/mcp")
TOKEN = os.environ.get("TRIP2G_TOKEN", "").strip()
PROTOCOL = "2024-11-05"

TOOL_NAME = "search_and_read_the_most_relevant_section_from_the_trip2g_knowledge_base"
TOOL_DESC = (
    "Why: returns ONLY the single most relevant section that answers your question — "
    "not the whole note — so you spend a fraction of the tokens and the answer lands "
    "at the top of the context window. "
    "When: use whenever you need a specific fact, definition, setting, or how-to from "
    "the trip2g knowledge base. "
    "How it works (one call): it searches the base, navigates the matched note's table "
    "of contents to the exact section, and reads just that section — so you never dump "
    "a whole note or grep blindly."
)

_WORD = re.compile(r"\w+", re.UNICODE)
_MARK = re.compile(r"</?mark>")


def log(*a):
    print("[trip2g-adapter]", *a, file=sys.stderr, flush=True)


def upstream(name, arguments):
    """Call one tool on the upstream HTTP MCP server (JSON-RPC 2.0)."""
    payload = json.dumps({
        "jsonrpc": "2.0", "id": 1, "method": "tools/call",
        "params": {"name": name, "arguments": arguments},
    }).encode()
    headers = {"Content-Type": "application/json", "Accept": "application/json"}
    if TOKEN:
        headers["Authorization"] = "Bearer " + TOKEN
    req = urllib.request.Request(UPSTREAM, data=payload, headers=headers, method="POST")
    with urllib.request.urlopen(req, timeout=30) as r:
        body = json.loads(r.read().decode())
    if "error" in body:
        raise RuntimeError(body["error"].get("message", str(body["error"])))
    return body["result"]


def text_of(result):
    content = result.get("content") or []
    return content[0]["text"] if content else ""


def words(s):
    return {w.lower() for w in _WORD.findall(s or "") if len(w) > 2}


def descend(pid, query_words):
    """Walk the note's TOC to the best-matching leaf via expand; return its toc_path.
    Resilient: if expand is unavailable (e.g. not deployed yet) it returns what it has
    so the caller can fall back to a focused chunk window or the whole note."""
    path = []
    for _ in range(8):
        try:
            res = upstream("expand", {"pid": pid, "toc_path": path})
        except Exception:
            return path
        children = (res.get("structuredContent") or {}).get("children") or []
        if not children:
            break
        pick = max(children, key=lambda c: len(words(c.get("title")) & query_words))
        path = pick.get("path") or path
        if not pick.get("has_children"):
            break
    return path


def run_composite(arguments):
    query = ((arguments or {}).get("query") or "").strip()
    if not query:
        return {"content": [{"type": "text", "text": "query is required"}], "isError": True}

    sr = upstream("search", {"query": query}).get("structuredContent") or {}
    results = sr.get("results") or []
    if not results:
        return {"content": [{"type": "text", "text": f"No results for: {query}"}]}

    top = results[0]
    pid = top.get("note_id")
    matches = top.get("matches") or []

    # Prefer the precise pointer the search already attached to the matched chunk;
    # otherwise navigate the TOC tree to the best section.
    toc_path = matches[0].get("toc_path") if matches else None
    if not toc_path:
        snippet = _MARK.sub("", matches[0]["snippet"]) if matches else ""
        toc_path = descend(pid, words(query + " " + snippet))

    section = text_of(upstream("note_html", {"pid": pid, "toc_path": toc_path})) if toc_path else ""
    if not section and matches:  # fall back to the focused chunk window
        section = text_of(upstream("note_html", {"pid": pid, "match_id": matches[0]["match_id"]}))
    if not section:  # last resort: the whole note
        section = text_of(upstream("note_html", {"pid": pid}))

    breadcrumb = " > ".join([top.get("title", "")] + (toc_path or []))
    header = f"Source: {top.get('title', '')} ({top.get('url', '')})\nSection: {breadcrumb}\n\n"
    return {
        "content": [{"type": "text", "text": header + section}],
        "structuredContent": {
            "note_id": pid,
            "note_path": top.get("note_path"),
            "url": top.get("url"),
            "title": top.get("title"),
            "toc_path": toc_path,
        },
    }


TOOLS = [{
    "name": TOOL_NAME,
    "description": TOOL_DESC,
    "inputSchema": {
        "type": "object",
        "properties": {
            "query": {"type": "string", "description": "Natural-language question or keywords to answer from the knowledge base."},
        },
        "required": ["query"],
    },
}]


def handle(msg):
    method = msg.get("method")
    mid = msg.get("id")

    if method == "initialize":
        client_proto = (msg.get("params") or {}).get("protocolVersion") or PROTOCOL
        return ok(mid, {
            "protocolVersion": client_proto,
            "capabilities": {"tools": {}},
            "serverInfo": {"name": "trip2g-adapter", "version": "1.0.0"},
        })
    if method == "notifications/initialized":
        return None
    if method == "tools/list":
        return ok(mid, {"tools": TOOLS})
    if method == "tools/call":
        params = msg.get("params") or {}
        if params.get("name") != TOOL_NAME:
            return err(mid, -32601, f"Unknown tool: {params.get('name')}")
        try:
            return ok(mid, run_composite(params.get("arguments")))
        except Exception as e:  # noqa: BLE001 — surface upstream errors as a tool error
            return ok(mid, {"content": [{"type": "text", "text": f"error: {e}"}], "isError": True})

    if mid is None:
        return None  # an unknown notification — nothing to answer
    return err(mid, -32601, f"Method not found: {method}")


def ok(mid, result):
    return {"jsonrpc": "2.0", "id": mid, "result": result}


def err(mid, code, message):
    return {"jsonrpc": "2.0", "id": mid, "error": {"code": code, "message": message}}


def main():
    log(f"ready; upstream={UPSTREAM}")
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            msg = json.loads(line)
        except Exception as e:  # noqa: BLE001
            log("parse error:", e)
            continue
        resp = handle(msg)
        if resp is not None:
            sys.stdout.write(json.dumps(resp) + "\n")
            sys.stdout.flush()


if __name__ == "__main__":
    main()
