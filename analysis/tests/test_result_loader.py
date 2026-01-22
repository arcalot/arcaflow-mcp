"""Tests for the result loader."""

# pylint: disable=import-error

from __future__ import annotations

from pathlib import Path

import pytest

from arcaflow_analysis.parser.result_loader import ResultLoader


def test_load_json_result(tmp_path: Path) -> None:
    result_file = tmp_path / "result.json"
    result_file.write_text('{"status": "ok"}', encoding="utf-8")

    loader = ResultLoader()
    entry = loader.load_file(result_file)

    assert entry.format == "json"
    assert entry.payload == {"status": "ok"}


def test_load_yaml_result(tmp_path: Path) -> None:
    result_file = tmp_path / "result.yaml"
    result_file.write_text("status: ok\n", encoding="utf-8")

    loader = ResultLoader()
    entry = loader.load_file(result_file)

    assert entry.format == "yaml"
    assert entry.payload == {"status": "ok"}


def test_load_log_result(tmp_path: Path) -> None:
    result_file = tmp_path / "result.log"
    result_file.write_text("line 1\nline 2\n", encoding="utf-8")

    loader = ResultLoader()
    entry = loader.load_file(result_file)

    assert entry.format == "log"
    assert entry.payload == "line 1\nline 2\n"


def test_cache_invalidation_on_change(tmp_path: Path) -> None:
    result_file = tmp_path / "result.json"
    result_file.write_text('{"status": "ok"}', encoding="utf-8")

    loader = ResultLoader()
    first = loader.load_file(result_file)
    result_file.write_text('{"status": "updated"}', encoding="utf-8")

    second = loader.load_file(result_file)

    assert first.payload != second.payload


def test_missing_file_raises(tmp_path: Path) -> None:
    loader = ResultLoader()
    missing_path = tmp_path / "missing.json"
    with pytest.raises(FileNotFoundError):
        loader.load_file(missing_path)


def test_load_text_parses_yaml() -> None:
    loader = ResultLoader()
    entry = loader.load_text("value: 4\n")
    assert entry.format == "yaml"
    assert entry.payload == {"value": 4}
