# vinote - Terminal Note-Taking System

## Context

Marcin uses Silverbullet for notes but is frustrated with visual glitches. He wants to move his workflow to the terminal, but **gradually** — both tools must work in parallel on the same `~/notes` directory. Existing notes and Silverbullet templates stay untouched.

**Setup:** AstroNvim v5 (lazy.nvim, snacks.nvim), notes in `~/notes` as Markdown with YAML frontmatter.

**Key constraints:**
- Coexistence with Silverbullet — no changes to existing files or template syntax
- Ergonomic UX — everything inside Neovim, zero context-switching
- Go CLI for indexing and data; Neovim plugin (Lua) for UI

---

## Architecture

Two components working together:

```
┌─────────────────────────────────────────────────┐
│  Neovim + vinote.lua plugin                     │
│                                                 │
│  <leader>vw  Weekly view (dynamic, from index)  │
│  <leader>vo  Open note (snacks.picker)          │
│  <leader>vs  Search content (snacks.picker)     │
│  <leader>vn  New note                           │
│  <leader>vt  Topics (snacks.picker, filtered)   │
│  <leader>vb  Backlinks for current buffer       │
│  gf          Follow [[wikilink]]                │
│                                                 │
│  Uses snacks.nvim pickers, floats, splits       │
└──────────────┬──────────────────────────────────┘
               │ calls (shell out, reads JSON)
               ▼
┌─────────────────────────────────────────────────┐
│  vn (Go binary — CLI, no TUI)                   │
│                                                 │
│  vn index          build/refresh index          │
│  vn query          filter notes (JSON output)   │
│  vn weekly         create weekly note from tpl  │
│  vn weekly-view    dynamic weekly data (JSON)   │
│  vn backlinks      backlinks for a note (JSON)  │
│  vn resolve        resolve wikilink → path      │
└──────────────┬──────────────────────────────────┘
               │ reads
               ▼
        ~/notes/ (shared with Silverbullet)
```

**Coexistence model:** vinote is a **viewer, navigator, and launcher**. It reads the same frontmatter and wikilinks that Silverbullet uses, but never modifies existing files. For weekly notes, vinote creates new files from the SB template (preserving `${query[[...]]}` syntax intact). Dynamic views (meetings, topics) are built from the index — vinote does not parse SB query syntax. Both tools render the same underlying data independently.

**Why not a standalone TUI?** The user already lives in Neovim with AstroNvim v5. A bubbletea TUI would mean: exit nvim → launch TUI → pick note → exit TUI → open nvim. Instead, everything happens inside nvim via snacks.nvim pickers and floats — zero context-switching, and snacks already provides fuzzy matching, live grep, and preview for free.

---

## UX: Neovim Keybindings

All bindings under `<leader>v` prefix (v for vinote):

| Key | Action | Implementation |
|---|---|---|
| `<leader>vw` | Weekly view | Float/split with dynamic weekly data from index |
| `<leader>vo` | Open note | snacks.picker over note titles (from `vn index`) |
| `<leader>vs` | Search content | snacks.picker with live grep |
| `<leader>vn` | New note | Prompt for path, create file, open buffer |
| `<leader>vt` | Topics | snacks.picker filtered by tag:topic, not archived |
| `<leader>vb` | Backlinks | snacks.picker showing notes linking to current buffer |
| `gf` | Follow wikilink | Resolve `[[link]]` under cursor via `vn resolve`, open file |

### Flow: Weekly View (`<leader>vw`)
1. Calls `vn weekly-view --week=current` → gets JSON with meetings, topics for this week
2. If weekly note file doesn't exist, calls `vn weekly --create` first
3. Opens a float/split showing the dynamic view:
   - Meetings this week (from index: tag:meeting + date in range)
   - Active topics (from index: tag:topic, not archived)
4. Each item is selectable — Enter opens that note
5. `e` opens the weekly note file itself for editing

### Flow: Open Note (`<leader>vo`)
1. snacks.picker fuzzy search over all note titles
2. Source: `vn query --all --json` (cached, fast)
3. Select → opens in buffer

### Flow: Search (`<leader>vs`)
1. snacks.picker with live grep (built-in snacks functionality)
2. Select result → opens file at matching line

### Flow: New Note (`<leader>vn`)
1. Prompt for path (with completion on existing dirs)
2. Optionally select tags
3. Creates file with frontmatter, opens buffer

### Flow: Topics (`<leader>vt`)
1. snacks.picker over results of `vn query --tag=topic --not=archived --json`
2. Shows title + last modified
3. Select → opens in buffer

### Flow: Backlinks (`<leader>vb`)
1. Calls `vn backlinks --note=<current buffer path> --json`
2. snacks.picker over results
3. Select → opens in buffer

