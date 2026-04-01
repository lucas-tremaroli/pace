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
          A productivity CLI for humans and Claude
   -----------------------------------------------------
```

A terminal-based productivity tool designed to work hand-in-hand with [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Manage tasks, take Markdown notes, and let your AI assistant track its own work through a built-in MCP server.

> [!NOTE]
> **Opinionated by design** — Pace is built specifically for Claude Code integration. It prioritizes structured MCP communication over raw CLI usage, and keeps everything local in a `.pace/` directory alongside your code.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install lucas-tremaroli/tap/pace
```

### Go

```bash
go install github.com/lucas-tremaroli/pace@latest
```

## Getting Started

```bash
# Initialize project-specific storage (creates .pace/ in your repo)
pace init

# Install the MCP server into Claude Code
pace mcp install
```

That's it. Claude Code can now manage tasks and notes for your project through MCP tools automatically.

## How It Works

Pace exposes an **MCP server** (`pace mcp run`) that Claude Code uses to manage tasks and notes. When you run `pace mcp install`, it registers the server with Claude Code, giving it access to 18 structured tools for task management, note-taking, dependency tracking, and more.

This means Claude Code can:
- **Track its own work** — create tasks, update status, log progress, and mark things done
- **Manage dependencies** — model blocking relationships so it knows what's actionable
- **Write and reference notes** — persist specs, ADRs, and decisions across sessions
- **Understand project context** — run `pace_context` to immediately grasp the project state

All data lives in `.pace/` inside your repo — no external services, no API tokens, no network requests.

## For Humans: The TUI

While Claude Code interacts with Pace through MCP, you get an interactive terminal UI to see and manage everything.

```bash
pace tui
```

A unified dashboard showing your tasks and notes side by side with a detail panel. Create, edit, and move tasks, browse notes, and manage your project — all from one screen.

## Recharge

Pace also believes in breaks. Take advantage of the focus timer and some truly terrible jokes:

```bash
# Start a 25-minute focus session
pace tick

# Take a quick 5-minute break
pace tick -m 5

# Read a joke while you rest
pace joke
```

## Project Storage

Pace uses per-project storage to keep tasks and notes isolated to each repository.

```bash
# Initialize in current directory (creates .pace/)
pace init

# Check storage location and project state
pace context

# Migrate between global and project storage
pace migrate --from global --to project
```

**Storage resolution:** Pace searches upward from your current directory for `.pace/`. If not found, it falls back to `~/.config/pace/` (global storage).

## Configuration

```bash
# Set custom task ID prefix (e.g., "AUTH" -> AUTH-1, AUTH-2, ...)
pace config set id_prefix "AUTH"

# View config
pace config list
```

## Contributing

Contributions welcome! Please open an issue to discuss changes before submitting PRs.

## License

MIT

## Acknowledgments

Built with [Charmbracelet](https://charm.sh/) libraries: Bubbletea, Bubbles, and Lipgloss.
