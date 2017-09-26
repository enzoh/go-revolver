/**
 * File        : ui.go
 * Description : User interface.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package main

var HTML = []byte(`<!DOCTYPE html>
<html>

	<!-- Header -->
	<head>

		<!-- Metadata -->
		<meta charset="utf-8">
		<meta content="width=device-width, initial-scale=1.0" name="viewport">

		<!-- Title -->
		<title>DFINITY | Network Topology Inspector</title>

		<!-- Fonts -->
		<link href="https://fonts.googleapis.com/css?family=Nunito:600" rel="stylesheet" type="text/css">
		<link href="https://fonts.googleapis.com/css?family=Proxima+Nova:400,700" rel="stylesheet" type="text/css">
		<link href="https://fonts.googleapis.com/css?family=Roboto+Mono:400" rel="stylesheet" type="text/css">

		<!-- Styles -->
		<link href="https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.3.7/css/bootstrap.min.css" rel="stylesheet" type="text/css">

		<!-- Custom -->
		<style>

		/**
		 * General Styles
		 */

		body {
			color: #FFF;
			font-family: 'Proxima Nova', sans-serif;
			font-size: 12px;
			font-weight: 400;
			-webkit-font-smoothing: antialiased;
		}

		h2 {
			font-size: 30px;
			font-weight: 700;
			margin-left: auto;
			margin-right: auto;
			position: relative;
			z-index: 1;
		}

		h3 {
			font-size: 24px;
			font-weight: 400;
			font-style: italic;
			margin-left: auto;
			margin-right: auto;
			padding-bottom: 15px;
			position: relative;
			z-index: 1;
		}

		input {
			color: #000;
			font-family: 'Roboto Mono', monospace;
			font-size: 14px;
			font-weight: 400;
		}

		/**
		 * Navigation
		 */

		.navbar-custom {
			background-color: transparent;
			border: 0;
		}

		.navbar-custom .navbar-brand {
			height: 56px;
			padding: 0 15px;
		}

		.navbar-custom .navbar-brand > img {
			display: inline;
			height: 56px;
			padding: 14px 0;
		}

		.navbar-custom .navbar-brand > span {
			color: #FFF;
			font-family: 'Nunito', sans-serif;
			font-size: 34px;
			font-weight: 600;
			left: 2px;
			position: relative;
			top: 8px;
		}

		.navbar-custom .navbar-toggle {
			top: 4px;
		}

		.navbar-custom .navbar-nav > li.active {
			background-color: transparent;
			border-bottom: 3px solid #FFF;
		}

		.navbar-custom .navbar-nav > li > a {
			color: #FFF;
			height: 53px;
			padding-bottom: 22px;
			padding-top: 22px;
			text-transform: uppercase;
		}

		.navbar-custom .navbar-nav > li > a:hover,
		.navbar-custom .navbar-nav > li > a:focus {
			background-color: transparent;
		}

		.navbar-color,
		.custom-collapse {
			background-color: #FFF;
			box-shadow: 0 0 4px rgba(0,0,0,0.2);
			-webkit-box-shadow: 0 0 4px rgba(0,0,0,0.2);
			padding: 0;
		}

		.navbar-color .navbar-brand > span,
		.custom-collapse .navbar-brand > span {
			color: #333;
		}

		.custom-collapse .navbar-nav {
			text-align: left;
		}

		.navbar-color .navbar-nav > li.active {
			border-bottom: 3px solid #1EFF00;
		}

		.custom-collapse .navbar-nav > li.active {
			border: 0;
		}

		.navbar-color .navbar-nav > li > a,
		.custom-collapse .navbar-nav > li > a {
			color: #333;
		}

		.navbar-color .navbar-toggle .icon-bar,
		.custom-collapse .navbar-toggle .icon-bar {
			background-color: #333;
		}

		/**
		 * Network
		 */

		#network {
			background-color: #333;
		}

		.canvas {
			position: absolute;
			z-index: 1;
		}

		.nodes circle {
			z-index: -1;
		}

		.links line {
			stroke: #FFF;
			stroke-opacity: 0.25;
			stroke-width: 0.25;
			z-index: -1;
		}

		.console {
			margin-top: 56px;
		}

		.tooltip {
			text-align: left;
			opacity: 0;
			z-index: 0;
		}

		.info {
			color: #1EFF00;
			font-family: 'Roboto Mono', monospace;
			font-size: 14px;
			font-weight: 400;
			display: inline;
		}

		.splash-footer {
			text-align: center;
		}

		.input-group {
			position: relative;
			z-index: 1;
		}

		</style>

	</head>

	<!-- Body -->
	<body>

		<div class="wrapper">

			<!-- Navigation -->
			<nav class="navbar navbar-custom navbar-fixed-top" role="navigation">

				<div class="container">

					<div class="navbar-header">

						<a class="navbar-brand" href="/">
							<img alt="" src="https://s3-us-west-2.amazonaws.com/dfinity/images/logo.svg">
							<span>DFINITY</span>
						</a>

						<button class="navbar-toggle" data-target="#navbar" data-toggle="collapse" type="button">
							<span class="sr-only">Toggle Navigation</span>
							<span class="icon-bar"></span>
							<span class="icon-bar"></span>
							<span class="icon-bar"></span>
						</button>

					</div>

					<div class="navbar-collapse collapse" id="navbar">
						<ul class="nav navbar-nav navbar-right">
							<li><a href="#network">Network</a></li>
						</ul>
					</div>

				</div>

			</nav>

			<!-- Canvas -->
			<svg class="canvas" width="900" height="900" viewBox="0 0 900 1350" preserveAspectRatio="xMidYMid"></svg>

			<!-- Network -->
			<section id="network">

				<div class="container">

					<div class="row splash-body">
						<div class="col-sm-6">
							<div class="console hidden-xs hidden-sm"></div>
						</div>
					</div>


					<div class="row splash-footer">
						<h2>The Decentralized Cloud</h2>
						<h3>A scalable, tamperproof<br>blockchain computer network</h3>
						<div class="col-sm-3"></div>
						<div class="col-sm-6">
							<div class="input-group">
								<span class="input-group-addon">Beacon</span>
								<input id="beacon" type="text" class="form-control input-md"/>
							</div>
						</div>
						<div class="col-sm-3"></div>
					</div>

				</div>

			</section>

		</div>

		<!-- Scripts -->
		<script src="https://cdnjs.cloudflare.com/ajax/libs/d3/4.10.0/d3.min.js" type="text/javascript"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/jquery/2.1.4/jquery.min.js" type="text/javascript"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/jquery-smooth-scroll/1.5.4/jquery.smooth-scroll.min.js" type="text/javascript"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.3.7/js/bootstrap.min.js" type="text/javascript"></script>
		<script>
		(function($) {

			function resize() {

				$('.splash-body').css('height', $(window).height() * 0.6);
				$('.splash-footer').css('height', $(window).height() * 0.4);

				$('.canvas').attr('height', $(window).height());
				$('.canvas').attr('width', $(window).width());

			}

			$(window).load(function() {
				resize('.canvas');
			});

			$(document).ready(function() {

				var navbar = $('.navbar');
				var navbar_height = navbar.height();

				if ($(window).width() <= 752) {
					navbar.addClass('custom-collapse');
				}

				$(window).resize(function() {
					if ($(this).width() <= 752) {
						navbar.addClass('custom-collapse');
					} else {
						navbar.removeClass('custom-collapse');
					}
				});

				$(window).scroll(function() {
					if ($(this).scrollTop() > navbar_height) {
						navbar.addClass('navbar-color');
					} else {
						navbar.removeClass('navbar-color');
					}
				});

				$(window).scrollspy({
					target: '.navbar-custom',
					offset: 50
				});

				$('a[href*=#]').bind('click', function(e) {
					e.preventDefault();
					var anchor = $(this);
					$('html, body').stop().animate({
						scrollTop: $(anchor.attr('href')).offset().top
					}, 500);
				});

				$(window).resize(function() {
					resize('.canvas');
				});

				var svg = d3.select('.canvas'), width = +svg.attr('width'), height = +svg.attr('height');
				var tip = d3.select('.terminal').append('div').attr('class', 'tooltip');

				var simulation = d3.forceSimulation()
					.force('center', d3.forceCenter(width / 2, height / 2))
					.force('charge', d3.forceManyBody())
					.force('link', d3.forceLink().id(function(d) { return d.NodeID; }));

				function drag(d) {
					d.fx = d3.event.x;
					d.fy = d3.event.y;
				}

				function start(d) {
					if (!d3.event.active) {
						simulation.alphaTarget(0.3).restart();
					}
					d.fx = d.x;
					d.fy = d.y;
				}

				function end(d) {
					if (!d3.event.active) {
						simulation.alphaTarget(0);
					}
					d.fx = null;
					d.fy = null;
				}

				function mouseover(d) {
					tip.transition().duration(100).style('opacity', 1);
					tip.html(
						'<span class="info">NodeID: ' +
						d.NodeID +
						'<br/>Addresses:<br/>- ' +
						d.Addrs.join('<br/>- ') +
						'<br/>Network: ' +
						d.Network +
						'<br/>Version: ' +
						d.Version +
						'</span>'
					);
				}

				function mouseout(d) {
					tip.transition().duration(500).style('opacity', 0);
				}

				d3.json('https://testnet.london.dfinity.build/api/v1/graph', function(err, graph) {

					if (err) throw err;

					var link = svg
						.append('g')
						.attr('class', 'links')
						.selectAll('line')
						.data(graph.links)
						.enter()
						.append('line');

					var node = svg
						.append('g')
						.attr('class', 'nodes')
						.selectAll('circle')
						.data(graph.nodes)
						.enter()
						.append('circle')
						.attr('data-bls-public-key', function(d, i) { return d.UserData; })
						.attr('fill', function(d, i) { return '#525252'; })
						.attr('r', 5)
						.on('mouseover', mouseover)
						.on('mouseout', mouseout)
						.call(d3.drag()
							.on('drag', drag)
							.on('start', start)
							.on('end', end));

					function tick() {

						link.attr('x1', function(d) {
							return d.source.x;
						}).attr('y1', function(d) {
							return d.source.y;
						}).attr('x2', function(d) {
							return d.target.x;
						}).attr('y2', function(d) {
							return d.target.y;
						});

						node.attr('cx', function(d) {
							return d.x;
						}).attr('cy', function(d) {
							return d.y;
						});

					}

					simulation.nodes(graph.nodes).on('tick', tick);
					simulation.force('link').links(graph.links);

				});

				$.ajax({

					error: function() {
						console.error('Cannot download genesis block ...');
					},

					success: function(genesis) {

						if (genesis == null) {
							console.log('Awaiting genesis block ...');
							return;
						}

						var event;
						var block;
						var group;

						var miners = genesis.block.setup[0];
						var groups = genesis.block.setup[1];
						var matrix = new Array(groups.length);
						var source = new EventSource('https://worker.london.dfinity.build/api/v1/block/events');

						for (i = 0; i < groups.length; i++) {
							matrix[i] = new Array(groups[i].groupMembers.length);
							for (j = 0; j < groups[i].groupMembers.length; j++) {
								matrix[i][j] = miners[groups[i].groupMembers[j][0]];
							}
						}

						source.addEventListener('message', function(message) {

							event = JSON.parse(message.data);
							block = event.block;
							group = event.group;

							$('#beacon').val(block.beacon);

							var key;
							$('circle').attr('fill', '#525252');
							$('circle').each(function() {
								key = $(this).data('bls-public-key');
								if (matrix[group].includes(key)) {
									$(this).attr('fill', '#1EFF00');
								}
							});

						}, false);

						source.addEventListener('error', function(err) {
							console.error(err);
						}, false);

					},

					timeout: 30000,
					type: 'GET',
					url: 'https://worker.london.dfinity.build/api/v1/block/genesis',

				});

			});

		})(jQuery);
		</script>

	</body>

</html>`)
