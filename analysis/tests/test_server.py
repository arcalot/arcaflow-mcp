"""Tests for analysis server scaffolding."""

import logging

import grpc
import pytest

from arcaflow_analysis.server import app

from arcaflow_analysis.server.app import (
    GrpcServer,
    ServerConfig,
    _parse_args,
    configure_logging,
)


def test_server_config_defaults() -> None:
    """Ensure default server config values are set."""
    config = ServerConfig()
    assert config.host == "0.0.0.0"
    assert config.port == 50051


def test_grpc_server_start_and_ping() -> None:
    """Validate the gRPC server starts and responds to ping."""
    server = GrpcServer(ServerConfig(host="127.0.0.1", port=0))
    try:
        server.start()
    except PermissionError as exc:
        pytest.skip(f"socket operations not permitted: {exc}")
    assert server.port is not None

    import analysis_pb2
    import analysis_pb2_grpc

    channel = grpc.insecure_channel(f"127.0.0.1:{server.port}")
    stub = analysis_pb2_grpc.AnalysisServiceStub(channel)
    response = stub.Ping(analysis_pb2.PingRequest(message="hello"))
    assert response.message == "hello"
    assert response.server_version
    channel.close()
    server.stop()


def test_grpc_server_start_with_http() -> None:
    """Ensure HTTP server startup does not raise lookup errors."""
    server = GrpcServer(
        ServerConfig(host="127.0.0.1", port=0, http_address="127.0.0.1:0")
    )
    try:
        server.start()
    except PermissionError as exc:
        pytest.skip(f"socket operations not permitted: {exc}")
    assert server.port is not None
    assert server.http_port is not None
    server.stop()


def test_configure_logging_sets_level(monkeypatch: pytest.MonkeyPatch) -> None:
    """Verify configure_logging respects ARCAFLOW_MCP_LOG_LEVEL."""
    monkeypatch.setenv("ARCAFLOW_MCP_LOG_LEVEL", "DEBUG")
    configure_logging()
    root_logger = logging.getLogger()
    assert root_logger.level == logging.DEBUG


def test_configure_logging_override_level() -> None:
    """Verify configure_logging respects explicit level overrides."""
    configure_logging("ERROR")
    root_logger = logging.getLogger()
    assert root_logger.level == logging.ERROR


def test_parse_args_debug_overrides_default() -> None:
    args = _parse_args(["--debug"])
    assert args.debug is True
    assert args.log_level == "INFO"


def test_parse_args_custom_http_address() -> None:
    args = _parse_args(["--http-address", "127.0.0.1:8081"])
    assert args.http_address == "127.0.0.1:8081"


def test_main_handles_keyboard_interrupt(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    class FakeServer:
        def __init__(self, config: ServerConfig) -> None:
            self.config = config
            self.stopped = False

        def start(self) -> None:
            return None

        def wait_for_termination(self) -> None:
            raise KeyboardInterrupt

        def stop(self, grace: float = 5.0) -> None:
            _ = grace
            self.stopped = True

    fake = FakeServer(ServerConfig())
    monkeypatch.setattr(app, "GrpcServer", lambda _config: fake)
    monkeypatch.setattr(app, "configure_logging", lambda _level=None: None)
    monkeypatch.setattr(app, "sys", type("Sys", (), {"argv": ["app"]}))

    app.main()
    assert fake.stopped is True
