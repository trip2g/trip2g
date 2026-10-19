(function () {
  var BASES = 5;

  function rng(seed) {
    return function () {
      seed = (seed * 16807) % 2147483647;
      return (seed - 1) / 2147483646;
    };
  }

  var rand = rng(23);
  var dots = [];
  for (var b = 0; b < BASES; b++) {
    var list = [];
    for (var i = 0; i < 7; i++) list.push({ a: rand() * Math.PI * 2, d: 0.25 + rand() * 0.6 });
    dots.push(list);
  }

  function clamp(v) { return v < 0 ? 0 : v > 1 ? 1 : v; }

  (window.trip2gCircleViz = window.trip2gCircleViz || {}).federation = function (ctx, t, w, h, c) {
    var cx = w / 2;
    var cy = h / 2;
    var rx = Math.min(w * 0.38, h * 1.25);
    var ry = h * 0.33;
    var br = Math.max(10, Math.min(h * 0.14, w * 0.07));
    var pos = [];
    for (var b = 0; b < BASES; b++) {
      var a = -Math.PI / 2 + (b / BASES) * Math.PI * 2;
      pos.push({ x: cx + Math.cos(a) * rx, y: cy + Math.sin(a) * ry });
    }

    ctx.lineWidth = 1;
    ctx.setLineDash([2, 5]);
    ctx.strokeStyle = c.muted;
    ctx.globalAlpha = 0.6;
    for (var i = 0; i < BASES; i++) {
      ctx.beginPath();
      ctx.moveTo(cx, cy);
      ctx.lineTo(pos[i].x, pos[i].y);
      ctx.stroke();
    }
    ctx.setLineDash([]);
    ctx.globalAlpha = 1;

    var wave = (t % 3) / 3;
    for (var k = 0; k < BASES; k++) {
      var out = clamp(wave / 0.45);
      var back = clamp((wave - 0.5) / 0.45);
      var p = wave < 0.5 ? out : 1 - back;
      var dx = pos[k].x - cx;
      var dy = pos[k].y - cy;
      var len = Math.sqrt(dx * dx + dy * dy);
      var f = (len - br) / len;
      ctx.fillStyle = wave < 0.5 ? c.fg : c.accent;
      ctx.globalAlpha = wave > 0.95 ? 0 : 1;
      ctx.beginPath();
      ctx.arc(cx + dx * f * p, cy + dy * f * p, 2.2, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.globalAlpha = 1;

    for (var n = 0; n < BASES; n++) {
      var bx = pos[n].x;
      var by = pos[n].y;
      ctx.fillStyle = c.bg;
      ctx.strokeStyle = c.muted;
      ctx.lineWidth = 1.2;
      ctx.beginPath();
      ctx.arc(bx, by, br, 0, Math.PI * 2);
      ctx.fill();
      ctx.stroke();
      var visible = 4 + Math.floor(((t * 0.7 + n * 1.3) % 4));
      ctx.fillStyle = c.fg;
      for (var d = 0; d < dots[n].length && d < visible; d++) {
        ctx.beginPath();
        ctx.arc(bx + Math.cos(dots[n][d].a) * dots[n][d].d * br, by + Math.sin(dots[n][d].a) * dots[n][d].d * br, 1.4, 0, Math.PI * 2);
        ctx.fill();
      }
      var wa = t * (0.9 + n * 0.15) + n;
      ctx.fillStyle = c.accent;
      ctx.beginPath();
      ctx.arc(bx + Math.cos(wa) * br * 0.55, by + Math.sin(wa) * br * 0.55, 2.8, 0, Math.PI * 2);
      ctx.fill();
    }

    ctx.fillStyle = c.bg;
    ctx.strokeStyle = c.accent;
    ctx.lineWidth = 1.4;
    ctx.beginPath();
    ctx.moveTo(cx, cy - 7);
    ctx.lineTo(cx + 7, cy);
    ctx.lineTo(cx, cy + 7);
    ctx.lineTo(cx - 7, cy);
    ctx.closePath();
    ctx.fill();
    ctx.stroke();
  };
})();
