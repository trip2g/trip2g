#!/usr/bin/env python3
"""
team-activity-process — Claude Code Stop hook: records activity snapshots to trip2g.

Fires on every Stop hook but throttles to one snapshot per 10 minutes per project.
Aggregates git commits + transcript tool calls since the last snapshot.

Called by hook (async, non-blocking):
  echo "$HOOK_JSON" | python3 ~/.claude/skills/team-activity-process/process.py

Credentials (in priority order):
  1. TRIP2G_API_KEY + TRIP2G_API_URL env vars
  2. <vault>/.obsidian/plugins/trip2g/data.json
"""

import json
import os
import re
import subprocess
import sys
from datetime import datetime, timezone, timedelta
from pathlib import Path


THROTTLE_MINUTES = int(os.environ.get("TEAM_ACTIVITY_THROTTLE_MINUTES", "10"))
STATE_FILE = ".state.json"


# ---------------------------------------------------------------------------
# Credentials
# ---------------------------------------------------------------------------

def find_credentials(start: Path) -> tuple[str, str]:
    api_key = os.environ.get("TRIP2G_API_KEY", "")
    api_url = os.environ.get("TRIP2G_API_URL", "")
    if api_key and api_url:
        return api_key, api_url
    for directory in [start, *start.parents]:
        candidate = directory / ".obsidian" / "plugins" / "trip2g" / "data.json"
        if candidate.exists():
            try:
                data = json.loads(candidate.read_text())
                dirs = data.get("syncDirs", [])
                if dirs:
                    return dirs[0].get("apiKey", ""), dirs[0].get("apiUrl", "http://localhost:8081")
            except Exception:
                pass
    return "", ""


# ---------------------------------------------------------------------------
# Projects config (_projects.md frontmatter)
# ---------------------------------------------------------------------------

def parse_projects_md(content: str) -> dict:
    m = re.match(r"^---\s*\n(.*?)\n---", content, re.DOTALL)
    if not m:
        return {}
    yaml = m.group(1)

    result = {}
    for key in ("user", "display_name"):
        km = re.search(rf"^{key}:\s*(.+)$", yaml, re.MULTILINE)
        if km:
            result[key] = km.group(1).strip().strip('"').strip("'")

    projects = []
    blocks = re.split(r"\n  - team_name:", yaml)
    for block in blocks[1:]:
        lines = block.split("\n")
        proj = {
            "team_name": lines[0].strip().strip('"').strip("'"),
            "cwd_patterns": [],
            "visibility": "private",
        }
        in_patterns = False
        for line in lines[1:]:
            if re.match(r"    cwd_patterns:", line):
                in_patterns = True
            elif in_patterns and re.match(r"      - ", line):
                pat = re.sub(r"^      - ", "", line).strip().strip('"').strip("'")
                proj["cwd_patterns"].append(os.path.expanduser(pat))
            elif re.match(r"    visibility:", line):
                in_patterns = False
                proj["visibility"] = re.sub(r"^    visibility:\s*", "", line).strip().strip('"').strip("'")
            elif not line.startswith("    ") and line.strip():
                in_patterns = False
        projects.append(proj)

    result["projects"] = projects
    return result


def match_project(cwd: str, projects: list) -> dict | None:
    cwd_abs = os.path.abspath(os.path.expanduser(cwd))
    wildcard = None
    for proj in projects:
        for pat in proj.get("cwd_patterns", []):
            if pat == "*":
                wildcard = proj
                continue
            pat_abs = os.path.abspath(os.path.expanduser(pat))
            if cwd_abs == pat_abs or cwd_abs.startswith(pat_abs + os.sep):
                return proj
    return wildcard


def find_vault_root(start: Path) -> Path | None:
    env_root = os.environ.get("TRIP2G_VAULT_ROOT", "")
    if env_root:
        p = Path(os.path.expanduser(env_root))
        if (p / "team_activity" / "_projects.md").exists():
            return p
    for directory in [start, *start.parents]:
        if (directory / "team_activity" / "_projects.md").exists():
            return directory
    return None


