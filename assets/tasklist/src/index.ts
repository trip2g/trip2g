// tasklist widget — enables GFM task-list checkboxes for admins on the default
// template note page. Reads path + versionId from #tasklist-meta, un-disables
// the checkboxes found inside .content__body, and persists toggles via the
// GraphQL updateNotes patch mutation.
//
// Loaded only when: note has task-list items AND viewer is admin (gated by
// server-side emission in buildDefaultTemplateCtx / endpoint.go).

interface Meta {
  path: string;
  versionId: string;
}

interface NotePathResult {
  notePaths: Array<{ content: string }>;
}

interface UpdateNotesResult {
  updateNotes:
    | { __typename: 'UpdateNotesSuccessPayload'; paths: string[]; updated: Array<{ path: string; versionId: string }> }
    | { __typename: 'UpdateNotesHashMismatchPayload'; path: string; actualHash: string }
    | { __typename: 'UpdateNotesPatchNotFoundPayload'; path: string; find: string }
    | { __typename: 'ErrorPayload'; message: string };
}

const NOTE_CONTENT_QUERY = `
  query TaskListNoteContent($filter: NotePathsFilter) {
    notePaths(filter: $filter) {
      content
    }
  }
`;

const UPDATE_NOTES_MUTATION = `
  mutation TaskListUpdateNotes($input: UpdateNotesInput!) {
    updateNotes(input: $input) {
      __typename
      ... on UpdateNotesSuccessPayload {
        paths
        updated {
          path
          versionId
        }
      }
      ... on UpdateNotesHashMismatchPayload {
        path
        actualHash
      }
      ... on UpdateNotesPatchNotFoundPayload {
        path
        find
      }
      ... on ErrorPayload {
        message
      }
    }
  }
`;

