"""notify-service — the downstream service, pre-instrumented.

Both tracks call it. Participants never edit this file.
"""

import logging
import random
import time

from flask import Flask, jsonify, request

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)


def _deliver(tags: list[str]) -> str:
    if "urgent" in tags:
        time.sleep(random.uniform(0.75, 0.85))
        return "pager"
    time.sleep(random.uniform(0.015, 0.030))
    return "digest"


@app.get("/healthz")
def healthz():
    return jsonify(ok=True)


@app.post("/notify")
def notify():
    body = request.get_json(silent=True) or {}
    channel = _deliver(body.get("tags", []))
    logging.info("notification queued %s via %s", body.get("id"), channel)
    return jsonify(queued=True, channel=channel)


@app.post("/stats-ping")
def stats_ping():
    time.sleep(random.uniform(0.05, 0.12))
    return jsonify(ok=True)
