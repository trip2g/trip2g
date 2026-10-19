(function () {
  var AGENTS = 6;
  var PERIOD = 2.6;

  function clamp(v) { return v < 0 ? 0 : v > 1 ? 1 : v; }

  function glyph(ctx, x, y, s, stroke, bg) {
    ctx.fillStyle = bg;
    ctx.strokeStyle = stroke;
    ctx.lineWidth = 1.4;
    ctx.beginPath();
    ctx.rect(x - s, y - s * 0.8, s * 2, s * 1.6);
    ctx.fill();
    ctx.stroke();
    ctx.beginPath();
    ctx.moveTo(x, y - s * 0.8);
    ctx.lineTo(x, y - s * 1.3);
    ctx.stroke();
    ctx.fillStyle = stroke;
    ctx.fillRect(x - s * 0.5, y - s * 0.2, s * 0.3, s * 0.3);
    ctx.fillRect(x + s * 0.2, y - s * 0.2, s * 0.3, s * 0.3);
  }

  (window.trip2gCircleViz = window.trip2gCircleViz || {}).agents = function (ctx, t, w, h, c) {
    var cx = w / 2;
    var cy = h / 2;
    var rx = Math.min(w * 0.42, h * 1.3);
    var ry = h * 0.38;
    var core = Math.min(h * 0.16, 22);
    var s = Math.max(5, Math.min(9, h * 0.055));

    for (var i = 0; i < AGENTS; i++) {
      var a = (i / AGENTS) * Math.PI * 2 + t * 0.12;
      var x = cx + Math.cos(a) * rx;
      var y = cy + Math.sin(a) * ry;
      var phase = ((t + i * (PERIOD / AGENTS) * 1.7) % PERIOD) / PERIOD;
      var dx = cx - x;
      var dy = cy - y;
      var len = Math.sqrt(dx * dx + dy * dy);
      var ux = dx / len;
      var uy = dy / len;
      var sx = x + ux * s * 1.2;
      var sy = y + uy * s * 1.2;
      var ex = cx - ux * core * 1.1;
      var ey = cy - uy * core * 1.1;
      var active = phase < 0.8;

      ctx.strokeStyle = active ? c.accent : c.rule;
      ctx.globalAlpha = active ? 0.35 + 0.25 * Math.sin(phase * Math.PI) : 1;
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(sx, sy);
      ctx.lineTo(ex, ey);
      ctx.stroke();
      ctx.globalAlpha = 1;

      if (phase < 0.4) {
        var q = clamp(phase / 0.4);
        ctx.fillStyle = c.accent;
        ctx.beginPath();
        ctx.arc(sx + (ex - sx) * q, sy + (ey - sy) * q, 2.6, 0, Math.PI * 2);
        ctx.fill();
      } else if (phase < 0.8) {
        var r = clamp((phase - 0.4) / 0.4);
        ctx.fillStyle = c.fg;
        ctx.fillRect(ex + (sx - ex) * r - 3, ey + (sy - ey) * r - 2, 6, 4);
      }
      glyph(ctx, x, y, s, active ? c.fg : c.muted, c.bg);
    }

    for (var k = 2; k >= 0; k--) {
      ctx.fillStyle = c.bg;
      ctx.strokeStyle = k === 0 ? c.accent : c.muted;
      ctx.lineWidth = 1.4;
      ctx.beginPath();
      ctx.ellipse(cx, cy - core * 0.5 + k * core * 0.5, core * 1.2, core * 0.42, 0, 0, Math.PI * 2);
      ctx.fill();
      ctx.stroke();
    }
    ctx.globalAlpha = 0.5 + 0.5 * Math.sin(t * 3);
    ctx.fillStyle = c.accent;
    ctx.beginPath();
    ctx.arc(cx, cy - core * 0.5, 2.2, 0, Math.PI * 2);
    ctx.fill();
    ctx.globalAlpha = 1;
  };
})();
