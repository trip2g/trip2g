// Reads a note's YAML frontmatter in the browser. json(note.M().Raw()) fails
// on a nested map, so the calls layouts ship ContentString() and parse here.
// Covers the YAML that notes use: scalars, quoted keys, nested maps, lists
// of scalars and of maps, flow lists, and | / > blocks.
window.callsFrontmatter = (function () {
  const KEY = /^(?:"((?:[^"\\]|\\.)*)"|'([^']*)'|([^\s"'#-][^:]*?|-[^\s:][^:]*?))\s*:(?:\s+(.*)|\s*)$/;

  function indentOf(line) {
    return line.length - line.trimStart().length;
  }

  function skip(lines, i) {
    while (i < lines.length && (!lines[i].trim() || lines[i].trim().startsWith("#"))) {
      i++;
    }
    return i;
  }

  function isItem(line) {
    return /^-(\s|$)/.test(line.trim());
  }

  function scalar(s) {
    const v = s.trim();
    if (v === "" || v === "~") {
      return v === "~" ? null : "";
    }
    if (/^["[{]/.test(v) || /^(true|false|null|-?\d+(\.\d+)?)$/.test(v)) {
      try {
        return JSON.parse(v);
      } catch (e) {
      }
    }
    if (v[0] === "[" && v[v.length - 1] === "]") {
      return v.slice(1, -1).split(",").map(scalar).filter(x => x !== "");
    }
    const quoted = /^'(.*)'$/.exec(v);
    if (quoted) {
      return quoted[1].replace(/''/g, "'");
    }
    return v.replace(/\s+#.*$/, "");
  }

  function text(lines, i, indent, folded) {
    const out = [];
    while (i < lines.length && (!lines[i].trim() || indentOf(lines[i]) > indent)) {
      out.push(lines[i].trim());
      i++;
    }
    while (out.length && !out[out.length - 1]) {
      out.pop();
    }
    return [out.join(folded ? " " : "\n"), i];
  }

  function block(lines, i, minIndent) {
    i = skip(lines, i);
    if (i >= lines.length || indentOf(lines[i]) < minIndent) {
      return [null, i];
    }
    const indent = indentOf(lines[i]);
    if (isItem(lines[i])) {
      const out = [];
      while ((i = skip(lines, i)) < lines.length && indentOf(lines[i]) === indent && isItem(lines[i])) {
        const rest = lines[i].trim().replace(/^-\s*/, "");
        if (rest === "") {
          const [v, j] = block(lines, i + 1, indent + 1);
          out.push(v);
          i = j;
        } else if (KEY.test(rest)) {
          lines[i] = " ".repeat(indent + 2) + rest;
          const [v, j] = block(lines, i, indent + 2);
          out.push(v);
          i = j;
        } else {
          out.push(scalar(rest));
          i++;
        }
      }
      return [out, i];
    }
    const out = {};
    while ((i = skip(lines, i)) < lines.length && indentOf(lines[i]) === indent && !isItem(lines[i])) {
      const m = KEY.exec(lines[i].trim());
      if (!m) {
        i++;
        continue;
      }
      const key = m[1] !== undefined ? m[1] : (m[2] !== undefined ? m[2] : m[3]);
      const val = (m[4] || "").trim();
      if (/^[|>][-+]?$/.test(val)) {
        const [v, j] = text(lines, i + 1, indent, val[0] === ">");
        out[key] = v;
        i = j;
      } else if (val !== "") {
        out[key] = scalar(val);
        i++;
      } else {
        const next = skip(lines, i + 1);
        const sameLevelList = next < lines.length && indentOf(lines[next]) === indent && isItem(lines[next]);
        const [v, j] = block(lines, i + 1, sameLevelList ? indent : indent + 1);
        out[key] = v === null ? "" : v;
        i = j;
      }
    }
    return [out, i];
  }

  return function parse(md) {
    const s = String(md || "").replace(/\r\n/g, "\n");
    const m = /^---\n([\s\S]*?)\n---(?:\n|$)/.exec(s);
    if (!m) {
      return { meta: {}, body: s };
    }
    const [meta] = block(m[1].split("\n"), 0, 0);
    return { meta: meta && !Array.isArray(meta) ? meta : {}, body: s.slice(m[0].length) };
  };
})();