# ---------------------------------------------------------------------------
# State (throttle + git cursor)
# ---------------------------------------------------------------------------

def read_state(team_dir: Path) -> dict:
    state_path = team_dir / STATE_FILE
    if state_path.exists():
        try:
            return json.loads(state_path.read_text())
        except Exception:
            pass
    return {}


def write_state(team_dir: Path, state: dict) -> None:
    try:
        (team_dir / STATE_FILE).write_text(json.dumps(state, indent=2))
    except Exception:
        pass


def should_throttle(state: dict, now: datetime) -> bool:
    last = state.get("last_snapshot_at", "")
    if not last:
        return False
    try:
        last_dt = datetime.fromisoformat(last)
        return (now - last_dt) < timedelta(minutes=THROTTLE_MINUTES)
    except Exception:
        return False


# ---------------------------------------------------------------------------
# Git log
# ---------------------------------------------------------------------------

def get_new_commits(cwd: str, since_commit: str | None) -> list[str]:
    """Return list of 'hash subject' strings for commits since since_commit."""
    try:
        if since_commit:
            result = subprocess.run(
                ["git", "-C", cwd, "log", "--oneline", f"{since_commit}..HEAD",
                 "--format=%h %s"],
                capture_output=True, text=True, timeout=5
            )
        else:
            result = subprocess.run(
                ["git", "-C", cwd, "log", "--oneline", "-20", "--format=%h %s"],
                capture_output=True, text=True, timeout=5
            )
        if result.returncode != 0:
            return []
        lines = [l.strip() for l in result.stdout.splitlines() if l.strip()]
        return lines
    except Exception:
        return []


def get_head_commit(cwd: str) -> str | None:
    try:
        result = subprocess.run(
            ["git", "-C", cwd, "rev-parse", "HEAD"],
            capture_output=True, text=True, timeout=5
        )
        if result.returncode == 0:
            return result.stdout.strip()
    except Exception:
        pass
    return None


# ---------------------------------------------------------------------------
# Transcript parsing
# ---------------------------------------------------------------------------

def read_transcript_tail(path: str, max_bytes: int = 32_000) -> list:
    try:
        with open(path, "rb") as f:
            f.seek(0, 2)
            size = f.tell()
            f.seek(max(0, size - max_bytes))
            raw = f.read().decode("utf-8", errors="replace")
        events = []
        for line in raw.splitlines():
            line = line.strip()
            if not line:
                continue
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError:
                pass
        return events
    except (FileNotFoundError, IOError):
        return []


def extract_activity(events: list, since_ts: str | None = None) -> dict:
    """Extract tool calls and tasks, optionally filtering to events after since_ts."""
    since_dt = None
    if since_ts:
        try:
            since_dt = datetime.fromisoformat(since_ts)
        except Exception:
            pass

    tool_calls = []
    tasks = []

    for event in events:
        # Filter by timestamp if we have a cursor
        if since_dt:
            ts = event.get("timestamp", "")
            if ts:
                try:
                    event_dt = datetime.fromisoformat(ts.replace("Z", "+00:00"))
                    if event_dt <= since_dt:
                        continue
                except Exception:
                    pass

        msg = event.get("message", event)
        content = msg.get("content", [])
        if isinstance(content, str):
            continue
        for block in content:
            if not isinstance(block, dict):
                continue
            if block.get("type") != "tool_use":
                continue
            name = block.get("name", "")
            inp = block.get("input", {})

            if name == "Bash":
                cmd = inp.get("command", "")
                if cmd:
                    tool_calls.append(f"Bash: `{cmd[:120]}`")
            elif name in ("Edit", "Write"):
                fp = inp.get("file_path", "")
                if fp:
                    tool_calls.append(f"{name}: `{fp}`")
            elif name == "TaskCreate":
                subj = inp.get("subject", "")
                if subj:
                    tasks.append(subj)

    return {
        "tool_calls": [sanitize(tc) for tc in tool_calls[-12:]],
        "tasks": [sanitize(t) for t in tasks[-8:]],
    }


