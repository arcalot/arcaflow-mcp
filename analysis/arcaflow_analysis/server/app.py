"""gRPC server for the analysis service."""

from __future__ import annotations

import logging
import os
import socket
import sys
from concurrent import futures
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc

from arcaflow_analysis.server.http_server import AnalysisHTTPServer


def _add_generated_proto_path() -> Path:
    """Ensure generated protobuf modules are on sys.path."""
    repo_root = Path(__file__).resolve().parents[3]
    generated_path = repo_root / "api" / "generated" / "python"
    if str(generated_path) not in sys.path:
        sys.path.insert(0, str(generated_path))
    return generated_path


def _load_proto_modules() -> tuple[Any, Any]:
    """Load generated protobuf modules for the analysis service."""
    _add_generated_proto_path()
    try:
        import analysis_pb2
        import analysis_pb2_grpc
    except ImportError as exc:  # pragma: no cover - import error is surfaced in tests
        raise RuntimeError(
            "analysis protobuf modules not found; run scripts/proto-gen.sh"
        ) from exc
    return analysis_pb2, analysis_pb2_grpc


def _load_version() -> str:
    """Resolve the service version from the VERSION file."""
    repo_root = Path(__file__).resolve().parents[3]
    version_file = repo_root / "VERSION"
    try:
        return version_file.read_text(encoding="utf-8").strip()
    except FileNotFoundError:
        return "unknown"


def _resolve_bind_port(host: str, port: int) -> int:
    """Resolve an ephemeral port when port is set to 0."""
    if port != 0:
        return port
    addr_info = socket.getaddrinfo(host, 0, type=socket.SOCK_STREAM)
    family, socktype, proto, _, sockaddr = addr_info[0]
    with socket.socket(family, socktype, proto) as sock:
        sock.bind(sockaddr)
        return sock.getsockname()[1]


@dataclass(frozen=True)
class ServerConfig:
    """Configuration for the analysis service gRPC server."""

    host: str = "0.0.0.0"
    port: int = 50051
    max_workers: int = 8
    http_address: str | None = None


class AnalysisService:
    """Implements analysis RPCs."""

    def __init__(
        self,
        version: str,
        logger: logging.Logger,
        proto_module: Any,
    ) -> None:
        self._version = version
        self._logger = logger
        self._proto_module = proto_module

    def ping(self, request: Any, context: grpc.ServicerContext) -> Any:
        _ = context
        message = request.message or "pong"
        self._logger.info("analysis ping", extra={"message": message})
        return self._proto_module.PingResponse(
            message=message,
            server_version=self._version,
        )


class GrpcServer:
    """gRPC server for analysis operations with health checks."""

    def __init__(self, config: ServerConfig) -> None:
        self._config = config
        self._logger = logging.getLogger("arcaflow_analysis.server")
        self._server: grpc.Server | None = None
        self._http_server: AnalysisHTTPServer | None = None
        self.port: int | None = None
        self.http_port: int | None = None

    def start(self) -> None:
        """Start the gRPC server with health checks."""
        analysis_pb2, analysis_pb2_grpc = _load_proto_modules()
        _ = analysis_pb2

        self._server = grpc.server(
            futures.ThreadPoolExecutor(max_workers=self._config.max_workers)
        )
        service = AnalysisService(_load_version(), self._logger, analysis_pb2)

        class AnalysisServicer(analysis_pb2_grpc.AnalysisServiceServicer):
            def Ping(self, request: Any, context: grpc.ServicerContext) -> Any:
                return service.ping(request, context)

        analysis_pb2_grpc.add_AnalysisServiceServicer_to_server(
            AnalysisServicer(),
            self._server,
        )

        health_service = health.HealthServicer()
        health_pb2_grpc.add_HealthServicer_to_server(
            health_service,
            self._server,
        )
        health_service.set("", health_pb2.HealthCheckResponse.SERVING)
        service_name = analysis_pb2.DESCRIPTOR.services_by_name[
            "AnalysisService"
        ].full_name
        health_service.set(service_name, health_pb2.HealthCheckResponse.SERVING)

        port = _resolve_bind_port(self._config.host, self._config.port)
        bind_address = f"{self._config.host}:{port}"
        self.port = self._server.add_insecure_port(bind_address)
        if self.port == 0:
            raise RuntimeError(f"failed to bind {bind_address}")
        self._server.start()
        self._logger.info("analysis gRPC server started", extra={"port": self.port})

        if self._config.http_address:
            host, port = _split_host_port(self._config.http_address)
            self._http_server = AnalysisHTTPServer((host, port))
            self.http_port = self._http_server.port
            self._http_server.start()
            self._logger.info(
                "analysis HTTP server started",
                extra={"port": self.http_port},
            )

    def wait_for_termination(self) -> None:
        """Block until the gRPC server terminates."""
        if self._server is None:
            raise RuntimeError("gRPC server not started")
        self._server.wait_for_termination()

    def stop(self, grace: float = 5.0) -> None:
        """Stop the gRPC server."""
        if self._server is None:
            return
        self._server.stop(grace)
        self._logger.info("analysis gRPC server stopped")
        if self._http_server is not None:
            self._http_server.stop()


def configure_logging() -> None:
    """Configure structured logging for the analysis service."""
    log_level = os.getenv("ARCAFLOW_MCP_LOG_LEVEL", "INFO").upper()
    resolved_level = getattr(logging, log_level, logging.INFO)
    logging.basicConfig(
        level=resolved_level,
        format='{"level":"%(levelname)s","message":"%(message)s"}',
    )
    logging.getLogger().setLevel(resolved_level)


def main() -> None:
    """Entrypoint for the analysis service process."""
    configure_logging()
    http_address = os.getenv("ARCAFLOW_ANALYSIS_HTTP_ADDRESS")
    server = GrpcServer(ServerConfig(http_address=http_address))
    server.start()
    server.wait_for_termination()


if __name__ == "__main__":
    main()


def _split_host_port(address: str) -> tuple[str, int]:
    """Split host:port string into host and integer port."""
    if ":" not in address:
        raise ValueError("HTTP address must be host:port")
    host, port_text = address.rsplit(":", 1)
    return host, int(port_text)