### CLI Shortcuts (optional, for use outside nvim)
- `vn w` — create weekly note if missing, print path (useful for `nvim $(vn w)`)
- `vn o <query>` — fuzzy match, print path
- `vn s <query>` — search, print matches

---

## Project Structure

```
vinote/                          (~/dev/vinote)
├── cmd/
│   └── main.go                  # entry point, cobra root command
├── internal/
│   ├── config/
│   │   └── config.go            # notes dir, editor, paths
│   ├── index/
│   │   ├── index.go             # scan ~/notes, build note list
│   │   └── frontmatter.go       # YAML frontmatter parser
│   ├── query/
│   │   └── query.go             # filter by tag, frontmatter fields, date, path
│   ├── weekly/
│   │   └── weekly.go            # weekly note creation from SB template
│   ├── wikilink/
│   │   └── wikilink.go          # parse [[links]], resolve paths, find backlinks
│   └── cli/
│       ├── index.go             # vn index command
│       ├── query.go             # vn query command
│       ├── weekly.go            # vn weekly + vn weekly-view commands
│       ├── backlinks.go         # vn backlinks command
│       └── resolve.go           # vn resolve command
├── plugin/
│   └── vinote.lua               # Neovim plugin (→ symlinked or copied to nvim config)
├── go.mod
└── go.sum
```

---

## Core Components

### Note Index (`internal/index/`)
```go
type Note struct {
    Path        string            // relative, e.g. "Allegro/Tematy/Pigeon"
    Title       string            // from H1, frontmatter title, or filename
    Tags        []string          // from frontmatter "tags" (comma-separated or list)
    Frontmatter map[string]any    // all frontmatter fields
    Wikilinks   []string          // outgoing [[links]]
    ModTime     time.Time
}
```
- Scans `~/notes/**/*.md` recursively (all subdirectories), skips `_plug/`, `Library/`, `.git/`
- Parses YAML frontmatter between `---` delimiters
- Extracts wikilinks via regex `\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`
- Cached as `~/.config/vinote/index.json` with mtime-based invalidation
- Rebuild on startup if any file is newer than cache
- **Performance:** uses `filepath.WalkDir` (no symlink stat overhead), sequential frontmatter parsing. Optimize only if needed.

### Query (`internal/query/`)
Standalone filter functions on `[]Note`:
- `ByTag(notes []Note, tag string) []Note` — notes with given tag
- `ByPath(notes []Note, prefix string) []Note` — notes under a path prefix
- `ByFrontmatter(notes []Note, key, value string) []Note` — match frontmatter field
- `NotFrontmatter(notes []Note, key string) []Note` — exclude notes with field set to true
- `ByDateRange(notes []Note, field string, from, to time.Time) []Note` — date-based filtering
- Composable via chaining: `ByTag(NotFrontmatter(notes, "archived"), "topic")`

### Weekly (`internal/weekly/`)

**Creating a new weekly note:**
```go
func CreateFromTemplate(templatePath string, weekStart time.Time, targetDir string) (string, error)
```
1. Read the SB template file (path from config: `weekly_template`)
2. Replace date placeholders (week start, prev/next week links)
3. Write to `{weekly_dir}/{date}.md` — file keeps `${query[[...]]}` syntax intact
4. Return path to created file

**Building weekly view data (dynamic, from index):**
```go
func WeeklyView(idx *index.Index, weekStart time.Time) WeeklyData
```
```go
type WeeklyData struct {
    Week       string   // "2026-W11"
    DateRange  string   // "Mar 9–15, 2026"
    FilePath   string   // path to weekly note file (or empty if not created)
    FileExists bool
    Meetings   []Note   // tag:meeting, date in week range
    Topics     []Note   // tag:topic, not archived
}
```
This does **not** parse `${query[[...]]}` from the file. It builds the view directly from index queries — same data, independent rendering. SB and vinote both query the same notes, each with their own view logic.

### Wikilink (`internal/wikilink/`)
- `Parse(content string) []string` — extract all wikilinks from markdown
- `Resolve(link string, notesDir string) string` — resolve to file path
- `Backlinks(idx *index.Index, notePath string) []Note` — find notes linking to given note

### CLI Commands (`internal/cli/`)
All commands output JSON (for consumption by the Neovim plugin) or plain text (for terminal use).

| Command | Output | Description |
|---|---|---|
| `vn index` | status message | Rebuild index, print stats |
| `vn query --tag=X --not=Y --path=Z --json` | JSON `[]Note` | Filter notes |
| `vn weekly --create` | file path | Create weekly note from template if missing |
| `vn weekly-view [--week=2026-W11]` | JSON `WeeklyData` | Dynamic weekly data from index |
| `vn backlinks <note-path>` | JSON `[]Note` | Notes linking to given note |
| `vn resolve <wikilink>` | file path | Resolve wikilink to absolute path |

