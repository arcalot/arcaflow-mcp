"""Tests for analysis server scaffolding."""

import logging

import grpc
import pytest

from arcaflow_analysis.server.app import GrpcServer, ServerConfig, configure_logging


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


def test_configure_logging_sets_level(monkeypatch: pytest.MonkeyPatch) -> None:
    """Verify configure_logging respects ARCAFLOW_MCP_LOG_LEVEL."""
    monkeypatch.setenv("ARCAFLOW_MCP_LOG_LEVEL", "DEBUG")
    configure_logging()
    root_logger = logging.getLogger()
    assert root_logger.level == logging.DEBUG
