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
})();
