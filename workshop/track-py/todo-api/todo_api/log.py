import logging
import os

LEVEL = os.environ.get("LOG_LEVEL", "INFO").upper()

logging.basicConfig(level=LEVEL)

# Set the level on our own logger rather than relying on the root logger's.
# Under `opentelemetry-instrument` the root logger is configured before the
# application imports anything, which makes a later basicConfig() a no-op.
log = logging.getLogger("todo-api")
log.setLevel(LEVEL)
