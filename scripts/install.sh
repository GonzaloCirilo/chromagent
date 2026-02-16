#!/usr/bin/env bash
set -euo pipefail

echo "Building chromagent..."

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

go build -o chromagent ./cmd/chromagent

BINARY_PATH="$PROJECT_DIR/chromagent"
echo "Built: $BINARY_PATH"

CLAUDE_SETTINGS="$HOME/.claude/settings.json"
mkdir -p "$HOME/.claude"

echo ""
echo "To register hooks, merge the following into $CLAUDE_SETTINGS"
echo "  (or use /hooks inside Claude Code):"
echo ""
echo "  Replace /path/to/chromagent with:"
echo "  $BINARY_PATH"
echo ""
echo "  Example for a single event:"
echo '  "Stop": [{ "hooks": [{ "type": "command", "command": "'"$BINARY_PATH"'" }] }]'
echo ""
echo "  See example-settings.json for the full config."
echo ""
echo "Ensure Razer Synapse is running with Chroma SDK enabled."
