# Pace

```plaintext
        ___         ___           ___           ___
       /\  \       /\  \         /\__\         /\__\
      /::\  \     /::\  \       /:/  /        /:/ _/_
     /:/\:\__\   /:/\:\  \     /:/  /        /:/ /\__\
    /:/ /:/  /  /:/ /::\  \   /:/  /  ___   /:/ /:/ _/_
   /:/_/:/  /  /:/_/:/\:\__\ /:/__/  /\__\ /:/_/:/ /\__\
   \:\/:/  /   \:\/:/  \/__/ \:\  \ /:/  / \:\/:/ /:/  /
    \::/__/     \::/__/       \:\  /:/  /   \::/_/:/  /
     \:\  \      \:\  \        \:\/:/  /     \:\/:/  /
      \:\__\      \:\__\        \::/  /       \::/  /
       \/__/       \/__/         \/__/         \/__/

   -----------------------------------------------------
                  Built for humans and AI
   -----------------------------------------------------
```

A CLI productivity tool designed for developers **and** AI coding assistants. Manage tasks with a Kanban board, take markdown notes, and more.

## Why Pace?

Most productivity tools are built for humans clicking through GUIs. Pace is different—it's built CLI-first with structured JSON output, making it equally useful for:

- **Developers** who prefer terminal workflows and keyboard-driven interfaces
- **AI coding assistants** (Claude Code, Cursor, Copilot) that need to track work programmatically

## Features

- **JSON by default** — All commands output structured JSON, no flags needed
- **Project-specific storage** — Each repo gets its own `.pace/` directory
- **Task dependencies** — Model blocking relationships between tasks
- **Multiple task types** — `task`, `bug`, `feature`, `chore`, `docs`
- **Priority levels** — Urgent (1) through Low (4)
- **Markdown notes** — Store specs, ADRs, and documentation alongside tasks
- **Interactive TUI** — Kanban board and note picker for humans
- **Focus timer** — Pomodoro-style timer in the terminal

## Installation

### Homebrew (macOS/Linux)

```bash
brew install lucas-tremaroli/tap/pace
```

### Go

```bash
go install github.com/lucas-tremaroli/pace@latest
```

---

## For AI Coding Assistants

*This section is written from the perspective of Claude, an AI that uses Pace during coding sessions.*

### Why I Like Using Pace

When I work on a codebase, I need to track what I'm doing, what's blocked, and what decisions were made. Most humans use Linear, Jira, or GitHub Issues—but those require API tokens, network requests, and complex authentication. Pace gives me something simpler: a local SQLite database I can read and write with shell commands.

**What makes it work for me:**

1. **JSON output I can parse** — When I run `pace task list`, I get structured data back. No scraping terminal colors or parsing ASCII tables.

2. **Project-scoped context** — The `.pace/` directory travels with the repo. When I'm dropped into a new session, I run `pace info` and immediately understand the project state.

3. **Dependency tracking** — `pace task ready` shows me only unblocked tasks. I don't waste tokens figuring out what I can actually work on.

4. **Notes for specs** — I can write an ADR or spec to `pace note create`, then reference it later. This persists across sessions better than conversation context.

### My Typical Workflow

```bash
# First thing: understand the project state
pace info

# See what's ready to work on
pace task ready

# Before starting work, create or update a task
pace task create --title "Implement auth middleware" --type feature --priority 2

# Document decisions in notes
pace note create auth-approach -c "# Auth Approach\n\nUsing JWT because..."

# When blocked by something else
pace task dep add <blocker-id> <my-task-id>

# Mark progress
pace task update <id> --status in-progress
pace task update <id> --status done
```

### Example: Reading Project Context

```bash
$ pace info
{
  "success": true,
  "data": {
    "storage": { "type": "project", "path": "/repo/.pace" },
    "tasks": { "todo": 5, "in_progress": 2, "done": 12, "total": 19 },
    "notes": { "total": 4 },
    "config": { "id_prefix": "AUTH" }
  }
}
```

In one command, I know: this is a project with active work, there's some in-progress items, and tasks are prefixed with "AUTH".

### Example: Finding Actionable Work

```bash
$ pace task ready
[
  {
    "id": "AUTH-23",
    "title": "Add rate limiting to login endpoint",
    "status": "todo",
    "type": "feature",
    "priority": 2,
    "labels": ["security"]
  }
]
```

These tasks have no unresolved blockers—I can pick one and start.

### What's Coming Next

There are some features in the works that will make this even better for AI workflows.

---

## For Humans

*This section is written from my perspective as a human developer using Pace.*

I prefer to have everything on my terminal to minimize context switching, and Pace helps me stay organized whithout leaving the terminal. I can quickly jot down tasks and notes through the interactive TUI, and know that my AI coding assistant will be able to pick up where I left off later.

### Task Board

Launch an interactive Kanban board:

```bash
pace task tui
```

![Task Demo](./.github/assets/task.demo.gif)

### Notes

Create and manage markdown notes:

```bash
# Browse notes in TUI
pace note

# Create with editor
pace note create meeting-notes

# Create with content directly
pace note create todo -c "Review PRs"
```

![Notes Demo](./.github/assets/note.demo.gif)

### Recharging

I also believe that, to be productive, humans need breaks. We are not machines! Take advantage of Pace's focus timer and some truly terrible jokes:

```bash
# Start a 25-minute focus session
pace tick

# Take a quick 5-minute break to stretch
pace tick -m 5

# Read a joke while you rest
pace joke
```

---

## Project Storage

Pace supports per-project storage, keeping tasks and notes isolated to each repository.

```bash
# Initialize in current directory (creates .pace/)
pace init

# Check which storage you're using
pace status

# Migrate between global and project storage
pace migrate --from global --to project
```

**Storage resolution:** Pace searches upward from your current directory for `.pace/`. If not found, it falls back to `~/.config/pace/` (global storage).

---

## CLI Reference

### Core Commands

| Command | Description |
|---------|-------------|
| `pace task tui` | Launch Kanban TUI |
| `pace task list` | List all tasks (JSON) |
| `pace task create --title "..." --type feature` | Create a task |
| `pace task update <id> --status done` | Update task |
| `pace task ready` | Show unblocked tasks |
| `pace task dep add <blocker> <blocked>` | Add dependency |
| `pace note tui` | Launch note picker TUI |
| `pace note create <name> -c "content"` | Create note |
| `pace note read <name>` | Read note content |
| `pace info` | Project overview |
| `pace status` | Storage location |

### Task Flags

- `--status`: `todo`, `in-progress`, `done`
- `--type`: `task`, `bug`, `feature`, `chore`, `docs`
- `--priority`: `1` (urgent), `2` (high), `3` (normal), `4` (low)
- `--label`: string tag (repeatable)

---

## Configuration

```bash
# Set custom task ID prefix
pace config set id_prefix "AUTH"

# View config
pace config list
```

---

## Contributing

Contributions welcome! Please open an issue to discuss changes before submitting PRs.

## License

MIT

## Acknowledgments

Built with [Charmbracelet](https://charm.sh/) libraries: Bubbletea, Bubbles, and Lipgloss.
