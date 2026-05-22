(function () {
	var searchButtons = document.querySelectorAll(".js-toggle-search");
	var search = document.querySelector(".js-search");
	var close = document.querySelector(".search_close");
	var searchInput = search && search.querySelector('input[name="q"]');
	var liveSearchPanel = search && search.querySelector("[data-live-search-results]");
	var userEntry = document.querySelector(".user-entry");
	var userToggle = userEntry && userEntry.querySelector(".js-toggle-user-menu");
	var progressBar = document.querySelector("#bar, .scrollbar-progress");
	var progressFrame;
	var liveSearchIndex = null;
	var liveSearchLoading = null;
	var liveSearchTimer = null;

	function updateReadingProgress() {
		if (!progressBar) {
			return;
		}
		var doc = document.documentElement;
		var scrollable = Math.max(1, doc.scrollHeight - window.innerHeight);
		var progress = Math.min(1, Math.max(0, (window.scrollY || doc.scrollTop || 0) / scrollable));
		progressBar.style.width = "100%";
		progressBar.style.background = "linear-gradient(90deg, #fb98c0, #ffd7e7)";
		progressBar.style.transform = "scaleX(" + progress + ")";
	}

	function requestReadingProgressUpdate() {
		if (progressFrame) {
			return;
		}
		progressFrame = window.requestAnimationFrame(function () {
			progressFrame = null;
			updateReadingProgress();
		});
	}

	updateReadingProgress();
	window.addEventListener("scroll", requestReadingProgressUpdate, { passive: true });
	window.addEventListener("resize", requestReadingProgressUpdate);

	if (document.body.classList.contains("sakura-effects-on") && !window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
		var petalLayer = document.createElement("div");
		petalLayer.className = "sakura-petal-layer";
		petalLayer.setAttribute("aria-hidden", "true");
		document.body.appendChild(petalLayer);
		window.setInterval(function () {
			if (document.hidden || petalLayer.childElementCount > 18) {
				return;
			}
			var petal = document.createElement("span");
			petal.className = "sakura-petal";
			petal.style.setProperty("--petal-left", Math.round(Math.random() * 100) + "vw");
			petal.style.setProperty("--petal-size", Math.round(7 + Math.random() * 8) + "px");
			petal.style.setProperty("--petal-drift", Math.round(-70 + Math.random() * 140) + "px");
			petal.style.setProperty("--petal-rotate", Math.round(Math.random() * 180) + "deg");
			petal.style.setProperty("--petal-duration", Math.round(9 + Math.random() * 8) + "s");
			petal.addEventListener("animationend", function () {
				petal.remove();
			});
			petalLayer.appendChild(petal);
		}, 900);
	}

	userToggle && userToggle.addEventListener("click", function (event) {
		event.preventDefault();
		event.stopPropagation();
		var isOpen = userEntry.classList.toggle("is-open");
		userToggle.setAttribute("aria-expanded", isOpen ? "true" : "false");
	});

	document.addEventListener("click", function (event) {
		if (!userEntry || !userEntry.classList.contains("is-open") || userEntry.contains(event.target)) {
			return;
		}
		closeUserEntry();
	});

	document.addEventListener("keydown", function (event) {
		if (event.key === "Escape") {
			closeUserEntry();
			closeSearch();
		}
	});

	function closeUserEntry() {
		if (!userEntry) {
			return;
		}
		userEntry.classList.remove("is-open");
		userToggle && userToggle.setAttribute("aria-expanded", "false");
	}

	searchButtons.forEach(function (button) {
		button.addEventListener("click", function () {
			if (!search) {
				return;
			}
			closeUserEntry();
			search.classList.add("is-visible");
			document.body.classList.add("search-open");
			renderLiveSearch((searchInput && searchInput.value) || "");
			loadLiveSearchIndex();
			window.setTimeout(function () {
				searchInput && searchInput.focus();
			}, 80);
		});
	});

	close && close.addEventListener("click", function () {
		closeSearch();
	});

	search && search.addEventListener("submit", function () {
		if (searchInput) {
			searchInput.value = searchInput.value.trim();
		}
	});

	searchInput && searchInput.addEventListener("input", function () {
		window.clearTimeout(liveSearchTimer);
		liveSearchTimer = window.setTimeout(function () {
			renderLiveSearch(searchInput.value || "");
			loadLiveSearchIndex();
		}, 80);
	});

	function closeSearch() {
		if (!search) {
			return;
		}
		search.classList.remove("is-visible");
		document.body.classList.remove("search-open");
	}

	function loadLiveSearchIndex() {
		if (liveSearchIndex || liveSearchLoading || !liveSearchPanel) {
			return liveSearchLoading || Promise.resolve(liveSearchIndex);
		}
		try {
			var cached = window.sessionStorage && window.sessionStorage.getItem("koimoe_live_search_index");
			if (cached) {
				var parsed = JSON.parse(cached);
				if (parsed && parsed.cached_at && Date.now() - parsed.cached_at < 5 * 60 * 1000) {
					liveSearchIndex = parsed.index;
					renderLiveSearch((searchInput && searchInput.value) || "");
					return Promise.resolve(liveSearchIndex);
				}
			}
		} catch (error) {
			// Search still works without sessionStorage.
		}
		liveSearchLoading = fetch("/api/search-index", { headers: { Accept: "application/json" } })
			.then(function (response) {
				if (!response.ok) {
					throw new Error("search index failed");
				}
				return response.json();
			})
			.then(function (payload) {
				liveSearchIndex = payload || {};
				try {
					if (window.sessionStorage) {
						window.sessionStorage.setItem("koimoe_live_search_index", JSON.stringify({
							cached_at: Date.now(),
							index: liveSearchIndex
						}));
					}
				} catch (error) {
					// Cache failure should not block live search.
				}
				renderLiveSearch((searchInput && searchInput.value) || "");
				return liveSearchIndex;
			})
			.catch(function () {
				if (liveSearchPanel) {
					liveSearchPanel.innerHTML = '<p class="live-search-hint">Live search is resting for a moment. Press Enter for the full search page.</p>';
				}
			})
			.finally(function () {
				liveSearchLoading = null;
			});
		return liveSearchLoading;
	}

	function renderLiveSearch(rawQuery) {
		if (!liveSearchPanel) {
			return;
		}
		var query = normalizeLiveSearch(rawQuery);
		if (!query) {
			liveSearchPanel.innerHTML = '<p class="live-search-hint">Start typing to search posts, pages, categories, and tags.</p>';
			return;
		}
		if (!liveSearchIndex) {
			liveSearchPanel.innerHTML = '<p class="live-search-hint">Gathering tiny fragments...</p>';
			return;
		}
		var groups = [
			{ label: "Posts", icon: "file", items: matchPostItems(liveSearchIndex.posts || [], query).slice(0, 5) },
			{ label: "Pages", icon: "bookmark", items: matchPostItems(liveSearchIndex.pages || [], query).slice(0, 4) },
			{ label: "Categories", icon: "folder-o", items: matchTaxonomyItems(liveSearchIndex.categories || [], query).slice(0, 4) },
			{ label: "Tags", icon: "tag", items: matchTaxonomyItems(liveSearchIndex.tags || [], query).slice(0, 6) }
		];
		var html = "";
		var total = 0;
		groups.forEach(function (group) {
			if (!group.items.length) {
				return;
			}
			total += group.items.length;
			html += '<section class="live-search-section"><h3>' + escapeHTML(group.label) + '</h3>';
			group.items.forEach(function (item) {
				html += liveSearchItemHTML(item, query, group.icon);
			});
			html += '</section>';
		});
		if (!total) {
			liveSearchPanel.innerHTML = '<p class="live-search-hint">No tiny fragment matched yet. Press Enter to search deeper.</p>';
			return;
		}
		liveSearchPanel.innerHTML = html;
	}

	function matchPostItems(items, query) {
		return items.map(function (item) {
			var haystack = normalizeLiveSearch([
				item.title,
				item.excerpt,
				item.content,
				item.category,
				(item.tags || []).join(" ")
			].join(" "));
			return Object.assign({ _score: scoreLiveSearch(item.title, haystack, query) }, item);
		}).filter(function (item) {
			return item._score > 0;
		}).sort(function (a, b) {
			return b._score - a._score;
		});
	}

	function matchTaxonomyItems(items, query) {
		return items.map(function (item) {
			var haystack = normalizeLiveSearch(item.name || "");
			return Object.assign({ _score: scoreLiveSearch(item.name, haystack, query) }, item);
		}).filter(function (item) {
			return item._score > 0;
		}).sort(function (a, b) {
			return b._score - a._score;
		});
	}

	function scoreLiveSearch(title, haystack, query) {
		var normalizedTitle = normalizeLiveSearch(title || "");
		if (normalizedTitle === query) return 100;
		if (normalizedTitle.indexOf(query) === 0) return 80;
		if (normalizedTitle.indexOf(query) >= 0) return 60;
		if (haystack.indexOf(query) >= 0) return 30;
		var parts = query.split(/\s+/).filter(Boolean);
		if (parts.length > 1 && parts.every(function (part) { return haystack.indexOf(part) >= 0; })) {
			return 20;
		}
		return 0;
	}

	function liveSearchItemHTML(item, query, icon) {
		var title = item.title || item.name || "Untitled";
		var excerpt = item.excerpt || item.content || (typeof item.post_count === "number" ? item.post_count + " posts" : "");
		var cover = (item.cover_image || "").trim();
		if (excerpt.length > 112) {
			excerpt = excerpt.slice(0, 112).replace(/\s+\S*$/, "") + "...";
		}
		var visual = cover ?
			'<span class="live-search-icon live-search-thumb"><img src="' + escapeHTML(cover) + '" alt=""></span>' :
			'<span class="live-search-icon"><i class="fa fa-' + escapeHTML(icon) + '" aria-hidden="true"></i></span>';
		return '<a class="live-search-item" href="' + escapeHTML(item.url || "/search") + '">' +
			visual +
			'<span><strong>' + highlightSearchText(title, query) + '</strong>' +
			(excerpt ? '<em>' + highlightSearchText(excerpt, query) + '</em>' : '') +
			'</span></a>';
	}

	function highlightSearchText(value, query) {
		var safe = escapeHTML(value || "");
		var trimmed = (query || "").trim();
		if (!trimmed) return safe;
		var first = trimmed.split(/\s+/)[0];
		if (!first || first.length < 2) return safe;
		var escaped = first.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
		return safe.replace(new RegExp("(" + escaped + ")", "ig"), "<mark>$1</mark>");
	}

	function normalizeLiveSearch(value) {
		return (value || "").toString().toLowerCase().replace(/&nbsp;/g, " ").replace(/\s+/g, " ").trim();
	}

	function escapeHTML(value) {
		return (value || "").toString().replace(/[&<>"']/g, function (char) {
			return {
				"&": "&amp;",
				"<": "&lt;",
				">": "&gt;",
				'"': "&quot;",
				"'": "&#39;"
			}[char];
		});
	}

	var topButton = document.querySelector("#moblieGoTop, .cd-top");
	topButton && topButton.addEventListener("click", function () {
		window.scrollTo({ top: 0, behavior: "smooth" });
	});


	document.querySelectorAll(".post-like-button").forEach(function (button) {
		var postID = button.getAttribute("data-post-id");
		var likeKey = button.getAttribute("data-like-key");
		var count = button.querySelector("strong");
		try {
			if (likeKey && window.localStorage.getItem(likeKey)) {
				button.classList.add("is-liked");
				button.setAttribute("aria-pressed", "true");
			}
		} catch (error) {
			// Local storage can be unavailable in strict privacy modes.
		}
		button.addEventListener("click", function () {
			if (!postID || button.classList.contains("is-loading") || button.classList.contains("is-liked")) {
				return;
			}
			button.classList.add("is-loading");
			fetch("/api/posts/" + encodeURIComponent(postID) + "/like", {
				method: "POST",
				headers: { Accept: "application/json" }
			})
				.then(function (response) {
					if (!response.ok) {
						throw new Error("like failed");
					}
					return response.json();
				})
				.then(function (payload) {
					if (count && typeof payload.likes === "number") {
						count.textContent = payload.likes;
					}
					button.classList.add("is-liked");
					button.setAttribute("aria-pressed", "true");
					try {
						if (likeKey) {
							window.localStorage.setItem(likeKey, "1");
						}
					} catch (error) {
						// Keep the UI response even when persistence is unavailable.
					}
				})
				.catch(function () {
					button.classList.add("has-error");
					window.setTimeout(function () {
						button.classList.remove("has-error");
					}, 1200);
				})
				.finally(function () {
					button.classList.remove("is-loading");
				});
		});
	});

	var commentForm = document.querySelector(".comment-form");
	if (commentForm) {
		var parentInput = commentForm.querySelector('input[name="parent_id"]');
		var replyContext = commentForm.querySelector(".comment-reply-context");
		var replyName = replyContext && replyContext.querySelector("strong");
		var cancelReply = commentForm.querySelector(".comment-cancel-reply");
		var commentTextarea = commentForm.querySelector("textarea");
		document.querySelectorAll(".comment-reply-button").forEach(function (button) {
			button.addEventListener("click", function () {
				var commentID = button.getAttribute("data-comment-id") || "";
				var author = button.getAttribute("data-comment-author") || "this comment";
				if (parentInput) {
					parentInput.value = commentID;
				}
				if (replyContext) {
					replyContext.hidden = false;
				}
				if (replyName) {
					replyName.textContent = author;
				}
				commentForm.classList.add("is-replying");
				commentForm.scrollIntoView({ behavior: "smooth", block: "center" });
				window.setTimeout(function () {
					commentTextarea && commentTextarea.focus();
				}, 280);
			});
		});
		cancelReply && cancelReply.addEventListener("click", function () {
			if (parentInput) {
				parentInput.value = "";
			}
			if (replyContext) {
				replyContext.hidden = true;
			}
			if (replyName) {
				replyName.textContent = "";
			}
			commentForm.classList.remove("is-replying");
		});
	}
})();
