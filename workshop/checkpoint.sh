#!/usr/bin/env bash
# Catches you up to the end of an exercise.
#
#   ./checkpoint.sh 52        # end of exercise 5.2, for your track
#
# It copies the reference solution over your working directory. Your work is
# saved to .checkpoint-backup/ first, so nothing is lost. Restart your API
# container afterwards (Ctrl-C in its terminal, then ↑ Enter).
#
# If you are stuck for more than ten minutes, run this and keep moving.
# Nobody is judging.

source "$(dirname "$0")/scripts/common.sh"

CHECKPOINT="${1:-}"
TRACK="${TRACK:-}"

if [ -z "$CHECKPOINT" ]; then
  echo "usage: ./checkpoint.sh <NM>     e.g. ./checkpoint.sh 52" >&2
  echo >&2
  echo "Available checkpoints:" >&2
  for d in "$WORKSHOP_DIR"/solutions/*/; do
    [ -d "$d" ] || continue
    printf '  %s  %s\n' "$(basename "$d")" "$(head -n1 "$d/README.md" 2>/dev/null || true)" >&2
  done
  exit 2
fi

if [ -z "$TRACK" ]; then
  echo "Which track are you on?"
  select choice in "Node.js / Express" "Python / Flask"; do
    case "$REPLY" in
      1) TRACK=js; break ;;
      2) TRACK=py; break ;;
      *) echo "Pick 1 or 2." ;;
    esac
  done
fi

case "$TRACK" in
  js) TRACK_DIR="track-js/todo-api" ;;
  py) TRACK_DIR="track-py/todo-api" ;;
  *)  echo "TRACK must be js or py" >&2; exit 2 ;;
esac

SRC="$WORKSHOP_DIR/solutions/$CHECKPOINT/$TRACK"
if [ ! -d "$SRC" ]; then
  echo "❌ No solution for checkpoint $CHECKPOINT on the $TRACK track." >&2
  echo "   Looked in: solutions/$CHECKPOINT/$TRACK" >&2
  exit 1
fi

STAMP=$(date +%Y%m%d-%H%M%S)
BACKUP="$WORKSHOP_DIR/.checkpoint-backup/$STAMP-$TRACK"
mkdir -p "$BACKUP"
cp -R "$WORKSHOP_DIR/$TRACK_DIR/." "$BACKUP/"
echo "💾 Your work is backed up in .checkpoint-backup/$STAMP-$TRACK"

cp -R "$SRC/." "$WORKSHOP_DIR/$TRACK_DIR/"
echo "📋 Applied solutions/$CHECKPOINT/$TRACK"

# Some checkpoints also change the Collector config, which the whole stack shares.
COLLECTOR_CHANGED=0
if [ -d "$SRC/../collector" ]; then
  cp -R "$SRC/../collector/." "$WORKSHOP_DIR/collector/"
  echo "📋 Applied the Collector config for this checkpoint"
  COLLECTOR_CHANGED=1
fi

echo
echo "✅ You are at checkpoint $CHECKPOINT."
echo "   Restart your API to pick it up: Ctrl-C in its terminal, then ↑ Enter."
if [ "$COLLECTOR_CHANGED" = 1 ]; then
  echo "   This checkpoint also changed collector/config.yaml — restart the Collector"
  echo "   in its own terminal too: Ctrl-C, then ↑ Enter."
fi
if [ "$TRACK" = py ]; then
  echo "   If the checkpoint changed requirements.txt, re-run the pip install"
  echo "   command from Module 3.1 first."
else
  echo "   If the checkpoint changed package.json, re-run the npm install"
  echo "   command from Module 3.1 first."
fi
