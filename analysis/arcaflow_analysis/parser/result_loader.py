"""Load workflow result files from disk with caching."""

from __future__ import annotations

import json
from collections import OrderedDict
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml

ResultPayload = dict[str, Any] | list[Any] | str | None


@dataclass(frozen=True)
class ResultEntry:
    """Normalized result payload from a workflow result file."""

    path: Path
    format: str
    payload: ResultPayload
    raw_text: str


@dataclass(frozen=True)
class CacheKey:
    """Cache key derived from a result file's identity."""

    path: Path
    size: int
    modified_ns: int


class ResultCache:
    """Simple LRU cache keyed by file fingerprint."""

    def __init__(self, max_entries: int = 128) -> None:
        self._max_entries = max_entries
        self._entries: OrderedDict[CacheKey, ResultEntry] = OrderedDict()

    def get(self, key: CacheKey) -> ResultEntry | None:
        """Return a cached result entry if present."""
        entry = self._entries.get(key)
        if entry is None:
            return None
        self._entries.move_to_end(key)
        return entry

    def set(self, key: CacheKey, entry: ResultEntry) -> None:
        """Store a result entry in the cache."""
        self._entries[key] = entry
        self._entries.move_to_end(key)
        while len(self._entries) > self._max_entries:
            self._entries.popitem(last=False)


class ResultLoader:
    """Load workflow result files from disk with format detection."""

    def __init__(self, cache: ResultCache | None = None) -> None:
        self._cache = cache or ResultCache()

    def load_text(self, raw_text: str, format_hint: str | None = None) -> ResultEntry:
        """Parse raw result text without touching the filesystem."""
        format_name, payload = self._parse_payload(raw_text, format_hint or "")
        return ResultEntry(
            path=Path("<inline>"),
            format=format_name,
            payload=payload,
            raw_text=raw_text,
        )

    def load_file(self, path: str | Path) -> ResultEntry:
        """Load a workflow result file from disk."""
        result_path = Path(path)
        if not result_path.exists():
            raise FileNotFoundError(result_path)

        stat = result_path.stat()
        cache_key = CacheKey(
            path=result_path,
            size=stat.st_size,
            modified_ns=stat.st_mtime_ns,
        )
        cached = self._cache.get(cache_key)
        if cached is not None:
            return cached

        raw_text = result_path.read_text(encoding="utf-8")
        format_hint = result_path.suffix.lower().lstrip(".")
        format_name, payload = self._parse_payload(raw_text, format_hint)
        entry = ResultEntry(
            path=result_path,
            format=format_name,
            payload=payload,
            raw_text=raw_text,
        )
        self._cache.set(cache_key, entry)
        return entry

    def _parse_payload(
        self, raw_text: str, format_hint: str
    ) -> tuple[str, ResultPayload]:
        """Parse raw text based on format hints."""
        if format_hint in {"json"}:
            return "json", json.loads(raw_text)
        if format_hint in {"yaml", "yml"}:
            return "yaml", yaml.safe_load(raw_text)
        if format_hint in {"log", "txt"}:
            return "log", raw_text

        parsed = self._try_parse_structured(raw_text)
        if parsed is not None:
            return parsed
        return "log", raw_text

    def _try_parse_structured(self, raw_text: str) -> tuple[str, ResultPayload] | None:
        """Attempt to parse JSON or YAML from raw text."""
        try:
            return "json", json.loads(raw_text)
        except json.JSONDecodeError:
            pass
        try:
            return "yaml", yaml.safe_load(raw_text)
        except yaml.YAMLError:
            return None


def cache_key_for_path(path: Path) -> CacheKey:
    """Compute a cache key for a result file path."""
    stat = path.stat()
    return CacheKey(path=path, size=stat.st_size, modified_ns=stat.st_mtime_ns)
