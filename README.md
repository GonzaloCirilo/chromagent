# chromagent

**Razer Chroma RGB lighting driven by AI coding agents.**

Your keyboard, mouse, and peripherals react in real-time to what your AI agent is doing — flashing when tools run, pulsing on completion, alerting when it needs your attention. When the session ends, your Chroma Studio effects seamlessly resume.

Works with **Claude Code** and **Cursor**. New agents can be added by implementing a simple adapter interface.

---

## How It Works

chromagent runs as a **background service** that owns the Chroma SDK connection for as long as an AI agent session is active.

```
Your AI agent fires a hook
        ↓
chromagent (hook process, reads stdin JSON)
  • If daemon isn't running → auto-starts it in the background
  • Forwards the event to the daemon via local socket
  • Exits immediately (never blocks the agent)
        ↓
chromagent daemon (long-running background service)
  • Holds the Chroma SDK connection + sends heartbeats
  • Tracks active sessions
  • Dispatches RGB effects
  • When all sessions end → releases SDK → Chroma Studio resumes
```

The daemon auto-starts on first hook and shuts down 3 seconds after the last session ends. You never need to manage it manually.

---

## Requirements

- **Windows** with [Razer Synapse 3](https://www.razer.com/synapse-3) installed and Chroma SDK enabled
- **Go 1.22+** to build from source

---

## Installation

### 1. Build

```bat
git clone https://github.com/GonzaloCirilo/chromagent.git
cd chromagent
scripts\install.bat
```

This builds `chromagent.exe` in the project directory and prints the path you'll need for hook registration.

### 2. Register hooks

#### Claude Code

Add to `%APPDATA%\Claude\settings.json` (or use `/hooks` inside Claude Code):

```json
{
  "hooks": {
    "SessionStart":      [{ "hooks": [{ "type": "command", "command": "C:/path/to/chromagent.exe", "timeout": 10 }] }],
    "SessionEnd":        [{ "hooks": [{ "type": "command", "command": "C:/path/to/chromagent.exe", "timeout": 10 }] }],
    "UserPromptSubmit":  [{ "hooks": [{ "type": "command", "command": "C:/path/to/chromagent.exe", "timeout": 5  }] }],
    "PreToolUse":        [{ "matcher": "*", "hooks": [{ "type": "command", "command": "C:/path/to/chromagent.exe", "timeout": 5  }] }],
    "PostToolUse":       [{ "matcher": "*", "hooks": [{ "type": "command", "command": "C:/path/to/chromagent.exe", "timeout": 5  }] }],
    "Stop":              [{ "hooks": [{ "type": "command", "command": "C:/path/to/chromagent.exe", "timeout": 10 }] }],
    "SubagentStop":      [{ "hooks": [{ "type": "command", "command": "C:/path/to/chromagent.exe", "timeout": 10 }] }],
    "PreCompact":        [{ "hooks": [{ "type": "command", "command": "C:/path/to/chromagent.exe", "timeout": 5  }] }],
    "Notification": [
      { "matcher": "permission_prompt", "hooks": [{ "type": "command", "command": "C:/path/to/chromagent.exe", "timeout": 10 }] },
      { "matcher": "idle_prompt",       "hooks": [{ "type": "command", "command": "C:/path/to/chromagent.exe", "timeout": 10 }] }
    ]
  }
}
```

Replace `C:/path/to/chromagent.exe` with the path printed by `install.bat`. See `example-settings.json` for a standalone copy of this config.

#### Cursor

Configure Cursor's hook system to pipe events to the binary via stdin. The binary auto-detects Cursor events from the JSON shape — no flags needed.

### 3. Verify

Open a Claude Code session. Your keyboard should pulse blue on start. Run any tool and it should flash yellow → green. Done.

---

## Event → Effect Mapping

| Event              | Effect                    | Default Color       | Trigger                  |
|--------------------|---------------------------|---------------------|--------------------------|
| SessionStart       | Smooth pulse              | 🔵 Arc Reactor Blue | Agent session opened     |
| SessionEnd         | Fade out → off            | 🔵 → Off            | Agent session closed     |
| PromptAcknowledged | Brief flash               | 🩵 Cyan             | User prompt received     |
| ToolExecuting      | Static hold               | 🟡 Yellow           | Tool/action in progress  |
| ToolCompleted      | Brief flash               | 🟢 Green            | Tool/action done         |
| TaskComplete       | Smooth pulse              | 🟢 Green            | Agent finished task      |
| SubtaskComplete    | Brief flash               | 🟢 Green            | Subagent done            |
| NeedsAttention     | Triple alert flash + hold | 🔴 Red              | Needs your attention     |
| WaitingForInput    | Pulse                     | 🟠 Orange           | Idle, waiting for user   |
| ContextCompacting  | Brief flash               | 🟣 Purple           | Context compacting       |
| Notification       | Brief flash               | ⚪ White            | Generic notification     |

---

## Configuration

A config file with default colors is created on first run:

- **Windows:** `%AppData%\chromagent\config.json`
- **macOS:** `~/Library/Application Support/chromagent/config.json`
- **Linux:** `~/.config/chromagent/config.json`

Edit the `[R, G, B]` values to customize colors per event:

```json
{
  "events": {
    "session_start":       [80, 160, 255],
    "session_end":         [80, 160, 255],
    "prompt_acknowledged": [0, 255, 255],
    "tool_executing":      [255, 255, 0],
    "tool_completed":      [0, 255, 0],
    "task_complete":       [0, 255, 0],
    "subtask_complete":    [0, 255, 0],
    "needs_attention":     [255, 0, 0],
    "waiting_for_input":   [255, 165, 0],
    "context_compacting":  [128, 0, 255],
    "notification":        [255, 255, 255]
  }
}
```

---

## Troubleshooting

**No effects at all**
- Ensure Razer Synapse is running and Chroma SDK is enabled in its settings.
- Check that the binary path in your hook config is correct.

**Result 126 (`ERROR_MOD_NOT_FOUND`) in logs**
- Restart the Chroma SDK service:
  ```powershell
  Restart-Service 'Razer Chroma SDK Server' -Force
  Restart-Service 'Razer Chroma SDK Service' -Force
  ```

**Effects stop working mid-session**
- The daemon may have crashed. Check for a stale socket at `%TEMP%\chromagent.sock`.
- Delete it and the next hook will auto-restart the daemon.

**Chroma Studio effects don't resume after session ends**
- The daemon shuts down 3 seconds after the last `SessionEnd` hook fires.
- If `SessionEnd` never fires (e.g. Claude Code crashed), kill the daemon manually:
  ```powershell
  Get-Process chromagent | Stop-Process
  ```

---

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
4. Add a blank import in `internal/service/server.go`

The `Detect` method should check for JSON fields unique to your agent. `ParseEvent` maps raw events to canonical `AgentEvent` values.

---

## Chroma SDK Reference

This project uses the [Razer Chroma SDK REST API v4.0](https://assets.razerzone.com/dev_portal/REST/html/index.html).

| Endpoint | Method | Description |
|----------|--------|-------------|
| `http://localhost:54235/razer/chromasdk` | POST | Initialize session → returns session URI |
| `{uri}/heartbeat` | PUT | Keep session alive (15s timeout) |
| `{uri}/keyboard` | PUT | Apply effect to keyboard |
| `{uri}/mouse` | PUT | Apply effect to mouse |
| `{uri}/mousepad` | PUT | Apply effect to mousepad |
| `{uri}/headset` | PUT | Apply effect to headset |
| `{uri}/chromalink` | PUT | Apply effect to ChromaLink |
| `{uri}` | DELETE | Terminate session |

**Color format:** BGR packed integer — `(B << 16) | (G << 8) | R`

---

## License

MIT
