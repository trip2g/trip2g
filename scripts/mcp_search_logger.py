#!/usr/bin/env python3
"""Drive an LLM through the MCP search tools and log the walk.

A headless port of docs/_layouts/search_visualizer.html's task loop: same
system prompt, same tool filter, same [progress] nudges, same trace format, so
a saved trace can be imported into /search_visualizer and replayed. Use it to
reproduce an agent's navigation without a browser, or to record traces for a
demo.

    OPENROUTER_KEY=sk-or-... scripts/mcp_search_logger.py "<question>" out.json [model] [max_steps]

    MCP_ENDPOINT   defaults to https://trip2g.com/_system/mcp

Progress goes to stderr; the trace to out.json.
"""
import json, os, re, sys, time, urllib.request

ENDPOINT = os.environ.get("MCP_ENDPOINT", "https://trip2g.com/_system/mcp")
ALLOWED = ['search', 'similar', 'note_html', 'expand',
           'federated_search', 'federated_similar', 'federated_note_html', 'federated_expand']
KB_HINT = ' Nested bases use "/" in kb_id: e.g. kb_id="philosophers/machiavelli" routes through the philosophers peer into the base it federates.'
SYS_TASK = """You explore a documentation knowledge graph through MCP tools and answer the user's question from what you actually read — never from memory.
Loop: search(query) -> read only the matched section with note_html(pid, toc_path=[...]) -> expand(pid) if the pointer looks off. Use similar(pid) for neighbors.
If a search result has kind "federation_kb", or local search finds nothing, use federated_search (pass kb_id to target one base; follow-up reads need federated_note_html/expand/similar with the same kb_id). If you do not know a base's kb_id, call federated_search without kb_id to fan out: the [kb_id] headers in the response name the connected bases, then target one.
Hub, index, and MOC pages are only POINTERS — the real concepts, principles, and verbatim quotes live in the leaf corpora, one level deeper. Do NOT answer from hub cards alone: DESCEND into the specific base with federated_search(kb_id="philosophers/<author>") (nested bases chain with "/"), then read the concept note there.
To open any note, pass `path` (the note_path from a search result, e.g. "concepts/maska-i-glubina.md") or the result's `match_id`. NEVER invent a note_id — it is an internal integer, not a path; if you don't have one, omit it and use path.
Before you answer, ask yourself: am I deep enough? If your evidence comes from a hub card, an index, a MOC, or a topics matrix, that is a POINTER or a summary — not the source. You are deep enough only when you have opened the actual concept or principle note inside a leaf corpus and read its verbatim text. If you are still on pointers, descend one more level and read the real note before answering.
Think out loud briefly (1-2 sentences) before each tool call so an observer can follow your reasoning. Cite note URLs in the final answer. When you have enough, answer without calling more tools.
You have exactly {N} tool-call steps for this walk. Budget them deliberately — plan to use most of them, and reserve enough steps to verify what you found before answering. A progress line will report how many steps remain; when it tells you to wrap up, produce your final answer."""

# Server messages quote literal placeholders like <hub>/<base>; strip only real HTML tags.
HTML_TAG = re.compile(r"</?(?:a|b|i|u|s|p|br|hr|em|strong|small|sup|sub|del|ins|mark|code|pre|span|div|section|article"
                      r"|header|footer|nav|aside|main|h[1-6]|ul|ol|li|dl|dt|dd|table|thead|tbody|tfoot|tr|td|th"
                      r"|blockquote|figure|figcaption|img|details|summary|svg|path|input|button|label)\b[^>]*>", re.I)

rpc_id = 0


def post(url, body, headers=None):
    req = urllib.request.Request(url, data=json.dumps(body).encode(), method="POST",
                                 headers={"Content-Type": "application/json", **(headers or {})})
    with urllib.request.urlopen(req, timeout=120) as r:
        return json.load(r)


def rpc(method, params=None):
    global rpc_id
    rpc_id += 1
    j = post(ENDPOINT, {"jsonrpc": "2.0", "id": rpc_id, "method": method, "params": params or {}},
             {"Accept": "application/json, text/event-stream"})
    if "error" in j:
        raise RuntimeError(f"MCP error {j['error'].get('code')}: {j['error'].get('message')}")
    return j["result"]


