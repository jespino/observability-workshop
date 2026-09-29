import os

import requests

NOTIFY_URL = os.environ.get("NOTIFY_URL", "http://notify-js:8084")
TIMEOUT_S = int(os.environ.get("NOTIFY_TIMEOUT_MS", "2000")) / 1000


def _post(path: str, body: dict) -> dict:
    response = requests.post(
        f"{NOTIFY_URL}{path}",
        json=body,
        headers={"content-type": "application/json"},
        timeout=TIMEOUT_S,
    )
    return response.json()


def notify_urgent(todo: dict) -> dict:
    return _post("/notify", {
        "id": todo["id"],
        "list": todo["list"],
        "title": todo["title"],
        "tags": todo["tags"],
    })


def notify_stats(summary: list) -> dict:
    return _post("/stats-ping", {"lists": len(summary)})
