// fleetkit — helpers for the node blocks codellm executes.
//
// The twin of cmd/codellm/fleetkit/fleetkit.py, deliberately function-for-
// function and name-for-name identical: a role moved between the two languages
// should not have to relearn the API, so the names stay snake_case here rather
// than turning camelCase.
//
// A block's whole contract with the fleet is one line of JSON on stdout:
// {"changes": [...], "answer": "..."}. These build it, and the notes that go
// into it, so a role does not re-derive either.
//
//   const fleetkit = require('fleetkit');   // import works too
//   fleetkit.emit([fleetkit.note('a.md', {title: 'T'}, '# T\n')], 'wrote 1');

const fs = require('fs');
const YAML = require('yaml');

/**
 * Render a note: YAML frontmatter, then body. Key order is kept as given.
 *
 * version 1.1 matches PyYAML, which the python twin uses, and it is not
 * cosmetic: under the library's 1.2 default a timestamp-shaped string is
 * emitted bare and reads back as a timestamp, and 'yes' reads back as true.
 * The two twins have to write the same note for the same input.
 */
function render(meta, body = '') {
  if (!meta || Object.keys(meta).length === 0) return body;
  const front = YAML.stringify(meta, { version: '1.1' }).replace(/\n$/, '');
  return '---\n' + front + '\n---\n' + body;
}

/** A change that replaces the whole note at path. */
function write(path, content) {
  return { path, content };
}

/** A change that writes a rendered note — write(path, render(meta, body)). */
function note(path, meta, body = '') {
  return write(path, render(meta, body));
}

/** A change that swaps the first occurrence of find for replace. */
function patch(path, find, replace) {
  return { path, find, replace };
}

/**
 * Print the stdout contract. Call once, and last: codellm parses the stdout of
 * the final block, so printing anything else after it fails the run.
 */
function emit(changes = [], answer = '') {
  console.log(JSON.stringify({ changes: Array.from(changes), answer }));
}

/**
 * The delivery bag codellm exposes through FLEET_INPUT: changed_files,
 * change_file, attached_notes, depth. Empty object outside a delivery.
 */
function bag() {
  const path = process.env.FLEET_INPUT;
  if (!path) return {};
  return JSON.parse(fs.readFileSync(path, 'utf8'));
}

/** The frontmatter block of a markdown string, or {} when there is none. */
function parse_frontmatter(content) {
  if (!content.startsWith('---\n')) return {};
  const end = content.indexOf('\n---', 3);
  if (end < 0) return {};
  const loaded = YAML.parse(content.slice(4, end), { version: '1.1' });
  return loaded && typeof loaded === 'object' && !Array.isArray(loaded) ? loaded : {};
}

/**
 * Parse the frontmatter of a note the delivery bag carries. A block has no
 * vault filesystem, so this looks path up among attached_notes and
 * changed_files. A path the bag does not carry is a role misconfiguration, not
 * a finding, so it throws rather than reading as a note with no fields.
 */
function note_frontmatter(path) {
  const b = bag();
  for (const key of ['attached_notes', 'changed_files']) {
    for (const entry of b[key] || []) {
      if (entry.path === path) return parse_frontmatter(entry.content || '');
    }
  }
  throw new Error(
    `note '${path}' is not in the delivery bag; widen the role's attach_notes to cover it`,
  );
}

module.exports = { render, note, write, patch, emit, bag, note_frontmatter, parse_frontmatter };
