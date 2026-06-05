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
		initSoftSelects(document);
		initDateTimePickers(document);
		bindContentShellLinks(document);
		initSettingsAnchors();
		return true;
	}

	function replacePostListContent(doc) {
		var current = document.querySelector("[data-admin-post-list]");
		var next = doc.querySelector("[data-admin-post-list]");
		if (!current || !next) return false;
		var currentTable = current.querySelector("[data-admin-post-table]");
		var nextTable = next.querySelector("[data-admin-post-table]");
		var currentRows = current.querySelector("[data-admin-post-rows]");
		var nextRows = next.querySelector("[data-admin-post-rows]");
		var currentHead = currentTable ? currentTable.querySelector("thead") : null;
		var nextHead = nextTable ? nextTable.querySelector("thead") : null;
		var sameHead = currentHead && nextHead && currentHead.innerHTML.trim() === nextHead.innerHTML.trim();
		if (sameHead && currentRows && nextRows) {
			currentRows.replaceWith(nextRows);
		} else if (currentTable && nextTable) {
			currentTable.replaceWith(nextTable);
		} else {
			current.replaceWith(next);
		}
		if (doc.title) document.title = doc.title;
		syncPostFilterTabs(doc);
		bindContentShellLinks(document);
		return true;
	}

	function syncPostFilterTabs(doc) {
		var currentTabs = document.querySelector(".post-filter-tabs");
		var nextTabs = doc.querySelector(".post-filter-tabs");
		if (!currentTabs || !nextTabs) return;
		Array.prototype.slice.call(currentTabs.querySelectorAll("a")).forEach(function (link) {
			link.classList.remove("is-active");
			link.removeAttribute("aria-current");
		});
		Array.prototype.slice.call(nextTabs.querySelectorAll("a")).forEach(function (nextLink) {
			if (!nextLink.classList.contains("is-active")) return;
			var nextUrl = new URL(nextLink.href, window.location.origin);
			var match = Array.prototype.slice.call(currentTabs.querySelectorAll("a")).find(function (link) {
				var currentUrl = new URL(link.href, window.location.origin);
				return currentUrl.pathname === nextUrl.pathname && currentUrl.search === nextUrl.search;
			});
			if (match) {
				match.classList.add("is-active");
				match.setAttribute("aria-current", "page");
			}
		});
	}

	function isShellLink(link) {
		if (!link) return false;
		var url = new URL(link.href, window.location.origin);
		return url.origin === window.location.origin && url.pathname.indexOf("/admin") === 0 && url.pathname !== "/admin/login";
	}

	function sameUrlAsCurrent(url) {
		var target = new URL(url, window.location.origin);
		return target.origin === window.location.origin &&
			target.pathname.replace(/\/$/, "") === (window.location.pathname.replace(/\/$/, "") || "/") &&
			target.search === window.location.search &&
			target.hash === window.location.hash;
	}

	function isCurrentShellLink(link) {
		if (!link) return false;
		if (links.indexOf(link) !== -1 && currentLink() === link) return true;
		return sameUrlAsCurrent(link.href);
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
		var rows = region.querySelector("[data-admin-post-rows]");
		var loadingTarget = rows || region;
		loadingTarget.classList.add("is-loading");
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
						var nextRows = nextRegion.querySelector("[data-admin-post-rows]");
						var nextLoadingTarget = nextRows || nextRegion;
						nextLoadingTarget.classList.add("is-loading");
						window.requestAnimationFrame(function () {
							window.requestAnimationFrame(function () {
								nextLoadingTarget.classList.remove("is-loading");
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
					var latestRows = latestRegion.querySelector("[data-admin-post-rows]");
					(latestRows || latestRegion).classList.remove("is-loading");
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
			if (sameUrlAsCurrent(link.href)) return;
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
			if (isCurrentShellLink(link)) return;
			loadAdminPage(link.href);
		}, true);
	}

	function bindContentShellLinks(root) {
		root = root || document;
		Array.prototype.slice.call(root.querySelectorAll(".post-filter-tabs a[href^='/admin']")).forEach(bindPostFilterClick);
		Array.prototype.slice.call(root.querySelectorAll(".comment-filter-tabs a[href^='/admin']")).forEach(bindShellClick);
	}

	function initSoftSelects(root) {
		root = root || document;
		Array.prototype.slice.call(root.querySelectorAll(".comment-bulk-bar select[name='bulk_action'], .media-bulk-bar select[name='bulk_action']")).forEach(function (select) {
			if (select.dataset.softSelectBound === "1") return;
			select.dataset.softSelectBound = "1";
			select.classList.add("admin-native-select");

			var wrapper = document.createElement("div");
			wrapper.className = "admin-soft-select";
			var trigger = document.createElement("button");
			trigger.type = "button";
			trigger.className = "admin-soft-select-trigger";
			trigger.setAttribute("aria-haspopup", "listbox");
			trigger.setAttribute("aria-expanded", "false");
			var triggerLabel = document.createElement("span");
			trigger.appendChild(triggerLabel);
			var menu = document.createElement("div");
			menu.className = "admin-soft-select-menu";
			menu.setAttribute("role", "listbox");

			function selectedOption() {
				return select.options[select.selectedIndex] || select.options[0];
			}

			function setOpen(open) {
				wrapper.classList.toggle("is-open", open);
				trigger.setAttribute("aria-expanded", open ? "true" : "false");
			}

			function update() {
				var option = selectedOption();
				triggerLabel.textContent = option ? option.textContent : "";
				Array.prototype.slice.call(menu.querySelectorAll("[data-soft-select-value]")).forEach(function (item) {
					item.setAttribute("aria-selected", item.dataset.softSelectValue === select.value ? "true" : "false");
				});
			}

			Array.prototype.slice.call(select.options).forEach(function (option) {
				var item = document.createElement("button");
				item.type = "button";
				item.className = "admin-soft-select-option";
				item.dataset.softSelectValue = option.value;
				item.setAttribute("role", "option");
				item.textContent = option.textContent;
				item.addEventListener("click", function () {
					select.value = option.value;
					select.dispatchEvent(new Event("change", { bubbles: true }));
					update();
					setOpen(false);
					trigger.focus();
				});
				menu.appendChild(item);
			});

			trigger.addEventListener("click", function (event) {
				event.stopPropagation();
				setOpen(!wrapper.classList.contains("is-open"));
			});

			wrapper.addEventListener("keydown", function (event) {
				if (event.key === "Escape") {
					setOpen(false);
					trigger.focus();
				}
				if (event.key === "ArrowDown") {
					event.preventDefault();
					setOpen(true);
					var active = menu.querySelector("[aria-selected='true']") || menu.querySelector(".admin-soft-select-option");
					if (active) active.focus();
				}
			});

			document.addEventListener("click", function () {
				setOpen(false);
			});

			select.addEventListener("change", update);
			wrapper.appendChild(trigger);
			wrapper.appendChild(menu);
			select.insertAdjacentElement("afterend", wrapper);
			update();
		});
	}

	function initDateTimePickers(root) {
		root = root || document;
		Array.prototype.slice.call(root.querySelectorAll("input[type='datetime-local'][name='published_at']")).forEach(function (input) {
			if (input.dataset.dateTimePickerBound === "1") return;
			input.dataset.dateTimePickerBound = "1";
			input.classList.add("admin-native-datetime");

			function pad(number) {
				return String(number).padStart(2, "0");
			}

			function parseValue(value) {
				var match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/.exec(value || "");
				if (!match) {
					var now = new Date();
					return new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours(), now.getMinutes());
				}
				return new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]), Number(match[4]), Number(match[5]));
			}

			function toInputValue(date) {
				return date.getFullYear() + "-" + pad(date.getMonth() + 1) + "-" + pad(date.getDate()) + "T" + pad(date.getHours()) + ":" + pad(date.getMinutes());
			}

			function toDisplayValue(date) {
				return date.getFullYear() + "/" + pad(date.getMonth() + 1) + "/" + pad(date.getDate()) + " " + pad(date.getHours()) + ":" + pad(date.getMinutes());
			}

			var selected = parseValue(input.value);
			var viewYear = selected.getFullYear();
			var viewMonth = selected.getMonth();
			var wrapper = document.createElement("div");
			wrapper.className = "admin-datetime-picker";

			var trigger = document.createElement("button");
			trigger.type = "button";
			trigger.className = "admin-datetime-trigger";
			trigger.setAttribute("aria-haspopup", "dialog");
			trigger.setAttribute("aria-expanded", "false");
			var triggerText = document.createElement("span");
			trigger.appendChild(triggerText);

			var popover = document.createElement("div");
			popover.className = "admin-datetime-popover";
			popover.setAttribute("role", "dialog");
			popover.setAttribute("aria-label", "Choose publish time");

			var calendar = document.createElement("div");
			calendar.className = "admin-datetime-calendar";
			var timePanel = document.createElement("div");
			timePanel.className = "admin-datetime-time";
			popover.appendChild(calendar);
			popover.appendChild(timePanel);
			wrapper.appendChild(trigger);
			wrapper.appendChild(popover);
			input.insertAdjacentElement("afterend", wrapper);

			function commit() {
				input.value = toInputValue(selected);
				input.dispatchEvent(new Event("input", { bubbles: true }));
				input.dispatchEvent(new Event("change", { bubbles: true }));
				triggerText.textContent = toDisplayValue(selected);
			}

			function setOpen(open) {
				wrapper.classList.toggle("is-open", open);
				trigger.setAttribute("aria-expanded", open ? "true" : "false");
				if (open) render();
			}

			function renderCalendar() {
				calendar.innerHTML = "";
				var header = document.createElement("div");
				header.className = "admin-datetime-header";
				var title = document.createElement("strong");
				title.textContent = viewYear + "/" + pad(viewMonth + 1);
				var controls = document.createElement("div");
				var prev = document.createElement("button");
				var next = document.createElement("button");
				prev.type = "button";
				next.type = "button";
				prev.className = "admin-datetime-nav";
				next.className = "admin-datetime-nav";
				prev.setAttribute("aria-label", "Previous month");
				next.setAttribute("aria-label", "Next month");
				prev.textContent = "<";
				next.textContent = ">";
				prev.addEventListener("click", function () {
					viewMonth -= 1;
					if (viewMonth < 0) {
						viewMonth = 11;
						viewYear -= 1;
					}
					renderCalendar();
				});
				next.addEventListener("click", function () {
					viewMonth += 1;
					if (viewMonth > 11) {
						viewMonth = 0;
						viewYear += 1;
					}
					renderCalendar();
				});
				controls.appendChild(prev);
				controls.appendChild(next);
				header.appendChild(title);
				header.appendChild(controls);
				calendar.appendChild(header);

				var weekdays = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
				var weekRow = document.createElement("div");
				weekRow.className = "admin-datetime-weekdays";
				weekdays.forEach(function (day) {
					var item = document.createElement("span");
					item.textContent = day;
					weekRow.appendChild(item);
				});
				calendar.appendChild(weekRow);

				var grid = document.createElement("div");
				grid.className = "admin-datetime-days";
				var first = new Date(viewYear, viewMonth, 1);
				var start = new Date(viewYear, viewMonth, 1 - first.getDay());
				var today = new Date();
				for (var index = 0; index < 42; index += 1) {
					var date = new Date(start.getFullYear(), start.getMonth(), start.getDate() + index);
					var button = document.createElement("button");
					button.type = "button";
					button.textContent = date.getDate();
					if (date.getMonth() !== viewMonth) button.classList.add("is-muted");
					if (date.toDateString() === today.toDateString()) button.classList.add("is-today");
					if (date.toDateString() === selected.toDateString()) button.classList.add("is-selected");
					button.addEventListener("click", function (picked) {
						return function () {
							selected = new Date(picked.getFullYear(), picked.getMonth(), picked.getDate(), selected.getHours(), selected.getMinutes());
							viewYear = selected.getFullYear();
							viewMonth = selected.getMonth();
							commit();
							render();
						};
					}(date));
					grid.appendChild(button);
				}
				calendar.appendChild(grid);

				var footer = document.createElement("div");
				footer.className = "admin-datetime-footer";
				var todayButton = document.createElement("button");
				todayButton.type = "button";
				todayButton.textContent = "Today";
				todayButton.addEventListener("click", function () {
					var now = new Date();
					selected = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours(), now.getMinutes());
					viewYear = selected.getFullYear();
					viewMonth = selected.getMonth();
					commit();
					render();
				});
				var doneButton = document.createElement("button");
				doneButton.type = "button";
				doneButton.textContent = "Done";
				doneButton.addEventListener("click", function () {
					setOpen(false);
					trigger.focus();
				});
				footer.appendChild(todayButton);
				footer.appendChild(doneButton);
				calendar.appendChild(footer);
			}

			function renderTimeColumn(label, max, current, setter) {
				var column = document.createElement("div");
				column.className = "admin-datetime-time-column";
				var heading = document.createElement("span");
				heading.textContent = label;
				column.appendChild(heading);
				var list = document.createElement("div");
				list.className = "admin-datetime-time-list";
				for (var value = 0; value <= max; value += 1) {
					var button = document.createElement("button");
					button.type = "button";
					button.textContent = pad(value);
					if (value === current) button.classList.add("is-selected");
					button.addEventListener("click", function (picked) {
						return function () {
							setter(picked);
							commit();
							render();
						};
					}(value));
					list.appendChild(button);
				}
				column.appendChild(list);
				return column;
			}

			function renderTime() {
				timePanel.innerHTML = "";
				timePanel.appendChild(renderTimeColumn("Hour", 23, selected.getHours(), function (value) {
					selected.setHours(value);
				}));
				timePanel.appendChild(renderTimeColumn("Minute", 59, selected.getMinutes(), function (value) {
					selected.setMinutes(value);
				}));
				window.requestAnimationFrame(function () {
					Array.prototype.slice.call(timePanel.querySelectorAll(".admin-datetime-time-column")).forEach(function (column) {
						var active = column.querySelector(".is-selected");
						if (active) column.querySelector(".admin-datetime-time-list").scrollTop = Math.max(0, active.offsetTop - 70);
					});
				});
			}

			function render() {
				renderCalendar();
				renderTime();
			}

			trigger.addEventListener("click", function (event) {
				event.stopPropagation();
				setOpen(!wrapper.classList.contains("is-open"));
			});
			popover.addEventListener("click", function (event) {
				event.stopPropagation();
			});
			document.addEventListener("click", function () {
				setOpen(false);
			});
			document.addEventListener("keydown", function (event) {
				if (event.key === "Escape") setOpen(false);
			});
			input.addEventListener("change", function () {
				selected = parseValue(input.value);
				viewYear = selected.getFullYear();
				viewMonth = selected.getMonth();
				commit();
			});
			commit();
		});
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
		var dropdown = menu.querySelector(".admin-user-dropdown");
		var dropdownIndicator = null;
		var dropdownItems = dropdown ? Array.prototype.slice.call(dropdown.querySelectorAll("a, button")) : [];

		if (dropdown && !dropdown.querySelector(".admin-user-dropdown-indicator")) {
			dropdownIndicator = document.createElement("span");
			dropdownIndicator.className = "admin-user-dropdown-indicator";
			dropdownIndicator.setAttribute("aria-hidden", "true");
			dropdown.insertBefore(dropdownIndicator, dropdown.firstChild);
		} else if (dropdown) {
			dropdownIndicator = dropdown.querySelector(".admin-user-dropdown-indicator");
		}

		function moveDropdownIndicator(item, instant) {
			if (!dropdown || !dropdownIndicator || !item) return;
			if (instant) dropdown.classList.add("is-positioning");
			dropdownIndicator.style.width = item.offsetWidth + "px";
			dropdownIndicator.style.height = item.offsetHeight + "px";
			dropdownIndicator.style.setProperty("--dropdown-x", item.offsetLeft + "px");
			dropdownIndicator.style.setProperty("--dropdown-y", item.offsetTop + "px");
			dropdown.classList.add("is-ready");
			if (instant) {
				window.requestAnimationFrame(function () {
					dropdown.classList.remove("is-positioning");
				});
			}
		}

		function setOpen(open) {
			menu.classList.toggle("is-open", open);
			trigger.setAttribute("aria-expanded", open ? "true" : "false");
			if (!open && dropdown) dropdown.classList.remove("is-ready");
		}

		trigger.addEventListener("click", function (event) {
			event.stopPropagation();
			setOpen(!menu.classList.contains("is-open"));
		});

		menu.addEventListener("click", function (event) {
			event.stopPropagation();
		});

		dropdownItems.forEach(function (item) {
			item.addEventListener("mouseenter", function () {
				moveDropdownIndicator(item);
			});
			item.addEventListener("focus", function () {
				moveDropdownIndicator(item);
			});
		});

		if (dropdown) {
			dropdown.addEventListener("mouseleave", function () {
				dropdown.classList.remove("is-ready");
			});
		}

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
	initSoftSelects();
	initDateTimePickers();
})();
