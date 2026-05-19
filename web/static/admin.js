(function () {
	var nav = document.querySelector(".admin-topbar nav");
	if (!nav) return;

	var links = Array.prototype.slice.call(nav.querySelectorAll(":scope > a:not(.button)"));
	if (!links.length) return;

	var indicator = document.createElement("span");
	indicator.className = "admin-nav-indicator";
	indicator.setAttribute("aria-hidden", "true");
	nav.insertBefore(indicator, nav.firstChild);

	function normalizedPath(link) {
		try {
			return new URL(link.getAttribute("href"), window.location.origin).pathname.replace(/\/$/, "") || "/";
		} catch (err) {
			return "";
		}
	}

	function currentLink() {
		var path = window.location.pathname.replace(/\/$/, "") || "/";
		if (path === "/admin/posts/new" || path.indexOf("/admin/posts/") === 0) {
			return links.find(function (link) { return normalizedPath(link) === "/admin"; });
		}
		if (path.indexOf("/admin/categories") === 0 || path.indexOf("/admin/tags") === 0) {
			return links.find(function (link) { return normalizedPath(link) === "/admin/taxonomy"; });
		}
		var best = { link: null, score: -1 };
		links.forEach(function (link) {
			var linkPath = normalizedPath(link);
			var score = -1;
			if (!linkPath || linkPath === "/") return;
			if (path === linkPath) score = linkPath.length + 1000;
			else if (linkPath !== "/admin" && path.indexOf(linkPath + "/") === 0) score = linkPath.length;
			if (score > best.score) best = { link: link, score: score };
		});
		return best.link;
	}

	function moveTo(link) {
		if (!link) return;
		indicator.style.width = link.offsetWidth + "px";
		indicator.style.height = link.offsetHeight + "px";
		indicator.style.setProperty("--nav-x", link.offsetLeft + "px");
		indicator.style.setProperty("--nav-y", link.offsetTop + "px");
		nav.classList.add("is-ready");
	}

	var active = currentLink();
	if (active) {
		active.setAttribute("aria-current", "page");
	}

	window.requestAnimationFrame(function () {
		moveTo(active || links[0]);
	});

	links.forEach(function (link) {
		link.addEventListener("mouseenter", function () { moveTo(link); });
		link.addEventListener("focus", function () { moveTo(link); });
	});

	nav.addEventListener("mouseleave", function () { moveTo(active || links[0]); });
	window.addEventListener("resize", function () { moveTo(document.querySelector(".admin-topbar nav > a[aria-current='page']") || active || links[0]); });
})();
