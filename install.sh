#!/usr/bin/env bash
set -euo pipefail

echo "🔧 Building claude-chroma..."

# Build the Go binary
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"
go build -o claude-chroma ./cmd/claude-chroma

BINARY_PATH="$SCRIPT_DIR/claude-chroma"
echo "✅ Built: $BINARY_PATH"

# Determine Claude Code settings location
CLAUDE_SETTINGS="$HOME/.claude/settings.json"
mkdir -p "$HOME/.claude"

echo ""
echo "📋 To register hooks, merge the following into $CLAUDE_SETTINGS"
echo "   (or use /hooks inside Claude Code):"
echo ""
echo "   Replace /path/to/claude-chroma with:"
echo "   $BINARY_PATH"
echo ""
echo "   Example for a single event:"
echo '   "Stop": [{ "hooks": [{ "type": "command", "command": "'$BINARY_PATH'" }] }]'
echo ""
echo "   See example-settings.json for the full config."
echo ""
echo "🎮 Ensure Razer Synapse is running with Chroma SDK enabled."
echo "🚀 You're all set. Go build something, sir."
