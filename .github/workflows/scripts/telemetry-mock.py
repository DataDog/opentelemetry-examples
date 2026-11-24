import http.server


class Handler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        with open("/tmp/telemetry.json", "wb") as f:
            f.write(self.rfile.read(int(self.headers["Content-Length"])))
        self.send_response(202)
        self.end_headers()


http.server.HTTPServer(("127.0.0.1", 8126), Handler).serve_forever()
