(function () {
	var searchButtons = document.querySelectorAll(".js-toggle-search");
	var search = document.querySelector(".js-search");
	var close = document.querySelector(".search_close");

	searchButtons.forEach(function (button) {
		button.addEventListener("click", function () {
			search && search.classList.add("is-visible");
		});
	});

	close && close.addEventListener("click", function () {
		search && search.classList.remove("is-visible");
	});

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
