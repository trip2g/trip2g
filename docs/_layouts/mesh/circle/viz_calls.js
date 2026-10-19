(function () {
  var BARS = 34;
  var CUT = 2.2;
  var NOTES = [
    { x: 0.66, y: 0.28 },
    { x: 0.8, y: 0.5 },
    { x: 0.64, y: 0.74 },
    { x: 0.55, y: 0.5 },
    { x: 0.85, y: 0.22 },
    { x: 0.82, y: 0.8 }
  ];
  var LINKS = [[0, 1], [1, 2], [2, 3], [3, 0], [0, 4], [2, 5], [1, 5]];

  function clamp(v) { return v < 0 ? 0 : v > 1 ? 1 : v; }
  function ease(v) { v = clamp(v); return v < 0.5 ? 4 * v * v * v : 1 - Math.pow(-2 * v + 2, 3) / 2; }

  function amp(i, t) {
    return 0.25 + 0.75 * Math.abs(Math.sin(i * 0.9 + t * 3.1) * Math.cos(i * 0.37 - t * 1.7));
  }

  (window.trip2gCircleViz = window.trip2gCircleViz || {}).calls = function (ctx, t, w, h, c) {
    var left = w * 0.04;
    var right = w * 0.46;
    var cy = h / 2;
    var bw = (right - left) / BARS;
    var cut = ((t % CUT) / CUT);
    var segStart = Math.floor(BARS * 0.55);

    for (var i = 0; i < BARS; i++) {
      var a = amp(i, t) * h * 0.36;
      var inSeg = i >= segStart && cut < 0.45;
      ctx.fillStyle = inSeg ? c.accent : c.muted;
      ctx.globalAlpha = 0.35 + 0.65 * (i / BARS);
      ctx.fillRect(left + i * bw, cy - a, Math.max(1, bw * 0.55), a * 2);
    }
    ctx.globalAlpha = 1;

    var done = Math.floor(t / CUT);
    var visible = Math.min(NOTES.length, 2 + (done % (NOTES.length + 2)));
    var freshIndex = cut < 0.97 ? visible - 1 : -1;
    var grow = ease((cut - 0.8) / 0.17);
    ctx.strokeStyle = c.muted;
    ctx.lineWidth = 1;
    for (var l = 0; l < LINKS.length; l++) {
      var a = LINKS[l][0];
      var b = LINKS[l][1];
      if (a >= visible || b >= visible) continue;
      var g = 1;
      if (a === freshIndex || b === freshIndex) {
        if (grow <= 0) continue;
        g = grow;
        if (a === freshIndex) { a = LINKS[l][1]; b = LINKS[l][0]; }
      }
      var ax = NOTES[a].x * w;
      var ay = NOTES[a].y * h;
      ctx.beginPath();
      ctx.moveTo(ax, ay);
      ctx.lineTo(ax + (NOTES[b].x * w - ax) * g, ay + (NOTES[b].y * h - ay) * g);
      ctx.stroke();
    }

    var cw = Math.min(w * 0.06, h * 0.2);
    var ch = cw * 1.25;
    for (var n = 0; n < visible; n++) {
      var nx = NOTES[n].x * w;
      var ny = NOTES[n].y * h;
      var fresh = n === freshIndex;
      if (fresh) {
        var p = ease((cut - 0.45) / 0.25);
        nx = right + (nx - right) * p;
        ny = cy + (ny - cy) * p;
        if (cut < 0.45) continue;
      }
      ctx.fillStyle = c.bg;
      ctx.strokeStyle = fresh ? c.accent : c.fg;
      ctx.lineWidth = 1.3;
      ctx.fillRect(nx - cw / 2, ny - ch / 2, cw, ch);
      ctx.strokeRect(nx - cw / 2, ny - ch / 2, cw, ch);
      ctx.fillStyle = c.muted;
      ctx.fillRect(nx - cw / 2 + 3, ny - ch / 2 + 4, cw - 6, 1.5);
      ctx.fillRect(nx - cw / 2 + 3, ny - ch / 2 + 8, (cw - 6) * 0.7, 1.5);
      if (n % 2 === 0) {
        ctx.strokeStyle = fresh ? c.accent : c.muted;
        ctx.strokeRect(nx - cw / 2 + 3, ny + ch / 2 - 7, 4, 4);
      }
    }
  };
})();
