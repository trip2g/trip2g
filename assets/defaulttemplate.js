(function(){
  var header = document.querySelector('.site-header');
  if (header) {
    window.addEventListener('scroll', function(){ header.classList.toggle('is-scrolled', window.scrollY > 0); }, { passive: true });
  }
  var backdrop = document.getElementById('backdrop');
  var leftSidebar = document.querySelector('.layout__sidebar--left');
  var rightSidebar = document.querySelector('.layout__sidebar--right');
  function openSidebar(s){ s.classList.add('is-open'); if(backdrop) backdrop.classList.add('is-visible'); document.body.style.overflow='hidden'; }
  function closeAll(){ leftSidebar&&leftSidebar.classList.remove('is-open'); rightSidebar&&rightSidebar.classList.remove('is-open'); if(backdrop) backdrop.classList.remove('is-visible'); document.body.style.overflow=''; }
  var btnLeft = document.getElementById('btn-left');
  if (btnLeft && leftSidebar) btnLeft.addEventListener('click', function(){ leftSidebar.classList.contains('is-open') ? closeAll() : openSidebar(leftSidebar); });
  var btnRight = document.getElementById('btn-right');
  if (btnRight && rightSidebar) btnRight.addEventListener('click', function(){ rightSidebar.classList.contains('is-open') ? closeAll() : openSidebar(rightSidebar); });
  if (backdrop) backdrop.addEventListener('click', closeAll);
  document.querySelectorAll('.layout__sidebar a[href]').forEach(function(a){
    if (a.pathname === location.pathname) a.classList.add('is-active');
  });
  (function() {
    var header = document.querySelector('.site-header');
    var sidebars = document.querySelectorAll('.layout__sidebar');
    if (!sidebars.length) return;
    var topOffset = (header ? header.offsetHeight : 56) + 16;
    var states = Array.prototype.map.call(sidebars, function(sb) {
      sb.style.top = topOffset + 'px';
      return { el: sb, top: topOffset };
    });
    var lastScrollY = window.scrollY;
    window.addEventListener('scroll', function() {
      var scrollY = window.scrollY;
      var delta = scrollY - lastScrollY;
      lastScrollY = scrollY;
      var vh = window.innerHeight;
      states.forEach(function(s) {
        var h = s.el.offsetHeight;
        if (h <= vh - topOffset) return;
        var minTop = vh - h;
        var newTop = Math.min(topOffset, Math.max(minTop, s.top - delta));
        if (newTop !== s.top) {
          s.top = newTop;
          s.el.style.top = newTop + 'px';
        }
      });
    }, { passive: true });
  })();

  // Heading anchors. The ids are already rendered by the markdown pipeline and
  // the TOC widget already links to them; this only adds the affordance to copy
  // one. Done in the DOM on purpose: emitting the anchor from the renderer would
  // change the stored HTML of every note, which is also what MCP section reads
  // and search breadcrumbs match against.
  (function(){
    var content = document.querySelector('.content__body');
    if (!content) return;
    content.querySelectorAll('h1[id],h2[id],h3[id],h4[id],h5[id],h6[id]').forEach(function(h){
      var a = document.createElement('a');
      a.className = 'heading-anchor';
      a.href = '#' + h.id;
      a.setAttribute('aria-hidden', 'true');
      a.setAttribute('tabindex', '-1');
      h.appendChild(a);
    });
  })();

})();
