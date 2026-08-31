# nudge

[![Go Reference](https://img.shields.io/badge/go-1.26-blue)](go.mod)

A quiet CLI companion for building a healthier rhythm when focussing. nudge tracks one thing: **focus → pause
→ recover → return**, and gives you a small, well-timed cue at the edges of that cycle.
You stay in control throughout; nudge prompts, it never enforces.

```
nudge in/out/done  →  state.json updated  →  a detached watcher is scheduled for the
next cue  →  the command returns immediately  →  ...time passes...  →  the watcher
fires (bell + native notification) and keeps re-firing on a bounded schedule until you
act  →  next time you run bare `nudge`, if the session ran past its end, you're offered
a quick picker for what happens next
```

## Why

Most CLI timers are either a single blocking countdown you have to babysit, or a
full-screen TUI you leave open. nudge is neither:

- **No daemon.** State is a single JSON file, re-read and rewritten on every
  invocation — the same model `git` uses. Nothing runs in the background except a
  short-lived process spawned only while a timed cue is actually pending.
- **Every command returns immediately.** `nudge in 25m` updates state and hands your
  shell back in milliseconds — the cue arrives later via that detached watcher, not by
  nudge occupying your terminal. `nudge await -- <cmd>` and the overdue picker (below)
  are the two deliberate exceptions.
- **Transitions are always manual.** A cue tells you a session's up; it never flips
  your state for you. Nudge prompts, it doesn't nag.
- **A small, closed vocabulary.** `in`, `out`, `later`, `done`, `status`, `prompt`,
  `loop`, `config`, `await`, `help` — that's the whole surface, on purpose.

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

  ███████████████░░░░░░░ 68%
```

## Commands

| Command | Description |
|---|---|
| `nudge` | Show status, start the default rhythm if idle, or offer a picker if the last session ran past its end |
| `nudge in [duration]` | Begin a focus period (from idle, or returning from a break) |
| `nudge out [duration]` | Begin a break |
| `nudge later` | Postpone the next cue by 5 minutes, without changing state |
| `nudge done` | Stop tracking entirely, back to idle |
| `nudge status` | What state am I in, time remaining |
| `nudge prompt` | One-line plain-text status for embedding in a shell prompt/status bar |
| `nudge loop start` \| `stop` | Arm/disarm session counting and long-break escalation |
| `nudge config` | Show current defaults |
| `nudge config set <key> <value>` | Set a default — `focus`, `break`, `long-break`, `every`, `repeat`, `repeat-limit` |
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

## Cues: bell, notification, and repeat

When a timed phase ends, nudge fires two signals, not one: the terminal bell (works
everywhere, including over SSH) and a native macOS notification (`osascript display
notification`) — added after discovering the bell alone is easy to miss, since
Terminal.app ships with it disabled by default. If nothing acts on it, the cue repeats
on a bounded schedule instead of firing once and going quiet — `repeat` (default `5m`)
and `repeat-limit` (default `6`), both adjustable via `nudge config set`. Any real
transition — `in`, `out`, `done`, or `later` — supersedes the next scheduled repeat.

## What's next?

The next time you run bare `nudge` after a session has run past its end, instead of a
stuck `00:00` you get a small interactive picker
([`huh`](https://github.com/charmbracelet/huh)) offering the two transitions the state
machine actually supports from there — the natural next step (a break from FOCUS, back
to focus from REST) or stopping for now:

```
focus session done — what next?
> take a break
  stop for now
```

This only appears when both stdin and stdout are real terminals — a piped or scripted
invocation falls straight through to plain status instead. Cancelling (Esc/Ctrl-C)
leaves state untouched, same as every other no-op in nudge.

## Shell prompt embedding

`nudge prompt` prints a compact, unstyled one-liner — `focus · 18:42` / `break · 04:51`
— or nothing at all when idle, so an optional prompt segment just disappears when
nudge isn't tracking anything:

```bash
PS1='$(nudge prompt) $ '
```

(or wire it into a starship custom command, tmux `status-right`, etc.)

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
                     subprocess plumbing, and the interactive overdue picker
internal/nudge/      pure domain model — Phase, State, and every transition
                     (In/Out/Done/Later/Overdue/StartLoop/StopLoop/
                     CompletedFocusSession). No file I/O, no CLI, no rendering.
internal/store/       persists State to state.json, atomically
internal/config/      persists Config (durations, loop cadence, cue repeat
                        cadence) to config.json
internal/watch/         the detached-process cue: terminal bell + native
                        notification, repeated on a bounded schedule, keyed off
                        a NextCue timestamp so `later` can supersede a pending
                        watcher
internal/render/         turns State/Config into nudge's small, restrained text
                          output — status views, no-op echoes, help, and the
                          overdue picker's copy
```

## State and config files

| File | Location | Contents |
|---|---|---|
| `state.json` | `$XDG_STATE_HOME/nudge/` (falls back to `~/.local/state/nudge/`) | Live rhythm state — phase, timestamps, loop counter |
| `config.json` | `$XDG_CONFIG_HOME/nudge/` (falls back to `~/.config/nudge/`) | Durable defaults — durations, long-break cadence, cue repeat interval/limit |

Neither file needs to exist: a missing state file reads as fresh IDLE, and a missing
config file reads as nudge's built-in defaults (25m focus / 5m break / 15m long break /
every 4 sessions / repeat every 5m up to 6 times).
