# Pace

Your command line buddy to make work actually enjoyable.

![Pace Logo](.github/PACE.png)

## Features

- **Task Board** - Kanban-style task management in your terminal
- **Notes** - Create and browse markdown notes with a beautiful TUI
- **Focus Timer** - Pomodoro-style timer to help you enter flow state
- **Dad Jokes** - Because everyone needs a laugh sometimes

## Installation

### Using Go

```bash
go install github.com/lucas-tremaroli/pace@latest
```

### From Source

```bash
git clone https://github.com/lucas-tremaroli/pace.git
cd pace
make install
```

## Usage

### Task Board

Launch an interactive Kanban board to manage your tasks:

```bash
pace task
```

### Notes

Create and manage markdown notes:

```bash
# Open a TUI to browse and edit existing notes
pace note list

# Create a new note (opens in your default editor)
pace note create meeting-notes

# Create a note with content directly
pace note create todo -c "Buy groceries"
```

### Focus Timer

Start a focus timer to help you stay productive:

```bash
# Start a 25-minute timer (default)
pace tick

# Start a custom duration timer
pace tick -m 45
```

### Dad Jokes

Get a random dad joke to brighten your day:

```bash
pace joke
```
