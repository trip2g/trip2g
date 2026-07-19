// ============================================================
// HERO HUB — inputs → core → outputs, bound to a voiceover.
// The animation is a pure function of audio.currentTime: every frame
// reconcile(t) sets the exact visual state for time t, so dragging the
// audio scrubber seeks the animation both ways. Beat durs are the
// measured VO clip lengths (speech + 0.5s air) so audio and visuals
// stay in lock; they also drive a timer fallback for record mode.
// ============================================================
(function () {
  const stage = document.getElementById("hero-hub-stage");
  if (!stage) return;

  const hero = document.getElementById("hero-hub");
  const playBtn = document.getElementById("hero-hub-play");
  const replayBtn = document.getElementById("hero-hub-replay");
  const captionEl = document.getElementById("hero-hub-caption");
  const audioEl = document.getElementById("hero-hub-audio");
  if (audioEl) audioEl.volume = 0.5; // start at half volume
  const ppBtn = document.getElementById("hero-hub-pp");
  const seekEl = document.getElementById("hero-hub-seek");
  const timeEl = document.getElementById("hero-hub-time");
  const BLOCK = "mesh-hero_hub"; // matches the @did expansion of hero_hub.html

  const params = new URLSearchParams(location.search);
  const RECORD = params.get("hero") === "play" || location.hash === "#play";
  // ?herospeed=1.5 plays 1.5x faster; ?herodelay=2 waits 2s before auto-play in record mode
  const SPEED = parseFloat(params.get("herospeed") || "1") || 1;
  const PRE_ROLL = (parseFloat(params.get("herodelay") || "1.2") || 1.2) * 1000;
  const REDUCED = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const LEAD = 150; // show-then-tell: the visual lands ~150ms before its word

  // ---- beat sheet: per-beat VO clip lengths (ms, incl. 0.5s tail air), by language ----
  const LANG = (audioEl && audioEl.dataset.lang) || "en";
  const DURS = {
    en: { core: 2952, markdown: 2377, obsidian: 3187, api: 2847, git: 1985, website: 2299, telegram: 1829, mcp: 2377, rag: 2220, webhooks: 2534, realtime: 3657, inout: 3265, own_computer: 1985, own_server: 2064, own_federation: 3422, cta: 4232 },
    ru: { core: 2952, markdown: 1985, obsidian: 3422, api: 2691, git: 1907, website: 2377, telegram: 1750, mcp: 2534, rag: 2534, webhooks: 2220, realtime: 4232, inout: 2847, own_computer: 1907, own_server: 1750, own_federation: 3500, cta: 4127 },
  };
  const D = DURS[LANG] || DURS.en;
  const BEATS = ["core", "markdown", "obsidian", "api", "git", "website", "telegram", "mcp", "rag", "webhooks", "realtime", "inout", "own_computer", "own_server", "own_federation", "cta"]
    .map((id) => ({ id, dur: D[id] }));
  const starts = [];
  const idx = {};
  let total = 0;
  BEATS.forEach((b, i) => { starts.push(total); idx[b.id] = i; total += b.dur; });
  const beatStart = (id) => starts[idx[id]];

  // node → doc page (cells are clickable links), per language
  const LINKS_BY_LANG = {
    en: { core: "/en/user/getting_started", obsidian: "/en/user/two_way_sync", api: "/en/user/update_notes", git: "/en/user/git", website: "/en/user/publishing", telegram: "/en/user/telegram", mcp: "/en/user/mcp", rag: "/en/user/search", webhooks: "/en/user/webhooks" },
    ru: { core: "/ru/user/nachalo_rabotyi", obsidian: "/ru/user/nachalo_rabotyi", api: "/ru/user/update_notes", git: "/ru/user/git", website: "/ru/user/hosting", telegram: "/ru/user/telegram", mcp: "/ru/user/mcp", rag: "/ru/user/search", webhooks: "/ru/user/cron_webhooks" },
  };
  const LINKS = LINKS_BY_LANG[LANG] || LINKS_BY_LANG.en;

  const W = 1600, H = 900;
  const NODES = {
    core:     { x: 800,  y: 450, w: 240, h: 104, kind: "hub",    glyph: "✦", label: "trip2g",   sub: "your hub" },
    obsidian: { x: 280,  y: 235, w: 210, h: 76,  kind: "input",  glyph: "◆", label: "obsidian", sub: "markdown editor" },
    api:      { x: 218,  y: 450, w: 210, h: 76,  kind: "input",  glyph: "◉", label: "api",      sub: "agents push notes" },
    git:      { x: 280,  y: 665, w: 210, h: 76,  kind: "input",  glyph: "◇", label: "git",      sub: "plain git push" },
    website:  { x: 1322, y: 155, w: 210, h: 76,  kind: "output", glyph: "▭", label: "website",  sub: "your live site" },
    telegram: { x: 1368, y: 302, w: 210, h: 76,  kind: "output", glyph: "▷", label: "telegram", sub: "channel posts" },
    mcp:      { x: 1385, y: 450, w: 210, h: 76,  kind: "output", glyph: "✳", label: "mcp",      sub: "agents connect" },
    rag:      { x: 1368, y: 598, w: 210, h: 76,  kind: "output", glyph: "◎", label: "rag",      sub: "semantic search" },
    webhooks: { x: 1322, y: 745, w: 210, h: 76,  kind: "output", glyph: "↺", label: "webhooks", sub: "automation" },
  };
  const INPUTS = ["obsidian", "api", "git"];
  const OUTPUTS = ["website", "telegram", "mcp", "rag", "webhooks"];
  const MD_TOKENS = ["# heading", "[[wikilink]]", "- [ ] task", "**bold**", "> quote", "![image]", "`code`", "---"];

  // finale: nested ownership frames around the core (centered on it)
  const FRAMES = [
    { id: "own_computer",   w: 348, h: 190, label: "on your computer" },
    { id: "own_server",     w: 476, h: 296, label: "on your own server" },
    { id: "own_federation", w: 612, h: 404, label: "inside your federation" },
  ];

  // per-language overrides for node/frame text (brand/tech labels stay as-is)
  const L10N = {
    ru: {
      kind: { input: "вход", output: "выход", hub: "хаб" },
      nodes: {
        core:     { sub: "ваш хаб" },
        obsidian: { sub: "редактор markdown" },
        api:      { sub: "агенты пушат заметки" },
        git:      { sub: "просто git push" },
        website:  { label: "сайт", sub: "ваш живой сайт" },
        telegram: { sub: "посты в канал" },
        mcp:      { sub: "агенты подключаются" },
        rag:      { sub: "семантический поиск" },
        webhooks: { sub: "автоматизация" },
      },
      frames: {
        own_computer: "на вашем компьютере",
        own_server: "или на вашем сервере",
        own_federation: "внутри вашей федерации",
      },
    },
  };
  const LX = L10N[LANG] || {};

  const SVG_NS = "http://www.w3.org/2000/svg";
  function svg(tag, attrs = {}) {
    const el = document.createElementNS(SVG_NS, tag);
    for (const [k, v] of Object.entries(attrs)) {
      if (v !== undefined && v !== null && v !== false) el.setAttribute(k, v);
    }
    return el;
  }
  function txt(x, y, str, cls, anchor) {
    const t = svg("text", { x, y, class: cls, "text-anchor": anchor || "middle" });
    t.textContent = str;
    return t;
  }
  function easeInOut(p) { return p < 0.5 ? 2 * p * p : 1 - Math.pow(-2 * p + 2, 2) / 2; }
  function easeOut(p) { return 1 - Math.pow(1 - p, 3); }
  const now = () => performance.now();

  // link path: right edge of `a` → left edge of `b`, horizontal-tangent bezier
  function linkD(a, b) {
    const x1 = a.x + a.w / 2, y1 = a.y;
    const x2 = b.x - b.w / 2, y2 = b.y;
    const mx = x1 + (x2 - x1) * 0.5;
    return `M ${x1} ${y1} C ${mx} ${y1}, ${mx} ${y2}, ${x2} ${y2}`;
  }

  const SVG = svg("svg", { viewBox: `0 0 ${W} ${H}`, preserveAspectRatio: "xMidYMid meet", "aria-hidden": "true" });
  const linksG = svg("g");
  const framesG = svg("g");
  const nodesG = svg("g");
  const fxG = svg("g");
  SVG.appendChild(linksG);
  SVG.appendChild(framesG); // behind nodes so grey frames read as a back layer
  SVG.appendChild(nodesG);
  SVG.appendChild(fxG);
  stage.appendChild(SVG);

  // faint field of graph silhouettes, flashed across the whole hero on the finale ("trip to graphs")
  const graphFlashG = svg("g", { class: "hh-graphflash", "aria-hidden": "true" });
  SVG.insertBefore(graphFlashG, linksG);
  const GF = { nodes: [], edges: [], cx: 800, cy: 450 }; // node cloud + edges (animated)
  (function buildGraphField() {
    const N = 120;
    for (let i = 0; i < N; i++) {
      GF.nodes.push({
        bx: 80 + Math.random() * 1440, by: 70 + Math.random() * 760, bz: (Math.random() * 2 - 1) * 170,
        ph: Math.random() * 6.28, fr: 0.5 + Math.random() * 0.7,
        x: 0, y: 0, r: 3, el: null,
      });
    }
    const addEdge = (a, b) => { const el = svg("line", { class: "hh-gf-edge" }); graphFlashG.appendChild(el); GF.edges.push({ a, b, el }); };
    for (let i = 0; i < N; i++) {
      const near = GF.nodes.map((p, j) => ({ j, d: Math.hypot(p.bx - GF.nodes[i].bx, p.by - GF.nodes[i].by) }))
        .filter((o) => o.j !== i).sort((a, b) => a.d - b.d).slice(0, 3);
      for (const { j } of near) addEdge(i, j);
    }
    for (let k = 0; k < 55; k++) { const i = (Math.random() * N) | 0, j = (Math.random() * N) | 0; if (i !== j) addEdge(i, j); } // long-range tangle
    for (const n of GF.nodes) { n.el = svg("circle", { r: 3, class: "hh-gf-node" }); graphFlashG.appendChild(n.el); }
  })();
  // points float around their origin + the whole cloud sways in pseudo-3D
  function animateGraph(t) {
    const { cx, cy } = GF, FOCAL = 1600, A = 8;
    const th = 0.4 * Math.sin(t * 0.00009), cos = Math.cos(th), sin = Math.sin(th);
    for (const n of GF.nodes) {
      const dx = n.bx - cx;
      const rx = dx * cos + n.bz * sin, rz = -dx * sin + n.bz * cos;
      const s = FOCAL / (FOCAL + rz);
      n.x = Math.max(6, Math.min(W - 6, cx + rx * s + A * Math.sin(t * 0.001 * n.fr + n.ph)));
      n.y = Math.max(6, Math.min(H - 6, cy + (n.by - cy) * s + A * Math.cos(t * 0.001 * n.fr + n.ph)));
      n.r = Math.max(0.6, 3 * s);
      n.el.setAttribute("cx", n.x.toFixed(1));
      n.el.setAttribute("cy", n.y.toFixed(1));
      n.el.setAttribute("r", n.r.toFixed(2));
    }
    for (const e of GF.edges) {
      const a = GF.nodes[e.a], b = GF.nodes[e.b];
      e.el.setAttribute("x1", a.x.toFixed(1)); e.el.setAttribute("y1", a.y.toFixed(1));
      e.el.setAttribute("x2", b.x.toFixed(1)); e.el.setAttribute("y2", b.y.toFixed(1));
    }
  }
  function graphFlash() {
    graphFlashG.classList.remove("is-flash");
    void SVG.getBoundingClientRect(); // restart the one-shot animation
    graphFlashG.classList.add("is-flash");
  }

  // markdown stream path (beat 2): off-screen left into the core
  const coreLeft = NODES.core.x - NODES.core.w / 2 - 6;
  const MD_D = `M -80 450 C 240 434, 460 466, ${coreLeft} 450`;
  const mdLine = svg("path", { d: MD_D, class: "hh-flow" });
  linksG.appendChild(mdLine);

  const PATHS = {};   // id -> { el, len } for particle travel
  const linkEls = {}; // id -> { base, flow }
  {
    const md = svg("path", { d: MD_D, fill: "none", stroke: "none" });
    linksG.appendChild(md);
    PATHS.md = { el: md, len: md.getTotalLength() };
  }
  for (const id of INPUTS.concat(OUTPUTS)) {
    const d = INPUTS.includes(id) ? linkD(NODES[id], NODES.core) : linkD(NODES.core, NODES[id]);
    const base = svg("path", { d, class: "hh-link" });
    const flow = svg("path", { d, class: "hh-flow" });
    linksG.appendChild(base);
    linksG.appendChild(flow);
    const len = base.getTotalLength();
    base.style.setProperty("--hh-len", len.toFixed(1));
    PATHS[id] = { el: base, len };
    linkEls[id] = { base, flow };
  }

  // ownership frames (hidden until their beat)
  const frameEls = {};
  for (const f of FRAMES) {
    const g = svg("g", { class: "hh-frame" });
    const x = NODES.core.x - f.w / 2, y = NODES.core.y - f.h / 2;
    g.appendChild(svg("rect", { x, y, width: f.w, height: f.h, rx: 4, class: "hh-frame-box" }));
    g.appendChild(txt(x + 12, y + 20, (LX.frames && LX.frames[f.id]) || f.label, "hh-frame-label", "start"));
    framesG.appendChild(g);
    frameEls[f.id] = g;
  }

  const nodeEls = {};
  for (const [id, n] of Object.entries(NODES)) {
    const g = svg("g", { class: `hh-node hh-node--${n.kind}` });
    const x = n.x - n.w / 2, y = n.y - n.h / 2;
    g.appendChild(svg("rect", { x: x + 3, y: y + 5, width: n.w, height: n.h, rx: 2, class: "hh-card-shadow" }));
    g.appendChild(svg("rect", { x, y, width: n.w, height: n.h, rx: 2, class: "hh-card" }));
    const lo = (LX.nodes && LX.nodes[id]) || {};
    const kindLabel = (LX.kind && LX.kind[n.kind]) || n.kind;
    g.appendChild(txt(n.x, y + 19, `${n.glyph}  ${kindLabel.toUpperCase()}`, "hh-kind"));
    g.appendChild(txt(n.x, n.y + (n.kind === "hub" ? 9 : 7), lo.label || n.label, "hh-label"));
    g.appendChild(txt(n.x, y + n.h - 12, lo.sub || n.sub, "hh-sub"));
    const href = LINKS[id]; // wrap in a doc link so cells are clickable
    if (href) {
      const a = svg("a", { class: "hh-node-link" });
      a.setAttributeNS("http://www.w3.org/1999/xlink", "xlink:href", href);
      a.setAttribute("href", href);
      a.appendChild(g);
      nodesG.appendChild(a);
    } else {
      nodesG.appendChild(g);
    }
    nodeEls[id] = g;
  }

  // ---- transient fx state ----
  const particles = []; // { path, t0, dur, kind: "dot"|"token", text, dy, md, pulse }
  const rings = [];     // { x, y, t0, dur }
  const emitters = {};  // id -> { spawn, every, next }
  function ring(x, y, dur) { rings.push({ x, y, t0: now(), dur: dur || 700 }); }
  function addEmitter(id, spawn, every) {
    if (emitters[id]) return;
    emitters[id] = { spawn, every, next: now() + every * Math.random() };
  }
  function removeEmitter(id) { delete emitters[id]; }

  function setCaption(text) {
    captionEl.classList.remove("is-on");
    captionEl.textContent = text;
    void captionEl.offsetWidth;
    captionEl.classList.add("is-on");
  }

  // input flow: dots travelling into the hub, no ripple ring
  function inputDot(id) {
    particles.push({ path: id, t0: now() + 400 / SPEED, dur: 1300 / SPEED, kind: "dot" });
  }
  function inputEmitter(id) {
    addEmitter(id, () => particles.push({ path: id, t0: now(), dur: 1600, kind: "dot" }), 2600);
  }
  // one-shot reveal fx (suppressed while scrubbing) — outputs only
  function outputFx(id) {
    particles.push({ path: id, t0: now() + 350 / SPEED, dur: 1300 / SPEED, kind: "dot" });
  }
  function outputEmitter(id) {
    addEmitter(id, () => particles.push({ path: id, t0: now(), dur: 1600, kind: "dot" }), 3400);
  }
  function mdEmitter() {
    addEmitter("md", () => particles.push({
      path: "md", t0: now(), dur: 2400 / SPEED, kind: "file",
      dy: (Math.random() * 2 - 1) * 55,
    }), 440);
  }
  // realtime beat: a fast in→core→out burst that reads as sub-second processing
  function realtimePulse() {
    for (const id of OUTPUTS) particles.push({ path: id, t0: now() + 240 / SPEED, dur: 520 / SPEED, kind: "dot" });
  }

  // toggle a node/frame's is-on idempotently; fire fx only on a live forward reveal
  function setNode(id, on, seek) {
    const g = nodeEls[id];
    const was = g.classList.contains("is-on");
    if (on && !was) {
      g.classList.add("is-on");
      linkEls[id] && (linkEls[id].base.classList.add("is-on"), linkEls[id].flow.classList.add("is-on"));
      // no reveal ripple: ripples fire only on the block being described (onset rings below)
      if (INPUTS.includes(id)) { if (!seek) inputDot(id); inputEmitter(id); } // dots flow in
      else if (OUTPUTS.includes(id)) { if (!seek) outputFx(id); outputEmitter(id); }
    } else if (!on && was) {
      g.classList.remove("is-on");
      if (linkEls[id]) { linkEls[id].base.classList.remove("is-on"); linkEls[id].flow.classList.remove("is-on"); }
      removeEmitter(id);
    }
  }
  function setFrame(id, on, seek) {
    const g = frameEls[id];
    const was = g.classList.contains("is-on");
    if (on && !was) g.classList.add("is-on");
    else if (!on && was) g.classList.remove("is-on");
  }

  // ---- the whole diagram as a pure function of time ----
  let lastT = -1, lastCaption = -1, currentNodeId = "core"; // the node the voice is describing
  function reconcile(tMs) {
    const seek = lastT >= 0 && Math.abs(tMs - lastT) > 400; // a jump = scrubbed
    if (seek) { particles.length = 0; rings.length = 0; }
    const tt = tMs + LEAD; // reveal slightly before the word
    setNode("core", tt >= beatStart("core"), seek);
    for (const id of INPUTS) setNode(id, tt >= beatStart(id), seek);
    for (const id of OUTPUTS) setNode(id, tt >= beatStart(id), seek);
    // ownership frames: reveal each, then keep only the outermost bright and dim the inner ones (zoom-out feel)
    let outer = -1;
    FRAMES.forEach((f, i) => { const on = tt >= beatStart(f.id); setFrame(f.id, on, seek); if (on) outer = i; });
    FRAMES.forEach((f, i) => {
      const g = frameEls[f.id];
      const on = g.classList.contains("is-on");
      g.classList.toggle("is-active", i === outer);
      g.classList.toggle("is-dim", on && i !== outer);
    });
    // markdown files stream in during the pour beat
    const mdOn = tMs >= beatStart("markdown") && tMs < beatStart("obsidian");
    if (mdOn && !mdLine.classList.contains("is-on")) { mdLine.classList.add("is-on"); if (!seek) mdEmitter(); }
    else if (!mdOn && mdLine.classList.contains("is-on")) { mdLine.classList.remove("is-on"); removeEmitter("md"); }
    // caption = current beat; fire the realtime burst on a live forward cross
    let ci = 0;
    while (ci + 1 < BEATS.length && starts[ci + 1] <= tMs) ci++;
    if (ci !== lastCaption) {
      const bid = BEATS[ci].id;
      currentNodeId = (INPUTS.includes(bid) || OUTPUTS.includes(bid)) ? bid : "core";
      if (!seek && bid === "realtime") realtimePulse();
      if (!seek && bid === "cta") graphFlash(); // finale: graph silhouettes bloom across the hero
      lastCaption = ci;
    }
    lastT = tMs;
  }

  // ---- playback ----
  // when the audio exists (and we're not recording), its currentTime is the ONE
  // clock: playing or scrubbing the audio drives the diagram. Timer only for record.
  const USE_AUDIO = !!audioEl && !RECORD && !REDUCED;
  let activated = false, playT0 = 0, endedAt = 0, resetting = false;

  // loudness envelope: the animation pulses with the voice (reactive even while scrubbing)
  let ENV = null, smoothE = 0, prevRaw = 0, lastOnset = 0, prevTickT = 0;
  if (audioEl && audioEl.dataset.env) {
    fetch(audioEl.dataset.env).then((r) => r.json()).then((j) => { ENV = j; }).catch(() => {});
  }
  function energyAt(tMs) {
    if (!ENV) return 0;
    const i = Math.floor((tMs / 1000) * ENV.hz);
    return (i >= 0 && i < ENV.peaks.length) ? ENV.peaks[i] : 0;
  }

  function snap(fn) {
    SVG.classList.add("hh-snap");
    SVG.classList.remove("hh-poster");
    fn();
    void SVG.getBoundingClientRect();
    SVG.classList.remove("hh-snap");
  }

  // full reset back to the initial poster state (diagram dimmed, graph gone, play button shown)
  function resetToPoster() {
    hero.classList.remove(`${BLOCK}--playing`, `${BLOCK}--done`);
    for (const el of Object.values(nodeEls)) el.classList.remove("is-on");
    for (const el of Object.values(frameEls)) el.classList.remove("is-on", "is-active", "is-dim");
    for (const l of Object.values(linkEls)) { l.base.classList.remove("is-on"); l.flow.classList.remove("is-on"); }
    mdLine.classList.remove("is-on");
    graphFlashG.classList.remove("is-flash");
    particles.length = 0; rings.length = 0;
    for (const k of Object.keys(emitters)) delete emitters[k];
    smoothE = 0; activated = false; endedAt = 0; lastT = -1; lastCaption = -1; currentNodeId = "core";
    SVG.classList.add("hh-poster");
    resetting = true; // ignore the play/seeking events our own currentTime=0 triggers
    if (audioEl) { try { audioEl.pause(); audioEl.currentTime = 0; } catch (e) {} }
  }

  async function play() {
    if (REDUCED) { showFinal(); return; }
    resetting = false; // user-initiated: allow play/seeking to activate again
    hero.classList.remove(`${BLOCK}--done`);
    hero.classList.add(`${BLOCK}--playing`);
    endedAt = 0;
    lastT = -1; lastCaption = -1;
    activated = true;
    SVG.classList.remove("hh-poster");
    if (USE_AUDIO) {
      try { audioEl.currentTime = 0; audioEl.playbackRate = SPEED; await audioEl.play(); } catch (e) { /* scrub still works */ }
    } else {
      playT0 = now();
    }
  }

  function showFinal() {
    snap(() => {
      for (const el of Object.values(nodeEls)) el.classList.add("is-on");
      for (const el of Object.values(frameEls)) el.classList.add("is-on");
      for (const id of INPUTS.concat(OUTPUTS)) { linkEls[id].base.classList.add("is-on"); linkEls[id].flow.classList.add("is-on"); }
    });
    hero.classList.add(`${BLOCK}--reduced`);
    activated = true;
  }

  // ---- render loop ----
  function tick() {
    const t = now();
    if (activated) {
      const tMs = USE_AUDIO ? audioEl.currentTime * 1000 : (t - playT0) * SPEED;
      const ct = Math.min(tMs, total);
      reconcile(ct);
      const ended = USE_AUDIO ? audioEl.ended : tMs >= total;
      hero.classList.toggle(`${BLOCK}--done`, ended);
      if (ended) {
        hero.classList.remove(`${BLOCK}--playing`);
        if (!endedAt) endedAt = t;
        else if (t - endedAt > 5000) resetToPoster(); // 5s after the end: full reset + play button
      } else if (endedAt) {
        endedAt = 0;
      }
      // audio-reactive energy: pulse the hub with the voice, ring on onsets
      const jump = Math.abs(ct - prevTickT) > 400;
      const energy = energyAt(ct);
      smoothE += (energy - smoothE) * 0.28;
      if (!jump && energy > 0.5 && energy > prevRaw + 0.06 && t - lastOnset > 150 && !INPUTS.includes(currentNodeId)) {
        const cn = NODES[currentNodeId];
        ring(cn.x, cn.y, 420 + energy * 220);
        lastOnset = t;
      }
      prevRaw = energy;
      prevTickT = ct;
    } else {
      smoothE += (0 - smoothE) * 0.2;
    }
    // louder voice → busier links
    for (const e of Object.values(emitters)) {
      if (t >= e.next) { e.spawn(); e.next = t + e.every * (0.7 + Math.random() * 0.6) / (0.7 + smoothE * 1.6); }
    }
    if (graphFlashG.classList.contains("is-flash")) animateGraph(t); // float + pseudo-3D sway

    while (fxG.firstChild) fxG.removeChild(fxG.firstChild);
    // halo breathes around the node the voice is describing (not always the hub)
    const cg = nodeEls[currentNodeId];
    if (smoothE > 0.03 && cg && cg.classList.contains("is-on")) {
      const pad = 7 + smoothE * 24, c = NODES[currentNodeId];
      fxG.appendChild(svg("rect", {
        x: c.x - c.w / 2 - pad, y: c.y - c.h / 2 - pad, width: c.w + pad * 2, height: c.h + pad * 2, rx: 6,
        fill: "none", stroke: "var(--accent)", "stroke-width": 1.2, opacity: (0.05 + smoothE * 0.24).toFixed(3),
      }));
    }
    for (let i = particles.length - 1; i >= 0; i--) {
      const p = particles[i];
      if (t < p.t0) continue;
      const prog = (t - p.t0) / p.dur;
      if (prog >= 1) {
        if (p.pulse) {
          const path = PATHS[p.path];
          const end = path.el.getPointAtLength(path.len);
          ring(end.x, end.y, 700);
        }
        particles.splice(i, 1);
        continue;
      }
      const path = PATHS[p.path];
      const pt = path.el.getPointAtLength(easeInOut(prog) * path.len);
      const fade = Math.max(0, Math.min(1, prog / 0.12, (1 - prog) / 0.15));
      if (p.kind === "dot") {
        fxG.appendChild(svg("circle", { cx: pt.x, cy: pt.y, r: 9, fill: "var(--accent)", opacity: (0.18 * fade).toFixed(3) }));
        fxG.appendChild(svg("circle", { cx: pt.x, cy: pt.y, r: 4.5, fill: "var(--accent)", opacity: fade.toFixed(3) }));
      } else if (p.kind === "file") {
        // a markdown document icon (dog-eared corner + MD) floating into the hub
        const fx = pt.x, fy = pt.y + (p.dy || 0);
        const w = 36, h = 46, f = 13, x = fx - w / 2, y = fy - h / 2;
        const g = svg("g", { opacity: (0.95 * fade).toFixed(3) });
        g.appendChild(svg("path", { d: `M ${x} ${y} L ${x + w - f} ${y} L ${x + w} ${y + f} L ${x + w} ${y + h} L ${x} ${y + h} Z`, fill: "var(--bg)", stroke: "var(--accent)", "stroke-width": 1.4 }));
        g.appendChild(svg("path", { d: `M ${x + w - f} ${y} L ${x + w} ${y + f} L ${x + w - f} ${y + f} Z`, fill: "none", stroke: "var(--accent)", "stroke-width": 1.4 }));
        g.appendChild(txt(fx, fy + 9, "MD", "hh-md-badge"));
        fxG.appendChild(g);
      } else {
        const tok = txt(pt.x, pt.y + (p.dy || 0), p.text, "hh-token");
        tok.setAttribute("opacity", (0.9 * fade).toFixed(3));
        fxG.appendChild(tok);
      }
    }
    for (let i = rings.length - 1; i >= 0; i--) {
      const r = rings[i];
      const prog = (t - r.t0) / r.dur;
      if (prog >= 1) { rings.splice(i, 1); continue; }
      fxG.appendChild(svg("circle", {
        cx: r.x, cy: r.y, r: 30 + 100 * easeOut(prog),
        fill: "none", stroke: "var(--accent)", "stroke-width": 1.2,
        opacity: ((1 - prog) * 0.5).toFixed(3),
      }));
    }
    requestAnimationFrame(tick);
  }

  // ---- init ----
  if (REDUCED) {
    showFinal();
  } else if (RECORD) {
    hero.classList.add(`${BLOCK}--record`);
    const bar = document.querySelector(".mesh-bar");
    if (bar) bar.style.display = "none";
    window.scrollTo(0, 0);
    setTimeout(play, PRE_ROLL);
  } else {
    SVG.classList.add("hh-poster");
  }
  requestAnimationFrame(tick);

  playBtn.addEventListener("click", play);
  replayBtn.addEventListener("click", play);
  // the visible audio scrubber IS the animation timeline: seeking scrubs the diagram
  if (USE_AUDIO) {
    audioEl.addEventListener("play", () => {
      if (resetting) return;
      activated = true;
      SVG.classList.remove("hh-poster");
      hero.classList.add(`${BLOCK}--playing`);
      hero.classList.remove(`${BLOCK}--done`);
    });
    audioEl.addEventListener("seeking", () => {
      if (resetting) return;
      activated = true;
      SVG.classList.remove("hh-poster");
      hero.classList.add(`${BLOCK}--playing`);
    });
  }
  // custom dark player (native audio controls can't be themed to match the site)
  if (audioEl && ppBtn && seekEl && timeEl) {
    let scrubbing = false;
    const fmt = (s) => { s = Math.max(0, s | 0); return `${(s / 60) | 0}:${String(s % 60).padStart(2, "0")}`; };
    ppBtn.addEventListener("click", () => { if (audioEl.paused) audioEl.play(); else audioEl.pause(); });
    audioEl.addEventListener("play", () => { ppBtn.textContent = "❚❚"; });
    audioEl.addEventListener("pause", () => { ppBtn.textContent = "▶"; });
    audioEl.addEventListener("ended", () => { ppBtn.textContent = "▶"; });
    audioEl.addEventListener("timeupdate", () => {
      if (!scrubbing && audioEl.duration) seekEl.value = ((audioEl.currentTime / audioEl.duration) * 1000) | 0;
      timeEl.textContent = `${fmt(audioEl.currentTime)} / ${fmt(audioEl.duration || 0)}`;
    });
    seekEl.addEventListener("input", () => { scrubbing = true; if (audioEl.duration) audioEl.currentTime = (seekEl.value / 1000) * audioEl.duration; });
    seekEl.addEventListener("change", () => { scrubbing = false; });
  }
  document.addEventListener("keydown", (e) => {
    if (e.key !== "r" || /input|textarea/i.test(e.target.tagName)) return;
    if (hero.classList.contains(`${BLOCK}--done`)) play();
  });
})();
