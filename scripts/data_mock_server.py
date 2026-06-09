#!/usr/bin/env python3
"""Mock data adapter for the trip2g datachart `url` source (local demo).

Mimics trip2g_agent_queue's `POST /v1/query` contract:
  request:  {"sql": "SELECT ..."}
  response: a flat JSON array of row objects (SELECT-only; else 400).

Returns canned demo datasets matching the queries in docs/demo/datachart_*.md,
chosen by the table named in the SQL. No external deps (stdlib only).
"""
import json
import re
from http.server import BaseHTTPRequestHandler, HTTPServer

# Datasets keyed by the table referenced in the demo SQL queries.
DATASETS = {
    "stats": [  # datachart_fenced.md: SELECT day, revenue FROM stats
        {"day": "2026-06-01", "revenue": 1234},
        {"day": "2026-06-02", "revenue": 1980},
        {"day": "2026-06-03", "revenue": 1520},
        {"day": "2026-06-04", "revenue": 2410},
        {"day": "2026-06-05", "revenue": 2090},
    ],
    "sales": [  # datachart_types.md (bar): SELECT product, revenue FROM sales
        {"product": "Pro plan", "revenue": 5200},
        {"product": "Team plan", "revenue": 3800},
        {"product": "Starter", "revenue": 2600},
        {"product": "Add-ons", "revenue": 1400},
        {"product": "Support", "revenue": 900},
    ],
    "traffic": [
        {"source": "direct", "hits": 30},
        {"source": "search", "hits": 50},
        {"source": "social", "hits": 20},
    ],
}


def rows_for_sql(sql: str):
    s = sql.lower()
    for table, rows in DATASETS.items():
        if re.search(r"\bfrom\s+" + re.escape(table) + r"\b", s):
            return rows
    # Generic fallback so unknown queries still render something.
    return [{"x": i, "y": (i * 7) % 13} for i in range(1, 8)]


class Handler(BaseHTTPRequestHandler):
    def _send(self, code, payload):
        body = json.dumps(payload).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/healthz":
            self._send(200, {"ok": True})
        else:
            self._send(404, {"error": "not found"})

    def do_POST(self):
        if self.path != "/v1/query":
            self._send(404, {"error": "not found"})
            return
        length = int(self.headers.get("Content-Length") or 0)
        raw = self.rfile.read(length) if length else b"{}"
        try:
            req = json.loads(raw or b"{}")
        except json.JSONDecodeError:
            self._send(400, {"error": "invalid JSON body"})
            return
        sql = (req.get("sql") or "").strip()
        if not sql.lower().startswith("select"):
            self._send(400, {"error": "only SELECT queries allowed"})
            return
        self._send(200, rows_for_sql(sql))

    def log_message(self, fmt, *args):  # keep the container log quiet
        pass


if __name__ == "__main__":
    print("data_mock_server listening on :8090 (POST /v1/query)", flush=True)
    HTTPServer(("0.0.0.0", 8090), Handler).serve_forever()
