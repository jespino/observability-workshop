# Business logic with no library instrumentation of its own.
# Module 7.2 wraps this in the first custom span.

import re
from datetime import datetime, timezone

PRIORITY_LABELS = {1: "high", 2: "normal", 3: "low"}


def enrich(todo) -> dict:
    title = todo.title or ""

    first_word, *rest = title.split()
    slug = f"{first_word[0].upper()}-{re.sub(r'[^a-z0-9]+', '-', title.strip().lower())[:24]}"

    overdue = (
        todo.due_at is not None
        and todo.completed_at is None
        and todo.due_at < datetime.now(timezone.utc)
    )

    data = todo.to_dict()
    data.update(
        slug=slug,
        priority_label=PRIORITY_LABELS.get(todo.priority, "normal"),
        overdue=overdue,
        source="enricher",
    )
    return data
