(function () {
  function rng(seed) {
    return function () {
      seed = (seed * 16807) % 2147483647;
      return (seed - 1) / 2147483646;
    };
  }

  var rand = rng(11);
  var notes = [];
  for (var i = 0; i < 22; i++) {
    var a = rand() * Math.PI * 2;
    var d = Math.sqrt(rand());
    notes.push({ a: a, d: d, paid: d > 0.35 });
  }
  var agents = [
    { ang: -0.55, sub: true, phase: 0 },
    { ang: 0.0, sub: true, phase: 1.3 },
    { ang: 0.55, sub: false, phase: 2.1 },
    { ang: 1.05, sub: true, phase: 3.0 }
  ];

  function clamp(v) { return v < 0 ? 0 : v > 1 ? 1 : v; }

  function agent(ctx, x, y, s, color, bg) {
    ctx.fillStyle = bg;
    ctx.strokeStyle = color;
    ctx.lineWidth = 1.4;
    ctx.beginPath();
    ctx.rect(x - s, y - s * 0.8, s * 2, s * 1.6);
    ctx.fill();
    ctx.stroke();
    ctx.fillStyle = color;
    ctx.fillRect(x - s * 0.5, y - s * 0.2, s * 0.3, s * 0.3);
    ctx.fillRect(x + s * 0.2, y - s * 0.2, s * 0.3, s * 0.3);
  }

  (window.trip2gCircleViz = window.trip2gCircleViz || {}).subscription = function (ctx, t, w, h, c) {
    var R = Math.min(h * 0.4, w * 0.22);
    var cx = w * 0.36;
    var cy = h / 2;
    var membrane = R * 1.18;

    for (var n = 0; n < notes.length; n++) {
      var nx = cx + Math.cos(notes[n].a) * notes[n].d * R;
      var ny = cy + Math.sin(notes[n].a) * notes[n].d * R;
      ctx.fillStyle = notes[n].paid ? c.accent : c.fg;
      ctx.globalAlpha = notes[n].paid ? 0.85 : 1;
      ctx.beginPath();
      ctx.arc(nx, ny, 2.2, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.globalAlpha = 1;

    ctx.strokeStyle = c.accent;
    ctx.lineWidth = 1.2;
    ctx.setLineDash([3, 4]);
    ctx.lineDashOffset = -t * 8;
    ctx.beginPath();
    for (var s = 0; s <= 64; s++) {
      var ang = (s / 64) * Math.PI * 2;
      var rr = membrane + Math.sin(ang * 6 + t * 1.5) * 2.5;
      var mx = cx + Math.cos(ang) * rr;
      var my = cy + Math.sin(ang) * rr;
      if (s === 0) ctx.moveTo(mx, my); else ctx.lineTo(mx, my);
    }
    ctx.stroke();
    ctx.setLineDash([]);

    var ax = w * 0.86;
    var size = Math.max(5, Math.min(9, h * 0.05));
    for (var i = 0; i < agents.length; i++) {
      var ag = agents[i];
      var ay = cy + Math.sin(ag.ang) * h * 0.42;
      var color = ag.sub ? c.accent : c.dim;
      var hitA = Math.atan2(ay - cy, ax - cx);
      var edgeX = cx + Math.cos(hitA) * membrane;
      var edgeY = cy + Math.sin(hitA) * membrane;
      var endX = ag.sub ? cx + Math.cos(hitA) * R * 0.45 : edgeX;
      var endY = ag.sub ? cy + Math.sin(hitA) * R * 0.45 : edgeY;

      ctx.strokeStyle = color;
      ctx.globalAlpha = 0.45;
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(ax - size, ay);
      ctx.lineTo(endX, endY);
      ctx.stroke();

      var cyc = ((t + ag.phase) % 2.4) / 2.4;
      var p = clamp(cyc / 0.7);
      var bx = ax - size + (endX - ax + size) * p;
      var by = ay + (endY - ay) * p;
      ctx.globalAlpha = ag.sub ? 1 : 1 - clamp((cyc - 0.6) / 0.3);
      ctx.fillStyle = color;
      ctx.beginPath();
      ctx.arc(bx, by, 2.6, 0, Math.PI * 2);
      ctx.fill();
      if (!ag.sub && p >= 1) {
        ctx.strokeStyle = c.muted;
        ctx.globalAlpha = 1 - clamp((cyc - 0.7) / 0.3);
        ctx.beginPath();
        ctx.moveTo(edgeX - 4, edgeY - 4);
        ctx.lineTo(edgeX + 4, edgeY + 4);
        ctx.moveTo(edgeX + 4, edgeY - 4);
        ctx.lineTo(edgeX - 4, edgeY + 4);
        ctx.stroke();
      }
      ctx.globalAlpha = 1;
      agent(ctx, ax, ay, size, color, c.bg);
    }
  };
})();
