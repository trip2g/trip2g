import json, os, sys, time, urllib.request, urllib.error

MCP = "https://trip2g.com/_system/mcp"
MODEL = "anthropic/claude-haiku-4.5"
REPEATS = 3
MAX_STEPS = 8

def load_key():
    for p in [os.path.expanduser("~/krisp-run/.env"), "/home/alexes/projects2/trip2g_agent_queue/.env"]:
        try:
            for line in open(p):
                if line.startswith("OPENROUTER_API_KEY="):
                    return line.strip().split("=", 1)[1]
        except OSError:
            pass
    return None

KEY = load_key()

# Instructions note (frontmatter stripped)
note = open(os.environ.get("INIT_NOTE", "docs/_mcp_initialize.md")).read()
NOTE = note.split("---", 2)[2].strip()

def mcp(method, params=None, _id=1):
    body = {"jsonrpc": "2.0", "method": method, "id": _id}
    if params is not None:
        body["params"] = params
    req = urllib.request.Request(MCP, data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json"})
    for attempt in range(3):
        try:
            return json.load(urllib.request.urlopen(req, timeout=30))
        except Exception as e:
            if attempt == 2:
                return {"error": str(e)}
            time.sleep(1.5)

# Fetch live tool schemas for the OpenAI-format tool list
tl = mcp("tools/list")["result"]["tools"]
KEEP = {"search", "similar", "note_html", "expand"}
TOOLS = []
for t in tl:
    if t["name"] not in KEEP:
        continue
    TOOLS.append({"type": "function", "function": {
        "name": t["name"],
        "description": t.get("description", "")[:900],
        "parameters": t.get("inputSchema", {"type": "object", "properties": {}}),
    }})

def call_tool(name, args):
    r = mcp("tools/call", {"name": name, "arguments": args}, _id=99)
    if "result" not in r:
        return "TOOL ERROR: " + json.dumps(r)[:300], 0
    text = ""
    for c in r["result"].get("content", []):
        if c.get("type") == "text":
            text += c["text"]
    return text, len(text)

def chat(messages, tools):
    body = {"model": MODEL, "max_tokens": 700, "messages": messages, "tools": tools, "tool_choice": "auto"}
    req = urllib.request.Request("https://openrouter.ai/api/v1/chat/completions",
        data=json.dumps(body).encode(),
        headers={"Authorization": "Bearer " + KEY, "Content-Type": "application/json"})
    for attempt in range(4):
        try:
            return json.load(urllib.request.urlopen(req, timeout=90))
        except urllib.error.HTTPError as e:
            if attempt == 3:
                raise
            time.sleep(3)
        except Exception:
            if attempt == 3:
                raise
            time.sleep(3)

BARE_SYS = ("You are an assistant with access to a knowledge base through MCP tools "
            "(search, note_html, expand, similar). Use them to answer the user's question, "
            "then give a final answer citing the source.")

DUMP_THRESHOLD = 6000  # chars; a whole-note fallback is far bigger than a section

def run_one(question, expect_paths, with_note):
    sys_prompt = NOTE if with_note else BARE_SYS
    messages = [{"role": "system", "content": sys_prompt},
                {"role": "user", "content": question}]
    tool_calls = 0
    in_tok = out_tok = 0
    dumps = 0
    hit_path = False
    hit_section = False
    trace = []
    for step in range(MAX_STEPS):
        resp = chat(messages, TOOLS)
        u = resp.get("usage", {})
        in_tok += u.get("prompt_tokens", 0)
        out_tok += u.get("completion_tokens", 0)
        msg = resp["choices"][0]["message"]
        tcs = msg.get("tool_calls") or []
        if not tcs:
            final = msg.get("content", "") or ""
            trace.append({"final": final[:500]})
            break
        messages.append({"role": "assistant", "content": msg.get("content") or "", "tool_calls": tcs})
        for tc in tcs:
            tool_calls += 1
            name = tc["function"]["name"]
            try:
                args = json.loads(tc["function"]["arguments"] or "{}")
            except json.JSONDecodeError:
                args = {}
            out, ln = call_tool(name, args)
            # detect right note/section
            for ep in expect_paths:
                if ep.lower() in out.lower() or ep.lower() in json.dumps(args).lower():
                    hit_path = True
            if name == "note_html":
                if "toc_path" in args and args.get("toc_path"):
                    hit_section = True
                if ln > DUMP_THRESHOLD:
                    dumps += 1
            trace.append({"tool": name, "args": args, "out_len": ln})
            messages.append({"role": "tool", "tool_call_id": tc["id"], "content": out[:4000]})
    return {
        "tool_calls": tool_calls, "in_tok": in_tok, "out_tok": out_tok,
        "dumps": dumps, "hit_path": hit_path, "hit_section": hit_section,
        "trace": trace,
    }

QUESTIONS = [
    {"q": "How do I set up federation with a private peer using an HMAC secret?",
     "expect": ["federation", "HMAC", "inbound secret"]},
    {"q": "What tools does the trip2g MCP server expose?",
     "expect": ["search", "expand", "note_html", "federated"]},
    {"q": "How do I publish an Obsidian vault as a website with trip2g?",
     "expect": ["obsidian", "publish", "sync"]},
    {"q": "What is a subgraph in trip2g and how does access control work?",
     "expect": ["subgraph", "access", "subscrib"]},
    {"q": "How does the token economy toc_path drill-down save tokens?",
     "expect": ["toc_path", "token", "section"]},
    {"q": "What is a fuzzy pointer and how does search find the exact section?",
     "expect": ["fuzzy", "breadcrumb", "toc_path"]},
    {"q": "How do I connect trip2g to Telegram for publishing?",
     "expect": ["telegram", "publish", "channel"]},
    {"q": "What does the expand tool do and when should I use it?",
     "expect": ["expand", "table of contents", "progressive"]},
]

def main():
    if not KEY:
        print("NO KEY", file=sys.stderr); sys.exit(1)
    results = {"with": [], "without": []}
    for qi, item in enumerate(QUESTIONS):
        for variant, wn in [("with", True), ("without", False)]:
            for rep in range(REPEATS):
                try:
                    r = run_one(item["q"], item["expect"], wn)
                except Exception as e:
                    r = {"error": str(e), "tool_calls": 0, "in_tok": 0, "out_tok": 0,
                         "dumps": 0, "hit_path": False, "hit_section": False, "trace": []}
                r["q_idx"] = qi
                r["question"] = item["q"]
                r["variant"] = variant
                r["rep"] = rep
                results[variant].append(r)
                print(f"[{variant:7}] q{qi} rep{rep}: calls={r['tool_calls']} "
                      f"dumps={r['dumps']} hit={r['hit_path']} sec={r['hit_section']} "
                      f"tok={r['in_tok']+r['out_tok']}", flush=True)
                time.sleep(0.5)
    json.dump(results, open("ab_results.json", "w"), indent=1)
    # aggregate
    def agg(rows):
        n = len(rows)
        return {
            "n": n,
            "correct_pct": 100.0 * sum(r["hit_path"] for r in rows) / n,
            "avg_calls": sum(r["tool_calls"] for r in rows) / n,
            "avg_tok": sum(r["in_tok"] + r["out_tok"] for r in rows) / n,
            "avg_dumps": sum(r["dumps"] for r in rows) / n,
            "section_pct": 100.0 * sum(r["hit_section"] for r in rows) / n,
            "total_tok": sum(r["in_tok"] + r["out_tok"] for r in rows),
        }
    A = agg(results["with"]); B = agg(results["without"])
    print("\n=== AGGREGATE ===")
    print("WITH   :", A)
    print("WITHOUT:", B)
    json.dump({"with": A, "without": B}, open("ab_agg.json", "w"), indent=1)

if __name__ == "__main__":
    main()
