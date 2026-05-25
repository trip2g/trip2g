#!/usr/bin/env python3
"""
formsubmits — CLI for listing and marking admin form submissions.

Reads API key and URL from .obsidian/plugins/trip2g/data.json
(walks up from cwd) or from environment variables TRIP2G_API_KEY / TRIP2G_API_URL.

Usage examples:
  formsubmits list
  formsubmits list --unprocessed
  formsubmits list --unprocessed --limit 10
  formsubmits list --note-path-id 42 --status pending
  formsubmits list --form-id contact --created-after 2026-05-01
  formsubmits show 17
  formsubmits mark 17 --comment "spam, ignored"
  formsubmits count --unprocessed
"""

import argparse
import json
import os
import sys
import urllib.request
import urllib.error
from pathlib import Path


def find_config() -> dict:
    current = Path.cwd()
    for directory in [current, *current.parents]:
        candidate = directory / ".obsidian" / "plugins" / "trip2g" / "data.json"
        if candidate.exists():
            with open(candidate) as f:
                data = json.load(f)
            sync_dirs = data.get("syncDirs", [])
            if sync_dirs:
                return {
                    "api_key": sync_dirs[0].get("apiKey", ""),
                    "api_url": sync_dirs[0].get("apiUrl", "http://localhost:8081"),
                }
    return {}


def graphql(api_url: str, api_key: str, query: str, variables: dict, cookie: str = "") -> dict:
    body = json.dumps({"query": query, "variables": variables}).encode()
    headers: dict = {"Content-Type": "application/json"}
    if cookie:
        headers["Cookie"] = cookie
    else:
        headers["X-API-Key"] = api_key
    req = urllib.request.Request(
        f"{api_url}/_system/graphql",
        data=body,
        headers=headers,
        method="POST",
    )
    try:
        with urllib.request.urlopen(req) as resp:
            payload = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        payload = json.loads(e.read())

    if payload.get("errors"):
        print("GraphQL errors:", file=sys.stderr)
        for err in payload["errors"]:
            print(" ", err.get("message", err), file=sys.stderr)
        sys.exit(1)

    return payload.get("data") or {}


LIST_QUERY = """
query AdminFormSubmits($filter: AdminFormSubmitsFilterInput) {
  admin {
    formSubmits(filter: $filter) {
      totalCount
      nodes {
        id
        noteVersionId
        formId
        ip
        status
        createdAt
        processedAt
        comment
        fields {
          __typename
          ... on AdminFormStringValue { name value }
          ... on AdminFormIntValue    { name iv: value }
          ... on AdminFormBoolValue   { name bv: value }
        }
      }
    }
  }
}
"""

MARK_MUTATION = """
mutation MarkProcessed($input: MarkFormSubmitProcessedInput!) {
  admin {
    markFormSubmitProcessed(input: $input) {
      __typename
      ... on MarkFormSubmitProcessedPayload {
        submit { id processedAt comment }
      }
      ... on ErrorPayload {
        message
        fields { name message }
      }
    }
  }
}
"""


def build_filter(args) -> dict:
    f = {}
    if args.note_path_id is not None:
        f["notePathId"] = args.note_path_id
    if args.form_id is not None:
        f["formId"] = args.form_id
    if args.status is not None:
        f["status"] = args.status
    if args.unprocessed:
        f["processed"] = False
    elif args.processed:
        f["processed"] = True
    if args.created_after or args.created_before:
        f["createdAt"] = {}
        if args.created_after:
            f["createdAt"]["gte"] = args.created_after
        if args.created_before:
            f["createdAt"]["lte"] = args.created_before
    if args.limit is not None:
        f["limit"] = args.limit
    if args.offset is not None:
        f["offset"] = args.offset
    return f


def format_field(field: dict) -> str:
    t = field["__typename"]
    if t == "AdminFormStringValue":
        return f"{field['name']}={field['value']!r}"
    if t == "AdminFormIntValue":
        return f"{field['name']}={field['iv']}"
    if t == "AdminFormBoolValue":
        return f"{field['name']}={field['bv']}"
    return f"{field['name']}=?"


def print_submit(s: dict, full: bool):
    processed = "✓" if s.get("processedAt") else "·"
    head = f"[{processed}] #{s['id']} {s['createdAt']}  form={s['formId'] or '-'}  status={s['status']}  ip={s['ip']}"
    print(head)
    if full:
        if s.get("processedAt"):
            print(f"    processed_at={s['processedAt']}  comment={s.get('comment') or ''!r}")
        for f in s.get("fields") or []:
            print(f"    {format_field(f)}")


