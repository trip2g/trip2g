(function () {
  var FILES = 5;
  var DROP = 1.1;
  var CYCLE = FILES * DROP + 2.4;

  function clamp(v) { return v < 0 ? 0 : v > 1 ? 1 : v; }
  function ease(v) { v = clamp(v); return 1 - Math.pow(1 - v, 3); }

  function sheet(ctx, x, y, w, h, c, hot) {
    var fold = w * 0.28;
    ctx.fillStyle = c.bg;
    ctx.strokeStyle = hot ? c.accent : c.fg;
    ctx.lineWidth = 1.3;
    ctx.beginPath();
    ctx.moveTo(x, y);
    ctx.lineTo(x + w - fold, y);
    ctx.lineTo(x + w, y + fold);
    ctx.lineTo(x + w, y + h);
    ctx.lineTo(x, y + h);
    ctx.closePath();
    ctx.fill();
    ctx.stroke();
    ctx.fillStyle = hot ? c.accent : c.muted;
    ctx.font = "600 " + Math.round(h * 0.28) + "px 'IBM Plex Mono', monospace";
    ctx.fillText("#", x + 4, y + h * 0.42);
    ctx.fillStyle = c.muted;
    ctx.fillRect(x + 4, y + h * 0.58, w - 8, 1.5);
    ctx.fillRect(x + 4, y + h * 0.74, (w - 8) * 0.7, 1.5);
  }

  (window.trip2gCircleViz = window.trip2gCircleViz || {}).files = function (ctx, t, w, h, c) {
    var local = t % CYCLE;
    var fade = local > CYCLE - 0.6 ? (CYCLE - local) / 0.6 : 1;
    var fw = Math.min(w * 0.11, h * 0.34);
    var fh = fw * 0.62;
    var stackX = w * 0.1;
    var floor = h * 0.92;

    var pageW = Math.min(w * 0.42, h * 1.3);
    var pageX = w * 0.92 - pageW;
    var pageY = h * 0.08;
    var pageH = h * 0.84;

    ctx.globalAlpha = fade;
    ctx.strokeStyle = c.muted;
    ctx.lineWidth = 1.2;
    ctx.strokeRect(pageX, pageY, pageW, pageH);
    ctx.fillStyle = c.rule;
    ctx.fillRect(pageX, pageY, pageW, Math.max(8, pageH * 0.1));
    for (var d = 0; d < 3; d++) {
      ctx.fillStyle = c.muted;
      ctx.beginPath();
      ctx.arc(pageX + 7 + d * 7, pageY + Math.max(8, pageH * 0.1) / 2, 1.8, 0, Math.PI * 2);
      ctx.fill();
    }

    var arrowY = h / 2;
    var ax0 = stackX + fw + w * 0.06;
    var ax1 = pageX - w * 0.04;
    ctx.strokeStyle = c.dim;
    ctx.setLineDash([3, 4]);
    ctx.lineDashOffset = -t * 10;
    ctx.beginPath();
    ctx.moveTo(ax0, arrowY);
    ctx.lineTo(ax1, arrowY);
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.beginPath();
    ctx.moveTo(ax1 - 5, arrowY - 4);
    ctx.lineTo(ax1, arrowY);
    ctx.lineTo(ax1 - 5, arrowY + 4);
    ctx.stroke();

    var rowsTop = pageY + Math.max(8, pageH * 0.1) + 6;
    var rowH = (pageY + pageH - rowsTop - 4) / FILES;
    for (var i = 0; i < FILES; i++) {
      var landed = local - i * DROP - 0.6;
      var p = ease((local - i * DROP) / 0.6);
      if (p <= 0) continue;
      var y = -fh + (floor - fh * (i + 1) - 2 * i + fh) * p;
      sheet(ctx, stackX + (i % 2) * 3, y, fw, fh, c, landed > 0 && landed < 0.8);

      var r = ease(landed / 0.5);
      if (r > 0) {
        var ry = rowsTop + i * rowH;
        ctx.fillStyle = i === 0 ? c.fg : c.muted;
        var lw = (pageW - 16) * (i === 0 ? 0.6 : 0.95 - (i % 3) * 0.15) * r;
        ctx.fillRect(pageX + 8, ry + rowH * 0.25, lw, i === 0 ? Math.max(3, rowH * 0.32) : 2);
        if (i > 0) ctx.fillRect(pageX + 8, ry + rowH * 0.6, lw * 0.7, 2);
      }
    }
    ctx.globalAlpha = 1;
  };
})();
