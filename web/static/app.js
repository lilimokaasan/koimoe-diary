(function () {
	var searchButtons = document.querySelectorAll(".js-toggle-search");
	var search = document.querySelector(".js-search");
	var close = document.querySelector(".search_close");
	var searchInput = search && search.querySelector('input[name="q"]');
	var liveSearch = search && search.querySelector("[data-live-search]");
	var liveSearchStatus = search && search.querySelector("[data-live-search-status]");
	var liveSearchResults = search && search.querySelector("[data-live-search-results]");
	var searchIndexPromise;
	var searchIndex;

	searchButtons.forEach(function (button) {
		button.addEventListener("click", function () {
			if (!search) {
				return;
			}
			search.classList.add("is-visible");
			window.setTimeout(function () {
				searchInput && searchInput.focus();
			}, 80);
			prepareLiveSearch();
		});
	});

	close && close.addEventListener("click", function () {
		search && search.classList.remove("is-visible");
	});

	var topButton = document.querySelector("#moblieGoTop, .cd-top");
	topButton && topButton.addEventListener("click", function () {
		window.scrollTo({ top: 0, behavior: "smooth" });
	});

	function prepareLiveSearch() {
		if (!liveSearch || !liveSearchStatus || !liveSearchResults || searchIndexPromise) {
			return;
		}
		liveSearch.hidden = false;
		liveSearchStatus.textContent = "Loading KoiMoe search index...";
		searchIndexPromise = fetch("/api/search-index", {
			headers: { Accept: "application/json" }
		})
			.then(function (response) {
				if (!response.ok) {
					throw new Error("search index failed");
				}
				return response.json();
			})
			.then(function (payload) {
				searchIndex = payload || {};
				renderLiveSearch(searchInput ? searchInput.value : "");
			})
			.catch(function () {
				liveSearchStatus.textContent = "Search index is resting. Press Enter for full search.";
			});
	}

	function normalizeSearchText(value) {
		return String(value || "").toLowerCase().trim();
	}

	function escapeHTML(value) {
		return String(value || "")
			.replace(/&/g, "&amp;")
			.replace(/</g, "&lt;")
			.replace(/>/g, "&gt;")
			.replace(/"/g, "&quot;")
			.replace(/'/g, "&#39;");
	}

	function includesQuery(parts, query) {
		return parts.some(function (part) {
			return normalizeSearchText(part).indexOf(query) !== -1;
		});
	}

	function renderLiveSearch(query) {
		if (!liveSearch || !liveSearchStatus || !liveSearchResults) {
			return;
		}
		var normalized = normalizeSearchText(query);
		liveSearch.hidden = false;
		liveSearchResults.innerHTML = "";
		if (normalized.length < 2) {
			liveSearchStatus.textContent = "Type at least 2 characters to search KoiMoe Diary.";
			return;
		}
		if (!searchIndex) {
			liveSearchStatus.textContent = "Loading KoiMoe search index...";
			prepareLiveSearch();
			return;
		}

		var posts = (searchIndex.posts || [])
			.filter(function (post) {
				return includesQuery([
					post.title,
					post.excerpt,
					post.content,
					post.category,
					(post.tags || []).join(" ")
				], normalized);
			})
			.slice(0, 8);
		var categories = filterTaxonomy(searchIndex.categories, normalized).slice(0, 4);
		var tags = filterTaxonomy(searchIndex.tags, normalized).slice(0, 6);

		if (!posts.length && !categories.length && !tags.length) {
			liveSearchStatus.textContent = "No matching fragments yet. Press Enter for full search.";
			return;
		}

		liveSearchStatus.textContent = "";
		liveSearchResults.innerHTML = [
			renderSearchGroup("Posts", posts, renderPostResult),
			renderSearchGroup("Categories", categories, renderTaxonomyResult),
			renderSearchGroup("Tags", tags, renderTaxonomyResult)
		].join("");
	}

	function filterTaxonomy(items, query) {
		return (items || []).filter(function (item) {
			return includesQuery([item.name], query);
		});
	}

	function renderSearchGroup(title, items, renderer) {
		if (!items.length) {
			return "";
		}
		return '<section class="koimoe-live-search__group">' +
			'<h2 class="koimoe-live-search__title">' + escapeHTML(title) + "</h2>" +
			'<div class="koimoe-live-search__list">' + items.map(renderer).join("") + "</div>" +
			"</section>";
	}

	function renderPostResult(post) {
		var meta = [
			post.category || "Diary",
			typeof post.views === "number" ? post.views + " views" : "",
			typeof post.comment_count === "number" ? post.comment_count + " comments" : ""
		].filter(Boolean).join(" · ");
		return '<a class="koimoe-live-search__item" href="' + escapeHTML(post.url || "#") + '">' +
			'<span class="koimoe-live-search__item-title">' + escapeHTML(post.title || "Untitled") + "</span>" +
			'<span class="koimoe-live-search__meta">' + escapeHTML(meta) + "</span>" +
			'<span class="koimoe-live-search__excerpt">' + escapeHTML(post.excerpt || post.content || "") + "</span>" +
			"</a>";
	}

	function renderTaxonomyResult(item) {
		var count = typeof item.post_count === "number" ? item.post_count + " posts" : "Archive";
		return '<a class="koimoe-live-search__item" href="' + escapeHTML(item.url || "#") + '">' +
			'<span class="koimoe-live-search__item-title">' + escapeHTML(item.name || "Archive") + "</span>" +
			'<span class="koimoe-live-search__meta">' + escapeHTML(count) + "</span>" +
			"</a>";
	}

	searchInput && searchInput.addEventListener("input", function () {
		renderLiveSearch(searchInput.value);
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
})();
