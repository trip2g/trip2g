(function () {
  var STEP = 0.42;

  function rng(seed) {
    return function () {
      seed = (seed * 16807) % 2147483647;
      return (seed - 1) / 2147483646;
    };
  }

  function clamp(v) { return v < 0 ? 0 : v > 1 ? 1 : v; }
  function ease(v) { v = clamp(v); return 1 - Math.pow(1 - v, 3); }

  var pages = [
    { x: -0.62, y: -0.72, depth: 0 },
    { x: -0.5, y: -0.36, depth: 1 },
    { x: -0.5, y: 0.0, depth: 1 },
    { x: -0.38, y: 0.36, depth: 2 },
    { x: -0.5, y: 0.72, depth: 1 }
  ];

  var rand = rng(11);
  var notes = [];
  while (notes.length < 11) {
    var nx = 0.1 + rand() * 0.8;
    var ny = -0.8 + rand() * 1.6;
    var free = notes.every(function (o) {
      return (o.x - nx) * (o.x - nx) + (o.y - ny) * (o.y - ny) * 0.3 > 0.026;
    });
    if (free) notes.push({ x: nx, y: ny, r: 2 + rand() * 2.2, links: [] });
  }
  for (var k = 1; k < notes.length; k++) {
    var best = 0;
    var bd = 1e9;
    for (var j = 0; j < k; j++) {
      var dx = notes[k].x - notes[j].x;
      var dy = notes[k].y - notes[j].y;
      var d = dx * dx + dy * dy;
      if (d < bd) { bd = d; best = j; }
    }
    notes[k].links.push(best);
  }

  var cross = [[1, 2], [2, 5], [3, 8], [4, 10]];

  var order = [];
  var p = 0;
  var n = 0;
  while (p < pages.length || n < notes.length) {
    if (p < pages.length) order.push({ kind: "page", i: p++ });
    if (n < notes.length) order.push({ kind: "note", i: n++ });
    if (n < notes.length) order.push({ kind: "note", i: n++ });
  }
  var born = { page: [], note: [] };
  order.forEach(function (o, idx) { born[o.kind][o.i] = idx * STEP; });

  var GROWN = order.length * STEP + 0.6;
  var CYCLE = GROWN + 3.2;

  function draw(ctx, t, w, h, c) {
    var local = t % CYCLE;
    var fade = local > CYCLE - 0.8 ? (CYCLE - local) / 0.8 : 1;
    var cx = w / 2;
    var cy = h / 2;
    var sx = w * 0.46;
    var sy = h * 0.44;
    var pw = Math.min(w * 0.17, h * 0.5);
    var ph = Math.max(10, h * 0.13);

    function X(v) { return cx + v * sx; }
    function Y(v) { return cy + v * sy; }
    function grown(at, len) { return ease((local - at) / len); }

    ctx.lineWidth = 1;
    ctx.strokeStyle = c.muted;
    for (var a = 1; a < pages.length; a++) {
      var g = grown(born.page[a], 0.4);
      if (g <= 0) continue;
      var parent = pages[a].depth === 2 ? pages[a - 1] : pages[0];
      var x0 = X(parent.x) + 6;
      var y0 = Y(parent.y) + ph / 2;
      var y1 = Y(pages[a].y);
      var x1 = X(pages[a].x);
      ctx.globalAlpha = fade * 0.6;
      ctx.beginPath();
      ctx.moveTo(x0, y0);
      ctx.lineTo(x0, y0 + (y1 - y0) * g);
      if (g >= 1) ctx.lineTo(x1, y1);
      ctx.stroke();
    }

    for (var m = 0; m < notes.length; m++) {
      var gl = grown(born.note[m] + 0.15, 0.45);
      if (gl <= 0) continue;
      for (var l = 0; l < notes[m].links.length; l++) {
        var to = notes[notes[m].links[l]];
        ctx.globalAlpha = fade * 0.5;
        ctx.strokeStyle = c.muted;
        ctx.beginPath();
        ctx.moveTo(X(notes[m].x), Y(notes[m].y));
        ctx.lineTo(X(notes[m].x) + (X(to.x) - X(notes[m].x)) * gl, Y(notes[m].y) + (Y(to.y) - Y(notes[m].y)) * gl);
        ctx.stroke();
      }
    }

    for (var e = 0; e < cross.length; e++) {
      var pg = pages[cross[e][0]];
      var nt = notes[cross[e][1]];
      var at = Math.max(born.page[cross[e][0]], born.note[cross[e][1]]) + 0.3;
      var gc = grown(at, 0.7);
      if (gc <= 0) continue;
      var sx0 = X(pg.x) + pw;
      var sy0 = Y(pg.y) + ph / 2;
      var ex = X(nt.x);
      var ey = Y(nt.y);
      var mx = (sx0 + ex) / 2;
      var hot = local - at < 1.4;
      ctx.globalAlpha = fade * (hot ? 0.9 : 0.55);
      ctx.strokeStyle = hot ? c.accent : c.dim;
      ctx.setLineDash([3, 3]);
      ctx.beginPath();
      ctx.moveTo(sx0, sy0);
      var steps = Math.max(2, Math.round(16 * gc));
      for (var s = 1; s <= steps; s++) {
        var u = (s / steps) * gc;
        var bx = (1 - u) * (1 - u) * sx0 + 2 * (1 - u) * u * mx + u * u * ex;
        var by = (1 - u) * (1 - u) * sy0 + 2 * (1 - u) * u * ((sy0 + ey) / 2 - h * 0.08) + u * u * ey;
        ctx.lineTo(bx, by);
      }
      ctx.stroke();
      ctx.setLineDash([]);
    }

    for (var q = 0; q < pages.length; q++) {
      var gp = grown(born.page[q], 0.4);
      if (gp <= 0) continue;
      var px = X(pages[q].x);
      var py = Y(pages[q].y);
      var fresh = local - born.page[q] < 1.2;
      ctx.globalAlpha = fade * gp;
      ctx.fillStyle = c.bg;
      ctx.strokeStyle = fresh ? c.accent : c.fg;
      ctx.lineWidth = 1.2;
      ctx.fillRect(px, py, pw, ph);
      ctx.strokeRect(px, py, pw, ph);
      ctx.fillStyle = fresh ? c.accent : c.fg;
      ctx.fillRect(px + 4, py + ph * 0.22, (pw - 8) * 0.55 * gp, Math.max(2, ph * 0.2));
      ctx.fillStyle = c.muted;
      ctx.fillRect(px + 4, py + ph * 0.62, (pw - 8) * 0.85 * gp, 1.5);
      ctx.lineWidth = 1;
    }

    for (var r = 0; r < notes.length; r++) {
      var gn = grown(born.note[r], 0.35);
      if (gn <= 0) continue;
      var nfresh = local - born.note[r] < 1.2;
      var pulse = 1 + 0.12 * Math.sin(t * 2 + r);
      ctx.globalAlpha = fade;
      ctx.fillStyle = nfresh ? c.accent : c.fg;
      ctx.beginPath();
      ctx.arc(X(notes[r].x), Y(notes[r].y), notes[r].r * gn * pulse, 0, Math.PI * 2);
      ctx.fill();
    }

    ctx.globalAlpha = 1;
  }

  draw.still = GROWN + 1.6;
  (window.trip2gCircleViz = window.trip2gCircleViz || {}).selfsite = draw;
})();
