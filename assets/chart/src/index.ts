// datachart widget — finds `.chart` containers, reads the sibling
// `.chart__data` JSON ({config, data?}), resolves the rows (inline data, or
// fetch data-src as csv/json), auto-injects them into the ECharts config, draws.
//
// ECharts itself is loaded lazily (only when a page actually has charts) from
// /assets/echarts.min.js, so chart-free pages pay nothing but this tiny glue.

interface ChartSpec {
  config?: any;
  data?: unknown;
}

function csvToRows(text: string): unknown[] {
  const lines = text.trim().split(/\r?\n/).filter((l) => l.length > 0);
  if (lines.length === 0) return [];
  return lines.map((line) =>
    line.split(',').map((cell) => {
      const t = cell.trim();
      const n = Number(t);
      return t !== '' && !Number.isNaN(n) ? n : t;
    }),
  );
}

// Replace every "$data" placeholder in the config with rows; if none present,
// auto-inject into config.dataset.source.
function injectData(config: any, rows: unknown): any {
  if (rows == null) return config;

  let replaced = false;
  const walk = (o: any) => {
    if (Array.isArray(o)) {
      o.forEach((v, i) => {
        if (v === '$data') { o[i] = rows; replaced = true; } else walk(v);
      });
    } else if (o && typeof o === 'object') {
      for (const k of Object.keys(o)) {
        if (o[k] === '$data') { o[k] = rows; replaced = true; } else walk(o[k]);
      }
    }
  };
  walk(config);

  if (!replaced) {
    if (!config.dataset) config.dataset = { source: rows };
    else if (!Array.isArray(config.dataset) && config.dataset.source == null) {
      config.dataset.source = rows;
    }
  }
  return config;
}

async function resolveRows(el: HTMLElement, spec: ChartSpec): Promise<unknown> {
  if (spec.data != null) return spec.data;
  const src = el.getAttribute('data-src');
  if (!src) return null; // url/internal not yet wired → loader
  const res = await fetch(src);
  if (el.getAttribute('data-format') === 'csv') return csvToRows(await res.text());
  return res.json();
}

function renderChart(el: HTMLElement) {
  const dataEl = el.parentElement?.querySelector<HTMLScriptElement>('.chart__data');
  if (!dataEl) return;

  let spec: ChartSpec;
  try { spec = JSON.parse(dataEl.textContent || '{}'); } catch { return; }

  if (!el.style.height) el.style.height = '320px';

  const echarts = (window as any).echarts;
  const instance = echarts.init(el);
  el.classList.add('chart--loading');

  resolveRows(el, spec)
    .then((rows) => {
      el.classList.remove('chart--loading');
      if (rows == null && spec.data == null) return; // no data yet — keep loader
      instance.setOption(injectData(spec.config || {}, rows));
    })
    .catch(() => el.classList.remove('chart--loading'));

  window.addEventListener('resize', () => instance.resize());
}

function loadECharts(cb: () => void) {
  if ((window as any).echarts) { cb(); return; }
  let s = document.getElementById('echarts-lib') as HTMLScriptElement | null;
  if (s) { s.addEventListener('load', cb); return; }
  s = document.createElement('script');
  s.id = 'echarts-lib';
  s.src = '/assets/echarts.min.js';
  s.onload = cb;
  document.head.appendChild(s);
}

function initCharts() {
  const els = Array.from(document.querySelectorAll<HTMLElement>('.chart'));
  if (els.length === 0) return;
  loadECharts(() => els.forEach(renderChart));
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initCharts);
} else {
  initCharts();
}
