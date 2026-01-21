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
   Focus Timer · Kanban Board · Notes Management · Jokes
   -----------------------------------------------------
```

## Installation

### Homebrew (macOS/Linux)

```bash
brew install lucas-tremaroli/tap/pace
```

### Go

```bash
go install github.com/lucas-tremaroli/pace@latest
```

## Usage

### Task Board

Launch an interactive Kanban board to manage your tasks:

```bash
pace task
```

![Task Demo](./.github/assets/task.demo.gif)

### Notes

Create and manage markdown notes:

```bash
# Open a TUI to browse and edit existing notes
pace note list

# Create a new note (opens in your default editor)
pace note create meeting-notes

# Create a note with content directly
pace note create todo -c "Review PRs"
```

![Notes Demo](./.github/assets/note.demo.gif)

### Focus Timer

Start a focus timer to help you stay productive:

```bash
# Start a 25-minute timer (default)
pace tick

# Start a custom duration timer
pace tick -m 45
```

![Focus Timer Demo](./.github/assets/tick.demo.gif)

### Dad Jokes

Get a random dad joke to brighten your day:

```bash
pace joke
```

![Joke Demo](./.github/assets/joke.demo.gif)
