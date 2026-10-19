(function () {
  var STEP = 1.5;
  var SCRIPT = [
    { me: true, w: 0.42 },
    { me: false, w: 0.62, note: true, chips: 2 },
    { me: true, w: 0.3 },
    { me: false, w: 0.55, note: true, chips: 3 },
    { me: true, w: 0.36 },
    { me: false, w: 0.6, note: true, chips: 2 }
  ];

  function clamp(v) { return v < 0 ? 0 : v > 1 ? 1 : v; }

  function roundRect(ctx, x, y, w, h, r) {
    ctx.beginPath();
    ctx.moveTo(x + r, y);
    ctx.arcTo(x + w, y, x + w, y + h, r);
    ctx.arcTo(x + w, y + h, x, y + h, r);
    ctx.arcTo(x, y + h, x, y, r);
    ctx.arcTo(x, y, x + w, y, r);
    ctx.closePath();
  }

  function height(msg, u) {
    return msg.me ? u * 0.9 : u * 1.7 + (msg.chips ? u * 0.85 : 0);
  }

  (window.trip2gCircleViz = window.trip2gCircleViz || {}).bot = function (ctx, t, w, h, c) {
    var u = Math.max(10, Math.min(h / 8, 18));
    var winW = Math.min(w * 0.9, u * 18);
    var x0 = (w - winW) / 2;
    var gap = u * 0.4;
    var total = Math.floor(t / STEP);
    var into = (t % STEP) / STEP;

    ctx.strokeStyle = c.rule;
    ctx.lineWidth = 1;
    roundRect(ctx, x0, 2, winW, h - 4, 8);
    ctx.stroke();

    ctx.save();
    ctx.beginPath();
    ctx.rect(x0 + 1, 3, winW - 2, h - 6);
    ctx.clip();

    var y = h - u * 0.6;
    var last = SCRIPT[total % SCRIPT.length];
    var rise = (1 - Math.pow(1 - clamp(into / 0.35), 3)) * (height(last, u) + gap);
    y += height(last, u) + gap - rise;

    for (var i = total; i > total - 8; i--) {
      var msg = SCRIPT[((i % SCRIPT.length) + SCRIPT.length) % SCRIPT.length];
      var hh = height(msg, u);
      y -= hh + gap;
      if (y + hh < 0) break;
      var bw = (winW - u * 1.4) * msg.w;
      var bx = msg.me ? x0 + winW - u * 0.7 - bw : x0 + u * 0.7;
      var bubbleH = msg.me ? hh : u * 1.7;
      ctx.globalAlpha = i === total ? clamp(into / 0.25) : 1;
      ctx.fillStyle = msg.me ? c.rule : c.bg;
      ctx.strokeStyle = msg.me ? c.rule : c.accent;
      roundRect(ctx, bx, y, bw, bubbleH, 7);
      ctx.fill();
      ctx.stroke();
      if (msg.note) {
        ctx.strokeStyle = c.fg;
        ctx.strokeRect(bx + u * 0.4, y + u * 0.35, u * 0.8, u * 1.0);
        ctx.fillStyle = c.muted;
        ctx.fillRect(bx + u * 1.5, y + u * 0.45, bw - u * 2, 2);
        ctx.fillRect(bx + u * 1.5, y + u * 0.85, (bw - u * 2) * 0.8, 2);
        ctx.fillRect(bx + u * 1.5, y + u * 1.25, (bw - u * 2) * 0.55, 2);
      } else {
        ctx.fillStyle = c.muted;
        ctx.fillRect(bx + u * 0.4, y + hh / 2 - 1, bw - u * 0.8, 2);
      }
      if (msg.chips) {
        var cw = (bw - (msg.chips - 1) * 4) / msg.chips;
        for (var k = 0; k < msg.chips; k++) {
          ctx.strokeStyle = k === 0 && i < total ? c.accent : c.muted;
          roundRect(ctx, bx + k * (cw + 4), y + bubbleH + u * 0.2, cw, u * 0.6, 4);
          ctx.stroke();
        }
      }
    }
    ctx.restore();
    ctx.globalAlpha = 1;
  };
})();
