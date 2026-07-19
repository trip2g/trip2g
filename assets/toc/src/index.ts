// TOC widget - reads JSON data from widget__data script tag,
// renders links into #dt-toc-nav, tracks active heading via IntersectionObserver.

interface TOCItem {
  Text: string;
  Level: number;
  ID: string;
}

// Strip a trailing slash from a path (but keep root "/").
const normalize = (p: string) => (p.length > 1 ? p.replace(/\/$/, '') : p);

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
    a.className = 'toc__link';
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

function initSidebarActiveLink() {
  const current = normalize(location.pathname);

  const links = document.querySelectorAll<HTMLAnchorElement>(
    '.sidebar-nav a, .layout__sidebar--left .widget--content a'
  );

  links.forEach(a => {
    try {
      const url = new URL(a.href, location.href);
      if (normalize(url.pathname) === current) {
        a.classList.add('is-active');
        a.setAttribute('aria-current', 'page');
      }
    } catch {
      // skip malformed hrefs
    }
  });
}

function initAll() {
  initTOC();
  initSidebarActiveLink();
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initAll);
} else {
  initAll();
}
