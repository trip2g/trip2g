#!/usr/bin/env python3
"""
renderlayout — CLI wrapper for /_system/renderlayout.

Reads API key and URL from .obsidian/plugins/trip2g/data.json
(walks up from cwd) or from environment variables TRIP2G_API_KEY / TRIP2G_API_URL.

Usage examples:
  renderlayout --layout-file _layouts/iiworker/landing.html --note-src "hello"
  renderlayout --layout-file _layouts/iiworker/landing.html --note-path /landings/sales
  renderlayout --layout-src "{{ note.M().Debug() }}" --note-path /demo/simple
  renderlayout --layout-path /_layouts/iiworker/landing.html --note-path /demo/simple
"""

import argparse
import json
import os
import sys
import urllib.request
import urllib.error
from pathlib import Path


def find_config() -> dict:
    """Walk up from cwd looking for .obsidian/plugins/trip2g/data.json."""
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


def main():
    parser = argparse.ArgumentParser(
        description="Render a Jet layout template via /_system/renderlayout"
    )

    parser.add_argument("--layout-path", help="Layout path on server, e.g. /_layouts/foo.html")
    parser.add_argument("--layout-file", help="Read layout src from this local file")
    parser.add_argument("--layout-src", help="Inline layout template source")

    parser.add_argument("--note-path", help="Note path on server, e.g. /my-note")
    parser.add_argument("--note-file", help="Read note markdown from this local file")
    parser.add_argument("--note-src", help="Inline note markdown source")

    parser.add_argument("--api-key", help="API key (overrides config)")
    parser.add_argument("--api-url", help="Server URL (overrides config)")
    parser.add_argument("--fetch", action="store_true", help="Fetch and print rendered HTML")

    args = parser.parse_args()

    # Resolve credentials.
    config = find_config()
    api_key = args.api_key or os.environ.get("TRIP2G_API_KEY") or config.get("api_key", "")
    api_url = args.api_url or os.environ.get("TRIP2G_API_URL") or config.get("api_url", "http://localhost:8081")
    api_url = api_url.rstrip("/")

    if not api_key:
        print("Error: no API key found. Set TRIP2G_API_KEY or provide --api-key.", file=sys.stderr)
        sys.exit(1)

    # Build layout object.
    layout_path = args.layout_path
    if args.layout_file:
        layout_path = layout_path or ("/" + args.layout_file.lstrip("/"))
        with open(args.layout_file) as f:
            layout_src = f.read()
    else:
        layout_src = args.layout_src

    if not layout_path:
        print("Error: --layout-path or --layout-file is required.", file=sys.stderr)
        sys.exit(1)

    layout = {"path": layout_path}
    if layout_src is not None:
        layout["src"] = layout_src

    # Build note object.
    note = None
    if args.note_path:
        note = {"path": args.note_path}
    elif args.note_file:
        with open(args.note_file) as f:
            note = {"src": f.read()}
    elif args.note_src is not None:
        note = {"src": args.note_src}

    payload = {"layout": layout}
    if note is not None:
        payload["note"] = note

    # POST request.
    body = json.dumps(payload).encode()
    req = urllib.request.Request(
        f"{api_url}/_system/renderlayout",
        data=body,
        headers={"Content-Type": "application/json", "X-API-Key": api_key},
        method="POST",
    )

    try:
        with urllib.request.urlopen(req) as resp:
            result = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        result = json.loads(e.read())

    # Print warnings if any.
    warnings = result.get("warnings", {})
    layout_warns = warnings.get("layout", [])
    if layout_warns:
        print("WARNINGS:", file=sys.stderr)
        for w in layout_warns:
            print(" ", w, file=sys.stderr)

    if "error" in result:
        print(f"ERROR: {result['error']}", file=sys.stderr)
        sys.exit(1)

    preview_url = result.get("previewURL", "")
    preview_id = result.get("previewID", "")
    full_url = f"{api_url}{preview_url}"

    if args.fetch:
        fetch_req = urllib.request.Request(full_url)
        with urllib.request.urlopen(fetch_req) as resp:
            print(resp.read().decode())
    else:
        print(full_url)
        if layout_warns:
            sys.exit(1)


if __name__ == "__main__":
    main()
