from flask import Flask, jsonify

from todo_api.db import wait_for_dependencies
from todo_api.routes import bp


def create_app() -> Flask:
    app = Flask(__name__)
    app.register_blueprint(bp)

    @app.get("/healthz")
    def healthz():
        return jsonify(ok=True)

    # The frontend is served from a different origin. `traceparent` is allowed
    # so that the browser-instrumentation bonus in Module 7.4 works.
    @app.after_request
    def cors(response):
        response.headers["access-control-allow-origin"] = "*"
        response.headers["access-control-allow-headers"] = (
            "content-type,traceparent,tracestate,x-client"
        )
        response.headers["access-control-allow-methods"] = (
            "GET,POST,PATCH,DELETE,OPTIONS"
        )
        return response

    return app


wait_for_dependencies()
app = create_app()
