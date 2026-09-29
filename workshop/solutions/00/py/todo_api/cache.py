"""A small in-process cache in front of the list read path.

No network, no extra container: just a dict with time-to-live entries.
"""

import time

TTL_SECONDS = 30

_store: dict[str, tuple[float, object]] = {}


def cache_key(list_name: str) -> str:
    return f"todos:summary:{list_name}:{int(time.time() * 1000)}"


def read_summary(list_name: str):
    key = cache_key(list_name)
    entry = _store.get(key)
    fresh = entry is not None and entry[0] > time.time()
    return fresh, key, (entry[1] if fresh else None)


def write_summary(list_name: str, value) -> str:
    key = cache_key(list_name)
    _store[key] = (time.time() + TTL_SECONDS, value)
    return key


def invalidate(list_name: str) -> None:
    prefix = f"todos:summary:{list_name}:"
    for key in [k for k in _store if k.startswith(prefix)]:
        del _store[key]


def size() -> int:
    return len(_store)
