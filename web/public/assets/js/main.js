$(document).ready(function() {
	$("#search-button").click(function(event) {
		var redirectUrl = location.protocol
			+ "//"
			+ location.host
			+ "/blocks/"
			+ $("#search-input").val();

		window.location.replace(redirectUrl);

		event.preventDefault();
	});

	makeShorter();
});

function makeShorter() {
	$(".shortcut").text($(".shortcut").text().substring(0, 12) + "...");
}