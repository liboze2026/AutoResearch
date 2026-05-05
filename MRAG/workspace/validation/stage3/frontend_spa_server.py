from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlparse
import os
import sys

root = Path(sys.argv[1]).resolve()
port = int(sys.argv[2])
os.chdir(root)

class SPAHandler(SimpleHTTPRequestHandler):
    def do_GET(self):
        parsed = urlparse(self.path)
        route_path = parsed.path or "/"
        candidate = root / route_path.lstrip("/")
        if route_path == "/" or (not candidate.exists() and not Path(route_path).suffix):
            self.path = "/index.html"
        return super().do_GET()

server = ThreadingHTTPServer(("127.0.0.1", port), SPAHandler)
server.serve_forever()
