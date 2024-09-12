package cache

const SpinnerTemplate = `
	<svg version="1.1" id="spinner" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" x="0px" y="0px"
		viewBox="0 0 100 100" enable-background="new 0 0 0 0" xml:space="preserve">
		<path fill="#888" d="M73,50c0-12.7-10.3-23-23-23S27,37.3,27,50 M30.9,50c0-10.5,8.5-19.1,19.1-19.1S69.1,39.5,69.1,50">
			<animateTransform
				attributeName="transform"
				attributeType="XML"
				type="rotate"
				dur="1s"
				from="0 50 50"
				to="360 50 50"
				repeatCount="indefinite" />
		</path>
	</svg>
`

const PageTemplate = `
<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8" />
		<!-- https://stackoverflow.com/a/62438464 - https://heroicons.com/ - https://fffuel.co/eeencode/ -->
		<link rel="icon" href="data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIGZpbGw9Im5vbmUiIHZpZXdCb3g9IjAgMCAyNCAyNCIgc3Ryb2tlLXdpZHRoPSIxLjUiIHN0cm9rZT0iY3VycmVudENvbG9yIiBjbGFzcz0idy02IGgtNiI+PHBhdGggc3Ryb2tlLWxpbmVjYXA9InJvdW5kIiBzdHJva2UtbGluZWpvaW49InJvdW5kIiBkPSJNMi4yNSAxMi43NVYxMkEyLjI1IDIuMjUgMCAwMTQuNSA5Ljc1aDE1QTIuMjUgMi4yNSAwIDAxMjEuNzUgMTJ2Ljc1bS04LjY5LTYuNDRsLTIuMTItMi4xMmExLjUgMS41IDAgMDAtMS4wNjEtLjQ0SDQuNUEyLjI1IDIuMjUgMCAwMDIuMjUgNnYxMmEyLjI1IDIuMjUgMCAwMDIuMjUgMi4yNWgxNUEyLjI1IDIuMjUgMCAwMDIxLjc1IDE4VjlhMi4yNSAyLjI1IDAgMDAtMi4yNS0yLjI1aC01LjM3OWExLjUgMS41IDAgMDEtMS4wNi0uNDR6IiAvPjwvc3ZnPg==" type="image/svg+xml" />
		<title>
			{{.title}}
		</title>
		<style>
			div {
				width:100%;
			}
			img {
				max-width: 100%;
				height:auto;
				max-height: 100%;
			}
			video {
				max-width: 100%;
				height: auto;
				max-height: 100%;
			}
			iframe {
				position: absolute;
				top: 0;
				left: 0;
				width: 100%;
				height: 100%;
				border: 0;
			}
			object {
				max-width: 100%;
				height: auto;
				max-height: 100%;
			}
			#spinner {
				width: 40px;
				height: 40px;
				margin: 20px;
				display:inline-block;
			}
		</style>
		<script>
			window.onload = function(e) {
				// replace placeholder addresses with document.location.href
				const container = document.getElementById("container")
				if (container) {
					const inner = container.innerHTML.replaceAll("{document.location.href}", document.location.href)
					container.innerHTML = inner
				}

				// open websocket
				const socket = new WebSocket((document.location.href).replace(/^https?:/, "ws:"))
				const sendMessage = function(obj) { socket.send(JSON.stringify(obj)) }
				socket.onopen = function(e) {}
				socket.onerror = function(error) {}
				socket.onclose = function(event) {}
				socket.onmessage = function(event) {
					const data = JSON.parse(event.data)
					console.log('Received:', data)
					location.reload()
				}
			}
		</script>
	</head>
	<body>
		<div id="container">{{.html}}</div>
	</body>
</html>
`
