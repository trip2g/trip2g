(function () {
  var SPACING = 1.6;
  var TRAVEL = 3.2;

  function clamp(v) { return v < 0 ? 0 : v > 1 ? 1 : v; }
  function ease(v) { v = clamp(v); return v < 0.5 ? 4 * v * v * v : 1 - Math.pow(-2 * v + 2, 3) / 2; }

  function roundRect(ctx, x, y, w, h, r) {
    ctx.beginPath();
    ctx.moveTo(x + r, y);
    ctx.arcTo(x + w, y, x + w, y + h, r);
    ctx.arcTo(x + w, y + h, x, y + h, r);
    ctx.arcTo(x, y + h, x, y, r);
    ctx.arcTo(x, y, x + w, y, r);
    ctx.closePath();
  }

  (window.trip2gCircleViz = window.trip2gCircleViz || {}).channel = function (ctx, t, w, h, c) {
    var u = Math.min(w / 10, h / 4.2);
    var lineY = h * 0.5;
    var x0 = w * 0.08;
    var phoneW = u * 2.6;
    var phoneX = w - phoneW - w * 0.06;
    var x1 = phoneX - u * 0.6;

    ctx.strokeStyle = c.rule;
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(x0, lineY);
    ctx.lineTo(x1, lineY);
    ctx.stroke();
    var tickSpan = (x1 - x0) / 8;
    var shift = (t * tickSpan / SPACING) % tickSpan;
    ctx.fillStyle = c.dim;
    for (var k = 0; k < 10; k++) {
      var tx = x0 + k * tickSpan + shift;
      if (tx > x1) continue;
      ctx.fillRect(tx, lineY - 3, 1, 6);
    }

    ctx.strokeStyle = c.muted;
    roundRect(ctx, phoneX, h * 0.08, phoneW, h * 0.84, 8);
    ctx.stroke();

    var newest = Math.floor(t / SPACING);
    for (var i = newest; i > newest - 8; i--) {
      var age = t - i * SPACING;
      var cardW = u * 1.1;
      var cardH = u * 1.3;
      if (age < TRAVEL) {
        var p = ease(age / TRAVEL);
        var x = x0 + (x1 - cardW / 2 - x0) * p;
        ctx.globalAlpha = clamp(age / 0.4);
        ctx.fillStyle = c.bg;
        ctx.strokeStyle = c.fg;
        roundRect(ctx, x - cardW / 2, lineY - cardH / 2, cardW, cardH, 3);
        ctx.fill();
        ctx.stroke();
        ctx.fillStyle = c.muted;
        for (var r = 0; r < 4; r++) {
          ctx.fillRect(x - cardW / 2 + 5, lineY - cardH / 2 + 7 + r * (cardH - 12) / 4, (cardW - 10) * (r === 0 ? 0.6 : 1 - r * 0.12), 2);
        }
      } else {
        var slot = (age - TRAVEL) / SPACING;
        var bubbleH = u * 0.62;
        var by = h * 0.86 - bubbleH - slot * (bubbleH + 6);
        if (by < h * 0.12) continue;
        var pop = ease((age - TRAVEL) / 0.35);
        var fresh = age - TRAVEL < SPACING * 0.9;
        ctx.globalAlpha = Math.min(1, (by - h * 0.12) / (bubbleH));
        ctx.fillStyle = fresh ? c.accent : c.rule;
        roundRect(ctx, phoneX + 6, by, (phoneW - 12) * (0.4 + 0.6 * pop), bubbleH, 6);
        ctx.fill();
        ctx.fillStyle = fresh ? c.bg : c.muted;
        ctx.fillRect(phoneX + 12, by + bubbleH * 0.35, (phoneW - 30) * pop * 0.8, 2);
        ctx.fillRect(phoneX + 12, by + bubbleH * 0.62, (phoneW - 30) * pop * 0.5, 2);
      }
    }
    ctx.globalAlpha = 1;
  };
})();
