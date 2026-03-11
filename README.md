# vinote

Terminal note-taking system: a Go CLI (`vn`) paired with a Neovim plugin.

## Features

- **Fast index** — scans 1000 notes in ~100ms with mtime-based cache invalidation
- **Query & filter** — search notes by tags, frontmatter fields, with sorting by mtime/title/path
- **Weekly view** — floating window with meetings grouped by day, week navigation
- **Contextual note creation** — create child/sibling notes with pre-filled paths
- **Wikilinks** — `[[wikilink]]` resolution, backlinks discovery, and `gf` navigation in Neovim
- **Wikilink autocomplete** — `[[` completion via [blink.cmp](https://github.com/saghen/blink.cmp) source

## Installation

### CLI

```sh
go install github.com/marad/vinote/cmd/vn@latest
```

### Neovim plugin

With [lazy.nvim](https://github.com/folke/lazy.nvim):

```lua
{
  "marad/vinote",
  ft = "markdown",
  opts = {
    -- notes_dir = "~/notes",  -- default
  },
}
```

Requires [snacks.nvim](https://github.com/folke/snacks.nvim) for picker UI.

## CLI commands

| Command | Description |
|---------|-------------|
| `vn index` | Build/refresh the notes index |
| `vn query` | Query notes by tags and frontmatter |
| `vn backlinks <path>` | Find notes linking to a given note |
| `vn resolve <wikilink>` | Resolve a wikilink to a file path |
| `vn weekly` | Get weekly note metadata |
| `vn weekly-view` | Get weekly view data (meetings + topics) |

All commands output JSON to stdout.

## Keybindings

| Key | Action |
|-----|--------|
| `<leader>vo` | Open note (fuzzy picker) |
| `<leader>vs` | Search notes (live grep) |
| `<leader>vnn` | New note |
| `<leader>vnc` | New child note (under current) |
| `<leader>vns` | New sibling note (beside current) |
| `<leader>vt` | Topics |
| `<leader>vb` | Backlinks for current note |
| `<leader>vw` | Weekly view |
| `gf` | Follow `[[wikilink]]` under cursor (markdown buffers) |

## Configuration

`~/.config/vinote/config.toml`:

```toml
notes_dir = "~/notes"
```

## License

MIT
