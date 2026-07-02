// ============================================================
// MESH GRAPH (vanilla SVG)
// ============================================================
const NODES = [
  { id: "you",     label: "you + agent",     kind: "self",     x:  60,  y: 360, sub: "MCP" },
  { id: "obs",     label: "obsidian",        kind: "glyph",    x:  60,  y: 540, sub: "sync", icon: "obsidian" },
  { id: "youhub",  label: "your trip2g",     kind: "hub",      x: 320,  y: 450, sub: "cms · proxy" },
  { id: "y_tg",    label: "telegram",        kind: "source",   x: 130,  y: 165, sub: "channel" },
  { id: "y_drive", label: "articles",        kind: "source",   x: 378,  y: 126, sub: "vector index" },
  { id: "y_book1", label: "'thinking fast'", kind: "book",     x: 510,  y: 272, sub: "principles" },
  { id: "y_book2", label: "project · q3",    kind: "book",     x: 463,  y: 628, sub: "ideas" },
  { id: "y_raw",   label: "raw MCP",         kind: "external", x: 258,  y: 723, sub: "non-trip2g" },
  { id: "f_book",  label: "'infinity'",      kind: "book",     x: 1092, y:  96, sub: "principles" },
  { id: "f_drive", label: "articles",        kind: "source",   x: 907,  y: 186, sub: "vector index" },
  { id: "f_proj",  label: "hr-2026",         kind: "book",     x: 819,  y: 557, sub: "resumes" },
  { id: "f_other", label: "raw MCP",         kind: "external", x: 915,  y: 670, sub: "non-trip2g" },
  { id: "f_tg",    label: "telegram",        kind: "source",   x: 1165, y: 672, sub: "channel" },
  { id: "frhub",   label: "alice.trip2g",    kind: "hub",      x: 1020, y: 450, sub: "cms · proxy" },
  { id: "friend",  label: "alice + agent",   kind: "peer",     x: 1188, y: 321, sub: "MCP" },
  { id: "friend2", label: "obsidian",        kind: "glyph",    x: 1227, y: 489, sub: "sync", icon: "obsidian" },
];
const EDGES = [
  ["obs", "youhub"], ["you", "youhub"],
  ["youhub", "y_tg"], ["youhub", "y_drive"], ["youhub", "y_book1"], ["youhub", "y_book2"], ["youhub", "y_raw"],
  ["youhub", "frhub"],
  ["frhub", "f_book"], ["frhub", "f_drive"], ["frhub", "f_proj"], ["frhub", "f_other"], ["frhub", "f_tg"],
  ["frhub", "friend"], ["frhub", "friend2"],
];
const LANES = [
  { route: ["you", "youhub", "frhub", "f_proj", "frhub", "youhub", "you"], offset: 0,   speed: 1.8 },
  { route: ["friend", "frhub", "youhub", "y_book1", "youhub", "frhub", "friend"], offset: 3.0, speed: 1.8 },
];
const KIND = {
  self:    { glyph: "◉", fill: "var(--fg)",     text: "var(--bg)", strong: true,  raised: true },
  peer:    { glyph: "◇", fill: "var(--bg)",     text: "var(--fg)", strong: false, raised: true },
  glyph:   { glyph: "",  fill: "transparent",   text: "var(--fg)", strong: false, raised: false, noChrome: true },
  hub:     { glyph: "✦", fill: "var(--accent)", text: "var(--fg)", strong: true,  raised: true,  accent: true },
  source:  { glyph: "▭", fill: "var(--bg)",     text: "var(--fg)", strong: false, raised: false },
  book:    { glyph: "▤", fill: "var(--bg)",     text: "var(--fg)", strong: false, raised: false },
  external:{ glyph: "◌", fill: "var(--bg)",     text: "var(--fg)", strong: false, raised: false, dashed: true },
};
function cardSize(k) {
  if (k === "hub") return { w: 180, h: 80 };
  if (k === "self" || k === "peer") return { w: 200, h: 64 };
  if (k === "glyph") return { w: 90, h: 90 };
  return { w: 200, h: 64 };
}
function nodeById(id) { return NODES.find(n => n.id === id); }
function edgePoint(a, b) {
  if (!a || !b) return { p1: { x: 0, y: 0 }, p2: { x: 0, y: 0 } };
  const sa = cardSize(a.kind), sb = cardSize(b.kind);
  const trim = (cx, cy, w, h, tx, ty) => {
    const dx = tx - cx, dy = ty - cy;
    if (dx === 0 && dy === 0) return { x: cx, y: cy };
    const sx = dx === 0 ? Infinity : (w/2)/Math.abs(dx);
    const sy = dy === 0 ? Infinity : (h/2)/Math.abs(dy);
    const s = Math.min(sx, sy);
    return { x: cx + dx*s, y: cy + dy*s };
  };
  return { p1: trim(a.x, a.y, sa.w, sa.h, b.x, b.y), p2: trim(b.x, b.y, sb.w, sb.h, a.x, a.y) };
}