# ---------------------------------------------------------------------------
# Sanitization
# ---------------------------------------------------------------------------

_SECRET_PATTERNS = [
    (re.compile(r'(--(?:api[-_]?key|token|secret|password|passwd|auth|credential)[=\s]+)\S+', re.I), r'\1[REDACTED]'),
    (re.compile(r'\b([A-Z][A-Z0-9_]*(?:KEY|TOKEN|SECRET|PASSWORD|PASSWD|AUTH|CREDENTIAL|API)[A-Z0-9_]*)=[^\s\'"]+', re.I), r'\1=[REDACTED]'),
    (re.compile(r'(export\s+\w*(?:KEY|TOKEN|SECRET|PASSWORD|PASSWD|AUTH)\w*=)\S+', re.I), r'\1[REDACTED]'),
    (re.compile(r'((?:Bearer|Basic|Token)\s+)[A-Za-z0-9+/=_\-\.]{16,}', re.I), r'\1[REDACTED]'),
    (re.compile(r'eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+'), '[JWT-REDACTED]'),
    (re.compile(r'\b(AKIA|ASIA|AROA)[A-Z0-9]{16}\b'), '[AWS-KEY-REDACTED]'),
    (re.compile(r'\b(ghp_|gho_|ghu_|ghs_|ghr_|github_pat_)[A-Za-z0-9_]{20,}\b'), '[GH-TOKEN-REDACTED]'),
    (re.compile(r'\b[0-9a-f]{32,}\b', re.I), '[HEX-REDACTED]'),
    (re.compile(r'[A-Za-z0-9+/=_\-]{40,}'), '[KEY-REDACTED]'),
]


def sanitize(text: str) -> str:
    for pattern, replacement in _SECRET_PATTERNS:
        text = pattern.sub(replacement, text)
    return text


# ---------------------------------------------------------------------------
# Project wikilink resolution
# ---------------------------------------------------------------------------

def resolve_project_wikilink(vault_root: Path, team_name: str, cwd: str) -> str | None:
    projects_dir = vault_root / "team_activity" / team_name / "projects"
    if not projects_dir.is_dir():
        return None
    cwd_abs = os.path.abspath(os.path.expanduser(cwd))
    for md_file in projects_dir.glob("*.md"):
        try:
            content = md_file.read_text()
            m = re.match(r"^---\s*\n(.*?)\n---", content, re.DOTALL)
            if not m:
                continue
            cwd_match = re.search(r"^cwd:\s*(.+)$", m.group(1), re.MULTILINE)
            if not cwd_match:
                continue
            proj_cwd = os.path.abspath(os.path.expanduser(cwd_match.group(1).strip().strip('"').strip("'")))
            if cwd_abs == proj_cwd or cwd_abs.startswith(proj_cwd + os.sep):
                proj_name = md_file.stem
                return f"[[{team_name}/projects/{proj_name}]]"
        except Exception:
            pass
    return None


# ---------------------------------------------------------------------------
# Content builders
# ---------------------------------------------------------------------------

def build_snapshot(activity: dict, commits: list[str], cwd: str,
                   session_id: str, now: datetime,
                   project_wikilink: str | None = None) -> str:
    lines = [
        "---",
        f"session_id: {session_id}",
        f"cwd: {cwd}",
        f"timestamp: {now.isoformat()}",
        "---",
        "",
        f"## {now.strftime('%Y-%m-%d %H:%M')} UTC",
        "",
    ]
    if project_wikilink:
        lines.append(f"**Project:** {project_wikilink}")
        lines.append("")
    if commits:
        lines.append("**Commits:**")
        for c in commits:
            lines.append(f"- {sanitize(c)}")
        lines.append("")
    if activity["tasks"]:
        lines.append("**Tasks:**")
        for t in activity["tasks"]:
            lines.append(f"- {t}")
        lines.append("")
    if activity["tool_calls"]:
        lines.append("**Recent actions:**")
        for tc in activity["tool_calls"]:
            lines.append(f"- {tc}")
        lines.append("")
    return "\n".join(lines)


