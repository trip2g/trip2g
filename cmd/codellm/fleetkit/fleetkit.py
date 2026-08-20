"""fleetkit — helpers for the python blocks codellm executes.

A block's whole contract with the fleet is one line of JSON on stdout:
``{"changes": [...], "answer": "..."}``. These helpers build that line, and the
markdown notes that go into it, so a role note does not re-derive either and
does not glue YAML together by hand.

Dockerfile.codellm installs this module on python's default sys.path
(/usr/lib/python3/dist-packages), inside the subtree landlock grants read-only,
so every block can just ``import fleetkit``.

    import fleetkit

    fleetkit.emit(
        [fleetkit.note('calls/2026-01-15.md', {'title': 'Q1 planning'}, '# Q1\n')],
        'wrote 1 note',
    )
"""

import json
import os

import yaml

__all__ = ["render", "note", "write", "patch", "emit", "bag"]


def render(meta, body=""):
    """Render a note: YAML frontmatter, then body.

    yaml.safe_dump decides the quoting, so a title carrying a colon, a quote or
    a leading digit survives instead of producing frontmatter that parses as
    something else entirely. Key order is kept as given, not sorted.
    """
    if not meta:
        return body
    front = yaml.safe_dump(meta, sort_keys=False, allow_unicode=True).rstrip()
    return "---\n" + front + "\n---\n" + body


def write(path, content):
    """A change that replaces the whole note at path."""
    return {"path": path, "content": content}


def note(path, meta, body=""):
    """A change that writes a rendered note — write(path, render(meta, body))."""
    return write(path, render(meta, body))


def patch(path, find, replace):
    """A change that swaps the first occurrence of find for replace."""
    return {"path": path, "find": find, "replace": replace}


def emit(changes=(), answer=""):
    """Print the stdout contract.

    Call once, and last: codellm parses the stdout of the final block, so an
    earlier print of anything else takes its place and fails the run.
    """
    print(json.dumps({"changes": list(changes), "answer": answer}))


def bag():
    """The delivery bag codellm exposes through FLEET_INPUT.

    Carries changed_files, change_file, attached_notes and depth (see
    internal/fleetinput). Returns {} when the block runs outside a delivery,
    so `for c in fleetkit.bag().get('changed_files', [])` is safe either way.
    """
    path = os.environ.get("FLEET_INPUT")
    if not path:
        return {}
    with open(path, "r", encoding="utf-8") as fh:
        return json.load(fh)
