# claude-chroma

**Razer Chroma RGB lighting driven by Claude Code hook events.**

Your keyboard, mouse, and peripherals react in real-time to what Claude Code is doing — so you always know the state of your agent at a glance, even from across the room.

## Event → Color Mapping

| Event              | Effect                  | Color              | Meaning                          |
|--------------------|-------------------------|--------------------|----------------------------------|
| SessionStart       | Smooth pulse            | 🔵 Arc Reactor Blue | Claude is booting up             |
| SessionEnd         | Fade out                | 🔵 → Off           | Session over                     |
| UserPromptSubmit   | Brief flash             | 🔵 Cyan             | Prompt acknowledged              |
| PreToolUse         | Static hold             | 🟡 Yellow           | Tool executing...                |
| PostToolUse        | Brief flash             | 🟢 Green            | Tool completed                   |
| Stop               | Smooth pulse            | 🟢 Green            | Agent finished — task complete   |
| SubagentStop       | Brief flash             | 🟢 Green            | Subagent finished                |
| Notification:      |                         |                    |                                  |
|   permission_prompt| Triple alert flash + hold| 🔴 Red             | **NEEDS YOUR ATTENTION**         |
|   idle_prompt      | Pulse                   | 🟠 Orange           | Waiting for your input           |
| PreCompact         | Brief flash             | 🟣 Purple           | Context compacting               |

## Requirements

- **Windows** with Razer Synapse installed (Chroma SDK enabled)
- **Go 1.22+** to build
- **Claude Code** with hooks support

## Install

```bash
git clone https://github.com/you/claude-chroma.git
cd claude-chroma
bash install.sh
```

Then register the binary path in `~/.claude/settings.json` (see `example-settings.json`), or use `/hooks` inside Claude Code.

## How It Works

1. Claude Code fires a hook event → pipes JSON to the binary via stdin
2. The Go binary parses the event, initializes a Chroma SDK REST session on `localhost:54235`
3. Maps the event to an RGB effect (static, flash, pulse, alert)
4. Sends PUT requests to the Chroma SDK REST API for all devices
5. Maintains a heartbeat to keep the session alive during animations
6. Always exits 0 — never blocks Claude's workflow

## Customization

Edit `cmd/claude-chroma/main.go` to change the color mapping, add new effects, or target specific devices. The `internal/chroma/effects.go` file has `Flash`, `Pulse`, and `AlertFlash` — add your own.

## Architecture

```
stdin (JSON) → main.go → route by hook_event_name
                              ↓
                    internal/chroma/client.go
                         ↓           ↓
                    POST /init    PUT /effect
                         ↓           ↓
                  Razer Chroma SDK REST API (localhost:54235)
                         ↓
                    🌈 Your RGB peripherals
```

## License

MIT — do whatever you want with it.
