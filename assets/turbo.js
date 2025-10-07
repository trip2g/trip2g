document.addEventListener("DOMContentLoaded", function () {
  // turbo
  var contentSelector = "#all-content";
  var prefetchCache = {};
  var PREFETCH_TTL = 60 * 1000;

  function now() { return Date.now(); }

  function remountMol() {
    if (typeof $mol_view !== "undefined") {
      $mol_view.auto();
    }
  }

  function replaceMolHref(href) {
    if (typeof $mol_state_arg !== "undefined") {
      $mol_state_arg.href( href ); // mol hack
    }
  }

  function cleanPrefetchCache() {
    var urls = Object.keys(prefetchCache);
    for (var i = 0; i < urls.length; i++) {
      var key = urls[i];
      if (now() - prefetchCache[key].time > PREFETCH_TTL) {
        delete prefetchCache[key];
      }
    }
  }

  function getPage(url) {
    cleanPrefetchCache();
    if (prefetchCache[url]) {
      return prefetchCache[url].promise;
    }
    var p = fetch(url, { headers: { "X-Turbo": "yes" } }).then(htmlResponse);
    prefetchCache[url] = { promise: p, time: now() };
    return p;
  }

  function htmlResponse(response) {
    return response.text().then(function (html) {
      return {
        html: html,
        turbo: response.headers.get("X-Turbo-Response") == "true"
      };
    });
  }

  function replaceContentFromHtml(data, scrollY) {
    try {
      var wrapper = document.createElement("div");
      wrapper.innerHTML = data.html;

      if (data.turbo) {
        var children = Array.prototype.slice.call(wrapper.children);
        for (var i = 0; i < children.length; i++) {
          var node = children[i];

          if (node.id === "title") {
            document.title = node.innerHTML;
            continue;
          }

          if (!node.id) {
            continue;
          }

          var target = document.getElementById(node.id);
          if (target && target.parentNode) {
            target.parentNode.replaceChild(node, target);
          } else {
            throw new Error("Target not found for turbo id: " + node.id);
          }
        }
      } else {
        var parser = new DOMParser();
        var newDoc = parser.parseFromString(data.html, "text/html");

        var newContent = newDoc.querySelector(contentSelector);
        if (!newContent) throw new Error("Missing content");

        var oldContent = document.querySelector(contentSelector);
        oldContent.parentNode.replaceChild(newContent, oldContent);
        document.title = newDoc.title;
      }

      remountMol();

      window.scrollTo(0, scrollY || 0);
    } catch (err) {
      console.error("Replace content error:", err);
      return false;
    }
    return true;
  }

  document.body.addEventListener("click", function (e) {
    // toc links should not be handled here
    if (e.target.tagName === "A" && e.target.classList.contains("toc__link")) {
      return; // handled separately
    }

    var link = e.target.closest ? e.target.closest("a") : null;
    if (!link || link.target === "_blank" || link.hasAttribute("download") || link.href.indexOf("mailto:") === 0) return;

    var origin = window.location.origin;
    if (link.href.indexOf(origin) !== 0) return;

    e.preventDefault();
    var url = link.href;

    var currentState = history.state || {};
    currentState.turbolinks = true;
    currentState.scrollY = window.scrollY;
    history.replaceState(currentState, "", window.location.href);

    getPage(url)
      .then(function (data) {
        if (!replaceContentFromHtml(data, 0)) {
          window.location.href = url;
          return;
        }
        history.pushState({ turbolinks: true, url: url, scrollY: 0 }, "", url);
        replaceMolHref(url + (link.hash || ""));
      })
      .catch(function (err) {
        console.error("Navigation error", err);
        window.location.href = url;
      });
  });

  window.addEventListener("popstate", function (event) {
    if (!event.state || !event.state.turbolinks) return;

    var url = event.state.url || window.location.href;

    getPage(url)
      .then(function (data) {
        if (!replaceContentFromHtml(data, event.state.scrollY)) {
          window.location.href = url;
        }
      })
      .catch(function () {
        window.location.href = url;
      });
  });

  function prefetchOnEvent(evt) {
    var link = evt.target.closest ? evt.target.closest("a") : null;
    if (!link || link.target === "_blank" || link.hasAttribute("download") || link.href.indexOf("mailto:") === 0) return;
    if (link.href.indexOf(window.location.origin) !== 0) return;
    if (prefetchCache[link.href]) return;
    getPage(link.href).catch(function () { /* ignore */ });
  }

  document.body.addEventListener("mouseenter", prefetchOnEvent, true);
  document.body.addEventListener("touchstart", prefetchOnEvent, { passive: true, capture: true });

  // toc
  document.addEventListener("click", function (e) {
    if (e.target.tagName !== "A" || !e.target.className.includes("toc__link")) {
      return;
    }

    e.preventDefault();
    e.stopPropagation();

    const id = e.target.getAttribute('href').substring(1); // remove leading '#'

    $mol_state_arg.value('anchor', id);

    const target = document.getElementById(id);
    if (target) {
      target.scrollIntoView({ behavior: "smooth" });
    }
  });
});
