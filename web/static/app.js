(function () {
	var searchButtons = document.querySelectorAll(".js-toggle-search");
	var search = document.querySelector(".js-search");
	var close = document.querySelector(".search_close");
	var searchInput = search && search.querySelector('input[name="q"]');
	var userEntry = document.querySelector(".user-entry");
	var userToggle = userEntry && userEntry.querySelector(".js-toggle-user-menu");
	var progressBar = document.querySelector("#bar, .scrollbar-progress");
	var progressFrame;

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

	function closeSearch() {
		if (!search) {
			return;
		}
		search.classList.remove("is-visible");
		document.body.classList.remove("search-open");
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
})();
