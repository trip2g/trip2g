(function () {
  var SESSION = 2.4;
  var SESSIONS = 6;
  var WRITES = 2;
  var CYCLE = SESSION * SESSIONS;

  function clamp(v) { return v < 0 ? 0 : v > 1 ? 1 : v; }
  function ease(v) { v = clamp(v); return 1 - Math.pow(1 - v, 3); }

  function note(ctx, x, y, w, h, stroke, bg, muted) {
    ctx.fillStyle = bg;
    ctx.strokeStyle = stroke;
    ctx.lineWidth = 1.2;
    ctx.fillRect(x, y, w, h);
    ctx.strokeRect(x, y, w, h);
    ctx.fillStyle = muted;
    ctx.fillRect(x + 3, y + h * 0.35, w - 6, 1.5);
    ctx.fillRect(x + 3, y + h * 0.65, (w - 6) * 0.6, 1.5);
  }

  (window.trip2gCircleViz = window.trip2gCircleViz || {}).memory = function (ctx, t, w, h, c) {
    var local = t % CYCLE;
    var fadeAll = local > CYCLE - 0.5 ? (CYCLE - local) / 0.5 : 1;
    var k = Math.floor(local / SESSION);
    var st = (local - k * SESSION) / SESSION;

    var nw = Math.min(w * 0.07, h * 0.2);
    var nh = nw * 0.72;
    var cols = WRITES * SESSIONS;
    var gap = 6;
    var rowW = cols * nw + (cols - 1) * gap;
    var gx = (w - rowW) / 2;
    var gy = h * 0.78;
    function slot(i) { return { x: gx + i * (nw + gap), y: gy }; }

    var sw = Math.min(w * 0.26, h * 0.9);
    var sh = h * 0.34;
    var sx = w * 0.12 + (k % 3) * ((w * 0.76 - sw) / 2);
    var sy = h * 0.06;
    var appear = clamp(st / 0.12);
    var vanish = 1 - clamp((st - 0.82) / 0.18);
    var alive = Math.min(appear, vanish);

    for (var g = 1; g <= 2 && k - g >= 0; g++) {
      ctx.globalAlpha = fadeAll * (0.22 / g);
      ctx.strokeStyle = c.muted;
      ctx.lineWidth = 1;
      ctx.strokeRect(w * 0.12 + ((k - g) % 3) * ((w * 0.76 - sw) / 2), sy, sw, sh);
    }

    ctx.globalAlpha = fadeAll;
    var stored = k * WRITES;

    if (stored > 0 && st > 0.1 && st < 0.45) {
      var r = ease((st - 0.1) / 0.3);
      for (var i = 0; i < stored; i++) {
        var p = slot(i);
        var fx = p.x + nw / 2;
        var fy = p.y;
        var tx = sx + sw / 2;
        var ty = sy + sh;
        ctx.strokeStyle = c.accent;
        ctx.globalAlpha = fadeAll * 0.25 * alive;
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.moveTo(fx, fy);
        ctx.lineTo(tx, ty);
        ctx.stroke();
        ctx.globalAlpha = fadeAll * alive;
        ctx.fillStyle = c.accent;
        ctx.beginPath();
        ctx.arc(fx + (tx - fx) * r, fy + (ty - fy) * r, 2, 0, Math.PI * 2);
        ctx.fill();
      }
    }

    ctx.globalAlpha = fadeAll * alive;
    ctx.fillStyle = c.bg;
    ctx.strokeStyle = c.fg;
    ctx.lineWidth = 1.3;
    ctx.fillRect(sx, sy, sw, sh);
    ctx.strokeRect(sx, sy, sw, sh);
    ctx.fillStyle = c.rule;
    ctx.fillRect(sx, sy, sw, Math.max(6, sh * 0.18));
    ctx.fillStyle = c.accent;
    ctx.font = "600 " + Math.round(Math.max(9, sh * 0.26)) + "px 'IBM Plex Mono', monospace";
    ctx.fillText(">_", sx + 6, sy + sh * 0.68);
    ctx.fillStyle = c.muted;
    ctx.fillRect(sx + sw * 0.35, sy + sh * 0.5, sw * 0.5 * clamp(st / 0.6), 2);
    ctx.fillRect(sx + sw * 0.35, sy + sh * 0.72, sw * 0.35 * clamp((st - 0.2) / 0.5), 2);

    for (var n = 0; n < stored + WRITES; n++) {
      var s = slot(n);
      if (n < stored) {
        ctx.globalAlpha = fadeAll;
        note(ctx, s.x, s.y, nw, nh, c.fg, c.bg, c.muted);
        continue;
      }
      var start = 0.48 + (n - stored) * 0.14;
      var f = ease((st - start) / 0.25);
      if (f <= 0) continue;
      var ox = sx + sw / 2 - nw / 2;
      var oy = sy + sh;
      ctx.globalAlpha = fadeAll;
      note(ctx, ox + (s.x - ox) * f, oy + (s.y - oy) * f, nw, nh, c.accent, c.bg, c.muted);
    }
    ctx.globalAlpha = 1;
  };
})();
