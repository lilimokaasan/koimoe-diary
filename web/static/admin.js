(function () {
	var nav = document.querySelector(".admin-topbar nav");
	var indicator = null;
	var links = [];

	function normalizedPath(link) {
		try {
			return new URL(link.getAttribute("href"), window.location.origin).pathname.replace(/\/$/, "") || "/";
		} catch (err) {
			return "";
		}
	}

	function currentLink() {
		if (!links.length) return null;
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

	function moveTo(link, instant) {
		if (!nav || !indicator || !link) return;
		if (instant) {
			nav.classList.add("is-positioning");
		}
		indicator.style.width = link.offsetWidth + "px";
		indicator.style.height = link.offsetHeight + "px";
		indicator.style.setProperty("--nav-x", link.offsetLeft + "px");
		indicator.style.setProperty("--nav-y", link.offsetTop + "px");
		nav.classList.add("is-ready");
		if (instant) {
			window.requestAnimationFrame(function () {
				nav.classList.remove("is-positioning");
			});
		}
	}

	function syncNav(instant) {
		if (!nav) return;
		var active = currentLink();
		links.forEach(function (link) {
			if (link === active) link.setAttribute("aria-current", "page");
			else link.removeAttribute("aria-current");
		});
		moveTo(active || links[0], instant);
	}

	function runInlineScripts(root) {
		Array.prototype.slice.call(root.querySelectorAll("script:not([src])")).forEach(function (script) {
			try {
				(new Function(script.textContent || ""))();
			} catch (err) {
				console.error("Admin inline script failed", err);
			}
		});
	}

	function replaceAdminContent(doc) {
		var nextHero = doc.querySelector(".admin-hero");
		var nextShell = doc.querySelector(".admin-shell");
		var hero = document.querySelector(".admin-hero");
		var shell = document.querySelector(".admin-shell");
		if (!nextHero || !nextShell || !hero || !shell) return false;
		hero.replaceWith(nextHero);
		shell.replaceWith(nextShell);
		if (doc.title) document.title = doc.title;
		runInlineScripts(doc);
		return true;
	}

	function isShellLink(link) {
		if (!link) return false;
		var url = new URL(link.href, window.location.origin);
		return url.origin === window.location.origin && url.pathname.indexOf("/admin") === 0 && url.pathname !== "/admin/login";
	}

	function loadAdminPage(url, options) {
		options = options || {};
		document.body.classList.add("admin-is-loading");
		return fetch(url, {
			credentials: "same-origin",
			headers: { "X-Requested-With": "fetch" }
		}).then(function (response) {
			if (!response.ok) throw new Error("HTTP " + response.status);
			return response.text();
		}).then(function (html) {
			var doc = new DOMParser().parseFromString(html, "text/html");
			if (!replaceAdminContent(doc)) {
				window.location.href = url;
				return;
			}
			if (!options.skipHistory) {
				window.history.pushState({ adminShell: true }, "", url);
			}
			window.scrollTo(0, 0);
			syncNav(true);
		}).catch(function () {
			window.location.href = url;
		}).finally(function () {
			document.body.classList.remove("admin-is-loading");
		});
	}

	function bindShellClick(link) {
		link.addEventListener("click", function (event) {
			if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button !== 0) return;
			if (!isShellLink(link)) return;
			event.preventDefault();
			loadAdminPage(link.href);
		});
	}

	function initNav() {
		if (!nav) return;
		links = Array.prototype.slice.call(nav.querySelectorAll(":scope > a:not(.button)"));
		if (!links.length) return;

		indicator = nav.querySelector(".admin-nav-indicator");
		if (!indicator) {
			indicator = document.createElement("span");
			indicator.className = "admin-nav-indicator";
			indicator.setAttribute("aria-hidden", "true");
			nav.insertBefore(indicator, nav.firstChild);
		}

		syncNav(true);
		window.setTimeout(function () { syncNav(true); }, 80);
		window.requestAnimationFrame(function () { syncNav(true); });

		links.forEach(function (link) {
			bindShellClick(link);
			link.addEventListener("mouseenter", function () { moveTo(link); });
			link.addEventListener("focus", function () { moveTo(link); });
		});

		Array.prototype.slice.call(document.querySelectorAll(".admin-global-actions a[href^='/admin']")).forEach(bindShellClick);

		nav.addEventListener("mouseleave", function () { syncNav(false); });
		window.addEventListener("resize", function () { syncNav(true); });
		window.addEventListener("popstate", function () { loadAdminPage(window.location.href, { skipHistory: true }); });
	}

	function initUserMenu() {
		var menu = document.querySelector(".admin-user-menu");
		if (!menu) return;

		var trigger = menu.querySelector(".admin-user-chip");
		if (!trigger) return;

		function setOpen(open) {
			menu.classList.toggle("is-open", open);
			trigger.setAttribute("aria-expanded", open ? "true" : "false");
		}

		trigger.addEventListener("click", function (event) {
			event.stopPropagation();
			setOpen(!menu.classList.contains("is-open"));
		});

		menu.addEventListener("click", function (event) {
			event.stopPropagation();
		});

		document.addEventListener("click", function () {
			setOpen(false);
		});

		document.addEventListener("keydown", function (event) {
			if (event.key === "Escape") {
				setOpen(false);
				trigger.focus();
			}
		});
	}

	initNav();
	initUserMenu();
})();