---

## Neovim Plugin (`plugin/vinote.lua`)

Leverages snacks.nvim (already in AstroNvim v5) for pickers, and vim.fn.system / vim.fn.jobstart for calling `vn`.

Key design decisions:
- **Async index:** calls `vn` commands via `jobstart` to avoid blocking nvim
- **snacks.picker:** used for fuzzy note selection, search results, backlinks, topics
- **Weekly float:** custom float window rendering `WeeklyData` JSON as formatted markdown with selectable items
- **gf override:** in markdown buffers, intercept `gf` to resolve `[[wikilinks]]` via `vn resolve`

```lua
-- Simplified structure
return {
  "vinote",
  ft = "markdown",
  keys = {
    { "<leader>vw", function() require("vinote").weekly() end, desc = "Weekly view" },
    { "<leader>vo", function() require("vinote").open() end, desc = "Open note" },
    { "<leader>vs", function() require("vinote").search() end, desc = "Search notes" },
    { "<leader>vn", function() require("vinote").new() end, desc = "New note" },
    { "<leader>vt", function() require("vinote").topics() end, desc = "Topics" },
    { "<leader>vb", function() require("vinote").backlinks() end, desc = "Backlinks" },
  },
}
```

---

## Weekly Note Template

**Uses the existing Silverbullet template file** as the single source of truth for file creation. vinote reads this template (path from config: `weekly_template`), replaces date placeholders, and writes the new weekly note file with `${query[[...]]}` syntax preserved. This means:
- You only maintain one template (the SB one)
- SB renders queries dynamically in the browser as before
- vinote builds its weekly view from the **index** (not by parsing the file's query syntax)
- Two independent renderers, same underlying data

---

## Configuration

File: `~/.config/vinote/config.toml`

```toml
notes_dir = "~/notes"
editor = "nvim"

# paths relative to notes_dir
weekly_dir = "Allegro/Journal/Week"
weekly_template = "templates/Weekly.md"  # SB template file

# directories to skip during indexing
skip_dirs = ["_plug", "Library", ".git", "archive"]
```

---

## Dependencies

### Go binary
- `github.com/spf13/cobra` — CLI subcommands
- `gopkg.in/yaml.v3` — YAML frontmatter parsing
- `github.com/BurntSushi/toml` — config file

### Neovim plugin
- `snacks.nvim` (already in AstroNvim v5) — pickers, floats
- No additional nvim plugin dependencies

---

## Implementation Order

### Step 1: Project skeleton + config
- `go mod init`, cobra setup, config loading
- Binary name: `vn`

### Step 2: Note index
- Scan `~/notes`, parse frontmatter, extract wikilinks
- JSON cache with mtime invalidation
- `vn index` command

### Step 3: Query system
- Standalone filter functions by tag, frontmatter, date range, path prefix
- `vn query` command with JSON output

### Step 4: Wikilinks + backlinks
- Parse, resolve, find backlinks
- `vn backlinks` and `vn resolve` commands

### Step 5: Neovim plugin — core
- `<leader>vo` open note (snacks.picker + `vn query`)
- `<leader>vs` search (snacks.picker live grep)
- `<leader>vt` topics picker
- `<leader>vb` backlinks picker
- `gf` wikilink resolution

### Step 6: Weekly note creation
- Read existing SB template, create weekly note file (preserving `${query[[...]]}` syntax)
- `vn weekly --create` command
- `<leader>vw` — opens weekly note file

### Step 7: Weekly dynamic view
- `vn weekly-view` command (dynamic data from index)
- `<leader>vw` float with meetings and topics
- Selectable items → open note

### Step 8: Neovim plugin — new note
- `<leader>vn` new note flow

---

## Verification

1. `vn index` → builds index, reports note count and timing
2. `vn query --tag=topic --not=archived --json` → returns active topics (matches SB)
3. `vn weekly --create` → creates file from SB template, `${query[[...]]}` intact
4. Same file opens fine in Silverbullet (queries render dynamically as before)
5. `<leader>vw` in nvim → float shows meetings and topics for this week (from index)
6. Select a meeting in weekly view → opens that meeting note
7. `<leader>vo` → fuzzy search finds "Pigeon", Enter opens in buffer
8. `<leader>vs` → typing "rate limiting" shows matching files with preview
9. `<leader>vt` → lists active topics (matches what SB shows)
10. `<leader>vb` → shows backlinks for current buffer's note
11. `gf` on `[[Pigeon]]` → opens the Pigeon note
