import argparse
import json
import threading
import time
from http.server import BaseHTTPRequestHandler, HTTPServer


class DummyPiState:
    def __init__(self):
        self._lock = threading.Lock()
        self._count = 0
        self._running = False

    def start(self):
        with self._lock:
            self._running = True

    def stop(self):
        with self._lock:
            self._running = False

    def reset(self):
        with self._lock:
            self._count = 0
            self._running = False

    def status(self):
        with self._lock:
            state = "running" if self._running else "stopped"
            return {"state": state, "count": self._count}

    def tick(self):
        with self._lock:
            if self._running:
                self._count += 1


class DummyPiHandler(BaseHTTPRequestHandler):
    state = DummyPiState()

    def do_GET(self):
        if self.path == "/status":
            self._send_json(200, self.state.status())
            return
        self._send_json(404, {"ok": False, "error": "not found"})

    def do_POST(self):
        if self.path == "/start":
            self.state.start()
            self._send_json(200, {"ok": True})
            return
        if self.path == "/stop":
            self.state.stop()
            self._send_json(200, {"ok": True})
            return
        if self.path == "/reset":
            self.state.reset()
            self._send_json(200, {"ok": True})
            return
        self._send_json(404, {"ok": False, "error": "not found"})

    def log_message(self, format, *args):
        return

    def _send_json(self, code, payload):
        body = json.dumps(payload).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


def main():
    parser = argparse.ArgumentParser(description="Dummy fuel counter Pi server.")
    parser.add_argument("--port", type=int, default=8000, help="Port to listen on (default: 8000)")
    parser.add_argument("--rate", type=float, default=2.0, help="Counts per second when running (default: 2)")
    args = parser.parse_args()

    tick_interval = 1.0 / max(args.rate, 0.1)

    def ticker():
        while True:
            time.sleep(tick_interval)
            DummyPiHandler.state.tick()

    threading.Thread(target=ticker, daemon=True).start()

    server = HTTPServer(("", args.port), DummyPiHandler)
    print(f"Dummy Pi listening on port {args.port}")
    server.serve_forever()


if __name__ == "__main__":
    main()