def cmd_list(args, api_url, api_key, cookie=""):
    data = graphql(api_url, api_key, LIST_QUERY, {"filter": build_filter(args)}, cookie)
    conn = data["admin"]["formSubmits"]
    nodes = conn["nodes"]
    if args.json:
        print(json.dumps(conn, indent=2, ensure_ascii=False))
        return
    print(f"total: {conn['totalCount']}  shown: {len(nodes)}")
    for s in nodes:
        print_submit(s, full=args.verbose)


def cmd_show(args, api_url, api_key, cookie=""):
    data = graphql(api_url, api_key, LIST_QUERY, {"filter": {"limit": 200}}, cookie)
    target = next((n for n in data["admin"]["formSubmits"]["nodes"] if n["id"] == args.id), None)
    if target is None:
        # Fall back: scan with offset until found or end (rare).
        print(f"submit #{args.id} not in first 200 results — narrow with filters", file=sys.stderr)
        sys.exit(1)
    print_submit(target, full=True)


def cmd_count(args, api_url, api_key, cookie=""):
    args.limit = 0  # nodes empty, totalCount still computed
    args.offset = 0
    data = graphql(api_url, api_key, LIST_QUERY, {"filter": build_filter(args)}, cookie)
    print(data["admin"]["formSubmits"]["totalCount"])


def cmd_mark(args, api_url, api_key):
    inp = {"submitId": args.id}
    if args.comment:
        inp["comment"] = args.comment
    data = graphql(api_url, api_key, MARK_MUTATION, {"input": inp})
    payload = data["admin"]["markFormSubmitProcessed"]
    if payload["__typename"] == "ErrorPayload":
        print(f"ERROR: {payload.get('message')}", file=sys.stderr)
        for fe in payload.get("fields") or []:
            print(f"  {fe['name']}: {fe['message']}", file=sys.stderr)
        sys.exit(1)
    s = payload["submit"]
    print(f"marked #{s['id']} processed at {s['processedAt']}")


def add_filter_args(p: argparse.ArgumentParser):
    p.add_argument("--note-path-id", type=int, help="Note path ID")
    p.add_argument("--form-id", help="Form ID")
    p.add_argument("--status", choices=["pending", "visible", "hidden"])
    g = p.add_mutually_exclusive_group()
    g.add_argument("--unprocessed", action="store_true", help="Only unprocessed (processed_at IS NULL)")
    g.add_argument("--processed", action="store_true", help="Only processed")
    p.add_argument("--created-after", help="RFC3339, e.g. 2026-05-01T00:00:00Z")
    p.add_argument("--created-before", help="RFC3339, e.g. 2026-05-31T23:59:59Z")
    p.add_argument("--limit", type=int, help="Page size (default 50, max 200)")
    p.add_argument("--offset", type=int, help="Page offset")


def main():
    parser = argparse.ArgumentParser(description="List and process admin form submissions")
    parser.add_argument("--api-key", help="API key (overrides config and env)")
    parser.add_argument("--api-url", help="Server URL (overrides config and env)")
    sub = parser.add_subparsers(dest="cmd", required=True)

    p_list = sub.add_parser("list", help="List submissions")
    add_filter_args(p_list)
    p_list.add_argument("-v", "--verbose", action="store_true", help="Show field values")
    p_list.add_argument("--json", action="store_true", help="Raw JSON output")
    p_list.set_defaults(func=cmd_list)

    p_show = sub.add_parser("show", help="Show one submission with fields")
    p_show.add_argument("id", type=int)
    p_show.set_defaults(func=cmd_show)

    p_count = sub.add_parser("count", help="Print totalCount for a filter")
    add_filter_args(p_count)
    p_count.set_defaults(func=cmd_count)

    p_mark = sub.add_parser("mark", help="Mark submission processed")
    p_mark.add_argument("id", type=int)
    p_mark.add_argument("--comment", help="Optional comment")
    p_mark.set_defaults(func=cmd_mark)

    args = parser.parse_args()

    config = find_config()
    api_key = args.api_key or os.environ.get("TRIP2G_API_KEY") or config.get("api_key", "")
    api_url = args.api_url or os.environ.get("TRIP2G_API_URL") or config.get("api_url", "http://localhost:8081")
    api_url = api_url.rstrip("/")

    if not api_key:
        print("Error: no API key found. Set TRIP2G_API_KEY or provide --api-key.", file=sys.stderr)
        sys.exit(1)

    args.func(args, api_url, api_key)


if __name__ == "__main__":
    main()