const SVG_NS = "http://www.w3.org/2000/svg";
function svg(tag, attrs = {}, children = []) {
  const el = document.createElementNS(SVG_NS, tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (v !== undefined && v !== null && v !== false) el.setAttribute(k, v);
  }
  for (const c of children) el.appendChild(c);
  return el;
}
function obsidianIcon(cx, cy, size) {
  const s = size, g = svg("g", { transform: `translate(${cx - s/2}, ${cy - s/2})` });
  g.appendChild(svg("path", {
    d: `M ${s*0.5},0 L ${s},${s*0.4} L ${s*0.7},${s} L ${s*0.3},${s} L 0,${s*0.4} Z`,
    fill: "#7c3aed", stroke: "var(--fg)", "stroke-width": "0.8"
  }));
  g.appendChild(svg("path", {
    d: `M ${s*0.5},0 L ${s*0.3},${s*0.45} L 0,${s*0.4}`,
    fill: "none", stroke: "#a78bfa", "stroke-width": "0.6"
  }));
  g.appendChild(svg("path", {
    d: `M ${s*0.5},0 L ${s*0.7},${s*0.45} L ${s},${s*0.4}`,
    fill: "none", stroke: "#5b21b6", "stroke-width": "0.6"
  }));
  return g;
}
function textNode(x, y, str, attrs) {
  const t = svg("text", Object.assign({ x, y }, attrs));
  t.textContent = str;
  return t;
}

// build static parts once
const root = document.getElementById("mesh-svg-mount");
const SVG = svg("svg", {
  viewBox: "0 0 1340 820",
  preserveAspectRatio: "xMidYMid meet",
  "aria-hidden": "true"
});
// defs (dot grid)
const defs = svg("defs");
const pat = svg("pattern", { id: "mg-grid-dot", x: 0, y: 0, width: 22, height: 22, patternUnits: "userSpaceOnUse" });
pat.appendChild(svg("circle", { cx: 1, cy: 1, r: 0.7, fill: "var(--muted-2)", opacity: 0.35 }));
defs.appendChild(pat);
SVG.appendChild(defs);
SVG.appendChild(svg("rect", { x: 0, y: 0, width: 1340, height: 820, fill: "url(#mg-grid-dot)" }));

// territorial labels
const labelG = svg("g", {
  "font-family": "JetBrains Mono, monospace", "font-size": 10, "letter-spacing": "0.16em", fill: "var(--muted-2)"
});
labelG.appendChild(textNode(320, 32, "YOUR HUB", { "text-anchor": "middle" }));
labelG.appendChild(textNode(1020, 32, "FRIEND'S HUB", { "text-anchor": "middle" }));
SVG.appendChild(labelG);

// pipes group — we'll redraw flow each frame, but base lines are static
const pipesG = svg("g");
SVG.appendChild(pipesG);
const pipeFlowEls = {}; // edgeKey -> flow line element
const pipeBaseEls = {}; // edgeKey -> base line element

for (const [a, b] of EDGES) {
  const na = nodeById(a), nb = nodeById(b);
  const { p1, p2 } = edgePoint(na, nb);
  const base = svg("line", { x1: p1.x, y1: p1.y, x2: p2.x, y2: p2.y, class: "mg-pipe-base" });
  pipesG.appendChild(base);
  pipeBaseEls[`${a}-${b}`] = base;
  const flow = svg("line", { x1: p1.x, y1: p1.y, x2: p2.x, y2: p2.y, class: "mg-pipe-flow" });
  pipesG.appendChild(flow);
  pipeFlowEls[`${a}-${b}`] = flow;
}

