package cache

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
		<style type="text/css">
			{{.style}}
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
					switch (data.action) {
						case "reload":
							sendMessage({ "action": "close" })
							requestAnimationFrame(() => { location.reload() })
							break
						case "shutdown":
							document.title = "Cannon preview";
							const container = document.getElementById("container");
							if (container) {
								const inner = "<p>Disconnected from server: " + document.location.href + "</p>";
								container.innerHTML = inner;
							}
							break
					}
				}
			}
		</script>
	</head>
	<body>
		<div id="container">{{.html}}</div>
	</body>
</html>
`
