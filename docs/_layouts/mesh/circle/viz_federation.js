(function () {
  var ACT = 3.6;
  var FADE = 0.35;
  var PEOPLE = 6;
  var BASES = 10;
  var ROUTERS = 5;
  var CAPTIONS = {
    en: ["everyone in one pile", "everyone to ten places", "through a router"],
    ru: ["все в одну", "каждый в десять мест", "через маршрутизатор"]
  };

  function clamp(v) { return v < 0 ? 0 : v > 1 ? 1 : v; }
  function ease(v) { v = clamp(v); return v < 0.5 ? 4 * v * v * v : 1 - Math.pow(-2 * v + 2, 3) / 2; }

  function dot(ctx, x, y, r, fill) {
    ctx.fillStyle = fill;
    ctx.beginPath();
    ctx.arc(x, y, r, 0, Math.PI * 2);
    ctx.fill();
  }

  function person(ctx, x, y, s, color, bg) {
    ctx.fillStyle = bg;
    ctx.strokeStyle = color;
    ctx.lineWidth = 1.3;
    ctx.beginPath();
    ctx.arc(x, y - s * 0.55, s * 0.45, 0, Math.PI * 2);
    ctx.fill();
    ctx.stroke();
    ctx.beginPath();
    ctx.arc(x, y + s * 0.75, s * 0.8, Math.PI, 0);
    ctx.stroke();
  }

  function along(ax, ay, bx, by, p) {
    return { x: ax + (bx - ax) * p, y: ay + (by - ay) * p };
  }

  function pile(ctx, t, w, h, c) {
    var cx = w / 2;
    var cy = h * 0.56;
    var R = Math.min(h * 0.17, 26);
    var s = Math.max(5, Math.min(8, h * 0.05));
    for (var i = 0; i < PEOPLE * 2; i++) {
      var a = (i / (PEOPLE * 2)) * Math.PI * 2 + 0.3;
      var crowd = 0.55 + 0.08 * Math.sin(t * 2.3 + i * 1.7);
      var px = cx + Math.cos(a) * w * 0.36 * crowd;
      var py = cy + Math.sin(a) * h * 0.36 * crowd;
      var wob = Math.sin(t * 3 + i) * 14;
      ctx.strokeStyle = c.muted;
      ctx.globalAlpha = 0.55;
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(px, py);
      ctx.quadraticCurveTo((px + cx) / 2 + wob, (py + cy) / 2 - wob, cx, cy);
      ctx.stroke();
      ctx.globalAlpha = 1;
      var q = ((t * 0.35 + i * 0.13) % 1) * 0.85;
      var pt = along(px, py, cx, cy, q);
      dot(ctx, pt.x, pt.y, 1.8, c.fg);
      person(ctx, px, py, s, c.muted, c.bg);
    }
    ctx.fillStyle = c.bg;
    ctx.strokeStyle = c.fg;
    ctx.lineWidth = 1.4;
    var jitter = 1 + 0.06 * Math.sin(t * 9);
    ctx.beginPath();
    ctx.arc(cx, cy, R * jitter, 0, Math.PI * 2);
    ctx.fill();
    ctx.stroke();
    for (var k = 0; k < 14; k++) {
      dot(ctx, cx + Math.cos(k * 2.4 + t * 2) * R * 0.6 * ((k % 4) / 4 + 0.2), cy + Math.sin(k * 2.4 + t * 2) * R * 0.6 * ((k % 4) / 4 + 0.2), 1.4, c.muted);
    }
  }

  function web(ctx, t, w, h, c) {
    var s = Math.max(5, Math.min(8, h * 0.05));
    var top = h * 0.3;
    var bottom = h * 0.84;
    var grow = clamp(t / (ACT * 0.33));
    var bases = [];
    var people = [];
    for (var b = 0; b < BASES; b++) bases.push({ x: w * (0.08 + 0.84 * b / (BASES - 1)), y: top });
    for (var p = 0; p < PEOPLE; p++) people.push({ x: w * (0.15 + 0.7 * p / (PEOPLE - 1)), y: bottom });
    ctx.lineWidth = 1;
    for (var i = 0; i < PEOPLE; i++) {
      for (var j = 0; j < BASES; j++) {
        var order = (i * BASES + j) / (PEOPLE * BASES);
        var g = clamp((grow - order * 0.7) / 0.3);
        if (g <= 0) continue;
        var e = along(people[i].x, people[i].y - s, bases[j].x, bases[j].y, g);
        ctx.strokeStyle = c.muted;
        ctx.globalAlpha = 0.32;
        ctx.beginPath();
        ctx.moveTo(people[i].x, people[i].y - s);
        ctx.lineTo(e.x, e.y);
        ctx.stroke();
      }
    }
    ctx.globalAlpha = 1;
    for (var k = 0; k < BASES; k++) {
      ctx.fillStyle = c.bg;
      ctx.strokeStyle = c.fg;
      ctx.lineWidth = 1.2;
      ctx.beginPath();
      ctx.arc(bases[k].x, bases[k].y, Math.max(5, h * 0.045), 0, Math.PI * 2);
      ctx.fill();
      ctx.stroke();
    }
    for (var m = 0; m < PEOPLE; m++) person(ctx, people[m].x, people[m].y, s, c.muted, c.bg);
  }

  function routed(ctx, t, w, h, c) {
    var cx = w / 2;
    var cy = h * 0.62;
    var rx = Math.min(w * 0.36, h * 1.2);
    var ry = h * 0.22;
    var br = Math.max(8, Math.min(h * 0.09, 15));
    var s = Math.max(5, Math.min(8, h * 0.05));
    var nodes = [];
    for (var i = 0; i < ROUTERS; i++) {
      var a = -Math.PI / 2 + (i / ROUTERS) * Math.PI * 2;
      var x = cx + Math.cos(a) * rx;
      var y = cy + Math.sin(a) * ry;
      nodes.push({ x: x, y: y, px: x + Math.cos(a) * (br + s * 2.2), py: y + Math.sin(a) * (br + s * 2.2) });
    }
    var PULSE = ACT / 3;
    var n = Math.floor(t / PULSE) % ROUTERS;
    var from = nodes[n];
    var to = nodes[(n + 2) % ROUTERS];
    var q = (t % PULSE) / PULSE;

    ctx.lineWidth = 1;
    ctx.setLineDash([2, 4]);
    ctx.strokeStyle = c.accent;
    ctx.globalAlpha = 0.35;
    for (var k = 0; k < ROUTERS; k++) {
      var nb = nodes[(k + 1) % ROUTERS];
      ctx.beginPath();
      ctx.moveTo(nodes[k].x, nodes[k].y);
      ctx.lineTo(nb.x, nb.y);
      ctx.stroke();
    }
    ctx.setLineDash([]);
    ctx.globalAlpha = 0.8;
    ctx.beginPath();
    ctx.moveTo(from.x, from.y);
    ctx.lineTo(to.x, to.y);
    ctx.stroke();
    ctx.globalAlpha = 1;
    for (var m = 0; m < ROUTERS; m++) {
      ctx.strokeStyle = c.accent;
      ctx.beginPath();
      ctx.moveTo(nodes[m].px, nodes[m].py);
      ctx.lineTo(nodes[m].x, nodes[m].y);
      ctx.stroke();
    }

    var pt;
    if (q < 0.2) pt = along(from.px, from.py, from.x, from.y, ease(q / 0.2));
    else if (q < 0.5) pt = along(from.x, from.y, to.x, to.y, ease((q - 0.2) / 0.3));
    else if (q < 0.8) pt = along(to.x, to.y, from.x, from.y, ease((q - 0.5) / 0.3));
    else pt = along(from.x, from.y, from.px, from.py, ease((q - 0.8) / 0.2));
    dot(ctx, pt.x, pt.y, 3, q < 0.5 ? c.accent : c.fg);

    for (var r = 0; r < ROUTERS; r++) {
      var active = r === n || nodes[r] === to;
      ctx.fillStyle = c.bg;
      ctx.strokeStyle = active ? c.accent : c.muted;
      ctx.lineWidth = 1.4;
      ctx.beginPath();
      ctx.arc(nodes[r].x, nodes[r].y, br, 0, Math.PI * 2);
      ctx.fill();
      ctx.stroke();
      for (var d = 0; d < 4; d++) {
        dot(ctx, nodes[r].x + Math.cos(d * 1.9 + r) * br * 0.5, nodes[r].y + Math.sin(d * 1.9 + r) * br * 0.5, 1.2, c.fg);
      }
      person(ctx, nodes[r].px, nodes[r].py, s, r === n ? c.accent : c.fg, c.bg);
    }
  }

  var ACTS = [pile, web, routed];

  function draw(ctx, t, w, h, c) {
    var total = ACT * ACTS.length;
    var local = t % total;
    var act = Math.floor(local / ACT);
    var at = local - act * ACT;
    var alpha = Math.min(clamp(at / FADE), clamp((ACT - at) / FADE));
    var lang = (document.documentElement.lang || "en").slice(0, 2);
    var captions = CAPTIONS[lang] || CAPTIONS.en;

    ctx.save();
    ctx.globalAlpha = alpha;
    ACTS[act](ctx, at, w, h, c);
    ctx.restore();

    ctx.globalAlpha = alpha;
    ctx.font = "500 11px 'IBM Plex Mono', monospace";
    ctx.textAlign = "center";
    ctx.textBaseline = "top";
    ctx.fillStyle = act === 2 ? c.accent : c.muted;
    ctx.fillText((act + 1) + "/3 · " + captions[act], w / 2, 2);
    ctx.textAlign = "start";
    ctx.textBaseline = "alphabetic";
    ctx.globalAlpha = 1;
  }

  draw.still = ACT * 2.5;
  (window.trip2gCircleViz = window.trip2gCircleViz || {}).federation = draw;
})();
