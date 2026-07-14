---
executor: code
write_patterns:
  - "calls/**"
requires_secrets:
  - KRISP_TOKEN
---

# Pull the latest Krisp calls

This is a codellm **python skill**: a role body whose fenced block codellm executes.
It pulls the latest meetings from the live Krisp API and writes each one as a note.

The block reads its input from `$FLEET_INPUT` (the delivery bag): a JSON object with
`krisp_token` (the Krisp bearer token) and `n` (how many latest calls to pull). It calls
the Krisp API (`https://api.krisp.ai/v2`) directly with the stdlib — no third-party
packages, so it runs under codellm's scrubbed sandbox env. It emits the codellm stdout
contract on the last line of stdout: `{"changes":[...],"answer":"..."}`.

```python
import json, os, urllib.request, sys

# --- codellm delivery bag: $FLEET_INPUT points at a temp file with the JSON bag ---
bag = {}
p = os.environ.get("FLEET_INPUT")
if p and os.path.exists(p):
    bag = json.loads(open(p).read() or "{}")

token = (bag.get("krisp_token") or "").strip()
n = int(bag.get("n") or 3)
if not token:
    # Fail loud: the productized path injects KRISP_TOKEN at codellm (requires_secrets
    # manifest). For this demo it rides the bag. Either way, no token => no run.
    print(json.dumps({"changes": [], "answer": "error: KRISP_TOKEN missing from $FLEET_INPUT"}))
    sys.exit(0)

KRISP_BASE = "https://api.krisp.ai/v2"


def krisp(method, path, body=None):
    req = urllib.request.Request(
        KRISP_BASE + path,
        data=json.dumps(body).encode() if body is not None else None, method=method,
        headers={"Authorization": "Bearer " + token, "krisp_header_app": "web",
                 "krisp_header_web_project": "note", "krisp_origin_timezone": "+00:00",
                 "Origin": "https://app.krisp.ai", "Content-Type": "application/json",
                 "Accept": "application/json, text/plain, */*"})
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read())


def list_latest(limit):
    data = krisp("POST", "/meetings/list",
                 {"sort": "desc", "sortKey": "created_at", "page": 1,
                  "limit": max(limit * 4, 20), "isOwner": True}).get("data", {})
    rows = [m for m in (data.get("rows", []) or [])
            if m.get("id") and not m.get("is_demo") and int(m.get("duration", 0) or 0) >= 60]
    rows.sort(key=lambda m: m.get("started_at") or "", reverse=True)
    return rows[:limit]


def speaker_names(m):
    out = {}
    for i, s in enumerate(m.get("speakers", []) or [], start=1):
        nm = (s.get("first_name") or s.get("name") or "").strip()
        if nm:
            out[i] = nm
    return out


def extract_rows(tree):
    rows = []

    def walk(x):
        if isinstance(x, dict):
            if "speakerIndex" in x and isinstance(x.get("speech"), dict):
                sp = x["speech"]
                t = str(sp.get("text", "")).strip()
                if t:
                    rows.append({"start": float(sp.get("start", 0) or 0),
                                 "spk": int(x.get("speakerIndex", 0) or 0), "text": t})
            for v in x.values():
                walk(v)
        elif isinstance(x, list):
            for v in x:
                walk(v)

    walk(tree)
    rows.sort(key=lambda r: r["start"])
    return rows


def rows_to_text(rows, names):
    parts = []
    for r in rows:
        mm, ss = int(r["start"]) // 60, int(r["start"]) % 60
        label = names.get(r["spk"]) or ("Speaker %d" % r["spk"])
        parts.append("%s | %02d:%02d\n%s" % (label, mm, ss, r["text"]))
    return "\n".join(parts)


changes = []
pulled = []
for m in list_latest(n):
    mid = m["id"]
    names = speaker_names(m)
    try:
        rows = extract_rows(krisp("GET", "/block/" + mid + "/tree"))
    except Exception as e:
        rows = []
        err = str(e)
    else:
        err = ""
    started = (m.get("started_at") or "")[:16]
    title = (m.get("name") or "(no name)").strip()
    dur_min = int(m.get("duration", 0) or 0) // 60
    body = ["---",
            "krisp_id: %s" % mid,
            "started_at: %s" % started,
            "duration_min: %d" % dur_min,
            "speakers: %d" % len(m.get("speakers", []) or []),
            "---",
            "",
            "# %s" % title,
            "",
            "Pulled from the live Krisp API (`/meetings/list` + `/block/{id}/tree`).",
            ""]
    if rows:
        body += ["## Transcript", "", rows_to_text(rows, names)]
    else:
        body += ["## Transcript", "", "_(transcript unavailable: %s)_" % (err or "empty")]
    path = "calls/%s-%s.md" % ((m.get("started_at") or "")[:10], mid[:8])
    changes.append({"path": path, "content": "\n".join(body)})
    pulled.append("%s (%s, %dm, %d turns)" % (title, started, dur_min, len(rows)))

answer = "Pulled %d latest Krisp call(s): %s" % (len(changes), "; ".join(pulled)) if changes \
    else "No Krisp calls found."
print(json.dumps({"changes": changes, "answer": answer}))
```