def update_report(existing: str | None, snap_rel: str, now: datetime) -> str:
    entry = f"- [{now.strftime('%Y-%m-%d %H:%M')}]({snap_rel}) UTC"
    if not existing:
        return f"# Activity Report\n\n{entry}\n"
    lines = existing.split("\n")
    out = []
    inserted = False
    for line in lines:
        out.append(line)
        if not inserted and line.startswith("# "):
            out.append("")
            out.append(entry)
            inserted = True
    if not inserted:
        out.insert(0, entry)
    return "\n".join(out)


# ---------------------------------------------------------------------------
# GraphQL
# ---------------------------------------------------------------------------

import urllib.request

PUSH_MUTATION = """
mutation PushNotes($input: PushNotesInput!) {
  pushNotes(input: $input) {
    ... on PushNotesPayload { updatedPaths }
    ... on ErrorPayload { message }
  }
}
"""


def push_notes(api_url: str, api_key: str, updates: list[dict]) -> None:
    body = json.dumps({
        "query": PUSH_MUTATION,
        "variables": {"input": {"skipCommit": False, "updates": updates}},
    }).encode()
    req = urllib.request.Request(
        f"{api_url.rstrip('/')}/graphql",
        data=body,
        headers={"Content-Type": "application/json", "X-API-Key": api_key},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            resp.read()
    except Exception:
        pass


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    try:
        payload = json.loads(sys.stdin.read())
    except (json.JSONDecodeError, EOFError):
        return

    transcript_path = payload.get("transcript_path", "")
    cwd = payload.get("cwd", os.getcwd())
    session_id = payload.get("session_id", "unknown")

    vault_root = find_vault_root(Path(cwd))
    if not vault_root:
        return

    try:
        cfg_text = (vault_root / "team_activity" / "_projects.md").read_text()
    except IOError:
        return

    config = parse_projects_md(cfg_text)
    project = match_project(cwd, config.get("projects", []))
    if not project:
        return

    team_name = project["team_name"]
    visibility = project.get("visibility", "private")

    team_dir = vault_root / "team_activity" / team_name
    team_dir.mkdir(parents=True, exist_ok=True)

    state = read_state(team_dir)
    now = datetime.now(timezone.utc)

    # Throttle: skip if last snapshot was less than THROTTLE_MINUTES ago
    if should_throttle(state, now):
        return

    # Git: collect commits since last snapshot
    last_commit = state.get("last_git_commit")
    commits = get_new_commits(cwd, last_commit)
    head_commit = get_head_commit(cwd)

    # Transcript: collect tool calls since last snapshot
    last_snapshot_at = state.get("last_snapshot_at")
    events = read_transcript_tail(transcript_path)
    activity = extract_activity(events, since_ts=last_snapshot_at)

    # Skip if nothing happened
    if not commits and not activity["tool_calls"] and not activity["tasks"]:
        return

    snap_filename = f"{now.strftime('%Y-%m-%dT%H-%M')}.md"
    snap_rel = f"{team_name}/{snap_filename}"

    project_wikilink = resolve_project_wikilink(vault_root, team_name, cwd)
    snap_content = build_snapshot(activity, commits, cwd, session_id, now, project_wikilink)

    report_path = team_dir / "report.md"
    existing_report = report_path.read_text() if report_path.exists() else None
    report_content = update_report(existing_report, snap_filename, now)

    (team_dir / snap_filename).write_text(snap_content)
    report_path.write_text(report_content)

    # Update state
    new_state = {
        "last_snapshot_at": now.isoformat(),
        "last_git_commit": head_commit or last_commit,
    }
    write_state(team_dir, new_state)

    # Push to trip2g if not private
    if visibility != "private":
        api_key, api_url = find_credentials(vault_root)
        if api_key and api_url:
            push_notes(api_url, api_key, [
                {"path": f"team_activity/{snap_rel}", "content": snap_content},
                {"path": f"team_activity/{team_name}/report.md", "content": report_content},
            ])


if __name__ == "__main__":
    main()
