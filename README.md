# nudge

![nudge](nudge-dithered.png)

[![Go Reference](https://img.shields.io/badge/go-1.26-blue)](go.mod)

A quiet CLI companion for a better focus rhythm. nudge tracks one cycle —
**focus → pause → recover → return** — and gives you a well-timed cue at each
edge of it. You stay in control: nudge prompts, it never enforces.

## Why

Most CLI timers are either a blocking countdown you have to babysit, or a
full-screen TUI you leave open. nudge is neither:

- **No daemon.** State lives in a single JSON file, read and rewritten on
  every command — the same model `git` uses. The only background process is a
  short-lived watcher, and only while a cue is pending.
- **Commands return immediately.** `nudge in 25m` updates state and hands
  your shell back right away; the cue arrives later. `nudge await` and the
  overdue picker (below) are the two exceptions.
- **Transitions are manual.** A cue tells you time's up — it never changes
  your state for you.
- **A small vocabulary.** `in`, `out`, `later`, `done`, `status`, `prompt`,
  `loop`, `config`, `await`, `help`. That's the whole surface.

## Quickstart

```bash
go build -o nudge ./cmd/nudge
./nudge          # idle → starts a 25-minute focus session
./nudge status   # check remaining time without changing anything
./nudge out      # end focus, start a break
./nudge done     # stop tracking entirely
```

Every command prints a small status block, never a dashboard:

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

Calling a command in the wrong state is a quiet no-op with a one-line echo
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

`in` covers both starting fresh from idle and returning from a break — nudge
knows which one applies from the current state. `done` is the only way back
to idle, from either state, and fully stops tracking (including an active
loop).

### Loop escalation

`nudge loop start` arms a completed-session counter. While active, every Nth
session (`every`, default 4) takes a `long-break` instead of a regular one.
An explicit duration (`nudge out 2m`) always overrides this.

## Cues: bell, notification, and repeat

When a timed phase ends, nudge fires both a terminal bell (works everywhere,
including over SSH) and a native macOS notification — the bell alone is easy
to miss, since Terminal.app ships with it disabled by default. If nothing
acts on the cue, it repeats on a bounded schedule instead of firing once and
going quiet: `repeat` (default `5m`) and `repeat-limit` (default `6`), both
adjustable via `nudge config set`. Any real transition cancels the next
scheduled repeat.

## What's next?

If a session runs past its end, the next bare `nudge` call opens a small
picker ([`huh`](https://github.com/charmbracelet/huh)) instead of a stuck
`00:00` — offering just the transitions that make sense from there:

```
focus session done — what next?
> take a break
  stop for now
```

Only shown when both stdin and stdout are real terminals; a piped or
scripted call falls straight through to plain status. Cancelling (Esc/Ctrl-C)
leaves state untouched, same as any other no-op.

## Shell prompt embedding

`nudge prompt` prints a compact one-liner — `focus · 18:42` / `break · 04:51`
— or nothing at all when idle, so an optional prompt segment just disappears
when nudge isn't tracking anything:

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
cmd/nudge/       entrypoint: argument dispatch, orchestration, the
                 await/watch subprocess plumbing, and the overdue picker
internal/nudge/  pure domain model — Phase, State, and every transition
internal/store/  persists State to state.json, atomically
internal/config/ persists Config (durations, loop cadence, cue repeat) to
                 config.json
internal/watch/  the detached-process cue: bell + notification, repeated on
                 a bounded schedule, keyed off a NextCue timestamp
internal/render/ turns State/Config into nudge's text output — status,
                 no-op echoes, help, and the overdue picker's copy
```

## State and config files

| File | Location | Contents |
|---|---|---|
| `state.json` | `$XDG_STATE_HOME/nudge/` (falls back to `~/.local/state/nudge/`) | Live rhythm state — phase, timestamps, loop counter |
| `config.json` | `$XDG_CONFIG_HOME/nudge/` (falls back to `~/.config/nudge/`) | Durable defaults — durations, long-break cadence, cue repeat interval/limit |

Neither file needs to exist: missing state reads as fresh IDLE, and missing
config reads as nudge's built-in defaults (25m focus / 5m break / 15m long
break / every 4 sessions / repeat every 5m up to 6 times).
