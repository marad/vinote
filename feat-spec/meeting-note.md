# Meeting Note Creation

## Problem

Creating a meeting note in Silverbullet involves a template system that auto-fills the date, time, and places the file in the daily journal folder. In Neovim there's no equivalent — the user would have to manually create the file, type the frontmatter, and navigate to the right directory.

## Solution

Add a `new_meeting` command (`<leader>vnm`) that prompts for a meeting name and creates a fully-formed meeting note in the daily journal folder.

### Behavior

1. Prompt the user for the meeting name via `vim.ui.input`.
2. Compute the file path: `Allegro/Journal/Day/<YYYY-MM-DD>/<name>.md`.
3. Create the file with the following content:

```markdown
---
tags: meeting
date: <today>
hour: "<HH:MM>"
---
# Agenda
**Temat**:

# Uczestnicy
-

# Minutki
-

# Ustalenia
-
```

4. `hour` is the current time rounded down to the nearest 15 minutes.
5. Open the file and place the cursor at the end of the "**Temat**:" line in insert mode.

### Edge cases

- **File already exists**: open it without overwriting.
- **User cancels or enters empty name**: do nothing.
- **Daily directory doesn't exist yet**: create it with `mkdir -p`.

### Keybinding

| Command | Keybinding | Description |
|---------|------------|-------------|
| New meeting note | `<leader>vnm` | Create meeting note in today's journal |

Fits within the existing `<leader>vn` which-key group ("New note").
