(function () {
	var nav = document.querySelector(".admin-topbar nav");
	var indicator = null;
	var links = [];
	var hoveredNavLink = null;
	var focusedNavLink = null;
	var pageLeaveDelay = 420;

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
		if (path === "/admin/pages/new" || path.indexOf("/admin/pages/") === 0) {
			return links.find(function (link) { return normalizedPath(link) === "/admin/pages"; });
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

	function preferredIndicatorLink(active) {
		if (hoveredNavLink && links.indexOf(hoveredNavLink) !== -1) return hoveredNavLink;
		if (focusedNavLink && links.indexOf(focusedNavLink) !== -1) return focusedNavLink;
		return active || links[0];
	}

	function syncNav(instant) {
		if (!nav) return;
		var active = currentLink();
		links.forEach(function (link) {
			if (link === active) link.setAttribute("aria-current", "page");
			else link.removeAttribute("aria-current");
		});
		if (indicator) {
			moveTo(preferredIndicatorLink(active), instant);
		}
	}

	function clearIndicator() {
		if (!nav) return;
		hoveredNavLink = null;
		nav.classList.remove("is-ready");
	}

	function runInlineScripts(root) {
		Array.prototype.slice.call(root.querySelectorAll("script:not([src])")).forEach(function (script) {
			if ((script.textContent || "").indexOf("document.write") !== -1) {
				return;
			}
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
		initMediaPicker();
		initDangerConfirmations();
		initCommentBulkActions();
		initMediaBulkActions();
		bindContentShellLinks(document);
		initSettingsAnchors();
		return true;
	}

	function replacePostListContent(doc) {
		var current = document.querySelector("[data-admin-post-list]");
		var next = doc.querySelector("[data-admin-post-list]");
		if (!current || !next) return false;
		current.replaceWith(next);
		if (doc.title) document.title = doc.title;
		bindContentShellLinks(document);
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
			return new Promise(function (resolve) {
				document.body.classList.add("admin-is-leaving");
				window.setTimeout(function () {
					var doc = new DOMParser().parseFromString(html, "text/html");
					document.body.classList.add("admin-is-entering");
					if (!replaceAdminContent(doc)) {
						window.location.href = url;
						resolve();
						return;
					}
					document.body.classList.remove("admin-is-leaving");
					if (!options.skipHistory) {
						window.history.pushState({ adminShell: true }, "", url);
					}
					var shell = document.querySelector(".admin-shell");
					if (shell) {
						shell.scrollTop = 0;
					}
					syncNav(true);
					window.requestAnimationFrame(function () {
						window.requestAnimationFrame(function () {
							document.body.classList.remove("admin-is-entering");
						});
					});
					resolve();
				}, pageLeaveDelay);
			});
		}).catch(function () {
			window.location.href = url;
		}).finally(function () {
			document.body.classList.remove("admin-is-leaving");
			document.body.classList.remove("admin-is-loading");
		});
	}

	function loadPostFilter(url, options) {
		options = options || {};
		var region = document.querySelector("[data-admin-post-list]");
		if (!region) return loadAdminPage(url, options);
		region.classList.add("is-loading");
		return fetch(url, {
			credentials: "same-origin",
			headers: { "X-Requested-With": "fetch" }
		}).then(function (response) {
			if (!response.ok) throw new Error("HTTP " + response.status);
			return response.text();
		}).then(function (html) {
			return new Promise(function (resolve) {
				window.setTimeout(function () {
					var doc = new DOMParser().parseFromString(html, "text/html");
					if (!replacePostListContent(doc)) {
						loadAdminPage(url, options).then(resolve);
						return;
					}
					if (!options.skipHistory) {
						window.history.pushState({ adminPostList: true }, "", url);
					}
					syncNav(true);
					var nextRegion = document.querySelector("[data-admin-post-list]");
					if (nextRegion) {
						nextRegion.classList.add("is-loading");
						window.requestAnimationFrame(function () {
							window.requestAnimationFrame(function () {
								nextRegion.classList.remove("is-loading");
							});
						});
					}
					resolve();
				}, 180);
			});
		}).catch(function () {
			window.location.href = url;
		}).finally(function () {
			var latestRegion = document.querySelector("[data-admin-post-list]");
			if (latestRegion) {
				window.setTimeout(function () {
					latestRegion.classList.remove("is-loading");
				}, 20);
			}
		});
	}

	function bindPostFilterClick(link) {
		if (!link || link.dataset.adminPostFilterBound === "1") return;
		link.dataset.adminPostFilterBound = "1";
		link.addEventListener("click", function (event) {
			if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button !== 0) return;
			if (!isShellLink(link)) return;
			event.preventDefault();
			event.stopPropagation();
			loadPostFilter(link.href);
		}, true);
	}

	function bindShellClick(link) {
		if (!link || link.dataset.adminShellBound === "1") return;
		link.dataset.adminShellBound = "1";
		link.addEventListener("click", function (event) {
			if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button !== 0) return;
			if (!isShellLink(link)) return;
			event.preventDefault();
			event.stopPropagation();
			loadAdminPage(link.href);
		}, true);
	}

	function bindContentShellLinks(root) {
		root = root || document;
		Array.prototype.slice.call(root.querySelectorAll(".post-filter-tabs a[href^='/admin']")).forEach(bindPostFilterClick);
		Array.prototype.slice.call(root.querySelectorAll(".comment-filter-tabs a[href^='/admin']")).forEach(bindShellClick);
	}

	function scrollAdminShellTo(target, instant) {
		var shell = document.querySelector(".admin-shell");
		if (!shell || !target) return;
		var top = target.getBoundingClientRect().top - shell.getBoundingClientRect().top + shell.scrollTop - 12;
		var reduceMotion = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
		shell.scrollTo({
			top: Math.max(0, top),
			behavior: instant || reduceMotion ? "auto" : "smooth"
		});
	}

	function initSettingsAnchors() {
		var links = Array.prototype.slice.call(document.querySelectorAll(".settings-index a[href^='#settings-']"));
		if (!links.length) return;
		links.forEach(function (link) {
			if (link.dataset.settingsAnchorBound === "1") return;
			link.dataset.settingsAnchorBound = "1";
			link.addEventListener("click", function (event) {
				var hash = link.getAttribute("href");
				var target = hash ? document.querySelector(hash) : null;
				if (!target) return;
				event.preventDefault();
				event.stopPropagation();
				links.forEach(function (item) { item.classList.toggle("is-active", item === link); });
				scrollAdminShellTo(target);
				window.history.replaceState(window.history.state, "", window.location.pathname + window.location.search + hash);
			}, true);
		});
		if (window.location.hash && window.location.hash.indexOf("#settings-") === 0) {
			var active = links.find(function (link) { return link.getAttribute("href") === window.location.hash; });
			var target = document.querySelector(window.location.hash);
			if (active) active.classList.add("is-active");
			if (target) {
				window.requestAnimationFrame(function () {
					scrollAdminShellTo(target, true);
				});
			}
		}
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
			link.addEventListener("mouseenter", function () {
				hoveredNavLink = link;
				moveTo(link);
			});
			link.addEventListener("focus", function () {
				focusedNavLink = link;
				moveTo(link);
			});
			link.addEventListener("blur", function () {
				if (focusedNavLink === link) focusedNavLink = null;
			});
		});

		Array.prototype.slice.call(document.querySelectorAll(".admin-global-actions a[href^='/admin']")).forEach(bindShellClick);
		bindContentShellLinks(document);

		nav.addEventListener("mouseleave", clearIndicator);
		window.addEventListener("resize", function () { syncNav(true); });
		window.addEventListener("popstate", function () {
			if (window.location.pathname.replace(/\/$/, "") === "/admin" && document.querySelector("[data-admin-post-list]")) {
				loadPostFilter(window.location.href, { skipHistory: true });
				return;
			}
			loadAdminPage(window.location.href, { skipHistory: true });
		});
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

		Array.prototype.slice.call(menu.querySelectorAll(".admin-user-dropdown a")).forEach(function (link) {
			link.addEventListener("click", function () {
				setOpen(false);
				link.blur();
			});
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

	function initMediaPicker() {
		Array.prototype.slice.call(document.querySelectorAll(".media-picker-modal")).forEach(function (modal) {
			if (modal.dataset.mediaPickerBound === "1") return;
			modal.dataset.mediaPickerBound = "1";
			var openButtons = Array.prototype.slice.call(document.querySelectorAll("[data-open-media-picker]"));
			var closeButtons = Array.prototype.slice.call(modal.querySelectorAll("[data-close-media-picker]"));

			function setOpen(open) {
				modal.classList.toggle("is-open", open);
				modal.setAttribute("aria-hidden", open ? "false" : "true");
				document.body.classList.toggle("media-picker-open", open);
			}

			openButtons.forEach(function (button) {
				button.addEventListener("click", function () {
					setOpen(true);
				});
			});
			closeButtons.forEach(function (button) {
				button.addEventListener("click", function () {
					setOpen(false);
				});
			});
			modal.addEventListener("click", function (event) {
				if (event.target === modal) setOpen(false);
			});
			modal.addEventListener("click", function (event) {
				if (event.target.closest("[data-use-cover], [data-insert-media]")) {
					setOpen(false);
				}
			});
			document.addEventListener("keydown", function (event) {
				if (event.key === "Escape" && modal.classList.contains("is-open")) {
					setOpen(false);
				}
			});
		});
	}

	function initDangerConfirmations() {
		Array.prototype.slice.call(document.querySelectorAll("[data-confirm-delete]")).forEach(function (form) {
			if (form.dataset.confirmBound === "1") return;
			form.dataset.confirmBound = "1";
			form.addEventListener("submit", function (event) {
				var message = form.getAttribute("data-confirm-delete") || "Delete this item?";
				if (!window.confirm(message)) {
					event.preventDefault();
				}
			});
		});
	}

	function initCommentBulkActions() {
		var form = document.querySelector("#comment-bulk-form");
		if (!form || form.dataset.bulkBound === "1") return;
		form.dataset.bulkBound = "1";
		var selectAll = form.querySelector("[data-comment-select-all]");
		var action = form.querySelector("select[name='bulk_action']");

		function items() {
			return Array.prototype.slice.call(document.querySelectorAll("[data-comment-select-item]"));
		}

		selectAll && selectAll.addEventListener("change", function () {
			items().forEach(function (item) {
				item.checked = selectAll.checked;
			});
		});

		form.addEventListener("submit", function (event) {
			var selected = items().filter(function (item) { return item.checked; });
			if (!action || !action.value || !selected.length) {
				event.preventDefault();
				return;
			}
			if (action.value === "delete" && !window.confirm("Delete selected comments?")) {
				event.preventDefault();
			}
			if (action.value === "spam" && !window.confirm("Move selected comments to spam?")) {
				event.preventDefault();
			}
		});
	}

	function initMediaBulkActions() {
		var form = document.querySelector("#media-bulk-form");
		if (!form || form.dataset.bulkBound === "1") return;
		form.dataset.bulkBound = "1";
		var selectAll = form.querySelector("[data-media-select-all]");
		var action = form.querySelector("select[name='bulk_action']");

		function items() {
			return Array.prototype.slice.call(document.querySelectorAll("[data-media-select-item]"));
		}

		selectAll && selectAll.addEventListener("change", function () {
			items().forEach(function (item) {
				item.checked = selectAll.checked;
			});
		});

		form.addEventListener("submit", function (event) {
			var selected = items().filter(function (item) { return item.checked; });
			if (!action || !action.value || !selected.length) {
				event.preventDefault();
				return;
			}
			if (action.value === "delete" && !window.confirm("Delete selected media assets and their uploaded files?")) {
				event.preventDefault();
			}
		});
	}

	initNav();
	initUserMenu();
	initMediaPicker();
	initDangerConfirmations();
	initCommentBulkActions();
	initMediaBulkActions();
	initSettingsAnchors();
})();