// cards group
const cardsG = svg("g");
SVG.appendChild(cardsG);
const cardStrokeEls = {}; // id -> rect (for highlight)

for (const n of NODES) {
  const s = KIND[n.kind];
  const { w, h } = cardSize(n.kind);
  const x = n.x - w/2, y = n.y - h/2;
  const g = svg("g");

  if (s.noChrome) {
    // icon-only
    if (n.icon === "obsidian") g.appendChild(obsidianIcon(n.x, n.y - 6, 44));
    g.appendChild(textNode(n.x, n.y + 36, n.label, {
      "text-anchor": "middle",
      "font-family": "Inter, system-ui, sans-serif",
      "font-size": 13, "font-weight": 700, fill: "var(--fg)",
      "letter-spacing": "-0.005em"
    }));
    if (n.sub) g.appendChild(textNode(n.x, n.y + 50, n.sub, {
      "text-anchor": "middle",
      "font-family": "JetBrains Mono, monospace",
      "font-size": 8.5, fill: "var(--muted-2)", "letter-spacing": "0.06em"
    }));
  } else {
    if (s.raised) {
      g.appendChild(svg("rect", { x: x+3, y: y+5, width: w, height: h, fill: "rgba(0,0,0,0.18)", rx: 2 }));
    }
    const rect = svg("rect", {
      x, y, width: w, height: h,
      fill: s.fill, stroke: "var(--fg)",
      "stroke-width": s.strong ? 1.2 : 1,
      "stroke-dasharray": s.dashed ? "4 3" : "0",
      rx: 2
    });
    g.appendChild(rect);
    cardStrokeEls[n.id] = rect;

    g.appendChild(textNode(n.x, y + 18, `${s.glyph}  ${n.kind.toUpperCase()}`, {
      "text-anchor": "middle",
      "font-family": "JetBrains Mono, monospace",
      "font-size": 9,
      fill: s.accent ? "var(--fg)" : "var(--muted-2)",
      "letter-spacing": "0.12em"
    }));
    g.appendChild(textNode(n.x, y + h/2 + 6, n.label, {
      "text-anchor": "middle",
      "font-family": "Inter, system-ui, sans-serif",
      "font-size": n.kind === "hub" ? 14 : 13,
      "font-weight": 700,
      fill: s.text,
      "letter-spacing": "-0.005em"
    }));
    if (n.sub) g.appendChild(textNode(n.x, y + h - 10, n.sub, {
      "text-anchor": "middle",
      "font-family": "JetBrains Mono, monospace",
      "font-size": 8.5,
      fill: s.accent ? "color-mix(in oklab, var(--fg) 75%, transparent)" : "var(--muted-2)",
      "letter-spacing": "0.06em"
    }));
    if (n.id === "you") {
      g.appendChild(textNode(n.x, y - 8, "● ONLINE", {
        "text-anchor": "middle",
        "font-family": "JetBrains Mono, monospace",
        "font-size": 9, fill: "var(--accent)", "letter-spacing": "0.1em"
      }));
    }
  }
  n._el = g;
  n._w = w; n._h = h;
  n._bx = n.x; n._by = n.y; // original build position
  cardsG.appendChild(g);
}

// pulse + packet group (animated)
const fxG = svg("g");
SVG.appendChild(fxG);
const queryLabel = textNode(0, 0, "QUERY", {
  "text-anchor": "middle",
  "font-family": "JetBrains Mono, monospace",
  "font-size": 9, fill: "var(--accent)", "letter-spacing": "0.06em"
});
SVG.appendChild(queryLabel);

root.appendChild(SVG);

