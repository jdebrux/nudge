# nudge

[![Go Reference](https://img.shields.io/badge/go-1.26-blue)](go.mod)

A quiet CLI companion for building a healthier rhythm when focussing. nudge tracks one thing: **focus → pause
→ recover → return**, and gives you a small, well-timed cue at the edges of that cycle.
You stay in control throughout; nudge prompts, it never enforces.

```
nudge in/out/done  →  state.json updated  →  a detached watcher is scheduled for the
next cue  →  the command returns immediately  →  ...time passes...  →  the watcher
wakes, checks state.json is still current, and rings the terminal bell
```

## Why

Most CLI timers are either a single blocking countdown you have to babysit, or a
full-screen TUI you leave open. nudge is neither:

- **No daemon.** State is a single JSON file, re-read and rewritten on every
  invocation — the same model `git` uses. Nothing runs in the background except a
  short-lived process spawned only while a timed cue is actually pending.
- **Every command returns immediately.** `nudge in 25m` updates state and hands your
  shell back in milliseconds — the proactive bell arrives later via that detached
  watcher, not by nudge occupying your terminal. `nudge await -- <cmd>` is the one
  deliberate exception (see below).
- **Transitions are always manual.** A cue tells you a session's up; it never flips
  your state for you. Nudge prompts, it doesn't nag.
- **A small, closed vocabulary.** `in`, `out`, `later`, `done`, `status`, `loop`,
  `config`, `await`, `help` — that's the whole surface, on purpose.

## Quickstart

```bash
go build -o nudge ./cmd/nudge
./nudge          # idle → starts a 25-minute focus session
./nudge status   # check remaining time without changing anything
./nudge out      # end focus, start a break
./nudge done     # stop tracking entirely
```

Every command prints a small, restrained status block — never a dashboard:

```
  focus · 18:42

  ███████████████░░░░░░░
```

## Commands

| Command | Description |
|---|---|
| `nudge` | Show status, or start the default rhythm if idle |
| `nudge in [duration]` | Begin a focus period (from idle, or returning from a break) |
| `nudge out [duration]` | Begin a break |
| `nudge later` | Postpone the next cue by 5 minutes, without changing state |
| `nudge done` | Stop tracking entirely, back to idle |
| `nudge status` | What state am I in, time remaining |
| `nudge loop start` \| `stop` | Arm/disarm session counting and long-break escalation |
| `nudge config` | Show current defaults |
| `nudge config set <key> <value>` | Set a default — `focus`, `break`, `long-break`, `every` |
| `nudge await -- <cmd>` | Run `<cmd>`, break for as long as it takes, auto-return to focus when it exits |
| `nudge help` / `-h` / `--help` | Show the command list |

A command called in the wrong state is always a quiet no-op with a one-line echo
(`already on a break, 04:51 left`), never an error.

## The rhythm model

Three states, two verbs move you between them:

```
              nudge in
  ┌──────┐  ────────────►  ┌───────┐
  │ IDLE │                 │ FOCUS │
  └──────┘  ◄────────────  └───────┘
       ▲        nudge done      │
       │                    nudge out
  nudge done                    │
       │                        ▼
       │                   ┌───────┐
       └────────────────── │ REST  │
                            └───────┘
              nudge in (same verb, returning)
```

`in` covers both "start fresh from idle" and "return from a break" — nudge already
knows which one applies from the current state. `done` is the only way back to idle,
from either state, and fully stops tracking (including disarming an active loop).

### Loop escalation

`nudge loop start` arms a completed-session counter. While active, a default-duration
`out` counts the session, and every Nth one (`every`, default 4) uses `long-break`
instead of `break`. An explicit duration (`nudge out 2m`) always bypasses this — your
call is never silently overridden.

## Development

```bash
go build ./...                # build everything
go test ./... -v               # run the full test suite
go vet ./...                    # static checks
go build -o nudge ./cmd/nudge     # build the binary for manual testing
```

## Project layout

```
cmd/nudge/          entrypoint: argument dispatch, orchestration, the await/watch
                     subprocess plumbing
internal/nudge/      pure domain model — Phase, State, and every transition
                     (In/Out/Done/Later/StartLoop/StopLoop/CompletedFocusSession).
                     No file I/O, no CLI, no rendering.
internal/store/       persists State to state.json, atomically
internal/config/      persists Config (durations, loop cadence) to config.json
internal/watch/        the detached-process terminal-bell cue, keyed off a
                        NextCue timestamp so `later` can supersede a pending watcher
internal/render/        turns State/Config into nudge's small, restrained text output
```

## State and config files

| File | Location | Contents |
|---|---|---|
| `state.json` | `$XDG_STATE_HOME/nudge/` (falls back to `~/.local/state/nudge/`) | Live rhythm state — phase, timestamps, loop counter |
| `config.json` | `$XDG_CONFIG_HOME/nudge/` (falls back to `~/.config/nudge/`) | Durable defaults — durations, long-break cadence |

Neither file needs to exist: a missing state file reads as fresh IDLE, and a missing
config file reads as nudge's built-in defaults (25m focus / 5m break / 15m long break
/ every 4 sessions).
