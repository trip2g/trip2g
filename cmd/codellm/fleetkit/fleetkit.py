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

__all__ = ["render", "note", "write", "patch", "emit", "bag", "note_frontmatter"]


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


class Frontmatter(dict):
    """A frontmatter mapping whose missing keys read as None.

    So `if not data.title` says what it looks like instead of raising, and
    matches what the node twin gets from a plain object for free. Still a dict:
    data["title"], data.get("title") and **data all work.
    """

    def __getattr__(self, name):
        return self.get(name)


def note_frontmatter(path):
    """Parse the frontmatter of a note the delivery bag carries.

    A block has no vault filesystem — notes arrive in the bag, so this looks
    `path` up among attached_notes and changed_files. A path the bag does not
    carry is a role misconfiguration, not a finding, so it raises rather than
    reading as a note with no fields.
    """
    for key in ("attached_notes", "changed_files"):
        for entry in bag().get(key) or []:
            if entry.get("path") == path:
                return parse_frontmatter(entry.get("content") or "")
    raise KeyError(
        "note %r is not in the delivery bag; widen the role's attach_notes to cover it" % path
    )


def parse_frontmatter(content):
    """The frontmatter block of a markdown string, or empty when there is none."""
    if not content.startswith("---\n"):
        return Frontmatter()
    end = content.find("\n---", 3)
    if end < 0:
        return Frontmatter()
    loaded = yaml.safe_load(content[4:end])
    return Frontmatter(loaded if isinstance(loaded, dict) else {})


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