// ============================================================
// ANIMATION LOOP
// ============================================================
const start = performance.now();
function frame(now) {
  const t = (now - start) / 1000;
  // clear fx
  while (fxG.firstChild) fxG.removeChild(fxG.firstChild);

  // reset all flow lines & card strokes
  Object.values(pipeFlowEls).forEach(el => el.classList.remove("is-active"));
  Object.values(cardStrokeEls).forEach(el => {
    el.setAttribute("stroke", "var(--fg)");
    el.setAttribute("stroke-width", "1");
  });

  const lanes = LANES.map(lane => {
    const LEG = lane.speed;
    const total = (lane.route.length - 1) * LEG;
    const tt = (t + lane.offset) % (total + 0.6);
    const legIdx = Math.min(Math.floor(tt / LEG), lane.route.length - 2);
    const legProgress = Math.min(1, (tt - legIdx*LEG) / LEG);
    const fromId = lane.route[legIdx], toId = lane.route[legIdx + 1];
    const e = legProgress < 0.5 ? 2*legProgress*legProgress : 1 - Math.pow(-2*legProgress + 2, 2)/2;
    const { p1, p2 } = edgePoint(nodeById(fromId), nodeById(toId));
    const px = p1.x + (p2.x - p1.x)*e;
    const py = p1.y + (p2.y - p1.y)*e;
    return { px, py, fromId, toId, legProgress };
  });

  // packets: trail + shrink-into-node on arrival
  for (let i = 0; i < lanes.length; i++) {
    const l = lanes[i];
    const LEG = LANES[i].speed;
    const { p1, p2 } = edgePoint(nodeById(l.fromId), nodeById(l.toId));

    // trail: 3 fading dots behind the packet
    for (let s = 1; s <= 4; s++) {
      const trailPct = Math.max(0, l.legProgress - s * 0.06);
      const tE = trailPct < 0.5 ? 2*trailPct*trailPct : 1 - Math.pow(-2*trailPct+2,2)/2;
      const tx = p1.x + (p2.x - p1.x) * tE;
      const ty = p1.y + (p2.y - p1.y) * tE;
      fxG.appendChild(svg("circle", {
        cx: tx, cy: ty, r: 2.5 - s * 0.4,
        fill: "var(--accent)", opacity: (0.45 - s * 0.09).toFixed(2)
      }));
    }

    // packet: scale to 0 as it enters node, spring in from start
    const arrFade  = l.legProgress > 0.8 ? Math.max(0, 1 - (l.legProgress - 0.8) / 0.2) : 1;
    const startFade = l.legProgress < 0.08 ? l.legProgress / 0.08 : 1;
    const ps = Math.min(arrFade, startFade);
    if (ps > 0.02) {
      const pkt = svg("g", { transform: `translate(${l.px}, ${l.py})` });
      pkt.appendChild(svg("circle", { r: 9 * ps, fill: "var(--accent)", opacity: 0.18 }));
      pkt.appendChild(svg("circle", { r: 5 * ps, fill: "var(--accent)" }));
      fxG.appendChild(pkt);
    }
  }
  // query label follows lane 0 with opacity+scale pulse
  const pulse = 0.65 + 0.35 * Math.sin(t * 4.2);
  const pscale = 0.88 + 0.12 * Math.sin(t * 3.1);
  const qx = lanes[0].px, qy = lanes[0].py - 12;
  queryLabel.setAttribute("x", qx);
  queryLabel.setAttribute("y", qy);
  queryLabel.setAttribute("opacity", pulse.toFixed(3));
  queryLabel.setAttribute("transform", `translate(${qx},${qy}) scale(${pscale.toFixed(3)}) translate(${-qx},${-qy})`);

  requestAnimationFrame(frame);
}
requestAnimationFrame(frame);

// ============================================================
// TRACE LOG (typewriter reveal)
// ============================================================
const LOG_LINES = [
  "» your agent: 'how do we prioritize Q3?'",
  "→ your trip2g: search markdown notes",
  "↳ book/'thinking, fast & slow' · 3 chunks",
  "→ proxy: any peers with this knowledge?",
  "↳ alice.trip2g responds (open protocol)",
  "→ proxy: alice/project 'hr-2026'",
  "↳ similar(0.81) on prioritization framework",
  "« assemble: 2 hubs, 4 chunks, mixed strategies",
  "« reply — second brain, distributed",
];
const traceEl = document.getElementById("trace-stream");
traceEl.innerHTML = LOG_LINES.map((line, i) =>
  `<div class="log-line" data-i="${i}"><span class="log-num">${String(i+1).padStart(2, "0")}</span><span>${line}</span></div>`
).join("");
const lineEls = traceEl.querySelectorAll(".log-line");
function tickLog(now) {
  const t = (now - start) / 1000;
  const phase = (t * 0.13) % 1;
  lineEls.forEach((el, i) => {
    el.classList.toggle("is-on", (i / LOG_LINES.length) <= phase);
  });
  requestAnimationFrame(tickLog);
}
requestAnimationFrame(tickLog);

