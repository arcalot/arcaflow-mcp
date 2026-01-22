"""HTTP server for analysis operations."""

from __future__ import annotations

import json
import logging
import threading
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any

from arcaflow_analysis.server.http_api import analyze_results


class AnalysisHTTPServer:
    """HTTP server exposing analysis endpoints."""

    def __init__(self, address: tuple[str, int]) -> None:
        self._logger = logging.getLogger("arcaflow_analysis.http")
        self._server = ThreadingHTTPServer(address, _AnalysisHandler)
        self.port = self._server.server_address[1]
        self._thread: threading.Thread | None = None

    def start(self) -> None:
        """Start the HTTP server."""
        if self._thread is not None and self._thread.is_alive():
            return
        self._thread = threading.Thread(
            target=self._server.serve_forever,
            name="analysis-http-server",
            daemon=True,
        )
        self._thread.start()
        self._logger.info("analysis HTTP server started", extra={"port": self.port})

    def stop(self) -> None:
        """Stop the HTTP server."""
        self._server.shutdown()
        self._logger.info("analysis HTTP server stopped")


class _AnalysisHandler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:  # noqa: N802 - BaseHTTPRequestHandler API
        """Serve health checks."""
        if self.path == "/healthz":
            self._send_json({"status": "ok"})
            return
        self.send_error(HTTPStatus.NOT_FOUND)

    def do_POST(self) -> None:  # noqa: N802 - BaseHTTPRequestHandler API
        """Handle analysis requests."""
        if self.path != "/analysis/summary":
            self.send_error(HTTPStatus.NOT_FOUND)
            return
        try:
            length = int(self.headers.get("Content-Length", "0"))
        except ValueError:
            self.send_error(HTTPStatus.BAD_REQUEST)
            return
        body = self.rfile.read(length).decode("utf-8")
        try:
            request = json.loads(body)
        except json.JSONDecodeError:
            self.send_error(HTTPStatus.BAD_REQUEST)
            return
        response = analyze_results(request)
        self._send_json(response)

    def _send_json(self, payload: dict[str, Any]) -> None:
        """Send JSON response with a 200 status."""
        data = json.dumps(payload).encode("utf-8")
        self.send_response(HTTPStatus.OK)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)
