window.$ = window.jQuery = require('../node_modules/jquery/dist/jquery');
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

	setThemeElements(localStorage.getItem('currentTheme') || 'light');

	$('input[name=theme]').change(function (event) {
		$('.loading').show();

		var themeName = $(this).val();

		if(['light','dark'].indexOf(themeName) === -1)
			themeName = 'light';

		localStorage.setItem('currentTheme', themeName);

		$('#theme-style').attr('href', 'assets/build/' + themeName + '-theme.css');
		setThemeElements(themeName);

		setTimeout(function () {
			$('.loading').hide();
		}, 500);
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

function setThemeElements(themeName) {
	$('.theme-toggle > *').removeClass('active');

	var $activeThemeRadio = $('input[name=theme][value=' + themeName + ']');
	$activeThemeRadio.attr('checked', 'checked');
	$activeThemeRadio.parent().addClass('active');
}