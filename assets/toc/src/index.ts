// TOC widget - reads JSON data from widget__data script tag,
// renders links into #dt-toc-nav, tracks active heading via IntersectionObserver.

interface TOCItem {
  Text: string;
  Level: number;
  ID: string;
}

function initTOC() {
  const dataEl = document.querySelector<HTMLScriptElement>('script.widget__data[type="application/json"]');
  if (!dataEl) return;

  let items: TOCItem[];
  try {
    items = JSON.parse(dataEl.textContent || '[]');
  } catch { return; }

  const nav = document.getElementById('dt-toc-nav');
  if (!nav) return;

  // Build links (server already rendered fallback, replace with enhanced version)
  nav.innerHTML = '';
  const links = new Map<string, HTMLAnchorElement>();

  for (const item of items) {
    const a = document.createElement('a');
    a.href = '#' + item.ID;
    a.textContent = item.Text;
    a.style.setProperty('--data-level', String(item.Level));
    a.dataset.level = String(item.Level);
    links.set(item.ID, a);

    const div = document.createElement('div');
    div.className = `toc__item toc__item--level${item.Level}`;
    div.appendChild(a);
    nav.appendChild(div);
  }

  // IntersectionObserver for active heading
  const headings = items
    .map(item => document.getElementById(item.ID))
    .filter(Boolean) as HTMLElement[];

  if (headings.length === 0) return;

  let activeID = '';

  const observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          activeID = entry.target.id;
        }
      }
      links.forEach((a, id) => {
        a.classList.toggle('active', id === activeID);
      });
    },
    { rootMargin: '0px 0px -80% 0px', threshold: 0 }
  );

  headings.forEach(h => observer.observe(h));
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initTOC);
} else {
  initTOC();
}
