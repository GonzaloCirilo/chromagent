# Chromagent Architecture

## Overview

Chromagent drives Razer Chroma RGB keyboard lighting based on AI coding agent events (Claude Code, Cursor). It runs as a **service/daemon** that holds the Chroma SDK session only while at least one AI agent session is active. When all sessions end, the SDK is released and Chroma Studio effects resume.

---

## Components

```
cmd/chromagent/main.go          — Binary entrypoint (two modes: serve / client)
internal/service/server.go      — Daemon: Unix socket HTTP server + session tracking
internal/chroma/client.go       — Chroma SDK REST client (session, heartbeat, effects)
internal/chroma/effects.go      — High-level effect helpers (Flash, Pulse, AlertFlash)
internal/chroma/types.go        — Enums, color helpers, grid dimensions
internal/agents/agent.go        — Canonical event types + adapter registry
internal/agents/claude/         — Claude Code hook parser
internal/agents/cursor/         — Cursor IDE hook parser
internal/hooks/events.go        — Hook input struct definitions (Claude Code)
internal/config/config.go       — User color config (%AppData%/chromagent/config.json)
```

---

## Data Flow

```
AI Agent Hook fires
        │
        ▼
chromagent (short-lived hook process)
  1. Read JSON from stdin
  2. Check Unix socket: os.TempDir()/chromagent.sock
     ├── Not ready → spawn `chromagent serve` detached, poll 100ms until ready
     └── Ready → skip
  3. POST http://chromagent/event (over Unix socket)
  4. Wait for response (effect completes in service)
  5. Exit 0
        │
        ▼
chromagent serve (long-running daemon)
  ┌─────────────────────────────────────────────┐
  │ /health   → 200 OK (readiness check)        │
  │ /event    → parse event, update sessions,   │
  │             dispatch Chroma effect          │
  └─────────────────────────────────────────────┘
        │
        ├── Session tracking (map[sessionID]struct{})
        │     EventSessionStart  → add to map, cancel shutdown timer
        │     EventSessionEnd    → remove from map; if empty → 3s shutdown timer
        │     Any other event    → if sessionID unknown, register it (mid-session start)
        │
        └── Chroma SDK (heartbeat every 10s, SDK timeout 15s)
              └── Effects: Flash, Pulse, AlertFlash, Static, Clear
```

---

## Unix Socket IPC

- **Socket path:** `%TEMP%\chromagent.sock` (Go: `os.TempDir() + "/chromagent.sock"`)
- **Protocol:** HTTP/1.1 over the socket (no TCP port needed)
- **Client transport:**
  ```go
  &http.Transport{
      DialContext: func(ctx, _, _) (net.Conn, error) {
          return net.Dial("unix", socketPath)
      },
  }
  ```
- **Why Unix socket over TCP:** No port conflicts, no firewall exposure, Windows 10+ native, auto-cleaned on shutdown.

---

## Session Lifecycle

```
[Agent starts]
   Hook: SessionStart → POST /event → add sessionID to map
   Service: init SDK (500ms delay), start heartbeat loop

[Agent working]
   Hooks: ToolExecuting, ToolCompleted, etc. → POST /event → RGB effects
   Service: heartbeat keeps SDK alive (10s interval, 15s SDK timeout)

[Agent ends]
   Hook: SessionEnd → POST /event → remove sessionID from map
   If map empty → 3-second grace timer starts
   After 3s (no new sessions) → SDK.Close() → socket removed → os.Exit(0)
   Chroma Studio effects resume immediately.

[Edge: service starts mid-session]
   If any hook fires with an unknown sessionID → registered automatically
   Prevents premature shutdown if SessionStart was missed.
```

---

## Daemon Auto-Start

The hook process (client mode) auto-starts the daemon transparently:

1. `net.DialTimeout("unix", socketPath, 500ms)` — fast check
2. If fails → `exec.Command(exe, "serve")` with `DETACHED_PROCESS | HideWindow`
3. Poll socket every 100ms, up to 10 seconds
4. Forward event once ready

---

## Chroma SDK Notes

- **URL:** `http://localhost:54235/razer/chromasdk`
- **Init:** POST app info → get session URI (e.g. `http://localhost:PORT/chromasdk`)
- **Device endpoints:** `{sessionURI}/keyboard` (NOT `/chromasdk/keyboard`)
- **Must wait 500ms after init** before sending effects
- **Heartbeat:** PUT `{sessionURI}/heartbeat` every 10s
- **Color format:** BGR (not RGB) — helper: `chroma.BGR(r, g, b)`
- **Result 126** = `ERROR_MOD_NOT_FOUND` → restart Chroma SDK services

---

## Config

User config: `%AppData%\chromagent\config.json` (auto-created with defaults on first run)

```json
{
  "events": {
    "session_start":       [80, 160, 255],
    "tool_executing":      [255, 255, 0],
    "needs_attention":     [255, 0, 0]
  }
}
```

Colors are RGB triplets. All event keys listed in `internal/config/config.go`.

---

## Adding a New Agent Adapter

1. Create `internal/agents/<name>/adapter.go`
2. Implement `agents.AgentAdapter` interface: `Name()`, `Detect()`, `ParseEvent()`
3. Call `agents.Register(&MyAdapter{})` in the package `init()` function
4. Add blank import in `internal/service/server.go`

---

## Building

```bash
export PATH="$PATH:/c/Program Files/Go/bin"
go build ./...
go build -o chromagent.exe ./cmd/chromagent
```
