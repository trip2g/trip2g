document.addEventListener("DOMContentLoaded", function () {
  var contentSelector = "#all-content";

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

      if (typeof $mol_view !== "undefined") {
        $mol_view.autobind(null);
      }

      window.scrollTo(0, scrollY || 0);
    } catch (err) {
      console.error("Replace content error:", err);
      return false;
    }
    return true;
  }

  function htmlResponse(response) {
    return response.text().then(function (html) {
      return {
        html: html,
        turbo: response.headers.get("X-Turbo-Response") == "true"
      };
    });
  }

  document.body.addEventListener("click", function (e) {
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

    fetch(url, { headers: { "X-Turbo": "yes" } })
      .then(htmlResponse)
      .then(function (data) {
        if (!replaceContentFromHtml(data, 0)) {
          window.location.href = url;
          return;
        }
        history.pushState({ turbolinks: true, url: url, scrollY: 0 }, "", url);
      })
      .catch(function (err) {
        console.error("Navigation error", err);
        window.location.href = url;
      });
  });

  window.addEventListener("popstate", function (event) {
    if (!event.state || !event.state.turbolinks) return;

    var url = event.state.url || window.location.href;

    fetch(url, { headers: { "X-Turbo": "yes" } })
      .then(htmlResponse)
      .then(function (data) {
        if (!replaceContentFromHtml(data, event.state.scrollY)) {
          window.location.href = url;
        }
      })
      .catch(function () {
        window.location.href = url;
      });
  });
});
