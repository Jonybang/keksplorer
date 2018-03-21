window.d3 = require('../node_modules/d3/build/d3');
window._ = require('../node_modules/lodash');

var DURATION = 1500;
var DELAY = 500;
var THEME = localStorage.getItem('currentTheme') || 'light';

(function () {
	getTransactionsData();
})();

document.body.addEventListener('change-theme', function (e) {
	getTransactionsData();
}, false);

function getTransactionsData() {
	var chartContainerId = 'pendingtx';
	var containerEl = document.getElementById(chartContainerId);
	containerEl.innerHTML = '';
	
	d3.request('/api/transactions', function (error, data) {
		var transactionsByDate =
			_.chain(JSON.parse(data.response))
				.groupBy(function (transaction) {
					var date = new Date();
					date.setTime(parseInt(transaction.timestamp) * 1000);

					var d = date.getDate(),
						m = date.getMonth() + 1,
						y = date.getFullYear(),
						h = date.getHours(),
						mm = date.getMinutes();

					return y + '-' + m + '-' + d + ' ' + h + ':' + mm;
				})
				.map(function (transactions, date) {
					return {
						date: new Date(date),
						value: transactions.length
					};
				})
				.value();

		drawLineChart(chartContainerId, transactionsByDate);
	});
}

function drawLineChart(elementId, data) {
	var containerEl = document.getElementById(elementId),
		width = containerEl.clientWidth,
		height = width * 0.4,
		margin = {
			top: 30,
			right: 10,
			left: 25
		},

		detailWidth = 175,
		detailHeight = 55,
		detailMargin = 10,

		container = d3.select(containerEl),
		svg = container.append('svg')
			.attr('class', 'lineChart')
			.attr('width', width)
			.attr('height', height + margin.top),

		x = d3.scaleTime()
			.range([0, width - margin.left]),

		xAxis = d3.axisBottom()
			.scale(x)
			.ticks(8)
			.tickSize(10),

		y = d3.scaleLinear()
			.rangeRound([height, 0]),

		yAxis = d3.axisRight()
			.scale(y)
			.ticks(10),

		area = d3.area()
			.curve(d3.curveLinear)
			.x(function (d) {
				return x(d.date) + margin.left;
			})
			.y0(height)
			.y1(function (d) {
				return y(d.value);
			}),

		line = d3.line()
			.curve(d3.curveLinear)
			.x(function (d) {
				return x(d.date) + margin.left;
			})
			.y(function (d) {
				return y(d.value);
			}),

		circleContainer;

	x.domain(d3.extent(data, function(d) { return d.date; }));
	var maxValue = d3.max(data, function (d) { return d.value; });
	y.domain([0, maxValue + maxValue/4]);

	svg.append('g')
		.attr('class', 'xAxis')
		.attr('transform', 'translate(' + margin.left + ',' + ( height + 7 ) + ')')
		.call(xAxis);

	svg.append('g')
		.attr('class', 'yAxis')
		.call(yAxis);

	// Add the area path.
	var gradient = svg.append("svg:defs").append("svg:linearGradient")
		.attr("id", "gradient")
		.attr("x2", "0%")
		.attr("y2", "100%");

	gradient.append("svg:stop")
		.attr("offset", "0%")
		.attr("stop-color", "steelblue")
		.attr("stop-opacity", .5);

	gradient.append("svg:stop")
		.attr("offset", "100%")
		.attr("stop-color", window.getComputedStyle( document.body ,null).getPropertyValue('background-color'))
		.attr("stop-opacity", 1);

	svg.append('path')
		.style("fill", "url(#gradient)")
		.datum(data)
		.attr('class', 'area')
		.attr('d', area)
		.transition()
		.duration(DURATION)
		.attrTween('d', tween(data, area));

	// Add the line path.
	svg.append('path')
		.attr('class', 'areaLine')
		.attr("fill", "none")
		.attr("stroke", "steelblue")
		.attr("stroke-width", 1.5)
		.datum(data)
		.attr('d', line)
		.transition()
		.duration(DURATION)
		.delay(DURATION / 2)
		.attrTween('d', tween(data, line))
		.each(function () { drawCircles(data); });

	// Helper functions
	function drawCircle(datum, index) {
		circleContainer.datum(datum)
			.append('circle')
			.attr('class', 'circle')
			.attr('r', 5)
			.attr('fill', 'transparent')
			.attr('cx', function (d) { return x(d.date) + margin.left; })
			.attr('cy', function (d) { return y(d.value); })
			.on('mouseenter', function (d) {
				d3.select(this)
					.attr('fill', 'steelblue')
					.attr('class','circle circle__highlighted');

				d.active = true;
				showCircleDetail(d);
			})
			.on('mouseout', function (d) {
				d3.select(this)
					.attr('fill', 'transparent')
					.attr('class','circle');

				if (d.active) {
					hideCircleDetails();
					d.active = false;
				}
			})
			.on('click touch', function (d) { return d.active ? showCircleDetail(d) : hideCircleDetails(); })
			.transition()
			.delay(DURATION / 10 * index)
			.attr('r', 5);
	}

	function drawCircles(data) {
		circleContainer = svg.append('g');
		data.forEach(drawCircle);
	}

	function hideCircleDetails() {
		circleContainer.selectAll('.bubble')
			.remove();
	}

	function showCircleDetail(data) {
		var details = circleContainer.append('g')
			.attr('class', 'bubble')
			.attr('transform', function () {
				var x_str = x(data.date) - detailWidth/3,
					y_str = y(data.value) - detailHeight - detailMargin;

				return 'translate(' + x_str + ', ' + y_str + ')';
			});

		var bubble = d3.area()
			.curve(d3.curveLinear)
			.x0(0)
			.x1(function (i) { return i[0];})
			.y0(0)
			.y1(function (i) { return i[1];});

		details.append('path')
			.attr('d', function () { return bubble([[0,0],[detailWidth,0],[detailWidth,detailHeight],[0,detailHeight],[0,0]]);})
			.attr('width', detailWidth)
			.attr('height', detailHeight)
			.attr('fill', 'white')
			.attr("stroke", "steelblue")
			.attr("stroke-width", 1)
			.attr('opacity', '0.8');

		var text = details.append('text')
			.attr('class', 'bubble--text');

		text.append('tspan')
			.attr('class', 'bubble--label')
			.attr('x', detailWidth / 2)
			.attr('y', detailHeight / 3)
			.attr('text-anchor', 'middle')
			.text(data.date.toLocaleString());

		text.append('tspan')
			.attr('class', 'bubble--value')
			.attr('x', detailWidth / 2)
			.attr('y', detailHeight / 4 * 3)
			.attr('text-anchor', 'middle')
			.text("[Txn count: " + data.value + "]");
	}

	function tween(b, callback) {
		return function (a) {
			var i = d3.interpolateArray(a, b);

			return function (t) {
				return callback(i(t));
			};
		};
	}
}