// ============================================================
// TRY-NOW · copy prompt
// ============================================================
const tryCopyBtn = document.getElementById('try-copy');
const tryPromptText = document.getElementById('try-prompt-text');
if (tryCopyBtn && tryPromptText) {
  tryCopyBtn.addEventListener('click', async () => {
    const text = tryPromptText.innerText;
    try {
      await navigator.clipboard.writeText(text);
    } catch (e) {
      const r = document.createRange();
      r.selectNodeContents(tryPromptText);
      const s = window.getSelection();
      s.removeAllRanges(); s.addRange(r);
      try { document.execCommand('copy'); } catch (_) {}
      s.removeAllRanges();
    }
    const original = tryCopyBtn.textContent;
    tryCopyBtn.textContent = 'copied ✓';
    setTimeout(() => { tryCopyBtn.textContent = original; }, 1800);
  });
}


(function() {
  const DOCS = [
    { path: "en/user/getting_started",  desc: "install + first sync · ~5 min",        tag: "getting started", size: "4.1K" },
    { path: "en/user/federation",        desc: "knowledge network · how bases connect", tag: "federation",      size: "12.8K" },
    { path: "en/user/mcp",               desc: "MCP server · connect your agent",       tag: "mcp",             size: "3.2K" },
    { path: "en/user/telegram",          desc: "publish to a channel · two-way echo",   tag: "telegram",        size: "5.7K" },
    { path: "en/user/monetization",      desc: "free: true / paid notes",               tag: "monetization",    size: "2.9K" },
    { path: "en/user/two_way_sync",      desc: "vault ⇄ server",                        tag: "sync",            size: "3.4K" },
    { path: "en/user/templates",         desc: "_layouts/, custom pages",               tag: "templates",       size: "8.0K" },
    { path: "en/user/webhooks",          desc: "events · cron",                         tag: "webhooks",        size: "4.6K" },
    { path: "en/user/selfhosted",        desc: "self-host on your own server",          tag: "self-host",       size: "2.1K" },
    { path: "en/user/use_cases",         desc: "solo · friends · company · b2b",        tag: "use cases",       size: "7.3K" },
  ];
  const docsList = document.getElementById("docs-list");
  if (docsList) {
    docsList.innerHTML = DOCS.map(d => `
      <a class="file" href="/${d.path}">
        <span class="file-path">${d.path}</span>
        <span class="file-tag">${d.tag}</span>
        <span class="file-desc">${d.desc}</span>
        <span class="file-size">${d.size}</span>
        <span class="file-arr">read →</span>
      </a>
    `).join("");
  }
})();

