(function () {
  var root = document.getElementById("circle");
  if (!root) return;

  var B = "mesh-circle";
  var stage = root.querySelector("." + B + "__stage");
  var ring = root.querySelector("." + B + "__ring");
  var tabsEl = root.querySelector("." + B + "__tabs");
  var tabs = Array.prototype.slice.call(root.querySelectorAll("." + B + "__tab"));
  var buttons = tabs.map(function (li) { return li.querySelector("." + B + "__vertex"); });
  var panels = Array.prototype.slice.call(root.querySelectorAll("." + B + "__panel"));
  var countEl = root.querySelector("." + B + "__count");
  var N = panels.length;
  if (N < 2) return;

  var reduced = window.matchMedia("(prefers-reduced-motion: reduce)");
  var narrow = window.matchMedia("(max-width: 760px)");
  var SVGNS = "http://www.w3.org/2000/svg";
  var DURATION = 600;
  var STEP = (Math.PI * 2) / N;

  var current = 0;
  var rotation = 0;
  var anim = null;
  var geo = { cx: 0, cy: 0, r: 0 };
  var viz = null;

  function pad(n) { return n < 10 ? "0" + n : String(n); }

  buttons.forEach(function (b, i) { b.querySelector("." + B + "__num").textContent = pad(i + 1); });
  panels.forEach(function (p, i) { if (i !== 0) p.hidden = true; });

  var polygon = document.createElementNS(SVGNS, "polygon");
  polygon.setAttribute("fill", "none");
  polygon.setAttribute("stroke", "var(--rule)");
  polygon.setAttribute("stroke-width", "1.5");
  var circle = document.createElementNS(SVGNS, "circle");
  circle.setAttribute("fill", "none");
  circle.setAttribute("stroke", "var(--rule)");
  circle.setAttribute("stroke-dasharray", "2 6");
  ring.appendChild(circle);
  ring.appendChild(polygon);

  root.classList.add(B + "--live");

  function easeInOutCubic(t) {
    return t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2;
  }

  function measure() {
    var box = ring.getBoundingClientRect();
    var w = box.width;
    var h = box.height;
    if (narrow.matches) {
      var wheelH = Math.round(Math.min(w, Math.max(300, window.innerHeight * 0.48)));
      root.style.setProperty("--wheel-h", wheelH + "px");
      box = ring.getBoundingClientRect();
      w = box.width;
      h = box.height;
      geo.r = Math.max(90, Math.min(w / 2 - 44, h / 2 - 26));
    } else {
      geo.r = Math.max(160, Math.min(w / 2 - 150, h / 2 - 54));
    }
    geo.cx = w / 2;
    geo.cy = h / 2;
    ring.setAttribute("viewBox", "0 0 " + w + " " + h);
    circle.setAttribute("cx", geo.cx);
    circle.setAttribute("cy", geo.cy);
    circle.setAttribute("r", geo.r + (narrow.matches ? 14 : 22));

    var inner = geo.r * Math.cos(Math.PI / N);
    if (narrow.matches) {
      root.style.setProperty("--center-w", Math.round(inner * 1.25) + "px");
      root.style.setProperty("--viz-h", Math.round(inner * 1.05) + "px");
    } else {
      var cw = Math.min(500, inner * 1.42);
      root.style.setProperty("--center-w", Math.round(cw) + "px");
      root.style.setProperty("--viz-h", Math.round(Math.max(90, Math.min(170, inner * 0.46))) + "px");
    }
  }

  function angleOf(i) { return -Math.PI / 2 + i * STEP + rotation; }

  function layout() {
    var pts = [];
    for (var i = 0; i < N; i++) {
      var a = angleOf(i);
      var c = Math.cos(a);
      var s = Math.sin(a);
      var x = geo.cx + geo.r * c;
      var y = geo.cy + geo.r * s;
      pts.push(x.toFixed(1) + "," + y.toFixed(1));
      tabs[i].style.transform = "translate(" + x.toFixed(1) + "px," + y.toFixed(1) + "px)";
      var gap = narrow.matches ? 6 : 14;
      buttons[i].style.transform =
        "translate(" + (-50 + c * 50).toFixed(2) + "%," + (-50 + s * 50).toFixed(2) + "%) " +
        "translate(" + (c * gap).toFixed(1) + "px," + (s * gap).toFixed(1) + "px)";
    }
    polygon.setAttribute("points", pts.join(" "));
  }

  function rotateTo(target) {
    if (anim) cancelAnimationFrame(anim.raf);
    var from = rotation;
    var delta = target - from;
    delta = Math.atan2(Math.sin(delta), Math.cos(delta));
    if (reduced.matches || Math.abs(delta) < 1e-4) {
      rotation = from + delta;
      anim = null;
      layout();
      return;
    }
    var start = performance.now();
    anim = { raf: 0 };
    function tick(now) {
      var t = Math.min(1, (now - start) / DURATION);
      rotation = from + delta * easeInOutCubic(t);
      layout();
      if (t < 1) {
        anim.raf = requestAnimationFrame(tick);
      } else {
        anim = null;
      }
    }
    anim.raf = requestAnimationFrame(tick);
  }

  function startViz(panel) {
    if (viz) viz.stop();
    viz = null;
    var name = panel.dataset.viz;
    var canvas = panel.querySelector("." + B + "__viz");
    var registry = window.trip2gCircleViz || {};
    canvas.hidden = !name || !registry[name];
    if (canvas.hidden) return;
    viz = run(canvas, registry[name]);
  }

  function run(canvas, draw) {
    var ctx = canvas.getContext("2d");
    var raf = 0;
    var t0 = performance.now();
    var colors = palette();
    function size() {
      var dpr = Math.min(2, window.devicePixelRatio || 1);
      var w = canvas.clientWidth;
      var h = canvas.clientHeight;
      if (canvas.width !== Math.round(w * dpr) || canvas.height !== Math.round(h * dpr)) {
        canvas.width = Math.round(w * dpr);
        canvas.height = Math.round(h * dpr);
      }
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      return { w: w, h: h };
    }
    function frame(now) {
      var s = size();
      ctx.clearRect(0, 0, s.w, s.h);
      draw(ctx, reduced.matches ? 4.2 : (now - t0) / 1000, s.w, s.h, colors);
      if (!reduced.matches) raf = requestAnimationFrame(frame);
    }
    raf = requestAnimationFrame(frame);
    return { stop: function () { cancelAnimationFrame(raf); } };
  }

  function palette() {
    var cs = getComputedStyle(root);
    var probe = document.createElement("span");
    probe.style.color = cs.getPropertyValue("--accent");
    root.appendChild(probe);
    var accent = getComputedStyle(probe).color;
    root.removeChild(probe);
    return {
      accent: accent,
      fg: cs.getPropertyValue("--fg").trim(),
      muted: cs.getPropertyValue("--muted").trim(),
      dim: cs.getPropertyValue("--muted-2").trim(),
      rule: cs.getPropertyValue("--rule").trim(),
      bg: cs.getPropertyValue("--bg").trim()
    };
  }

  function select(i, opts) {
    i = ((i % N) + N) % N;
    var changed = i !== current || (opts && opts.initial);
    current = i;
    tabs.forEach(function (li, k) { li.classList.toggle(B + "__tab--on", k === i); });
    buttons.forEach(function (b, k) {
      b.setAttribute("aria-selected", k === i ? "true" : "false");
      b.tabIndex = k === i ? 0 : -1;
    });
    panels.forEach(function (p, k) {
      p.hidden = k !== i;
      p.classList.toggle("is-entering", k === i && !(opts && opts.initial));
    });
    countEl.textContent = pad(i + 1) + " / " + pad(N);
    if (changed) startViz(panels[i]);
    var target = -i * STEP;
    if (opts && opts.initial) {
      rotation = target;
      layout();
    } else {
      rotateTo(target);
    }
    if (opts && opts.focus) buttons[i].focus();
    if (!(opts && opts.initial) && history.replaceState) {
      history.replaceState(null, "", "#" + panels[i].dataset.slug);
    }
  }

  function step(d, focus) { select(current + d, { focus: focus }); }

  buttons.forEach(function (b, i) {
    b.addEventListener("click", function () { select(i); });
  });

  Array.prototype.forEach.call(root.querySelectorAll("." + B + "__nav-btn"), function (btn) {
    btn.addEventListener("click", function () { step(parseInt(btn.dataset.step, 10)); });
  });

  tabsEl.addEventListener("keydown", function (e) {
    if (e.key === "Home") { e.preventDefault(); select(0, { focus: true }); }
    if (e.key === "End") { e.preventDefault(); select(N - 1, { focus: true }); }
  });

  document.addEventListener("keydown", function (e) {
    if (e.altKey || e.ctrlKey || e.metaKey) return;
    var tag = (e.target && e.target.tagName) || "";
    if (tag === "INPUT" || tag === "TEXTAREA" || (e.target && e.target.isContentEditable)) return;
    var inTabs = tabsEl.contains(e.target);
    if (e.key === "ArrowRight" || (inTabs && e.key === "ArrowDown")) { e.preventDefault(); step(1, inTabs); }
    if (e.key === "ArrowLeft" || (inTabs && e.key === "ArrowUp")) { e.preventDefault(); step(-1, inTabs); }
  });

  var touch = null;
  stage.addEventListener("pointerdown", function (e) {
    if (e.pointerType === "mouse") return;
    touch = { x: e.clientX, y: e.clientY, t: performance.now() };
  });
  stage.addEventListener("pointerup", function (e) {
    if (!touch) return;
    var dx = e.clientX - touch.x;
    var dy = e.clientY - touch.y;
    var fast = performance.now() - touch.t < 800;
    touch = null;
    if (fast && Math.abs(dx) > 40 && Math.abs(dx) > Math.abs(dy) * 1.4) step(dx < 0 ? 1 : -1);
  });
  stage.addEventListener("pointercancel", function () { touch = null; });

  var resizeRaf = 0;
  window.addEventListener("resize", function () {
    cancelAnimationFrame(resizeRaf);
    resizeRaf = requestAnimationFrame(function () { measure(); layout(); });
  });

  function initialIndex() {
    var slug = decodeURIComponent((location.hash || "").slice(1));
    var byHash = panels.findIndex(function (p) { return p.dataset.slug === slug; });
    if (byHash >= 0) return byHash;
    var byNote = panels.findIndex(function (p) { return p.dataset.id === root.dataset.current; });
    return byNote >= 0 ? byNote : 0;
  }

  measure();
  select(initialIndex(), { initial: true });
  window.addEventListener("hashchange", function () { select(initialIndex()); });
})();
