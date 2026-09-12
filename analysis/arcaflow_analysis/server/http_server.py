"""HTTP server for analysis operations."""

from __future__ import annotations

import json
import logging
import os
import threading
from urllib.parse import parse_qs, urlparse
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any

from arcaflow_analysis.db import DatabaseConfig, HistoryRepository, init_db
from arcaflow_analysis.server.http_api import (
    add_history,
    analyze_results,
    get_history,
    list_history,
)


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
    _history_repo: HistoryRepository | None = None
    _history_lock = threading.Lock()

    @classmethod
    def _ensure_history_repo(cls) -> HistoryRepository | None:
        """Initialize the history repository if configured."""
        db_url = os.getenv("ARCAFLOW_ANALYSIS_DB_URL")
        if not db_url:
            return None
        if cls._history_repo is not None:
            return cls._history_repo
        with cls._history_lock:
            if cls._history_repo is None:
                session_factory = init_db(DatabaseConfig(url=db_url))
                cls._history_repo = HistoryRepository(session_factory)
        return cls._history_repo

    def do_GET(self) -> None:  # noqa: N802 - BaseHTTPRequestHandler API
        """Serve health checks."""
        if self.path == "/healthz":
            self._send_json({"status": "ok"})
            return
        if self.path.startswith("/analysis/history"):
            repo = self._ensure_history_repo()
            if repo is None:
                self.send_error(HTTPStatus.BAD_REQUEST)
                return
            parsed = urlparse(self.path)
            parts = parsed.path.split("/")
            if len(parts) == 3:
                query = parse_qs(parsed.query)
                workflow_id = query.get("workflow_id", [None])[0]
                response = list_history(repo, workflow_id)
                self._send_json(response)
                return
            if len(parts) == 4 and parts[3]:
                response = get_history(repo, parts[3])
                self._send_json(response)
                return
            self.send_error(HTTPStatus.BAD_REQUEST)
            return
        self.send_error(HTTPStatus.NOT_FOUND)

    def do_POST(self) -> None:  # noqa: N802 - BaseHTTPRequestHandler API
        """Handle analysis requests."""
        if self.path not in {"/analysis/summary", "/analysis/history"}:
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
        if self.path == "/analysis/summary":
            response = analyze_results(request)
            self._send_json(response)
            return
        repo = self._ensure_history_repo()
        if repo is None:
            self.send_error(HTTPStatus.BAD_REQUEST)
            return
        workflow_id = request.get("workflow_id")
        input_payload = request.get("input_payload")
        metrics = request.get("metrics")
        if (
            not workflow_id
            or not isinstance(input_payload, dict)
            or not isinstance(metrics, dict)
        ):
            self.send_error(HTTPStatus.BAD_REQUEST)
            return
        response = add_history(repo, workflow_id, input_payload, metrics)
        self._send_json(response)

    def _send_json(self, payload: dict[str, Any]) -> None:
        """Send JSON response with a 200 status."""
        data = json.dumps(payload).encode("utf-8")
        self.send_response(HTTPStatus.OK)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)
