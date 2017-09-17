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
		<link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/4.7.0/css/font-awesome.min.css" rel="stylesheet" type="text/css">
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
		 * Home
		 */

		#home {
			background-color: #333;
		}

		.diagram {
			position: absolute;
			z-index: 1;
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
			opacity: 0.2
		}

		.nodes circle,
		.links line {
			stroke: #FFF;
			stroke-opacity: 0.8;
			stroke-width: 1.2;
		}





		.console {
			margin-top: 56px;
		}



		.test2 {
			text-align: center;
		}


		h2 {
			font-size: 30px;
			font-weight: 700;
			margin-left: auto;
			margin-right: auto;
		}

		h3 {
			font-size: 24px;
			font-weight: 400;
			font-style: italic;
			margin-left: auto;
			margin-right: auto;
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
							<li><a href="#home">Home</a></li>
						</ul>
					</div>
				</div>
			</nav>

			<!-- Diagram -->
			<svg class="diagram" width="1400" height="1400" viewBox="0 0 1400 2100" preserveAspectRatio="xMidYMid"></svg>

			<!-- Home -->
			<section id="home">
				<div class="container">




					<div class="row test1">
						<div class="col-sm-6">
							<div class="console hidden-xs hidden-sm"></div>
						</div>
					</div>


					<div class="row test2">
						<h2>The Decentralized Cloud<h2>
						<h3>A scalable, tamperproof<br>blockchain computer network<h3>
						<br>
						<i class="fa fa-arrow-down" aria-hidden="true"></i>
					</div>


				</div>
			</section>



















		</div>









		<!-- Scripts -->
		<script src="https://cdnjs.cloudflare.com/ajax/libs/jquery/2.1.4/jquery.min.js" type="text/javascript"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/jquery-smooth-scroll/1.5.4/jquery.smooth-scroll.min.js" type="text/javascript"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/d3/4.10.0/d3.min.js" type="text/javascript"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.3.7/js/bootstrap.min.js" type="text/javascript"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/typed.js/2.0.5/typed.min.js" type="text/javascript"></script>
		<script>
		(function($) {

			function setChartSize() {
				$('.diagram').attr('height', $(window).height());
				$('.diagram').attr('width' , $(window).width());
				$('.test1').css('height', $(window).height() * 0.6);
				$('.test2').css('height', $(window).height() * 0.4);
			}

			$(window).load(function() {
				setChartSize('.diagram');
			});

			$(document).ready(function() {










		/**
		 * Navigation
		 */

		var navbar = $('.navbar');
		var navbar_height = navbar.height();

		// Collapse navigation on mobile devices.
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

		// Change color based on scroll position.
		$(window).scroll(function() {
			if ($(this).scrollTop() > navbar_height) {
				navbar.addClass('navbar-color');
			} else {
				navbar.removeClass('navbar-color');
			}
		});

		// Change focus based on scroll position.
		$(window).scrollspy({
			target: '.navbar-custom',
			offset: 50
		});

		// Navigate to scroll position on click.
		$('a[href*=#]').bind('click', function(e) {
			e.preventDefault();
			var anchor = $(this);
			$('html, body').stop().animate({
				scrollTop: $(anchor.attr('href')).offset().top
			}, 500);
		});













				$(window).resize(function() {
					setChartSize('.diagram');
				});

				var svg = d3.select('.diagram'),
					width = +svg.attr('width'),
					height = +svg.attr('height');

				var color = d3.scaleOrdinal(d3.schemeCategory20);

				var simulation = d3.forceSimulation()
					.force('link', d3.forceLink().id(function(d) { return d.NodeID; }))
					.force('charge', d3.forceManyBody())
					.force('center', d3.forceCenter(width / 2, height / 2));

				var tip = d3.select('.console')
					.append('div')
					.attr('class', 'tooltip');

				function mouseover(d) {

					tip
						.transition()
						.duration(200)
						.style('opacity', 1);

					tip.html('<span class="info"></span>');

					var render = new Typed('.info', {
						contentType: 'html',
						loop: false,
						loopCount: false,
						onComplete: function() {
							$('.typed-cursor').css('display', 'none')
						},
						restart: false,
						strings: [
							'ClusterID: ' + d.ClusterID + '<br/>ProcessID: ' + d.ProcessID + '<br/>NodeID: ' + d.NodeID + '<br/>Addresses:<br/>- ' + d.Addrs.join('<br/>- ') + '<br/>Peers: ' + d.Peers + '<br/>Streams: ' + d.Streams + '<br/>Network: ' + d.Network + '<br/>Version: ' + d.Version
						],
						typeSpeed: 0,
					});

				}

				function mouseout(d) {

					tip
						.transition()
						.duration(500)
						.style('opacity', 0);

				}

				function start(d) {
					if (!d3.event.active) simulation.alphaTarget(0.3).restart();
					d.fx = d.x;
					d.fy = d.y;
				}

				function drag(d) {
					d.fx = d3.event.x;
					d.fy = d3.event.y;
				}

				function end(d) {
					if (!d3.event.active) simulation.alphaTarget(0);
					d.fx = null;
					d.fy = null;
				}

				d3.json('/graph', function(error, graph) {

					if (error) throw error;

					var link = svg
						.append('g')
						.attr('class', 'links')
						.selectAll('line')
						.data(graph.links)
						.enter()
						.append('line')
						.attr('stroke-width', function(d) { return 1; });

					var node = svg
						.append('g')
						.attr('class', 'nodes')
						.selectAll('circle')
						.data(graph.nodes)
						.enter()
						.append('circle')
						.attr('r', 5)
						.attr('fill', function(d, i) { return color(i); })
						.on('mouseover', mouseover)
						.on('mouseout', mouseout)
						.call(d3.drag()
							.on('start', start)
							.on('drag', drag)
							.on('end', end));

					function ticked() {

						link
							.attr('x1', function(d) { return d.source.x; })
							.attr('y1', function(d) { return d.source.y; })
							.attr('x2', function(d) { return d.target.x; })
							.attr('y2', function(d) { return d.target.y; });

						node
							.attr('cx', function(d) { return d.x; })
							.attr('cy', function(d) { return d.y; });

					}

					simulation
						.nodes(graph.nodes)
						.on('tick', ticked);

					simulation
						.force('link')
						.links(graph.links);

				});

			});

		})(jQuery);
		</script>

	</body>

</html>`)
