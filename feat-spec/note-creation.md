# Contextual Note Creation

## Problem

Creating a new note requires typing the full path from scratch. When working within a note hierarchy, the user usually wants to create a note either "under" (child) or "beside" (sibling) the current note. Typing the common prefix manually is tedious and error-prone.

## Solution

Add two new commands that pre-fill the path prompt based on the current buffer's location in the notes tree.

### Commands

| Command | Keybinding | Prompt prefill | Example |
|---------|------------|---------------|---------|
| New note (generic) | `<leader>vnn` | _(empty)_ | `█` |
| New child note | `<leader>vnc` | current note path + `/` | `Foo/Bar/█` |
| New sibling note | `<leader>vns` | current note's parent + `/` | `Foo/█` |

### Behavior

1. Determine the current buffer's path relative to `notes_dir`, without the `.md` extension.
2. Compute the prefill:
   - **Child**: `<relative_path>/` (e.g. `Allegro/Tematy/Pigeon/`)
   - **Sibling**: `<parent_dir>/` (e.g. `Allegro/Tematy/`)
3. Open `vim.ui.input` with the prefill as `default` value. The cursor is placed at the end, so the user can immediately type the new note name or edit the path.
4. On confirm, create the file with standard frontmatter (same as existing `M.new`) and open it.

### Edge cases

- **Buffer is not inside `notes_dir`**: show a warning notification, do nothing.
- **Sibling at root level** (note path has no `/`): prefill is empty (equivalent to generic new note).
- **User clears the prefill**: works like a generic new note — any valid path is accepted.

### Keybinding migration

`<leader>vn` currently maps to the generic new note command. It will be replaced by the `<leader>vn` group:

- `<leader>vnn` — generic new note (same behavior as current `<leader>vn`)
- `<leader>vnc` — new child note
- `<leader>vns` — new sibling note

For which-key, the `<leader>vn` prefix should be labeled "New note".
