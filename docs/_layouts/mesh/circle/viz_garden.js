(function () {
  var COUNT = 22;
  var GROW = 0.28;
  var CYCLE = COUNT * GROW + 3.5;

  function rng(seed) {
    return function () {
      seed = (seed * 16807) % 2147483647;
      return (seed - 1) / 2147483646;
    };
  }

  var rand = rng(7);
  var nodes = [];
  for (var i = 0; i < COUNT; i++) {
    var a = rand() * Math.PI * 2;
    var d = Math.sqrt(rand()) * 0.9 + 0.08;
    nodes.push({ x: Math.cos(a) * d, y: Math.sin(a) * d, r: 2 + rand() * 2.5, links: [] });
  }
  for (var k = 1; k < COUNT; k++) {
    var best = -1;
    var second = -1;
    var bd = 1e9;
    var sd = 1e9;
    for (var j = 0; j < k; j++) {
      var dx = nodes[k].x - nodes[j].x;
      var dy = nodes[k].y - nodes[j].y;
      var dist = dx * dx + dy * dy;
      if (dist < bd) { sd = bd; second = best; bd = dist; best = j; }
      else if (dist < sd) { sd = dist; second = j; }
    }
    nodes[k].links.push(best);
    if (second >= 0 && k % 3 === 0) nodes[k].links.push(second);
  }

  function ease(t) { return t < 0 ? 0 : t > 1 ? 1 : 1 - Math.pow(1 - t, 3); }

  (window.trip2gCircleViz = window.trip2gCircleViz || {}).garden = function (ctx, t, w, h, c) {
    var local = t % CYCLE;
    var fade = local > CYCLE - 0.8 ? (CYCLE - local) / 0.8 : 1;
    var sx = w * 0.42;
    var sy = h * 0.42;
    var cx = w / 2;
    var cy = h / 2;
    function px(n) { return cx + n.x * sx; }
    function py(n) { return cy + n.y * sy; }

    ctx.globalAlpha = fade;
    ctx.lineWidth = 1;
    for (var i = 0; i < COUNT; i++) {
      var born = i * GROW;
      var g = ease((local - born - 0.15) / 0.45);
      if (g <= 0) continue;
      for (var l = 0; l < nodes[i].links.length; l++) {
        var to = nodes[nodes[i].links[l]];
        ctx.strokeStyle = c.muted;
        ctx.globalAlpha = fade * 0.55;
        ctx.beginPath();
        ctx.moveTo(px(nodes[i]), py(nodes[i]));
        ctx.lineTo(px(nodes[i]) + (px(to) - px(nodes[i])) * g, py(nodes[i]) + (py(to) - py(nodes[i])) * g);
        ctx.stroke();
      }
    }
    for (var n = 0; n < COUNT; n++) {
      var s = ease((local - n * GROW) / 0.35);
      if (s <= 0) continue;
      var fresh = local - n * GROW < 1.2;
      var pulse = 1 + 0.12 * Math.sin(t * 2 + n);
      ctx.globalAlpha = fade;
      ctx.fillStyle = fresh ? c.accent : c.fg;
      ctx.beginPath();
      ctx.arc(px(nodes[n]), py(nodes[n]), nodes[n].r * s * pulse, 0, Math.PI * 2);
      ctx.fill();
      if (fresh) {
        ctx.globalAlpha = fade * (1 - (local - n * GROW) / 1.2) * 0.6;
        ctx.strokeStyle = c.accent;
        ctx.beginPath();
        ctx.arc(px(nodes[n]), py(nodes[n]), nodes[n].r + 10 * ease((local - n * GROW) / 1.2), 0, Math.PI * 2);
        ctx.stroke();
      }
    }
    ctx.globalAlpha = 1;
  };
})();
