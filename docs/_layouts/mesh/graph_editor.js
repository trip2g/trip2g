
// ============================================================
// DRAG NODES (dev tool — rearrange, get coords via console)
// ============================================================
(function() {
  const mount = document.getElementById("mesh-svg-mount");
  if (!mount) return;
  const svgEl = mount.querySelector("svg");
  if (!svgEl) return;

  let drag = null;

  function svgPoint(e) {
    const pt = svgEl.createSVGPoint();
    pt.x = e.clientX; pt.y = e.clientY;
    return pt.matrixTransform(svgEl.getScreenCTM().inverse());
  }

  function nodeAtPoint(svgX, svgY) {
    for (const n of NODES) {
      const hw = (n._w || 200) / 2, hh = (n._h || 64) / 2;
      if (svgX >= n.x - hw && svgX <= n.x + hw && svgY >= n.y - hh && svgY <= n.y + hh)
        return n;
    }
    return null;
  }

  function updatePipes(n) {
    for (const [a, b] of EDGES) {
      if (a !== n.id && b !== n.id) continue;
      const na = nodeById(a), nb = nodeById(b);
      const { p1, p2 } = edgePoint(na, nb);
      const key = `${a}-${b}`;
      const base = pipeBaseEls[key], flow = pipeFlowEls[key];
      if (base) { base.setAttribute("x1",p1.x); base.setAttribute("y1",p1.y); base.setAttribute("x2",p2.x); base.setAttribute("y2",p2.y); }
      if (flow) { flow.setAttribute("x1",p1.x); flow.setAttribute("y1",p1.y); flow.setAttribute("x2",p2.x); flow.setAttribute("y2",p2.y); }
    }
  }

  svgEl.addEventListener("pointerdown", e => {
    const p = svgPoint(e);
    const n = nodeAtPoint(p.x, p.y);
    if (!n) return;
    drag = { node: n, ox: p.x - n.x, oy: p.y - n.y, x0: n.x, y0: n.y };
    svgEl.setPointerCapture(e.pointerId);
    e.preventDefault();
  });

  svgEl.addEventListener("pointermove", e => {
    if (!drag) return;
    const p = svgPoint(e);
    const n = drag.node;
    n.x = Math.round(p.x - drag.ox);
    n.y = Math.round(p.y - drag.oy);
    // translate card group by delta from original build position
    if (n._el) n._el.setAttribute("transform", `translate(${n.x - n._bx},${n.y - n._by})`);
    updatePipes(n);
  });

  svgEl.addEventListener("pointerup", () => {
    if (!drag) return;
    console.log("NODES:\n" + NODES.map(n => `  { id:"${n.id}", x:${n.x}, y:${n.y} }`).join(",\n"));
    drag = null;
  });
})();
