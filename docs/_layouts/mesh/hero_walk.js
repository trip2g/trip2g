// HERO WALK — replays recorded MCP walks (walk_<lang>.json) as a terminal log.
// Labels and the scenarios URL come from data attributes on #hero-walk.
(function () {
  const pl = document.getElementById("hero-walk");
  if (!pl) return;
  const B = "mesh-hero_walk"; // matches the @did expansion of hero_walk.html
  const $ = (id) => document.getElementById("hero-walk-" + id);
  const log = $("log"), fin = $("final"), taskEl = $("task"), meta = $("meta"), live = $("live"), dots = $("dots");
  const L = pl.dataset;
  const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
  const MAX_LINES = 14;
  let scenarios = [], run = 0, paused = false, visible = true;

  const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
  const span = (cls, html) => '<span class="' + B + "__" + cls + '">' + html + "</span>";

  function fmtCall(e) {
    const kb = e.kb ? span("kb", esc(e.kb)) + " " : "";
    return span("name", esc(e.name)) + " " + kb + span("arg", esc(e.arg));
  }
  function fmtRet(e) {
    if (e.err) return esc(e.err);
    const ms = " " + span("ms", e.ms + " ms");
    if (e.read !== undefined) return "read: " + esc(e.read) + "…" + ms;
    const items = (e.items || []).map((it) => {
      const p = it.split(" · ");
      return esc(p[0]) + (p[1] ? " " + span("dom", esc(p[1])) : "");
    }).join(" · ");
    return span("n", e.n + " notes") + ms + (items ? " · " + items : "");
  }
  function line(e) {
    const d = document.createElement("div");
    let g = "·", html = "";
    if (e.t === "think") { g = "…"; html = esc(e.text); }
    if (e.t === "call") { g = "→"; html = fmtCall(e); }
    if (e.t === "ret") { g = "←"; html = fmtRet(e); }
    d.className = B + "__ln " + B + "__ln--" + e.t + (e.err ? " " + B + "__ln--err" : "");
    d.innerHTML = span("g", g) + span("body", html);
    log.appendChild(d);
    while (log.children.length > MAX_LINES) log.removeChild(log.firstChild);
    requestAnimationFrame(() => requestAnimationFrame(() => d.classList.add(B + "__ln--on")));
  }
  const sleep = (ms) => new Promise((r) => {
    const t0 = performance.now();
    (function tick() {
      if (paused || !visible) return setTimeout(tick, 120);
      if (performance.now() - t0 >= ms) r(); else setTimeout(tick, 40);
    })();
  });
  function delay(e) {
    if (e.t === "think") return 1000;
    if (e.t === "call") return 350;
    if (e.t === "ret") return Math.min(e.ms || 600, 1400) * 0.8 + 250;
    return 0;
  }
  async function typeFinal(text, myRun) {
    fin.innerHTML = span("final-lbl", esc(L.final)) + span("final-txt " + B + "__final-txt--caret", "");
    const t = fin.lastChild;
    if (reduced) { t.textContent = text; t.classList.remove(B + "__final-txt--caret"); return; }
    let i = 0;
    while (i < text.length && myRun === run) { t.textContent = text.slice(0, i += 2); await sleep(14); }
    t.textContent = text; t.classList.remove(B + "__final-txt--caret");
  }
  async function play(i) {
    const myRun = ++run; const s = scenarios[i];
    [...dots.children].forEach((d, k) => d.classList.toggle(B + "__dot--on", k === i));
    taskEl.textContent = s.task;
    meta.textContent = [L.recorded, s.model, s.calls, s.cost, s.shape].join(" · ");
    log.innerHTML = ""; fin.innerHTML = "";
    for (const e of s.events) {
      if (myRun !== run) return;
      if (e.t === "final") { await typeFinal(e.text, myRun); break; }
      line(e);
      if (!reduced) await sleep(delay(e));
    }
    if (reduced) return;
    await sleep(7000);
    if (myRun === run) play((i + 1) % scenarios.length);
  }

  pl.addEventListener("mouseenter", () => { paused = true; live.classList.add(B + "__live--paused"); live.textContent = L.paused; });
  pl.addEventListener("mouseleave", () => { paused = false; live.classList.remove(B + "__live--paused"); live.textContent = L.live; });
  if ("IntersectionObserver" in window) new IntersectionObserver((en) => { visible = en[0].isIntersecting; }, { threshold: 0.2 }).observe(pl);

  fetch(L.src).then((r) => r.json()).then((list) => {
    scenarios = list;
    scenarios.forEach((s, i) => {
      const b = document.createElement("button");
      b.className = B + "__dot"; b.type = "button";
      b.setAttribute("role", "tab"); b.setAttribute("aria-label", "scenario " + (i + 1));
      b.onclick = () => play(i);
      dots.appendChild(b);
    });
    play(0);
  }).catch(() => {});
})();
