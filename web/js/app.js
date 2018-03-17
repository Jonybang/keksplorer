require('../node_modules/jquery/jquery');
require('../node_modules/bootstrap/dist/js/bootstrap');
require('../node_modules/popper.js/dist/popper');

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

	convertTimestamp();
});

function convertTimestamp() {
	$(".age").each(function() {
		var now = new Date(+($(this).text())*1000); 
		var now_utc = now.getUTCDate()+'-'+(now.getUTCMonth()+1)+'-'+now.getUTCFullYear() + ' in ' + now.getUTCHours()+':'+now.getUTCMinutes()+':'+now.getUTCSeconds();
		$(this).text(now_utc);
	});
}

