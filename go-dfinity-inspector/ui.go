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

		<!-- Styles -->
		<style>
		body {
			background-color: #222222;
			color: white;
			font-family: monospace;
			font-size: 14px;
			margin: 0;
			overflow: hidden;
			padding: 0;
			text-align: center;
		}
		.container {
			left: 0;
			margin-left: auto;
			margin-right: auto;
			position: fixed;
			right: 0;
			z-index: 0;
		}
		.logo {
			padding: 50px 0 50px 0;
			width: 300px;
		}
		.tooltip {
			margin-left: 10%;
			text-align: left;
		}
		.info {
			color: #1eff00;
			display: inline;
		}
		.chart {
			bottom: 0;
			left: 0;
			position: fixed;
			right: 0;
			top: 0;
		}
		.nodes circle {
			stroke: #fff;
			stroke-width: 1.3px;
		}
		.links line {
			stroke: #fff;
			stroke-opacity: 0.7;
		}
		</style>

	</head>

	<!-- Body -->
	<body>

		<!-- Logo -->
		<div class="container">
			<img class="logo" src="https://s3-us-west-2.amazonaws.com/dfinity/images/dfinity-logo-large.png">
		</div>

		<!-- D3 -->
		<svg class="chart" width="1024" height="768" viewBox="0 0 1024 768" preserveAspectRatio="xMidYMid meet"></svg>

		<!-- Scripts -->
		<script src="https://cdnjs.cloudflare.com/ajax/libs/jquery/2.1.4/jquery.min.js" type="text/javascript"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/d3/4.10.0/d3.min.js" type="text/javascript"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/typed.js/2.0.5/typed.min.js" type="text/javascript"></script>
		<script>
		(function($) {

			function setChartSize() {
				$('.chart').attr('height', $(window).height());
				$('.chart').attr('width' , $(window).width());
			}

			$(window).load(function() {
				setChartSize('.chart');
			});

			$(document).ready(function() {

				$(window).resize(function() {
					setChartSize('.chart');
				});

				var svg = d3.select('.chart'),
					width = +svg.attr('width'),
					height = +svg.attr('height');

				var color = d3.scaleOrdinal(d3.schemeCategory20);

				var simulation = d3.forceSimulation()
					.force('link', d3.forceLink().id(function(d) { return d.NodeID; }))
					.force('charge', d3.forceManyBody())
					.force('center', d3.forceCenter(width / 2, height / 2));

				var tip = d3.select('.container')
					.append('div')
					.attr('class', 'tooltip')
					.style('opacity', 0)
					.style('z-index', 0);

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
