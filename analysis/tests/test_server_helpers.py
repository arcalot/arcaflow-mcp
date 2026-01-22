"""Tests for analysis server helpers."""

from __future__ import annotations

# pylint: disable=import-error

# pylint: disable=import-error

import sys

import pytest

from arcaflow_analysis.server import app

# pylint: disable=protected-access


def test_split_host_port() -> None:
    host, port = app._split_host_port("127.0.0.1:8081")
    assert host == "127.0.0.1"
    assert port == 8081


def test_split_host_port_invalid() -> None:
    with pytest.raises(ValueError, match="host:port"):
        app._split_host_port("invalid")


def test_load_version_reads_file() -> None:
    version = app._load_version()
    assert version


def test_add_generated_proto_path() -> None:
    path = app._add_generated_proto_path()
    assert str(path) in sys.path


def test_load_proto_modules() -> None:
    analysis_pb2, analysis_pb2_grpc = app._load_proto_modules()
    assert hasattr(analysis_pb2, "PingRequest")
    assert hasattr(analysis_pb2_grpc, "AnalysisServiceStub")
