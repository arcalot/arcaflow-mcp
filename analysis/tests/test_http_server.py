"""Tests for the HTTP server wrapper."""

from __future__ import annotations

# pylint: disable=import-error

from arcaflow_analysis.server import http_server


class _FakeHTTPServer:
    def __init__(self, address, handler):  # noqa: D401 - test helper
        self.server_address = ("127.0.0.1", 9000)
        self._handler = handler
        self.started = False
        self.stopped = False

    def serve_forever(self) -> None:
        self.started = True

    def shutdown(self) -> None:
        self.stopped = True


class _ImmediateThread:
    def __init__(self, target, name, daemon):  # noqa: D401 - test helper
        self._target = target
        self._alive = False

    def start(self) -> None:
        self._alive = True
        self._target()

    def is_alive(self) -> bool:
        return self._alive


def test_http_server_start_stop(monkeypatch) -> None:
    monkeypatch.setattr(http_server, "ThreadingHTTPServer", _FakeHTTPServer)
    monkeypatch.setattr(http_server.threading, "Thread", _ImmediateThread)

    server = http_server.AnalysisHTTPServer(("127.0.0.1", 0))
    server.start()
    assert server.port == 9000

    server.stop()
    assert server._server.stopped  # type: ignore[attr-defined]
