/**
 * File        : ui.go
 * Description : User interface.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package main

const UI = `<!DOCTYPE html>
<html lang="en">

	<!-- Header -->
	<head>

		<!-- Metadata -->
		<meta charset="UTF-8">
		<meta content="width=device-width, initial-scale=1.0" name="viewport">

		<!-- Title -->
		<title>Chirp Client</title>

		<!-- Styles -->
		<link href="https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.3.6/css/bootstrap.min.css" rel="stylesheet" type="text/css">
		<link href="https://fonts.googleapis.com/css?family=Open+Sans:400,700" rel="stylesheet" type="text/css">

		<!-- Custom -->
		<style>
		body {
			background-color: #222222;
			font: 400 14px/1.5 'Open Sans', sans-serif;
			-webkit-font-smoothing: antialiased;
		}
		div.preloader {
			background-color: #FFFFFF;
			bottom: 0;
			left: 0;
			position: fixed;
			right: 0;
			top: 0;
			z-index: 999;
		}
		div.preloader > div.graphic {
			background-image: url('https://s3-us-west-2.amazonaws.com/dfinity/images/loader.gif');
			background-position: center;
			background-repeat: no-repeat;
			height: 500px;
			left: 50%;
			margin: -250px 0 0 -250px;
			position: absolute;
			top: 50%;
			width: 500px;
		}
		div.logo {
			padding: 50px 0px 50px 0px;
			text-align: center;
		}
		div.logo > img {
			width: 300px;
		}
		div.panel {
			border-radius: 6px;
		}
		div.panel-body {
			height: 50%;
			overflow-y: scroll;
		}
		ul.chat {
			list-style: none;
			margin: 0;
			overflow-y: scroll;
			padding: 10px;
		}
		ul.chat > li {
			border-bottom: 1px dotted #B3A9A9;
			clear: both;
			margin-bottom: 10px;
			padding-bottom: 5px;
		}
		ul.chat > li > div.chat-body {
			clear: both;
		}
		ul.chat > li > div.chat-body > p {
			color: #777777;
			margin: 0;
		}
		a,
		a:focus,
		a:hover {
			color: #21A9E3;
		}
		span.input-group-addon {
			border: 1px solid #B2B2B2;
			border-radius: 6px;
			border-right: none;
		}
		div.timestamp {
			display: inline;
		}
		div.error {
			color: #FF0000;
		}
		input#username {
			background-color: #EEEEEE;
			border: none;
			box-shadow: none;
			-moz-box-shadow: none;
			-webkit-box-shadow: none;
			width: 90px;
		}
		input#message {
			appearance: none;
			-moz-appearance: none;
			-webkit-appearance: none;
			background-color: #F9F9F9;
			border: 1px solid #B2B2B2;
			color: #000000;
		}
		input#message:focus {
			border: 1px solid #21A9E3;
			box-shadow: none;
			-moz-box-shadow: none;
			-webkit-box-shadow: none;
		}
		button.btn {
			background-color: #21A9E3;
			border-color: #21A9E3;
			color: #F9F9F9;
		}
		button.btn:focus,
		button.btn.active:focus {
			background-color: #21A9E3;
			color: #F9F9F9;
			outline: none !important;
		}
		button.btn:hover {
			color: #F9F9F9;
		}
		</style>

	</head>

	<!-- Body -->
	<body>

		<!-- Preloader -->
		<div class="preloader">
			<div class="graphic"></div>
		</div>

		<section>
			<div class="container">
				<div class="row">
					<div class="col-md-2"></div>
					<div class="col-md-8">

						<!-- Logo -->
						<div class="logo">
							<img src="https://s3-us-west-2.amazonaws.com/dfinity/images/dfinity-logo-large.png"/>
						</div>

						<!-- Chat Widget -->
						<div class="panel">
							<div class="panel-body">
								<ul class="chat"></ul>
							</div>
							<div class="panel-footer">
								<div class="input-group">
									<span class="input-group-addon">
										<input id="username" type="text" class="form-control input-sm" placeholder="Username"/>
									</span>
									<input id="message" type="text" class="form-control input-lg" placeholder="Enter your message here ..."/>
									<span class="input-group-btn">
										<button class="btn btn-lg" id="btn-chat">Chirp</button>
									</span>
								</div>
							</div>
						</div>

					</div>
				</div>
			</div>
		</section>

		<!-- Scripts -->
		<script src="https://cdnjs.cloudflare.com/ajax/libs/jquery/2.1.4/jquery.min.js" type="text/javascript"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/jQuery-linkify/2.1.4/linkify.min.js" type="text/javascript"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/jQuery-linkify/2.1.4/linkify-html.min.js" type="text/javascript"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.3.6/js/bootstrap.min.js" type="text/javascript"></script>

		<!-- Custom -->
		<script type="text/javascript">
		(function($) {

			$(window).load(function() {
				$('div.preloader').fadeOut(500, 'linear');
			});

			$(document).ready(function() {

				var random = Math.floor(Math.random() * 90000) + 10000;
				$('input#username').val('Guest ' + String(random));

				function formatTimestamp(timestamp) {
					if (!(typeof timestamp == 'number')) {
						console.error('format: bad argument');
						return '';
					}
					var delta = Math.floor((Date.now() - timestamp) / 1000);
					if (delta < 30) {
						return 'just now';
					} else if (delta < 60) {
						return String(delta) + ' seconds ago';
					} else if (delta < 120) {
						return 'a minute ago';
					} else if (delta < 3600) {
						return String(Math.floor(delta / 60)) + ' minutes ago';
					} else if (Math.floor(delta / 3600) == 1) {
						return 'an hour ago';
					} else if (delta < 86400) {
						return String(Math.floor(delta / 3600)) + ' hours ago';
					} else if (delta < 172800) {
						return 'yesterday';
					} else {
						return String(Math.floor(delta / 86400)) + ' days ago';
					}
				}

				function log(data, timestamp) {
					document.querySelector('ul.chat').insertAdjacentHTML('beforeend', '<li class="left"><div class="chat-body"><div class="header"><strong class="primary-font">' + data.Username + '</strong><small class="pull-right text-muted"></span><div class="timestamp" data-timestamp="' + String(timestamp) + '">' + formatTimestamp(timestamp) + '</div></small></div><p>' + linkifyHtml(data.Data, {
						defaultProtocol: 'https',
					}) + '</p></div></li>');
					document.querySelector('div.panel-body').scrollTop = document.querySelector('div.panel-body').scrollHeight;
				}

				setInterval(function() {
					$('div.timestamp').each(function() {
						$(this).html(formatTimestamp($(this).data('timestamp')));
					});
				}, 5000);

				var ws;

				function initWebSocket() {
					var socket = new WebSocket('ws://' + location.hostname + (location.port ? ':' + location.port : '') + '/ws');
					socket.onopen = function() {
						log({
							Username: 'Chirp Bot',
							Data: 'Welcome to Chirp! Here you can chat with DFINITY developers, ask questions, and get answers.',
						}, Date.now());
					}
					socket.onmessage = function(event) {
						log(JSON.parse(event.data), Date.now());
					}
					socket.onclose = function() {
						log({
							Username: 'Chirp Bot',
							Data: '<div class="error">WebSocket closed. Is the Chirp client open in another window or tab?</div>',
						}, Date.now());
					}
					return socket;
				}

				if (window.WebSocket === undefined) {
					log({
						Username: 'Chirp Bot',
						Data: '<div class="error">Your browser does not support WebSockets.</div>',
					}, Date.now());
					return;
				} else {
					ws = initWebSocket();
				}

				var nonce = 0;

				function chirp() {
					var username = $('input#username').val();
					var message = $('input#message').val();
					if (username.length === 0 || message.length === 0) {
						return;
					}
					var data = {
						Username: username,
						Data: message,
						Nonce: nonce++,
					};
					ws.send(JSON.stringify(data));
					log(data, Date.now());
					$('input#message').val('');
				}

				$('button#btn-chat').click(function(event) {
					event.preventDefault();
					chirp();
				});

				$('input#message').keydown(function(event) {
					if (event.keyCode == 13) {
						event.preventDefault();
						chirp();
					}
				});

			});

		})(jQuery);
		</script>

	</body>

</html>`