async function graphql<T>(query: string, variables: unknown): Promise<T> {
  const res = await fetch('/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  });
  if (!res.ok) {
    throw new Error(`GraphQL HTTP ${res.status}`);
  }
  const json = await res.json();
  if (json.errors?.length) {
    throw new Error(json.errors[0].message);
  }
  return json.data as T;
}

// countTaskMarkers counts GFM task list markers in markdown source, skipping
// markers inside fenced code blocks. Mirrors the Go implementation in
// internal/model/note_tasklist.go so the DOM ↔ source count guard is
// consistent on both sides.
function countTaskMarkers(src: string): number {
  const lines = src.split('\n');
  let count = 0;
  let inFence = false;
  let fenceChar = '';
  let fenceLen = 0;

  for (const rawLine of lines) {
    const line = rawLine.replace(/\r$/, '');
    const stripped = line.replace(/^[ \t]+/, '');

    if (!inFence) {
      if (stripped.length >= 3) {
        const ch = stripped[0];
        if (ch === '`' || ch === '~') {
          let run = 0;
          while (run < stripped.length && stripped[run] === ch) run++;
          if (run >= 3) {
            inFence = true;
            fenceChar = ch;
            fenceLen = run;
            continue;
          }
        }
      }
    } else {
      if (stripped.length >= fenceLen) {
        const ch = stripped[0];
        if (ch === fenceChar) {
          let run = 0;
          while (run < stripped.length && stripped[run] === ch) run++;
          if (run >= fenceLen && stripped.slice(run).trim() === '') {
            inFence = false;
          }
        }
      }
      continue;
    }

    const rest = stripped;
    if (rest.length < 5) continue;
    const bullet = rest[0];
    if (bullet !== '-' && bullet !== '*' && bullet !== '+') continue;
    if (rest[1] !== ' ' && rest[1] !== '\t') continue;
    let i = 2;
    while (i < rest.length && rest[i] === ' ') i++;
    if (i + 3 > rest.length) continue;
    if (rest[i] === '[') {
      const inner = rest[i + 1];
      if (rest[i + 2] === ']' && (inner === ' ' || inner === 'x' || inner === 'X')) {
        count++;
      }
    }
  }

  return count;
}

// flipNthTaskMarker returns a new source string where the Nth (0-based) task
// marker is toggled ([ ] ↔ [x]). Returns null if the counts don't match the
// expected domCount (safety guard) or if n is out of range.
function flipNthTaskMarker(src: string, domCount: number, n: number): string | null {
  const lines = src.split('\n');
  let markerIndex = 0;
  let inFence = false;
  let fenceChar = '';
  let fenceLen = 0;

  for (let li = 0; li < lines.length; li++) {
    const rawLine = lines[li];
    const line = rawLine.replace(/\r$/, '');
    const stripped = line.replace(/^[ \t]+/, '');
    const leadingSpaces = rawLine.length - rawLine.replace(/^[ \t]+/, '').length; // not used directly

    if (!inFence) {
      if (stripped.length >= 3) {
        const ch = stripped[0];
        if (ch === '`' || ch === '~') {
          let run = 0;
          while (run < stripped.length && stripped[run] === ch) run++;
          if (run >= 3) {
            inFence = true;
            fenceChar = ch;
            fenceLen = run;
            continue;
          }
        }
      }
    } else {
      if (stripped.length >= fenceLen) {
        const ch = stripped[0];
        if (ch === fenceChar) {
          let run = 0;
          while (run < stripped.length && stripped[run] === ch) run++;
          if (run >= fenceLen && stripped.slice(run).trim() === '') {
            inFence = false;
          }
        }
      }
      continue;
    }

    const rest = stripped;
    if (rest.length < 5) continue;
    const bullet = rest[0];
    if (bullet !== '-' && bullet !== '*' && bullet !== '+') continue;
    if (rest[1] !== ' ' && rest[1] !== '\t') continue;
    let i = 2;
    while (i < rest.length && rest[i] === ' ') i++;
    if (i + 3 > rest.length) continue;
    if (rest[i] === '[') {
      const inner = rest[i + 1];
      if (rest[i + 2] === ']' && (inner === ' ' || inner === 'x' || inner === 'X')) {
        if (markerIndex === n) {
          // Found the marker to flip.
          // Find the position within the original rawLine.
          // stripped starts after the leading whitespace.
          const leadLen = rawLine.indexOf(stripped);
          const markerOffset = leadLen + i;
          const newInner = inner === ' ' ? 'x' : ' ';
          const newLine = rawLine.slice(0, markerOffset + 1) + newInner + rawLine.slice(markerOffset + 2);
          const newLines = [...lines];
          newLines[li] = newLine;
          return newLines.join('\n');
        }
        markerIndex++;
      }
    }
  }

  return null;
}

// buildPatch returns the updateNotes input for a line-level find/replace when
// the line is unique in source, or falls back to a full-content upsert.
function buildPatchInput(
  path: string,
  src: string,
  oldLine: string,
  newLine: string,
  expectedHash: string,
  newSrc: string,
): unknown {
  // Count occurrences of oldLine in source (as a whole line).
  const occurrences = src.split('\n').filter(l => l.replace(/\r$/, '') === oldLine.replace(/\r$/, '')).length;

  if (occurrences === 1) {
    // Unique line → use a targeted find/replace patch.
    // The server's patch is a literal substring find; use the whole line.
    return {
      changes: [{ patch: { path, find: oldLine, replace: newLine } }],
    };
  }

  // Ambiguous or multi-occurrence → fall back to full upsert with expectedHash guard.
  return {
    changes: [{ upsert: { path, content: newSrc, expectedHash } }],
  };
}

let currentVersionId: string = '';

async function toggleCheckbox(
  checkbox: HTMLInputElement,
  index: number,
  domCount: number,
  meta: Meta,
): Promise<void> {
  // Optimistic toggle.
  const wasChecked = checkbox.checked;
  checkbox.disabled = true;

  try {
    // Fetch raw source.
    const contentRes = await graphql<{ data: NotePathResult }>(NOTE_CONTENT_QUERY, {
      filter: { paths: [meta.path] },
    });
    const src = (contentRes as any).notePaths?.[0]?.content as string | undefined;
    if (!src) throw new Error('note content not found');

    // Safety guard: DOM count must match source count.
    const srcCount = countTaskMarkers(src);
    if (srcCount !== domCount) {
      throw new Error(`task count mismatch (DOM=${domCount}, src=${srcCount}) — page may be stale, please reload`);
    }

    // Find the line to flip.
    const markers = collectMarkers(src);
    if (index >= markers.length) throw new Error('index out of range');
    const oldLine = markers[index];
    const newSrc = flipNthTaskMarker(src, domCount, index);
    if (!newSrc) throw new Error('could not locate marker in source');

    // Derive new line from flipped source.
    const newMarkers = collectMarkers(newSrc);
    const newLine = newMarkers[index];

    const input = buildPatchInput(meta.path, src, oldLine, newLine, currentVersionId || meta.versionId, newSrc);

    const updateRes = await graphql<{ data: UpdateNotesResult }>(UPDATE_NOTES_MUTATION, { input });
    const payload = (updateRes as any).updateNotes as UpdateNotesResult['updateNotes'];

    if (payload.__typename === 'UpdateNotesSuccessPayload') {
      const updated = payload.updated.find(u => u.path === meta.path);
      if (updated) currentVersionId = updated.versionId;
      // Reflect new checked state.
      checkbox.checked = !wasChecked;
    } else if (payload.__typename === 'UpdateNotesHashMismatchPayload') {
      throw new Error('note was modified by another client — please reload');
    } else if (payload.__typename === 'UpdateNotesPatchNotFoundPayload') {
      throw new Error(`patch target not found in source: "${payload.find}"`);
    } else {
      throw new Error((payload as any).message || 'unknown error');
    }
  } catch (err) {
    // Revert optimistic toggle.
    checkbox.checked = wasChecked;
    const msg = err instanceof Error ? err.message : String(err);
    console.error('[tasklist]', msg);
    showError(checkbox, msg);
  } finally {
    checkbox.disabled = false;
  }
}

// collectMarkers returns the full lines (CR-stripped) of all task markers in
// source, in order, skipping fenced code blocks.
function collectMarkers(src: string): string[] {
  const lines = src.split('\n');
  const out: string[] = [];
  let inFence = false;
  let fenceChar = '';
  let fenceLen = 0;

  for (const rawLine of lines) {
    const line = rawLine.replace(/\r$/, '');
    const stripped = line.replace(/^[ \t]+/, '');

    if (!inFence) {
      if (stripped.length >= 3) {
        const ch = stripped[0];
        if (ch === '`' || ch === '~') {
          let run = 0;
          while (run < stripped.length && stripped[run] === ch) run++;
          if (run >= 3) { inFence = true; fenceChar = ch; fenceLen = run; continue; }
        }
      }
    } else {
      if (stripped.length >= fenceLen && stripped[0] === fenceChar) {
        let run = 0;
        while (run < stripped.length && stripped[run] === fenceChar) run++;
        if (run >= fenceLen && stripped.slice(run).trim() === '') { inFence = false; }
      }
      continue;
    }

    const rest = stripped;
    if (rest.length < 5) continue;
    const bullet = rest[0];
    if (bullet !== '-' && bullet !== '*' && bullet !== '+') continue;
    if (rest[1] !== ' ' && rest[1] !== '\t') continue;
    let i = 2;
    while (i < rest.length && rest[i] === ' ') i++;
    if (i + 3 > rest.length) continue;
    if (rest[i] === '[') {
      const inner = rest[i + 1];
      if (rest[i + 2] === ']' && (inner === ' ' || inner === 'x' || inner === 'X')) {
        out.push(line);
      }
    }
  }

  return out;
}

function showError(near: HTMLInputElement, msg: string): void {
  // Remove any previous inline error for this checkbox.
  const prev = near.parentElement?.querySelector('.tasklist-error');
  if (prev) prev.remove();

  const span = document.createElement('span');
  span.className = 'tasklist-error';
  span.style.cssText = 'color:#c0392b;font-size:0.85em;margin-left:0.4em;';
  span.textContent = '⚠ ' + msg;
  near.after(span);

  // Auto-remove after 6 seconds.
  setTimeout(() => span.remove(), 6000);
}

function initTaskList(): void {
  const metaEl = document.getElementById('tasklist-meta');
  if (!metaEl) return;

  let meta: Meta;
  try {
    meta = JSON.parse(metaEl.textContent || '{}') as Meta;
  } catch {
    return;
  }
  if (!meta.path || !meta.versionId) return;
  currentVersionId = meta.versionId;

  const body = document.querySelector<HTMLElement>('.content__body');
  if (!body) return;

  // Detect checkboxes: GFM task-list → <li><input type="checkbox" disabled ...>
  // goldmark default output: checkbox is first element child of the <li>.
  const checkboxes = Array.from(
    body.querySelectorAll<HTMLInputElement>('li > input[type="checkbox"]'),
  );

  if (checkboxes.length === 0) return;

  const domCount = checkboxes.length;

  checkboxes.forEach((cb, index) => {
    cb.removeAttribute('disabled');
    cb.style.cursor = 'pointer';

    cb.addEventListener('change', (e) => {
      e.preventDefault();
      // Revert the browser's optimistic toggle — we handle state ourselves
      // after the server confirms.
      cb.checked = !cb.checked;
      toggleCheckbox(cb, index, domCount, meta);
    });
  });
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initTaskList);
} else {
  initTaskList();
}