def mcp_init():
    rpc("initialize", {"protocolVersion": "2024-11-05", "capabilities": {},
                       "clientInfo": {"name": "mcp-graph-visualizer", "version": "1.0"}})
    tools = rpc("tools/list")["tools"]
    return [{"type": "function", "function": {
        "name": t["name"],
        "description": (t.get("description") or "")[:900] + (KB_HINT if t["name"].startswith("federated") else ""),
        "parameters": t.get("inputSchema") or {"type": "object", "properties": {}},
    }} for t in tools if t["name"] in ALLOWED]


def content_text(result):
    return "\n".join(c["text"] for c in (result or {}).get("content", []) if c.get("type") == "text")


def slim(r):
    m = (r.get("matches") or [{}])[0]
    return {"title": r.get("title"), "note_id": r.get("note_id"), "note_path": r.get("note_path"),
            "url": r.get("url"), "kind": r.get("kind"),
            "federation": {"kb_id": r["federation"]["kb_id"]} if r.get("federation") else None,
            "snippet": HTML_TAG.sub("", m.get("snippet") or "")[:300] or None}


def trim_structured(sc):
    if not sc:
        return None
    out = {}
    if sc.get("results"):
        out["results"] = [slim(r) for r in sc["results"]]
    if sc.get("source"):
        out["source"] = slim(sc["source"])
    if sc.get("query"):
        out["query"] = sc["query"]
    return out


def main():
    key = os.environ["OPENROUTER_KEY"]
    task, out = sys.argv[1], sys.argv[2]
    model = sys.argv[3] if len(sys.argv) > 3 else "openai/gpt-5.4-mini"
    max_turns = int(sys.argv[4]) if len(sys.argv) > 4 else 16
    trace = {"version": 1, "mode": "task", "task": task, "model": model, "endpoint": ENDPOINT,
             "recorded": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "events": [], "cost": 0}
    tools = mcp_init()
    print(f"connected — {len(tools)} tools", file=sys.stderr)
    messages = [{"role": "system", "content": SYS_TASK.replace("{N}", str(max_turns))},
                {"role": "user", "content": task}]
    steps = 0
    for turn in range(1, max_turns + 1):
        if steps > 0:
            left = max_turns - steps
            prog = f"[progress] Step {steps} of {max_turns} used; {left} remaining."
            if left <= 2:
                prog += " Wrap up now: produce your final answer on your next message."
            messages.append({"role": "user", "content": prog})
        j = post("https://openrouter.ai/api/v1/chat/completions",
                 {"model": model, "messages": messages, "tools": tools, "usage": {"include": True}},
                 {"Authorization": "Bearer " + key})
        if "error" in j:
            raise RuntimeError("OpenRouter: " + json.dumps(j["error"]))
        if j.get("usage", {}).get("cost") is not None:
            trace["cost"] += j["usage"]["cost"]
        msg = j["choices"][0]["message"]
        if msg.get("content"):
            trace["events"].append({"t": "assistant", "text": msg["content"]})
            print(f"[{turn}] 💭 {msg['content'][:200]}", file=sys.stderr)
        if not msg.get("tool_calls"):
            trace["events"].append({"t": "final", "text": msg.get("content") or "(no text)"})
            break
        messages.append(msg)
        for tc in msg["tool_calls"]:
            try:
                args = json.loads(tc["function"].get("arguments") or "{}")
            except Exception:
                args = {}
            t0 = time.time()
            result, err = None, None
            try:
                result = rpc("tools/call", {"name": tc["function"]["name"], "arguments": args})
            except Exception as e:
                err = str(e)
            steps += 1
            ms = round((time.time() - t0) * 1000)
            text = "" if err else content_text(result)
            structured = None if err else result.get("structuredContent")
            print(f"[{turn}] → {tc['function']['name']} {json.dumps(args, ensure_ascii=False)[:160]} {ms}ms", file=sys.stderr)
            print(f"      ← {(err or HTML_TAG.sub(' ', text))[:160]!r}", file=sys.stderr)
            trace["events"].append({"t": "tool", "name": tc["function"]["name"], "args": args, "ms": ms, "err": err,
                                    "result": None if err else {"text": HTML_TAG.sub(" ", text)[:400],
                                                                "structured": trim_structured(structured)}})
            messages.append({"role": "tool", "tool_call_id": tc["id"],
                             "content": ("ERROR: " + err) if err else text[:12000]})
        if turn == max_turns:
            trace["events"].append({"t": "final", "text": "(max turns reached)"})
    json.dump(trace, open(out, "w"), ensure_ascii=False, indent=1)
    print(f"done: {steps} tool calls, ${trace['cost']:.4f} → {out}", file=sys.stderr)


if __name__ == "__main__":
    main()
