# chromagent

**Razer Chroma RGB lighting driven by AI coding agents.**

Your keyboard, mouse, and peripherals react in real-time to what your AI agent is doing — so you always know its state
at a glance, even from across the room.

Works with **Claude Code**, **Cursor**, and any agent that pipes JSON events to stdin. New agents can be added by
implementing a simple adapter interface — no CLI flags needed, the binary auto-detects the agent from the JSON shape.

## Supported Agents

| Agent       | Detection                          | Status          |
|-------------|------------------------------------|-----------------|
| Claude Code | `hook_event_name` field present    | Fully supported |
| Cursor      | `event` + `conversation_id` fields | Fully supported |
| Others      | Implement `AgentAdapter` interface | Extensible      |

## Event → Color Mapping

All agents map to a set of canonical events with default colors. Colors are fully customizable via config file.

| Event              | Effect                    | Default Color       | Meaning                  |
|--------------------|---------------------------|---------------------|--------------------------|
| SessionStart       | Smooth pulse              | 🔵 Arc Reactor Blue | Agent booting up         |
| SessionEnd         | Fade out                  | 🔵 → Off            | Session over             |
| PromptAcknowledged | Brief flash               | 🔵 Cyan             | User input received      |
| ToolExecuting      | Static hold               | 🟡 Yellow           | Tool/action in progress  |
| ToolCompleted      | Brief flash               | 🟢 Green            | Tool/action done         |
| TaskComplete       | Smooth pulse              | 🟢 Green            | Agent finished task      |
| SubtaskComplete    | Brief flash               | 🟢 Green            | Subagent/subtask done    |
| NeedsAttention     | Triple alert flash + hold | 🔴 Red              | **Needs your attention** |
| WaitingForInput    | Pulse                     | 🟠 Orange           | Idle, waiting for user   |
| ContextCompacting  | Brief flash               | 🟣 Purple           | Compacting context       |
| Notification       | Brief flash               | ⚪ White             | Generic notification     |

## Chroma SDK REST API

This project communicates with the [Razer Chroma SDK REST API v4.0](https://assets.razerzone.com/dev_portal/REST/html/index.html).

**Endpoints:**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `http://localhost:54235/razer/chromasdk` | POST | Initialize session (returns session URI) |
| `{uri}/heartbeat` | PUT | Keep session alive (must be sent within 15s timeout) |
| `{uri}/keyboard` | PUT/POST | Apply effect to keyboard |
| `{uri}/mouse` | PUT/POST | Apply effect to mouse |
| `{uri}/mousepad` | PUT/POST | Apply effect to mousepad |
| `{uri}/headset` | PUT/POST | Apply effect to headset |
| `{uri}/keypad` | PUT/POST | Apply effect to keypad |
| `{uri}/chromalink` | PUT/POST | Apply effect to ChromaLink devices |
| `{uri}/effect` | PUT | Apply a previously created effect by ID |
| `{uri}/effect` | DELETE | Remove a previously created effect by ID |
| `{uri}` | DELETE | Terminate session |

**Effect types:** `CHROMA_NONE`, `CHROMA_STATIC`, `CHROMA_CUSTOM`, `CHROMA_CUSTOM2`, `CHROMA_CUSTOM_KEY`

PUT applies the effect immediately. POST creates the effect and returns an effect ID for later use.

**Color format:** BGR (Blue-Green-Red) packed as a single integer: `B<<16 | G<<8 | R`

**Device grid dimensions:**

| Device | Grid |
|--------|------|
| Keyboard | 6 rows × 22 cols |
| Mouse | 9 rows × 7 cols |
| Mousepad | 15 LEDs |
| Headset | 5 LEDs |
| Keypad | 4 rows × 5 cols |
| ChromaLink | 5 elements |

Full API reference: https://assets.razerzone.com/dev_portal/REST/html/index.html

## Requirements

- **Windows** with Razer Synapse installed (Chroma SDK enabled)
- **Go 1.22+** to build

## Install

```bash
git clone https://github.com/GonzaloCirilo/agent-chroma.git
cd agent-chroma
bash install.sh
```

### Claude Code

Register the binary path in `~/.claude/settings.json` (see `example-settings.json`), or use `/hooks` inside Claude Code.

### Cursor

Configure Cursor's hook system to pipe events to the binary via stdin. The binary auto-detects Cursor events from the
JSON shape — no flags needed.

## How It Works

1. Your AI agent fires an event → pipes JSON to the binary via stdin
2. The binary auto-detects which agent sent the event (by JSON shape)
3. The agent adapter maps the raw event to a canonical `AgentEvent`
4. The canonical event is routed to an RGB effect (static, flash, pulse, alert)
5. Effect colors are read from the user config (with sensible defaults)
6. PUT requests hit the Chroma SDK REST API on `localhost:54235` for all devices
7. Always exits 0 — never blocks the agent's workflow

## Configuration

On first run, a config file is created with default colors at:

- **macOS**: `~/Library/Application Support/agent-chroma/config.json`
- **Linux**: `~/.config/agent-chroma/config.json`
- **Windows**: `%AppData%/agent-chroma/config.json`

Edit the RGB values `[R, G, B]` to customize colors per event:

```json
{
	"events": {
		"session_start": [
			80,
			160,
			255
		],
		"session_end": [
			80,
			160,
			255
		],
		"prompt_acknowledged": [
			0,
			255,
			255
		],
		"tool_executing": [
			255,
			255,
			0
		],
		"tool_completed": [
			0,
			255,
			0
		],
		"task_complete": [
			0,
			255,
			0
		],
		"subtask_complete": [
			0,
			255,
			0
		],
		"needs_attention": [
			255,
			0,
			0
		],
		"waiting_for_input": [
			255,
			165,
			0
		],
		"context_compacting": [
			128,
			0,
			255
		],
		"notification": [
			255,
			255,
			255
		]
	}
}
```

## Adding a New Agent

1. Create `internal/agents/youragent/adapter.go`
2. Implement the `AgentAdapter` interface:

```go
type AgentAdapter interface {
Name() string
Detect(raw json.RawMessage) bool
ParseEvent(raw json.RawMessage) (agents.AgentEvent, error)
}
```

3. Register via `init()`: `agents.Register(&Adapter{})`
4. Add a blank import in `cmd/agent-chroma/main.go`

The `Detect` method should check for JSON fields unique to your agent. `ParseEvent` maps agent-specific events to
canonical `AgentEvent` values.

## Architecture

```
stdin (JSON) → auto-detect agent → adapter.ParseEvent()
                                        ↓
                                  AgentEvent (canonical)
                                        ↓
                              config.Load() → color lookup
                                        ↓
                              handleEvent() → Chroma effect
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