// ============================================================
// TURBO MODE — 50 req/s
// ============================================================
(function() {
  const mount = document.getElementById("mesh-svg-mount");
  if (!mount) return;

  const TURBO_ROUTES = [
    ["you","youhub","y_tg"],
    ["you","youhub","y_drive"],
    ["you","youhub","y_raw"],
    ["obs","youhub","y_book1"],
    ["obs","youhub","y_book2"],
    ["you","youhub","frhub","f_book"],
    ["you","youhub","frhub","f_drive"],
    ["you","youhub","frhub","f_other"],
    ["you","youhub","frhub","f_tg"],
    ["you","youhub","frhub","friend"],
    ["friend","frhub","friend2"],
    ["friend","frhub","youhub","obs"],
    ["friend","frhub","youhub","y_raw"],
    ["friend","frhub","f_proj","frhub","f_book"],
    ["you","youhub","frhub","f_drive","frhub","f_tg"],
    ["obs","youhub","frhub","friend2"],
    ["friend","frhub","youhub","y_tg"],
    ["friend","frhub","youhub","y_drive"],
    ["you","youhub","y_book1","youhub","y_book2"],
    ["obs","youhub","frhub","f_other","frhub","f_proj"],
  ];

  let turboOn = false;
  const turboPackets = [];
  let spawnTimer = null;

  function enableTurbo() {
    turboOn = true;
    spawnTimer = setInterval(spawnPacket, 20);
  }
  function disableTurbo() {
    turboOn = false;
    clearInterval(spawnTimer);
  }

  // Separate layer so we don't conflict with fxG clear/redraw
  const turboG = document.createElementNS("http://www.w3.org/2000/svg", "g");
  SVG.appendChild(turboG);

  function spawnPacket() {
    const route = TURBO_ROUTES[Math.floor(Math.random() * TURBO_ROUTES.length)];
    turboPackets.push({ route, t0: performance.now() / 1000, speed: 0.3 + Math.random() * 0.3 });
  }

  function renderTurbo(now) {
    const t = now / 1000;
    while (turboG.firstChild) turboG.removeChild(turboG.firstChild);

    for (let i = turboPackets.length - 1; i >= 0; i--) {
      const p = turboPackets[i];
      const LEG = p.speed;
      const total = (p.route.length - 1) * LEG;
      const tt = t - p.t0;
      if (tt > total + 0.2) { turboPackets.splice(i, 1); continue; }

      const legIdx = Math.min(Math.floor(tt / LEG), p.route.length - 2);
      const legPct = Math.min(1, (tt - legIdx * LEG) / LEG);
      const e = legPct < 0.5 ? 2*legPct*legPct : 1 - Math.pow(-2*legPct+2,2)/2;

      const fn = NODES.find(n => n.id === p.route[legIdx]);
      const tn = NODES.find(n => n.id === p.route[legIdx + 1]);
      if (!fn || !tn) continue;

      const { p1, p2 } = edgePoint(fn, tn);
      const px = p1.x + (p2.x - p1.x) * e;
      const py = p1.y + (p2.y - p1.y) * e;
      const fade = Math.min(1, (total - tt) / 0.2);
      const ts = 1 + 0.7 * (1 - Math.sin(legPct * Math.PI));

      const g = document.createElementNS("http://www.w3.org/2000/svg", "g");
      g.setAttribute("transform", `translate(${px},${py})`);
      const glow = document.createElementNS("http://www.w3.org/2000/svg", "circle");
      glow.setAttribute("r", String(6 * ts)); glow.setAttribute("fill","var(--accent)");
      glow.setAttribute("opacity", String(0.18 * fade));
      const dot = document.createElementNS("http://www.w3.org/2000/svg", "circle");
      dot.setAttribute("r", String(2.5 * ts)); dot.setAttribute("fill","var(--accent)");
      dot.setAttribute("opacity", String(0.75 * fade));
      g.appendChild(glow); g.appendChild(dot);
      turboG.appendChild(g);
    }
    requestAnimationFrame(renderTurbo);
  }
  requestAnimationFrame(renderTurbo);

  // Button: rectangle + square handle in bottom-right of graph frame
  const graphFrame = mount.closest(".mesh-network__graph") || mount.parentElement;
  graphFrame.style.position = "relative";

  const btn = document.createElement("button");
  btn.style.cssText = [
    "position:absolute","bottom:8px","right:8px",
    "display:flex","align-items:center","gap:5px",
    "background:none","border:none",
    "padding:4px 6px","cursor:pointer",
    "font-family:'JetBrains Mono',monospace","font-size:8px",
    "letter-spacing:0.14em","text-transform:uppercase","color:var(--muted-2)",
    "opacity:0.35","z-index:10","transition:opacity .2s,color .15s"
  ].join(";");
  btn.addEventListener("mouseenter", () => { if (!turboOn) btn.style.opacity = "0.75"; });
  btn.addEventListener("mouseleave", () => { if (!turboOn) btn.style.opacity = "0.35"; });

  const sq = document.createElement("span");
  sq.style.cssText = "display:inline-block;width:9px;height:9px;border:1px solid var(--muted-2);background:transparent;flex-shrink:0;transition:background .15s,border-color .15s";

  btn.appendChild(sq);
  btn.appendChild(document.createTextNode("turbo"));
  graphFrame.appendChild(btn);

  function setTurboStyle(on) {
    btn.style.opacity = on ? "1" : "0.35";
    btn.style.color = on ? "var(--accent)" : "var(--muted-2)";
    sq.style.background = on ? "var(--accent)" : "transparent";
  }

  btn.addEventListener("click", () => {
    turboOn = !turboOn;
    if (turboOn) enableTurbo(); else disableTurbo();
    setTurboStyle(turboOn);
  });

  // auto-start turbo, turn off after 3s
  enableTurbo();
  setTurboStyle(true);
  setTimeout(() => { disableTurbo(); setTurboStyle(false); }, 1000);
})();
