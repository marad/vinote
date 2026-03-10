# vinote - Terminal Note-Taking System

## Context

Marcin uses Silverbullet for notes but is frustrated with visual glitches. He wants to move his workflow to the terminal, but **gradually** - both tools must work in parallel on the same `~/notes` directory. Existing notes and Silverbullet templates stay untouched.

**Setup:** AstroNvim v5 (lazy.nvim, snacks.nvim), notes in `~/notes` as Markdown with YAML frontmatter.

**Key constraints:**
- Coexistence with Silverbullet - no changes to existing files or template syntax
- Ergonomic UX - interactive TUI, not long CLI commands
- Go language, single binary

---

## Architecture

```
┌─────────────────────────────────────────┐
│  vn (Go binary, bubbletea TUI)          │
│                                         │
│  [w] Weekly  [o] Open  [s] Search       │
│  [n] New     [t] Topics [b] Backlinks   │
│                                         │
│  ┌─────────────────────────────────┐    │
│  │  internal/index   - note index  │    │
│  │  internal/query   - tag/meta    │    │
│  │  internal/weekly  - gen weekly  │    │
│  │  internal/tui     - bubbletea   │    │
│  └─────────────────────────────────┘    │
└──────────────┬──────────────────────────┘
               │ reads
               ▼
        ~/notes/ (shared)
               ▲
               │ reads/writes
┌──────────────┴──────────────────────────┐
│  Silverbullet (browser)                 │
│  ${query[[...]]} renders dynamically    │
└─────────────────────────────────────────┘
```

**Coexistence model:** vinote is a **viewer, navigator, and launcher** — it reads the same frontmatter and wikilinks that Silverbullet uses, but never duplicates SB's template logic. For weekly notes, vinote creates new files by copying the existing SB template (preserving `${query[[...]]}` syntax), then shows a rendered preview in the TUI (resolving queries via its own index). The actual file keeps SB syntax so both tools work with the same file. Pressing Enter opens the file in nvim for editing. Single source of truth = the SB template file.

---

## UX: Interactive TUI

Entry point: `vn` (short alias, binary name)

### Main Menu
```
┌─ vinote ──────────────────────────────┐
│                                       │
│  [w] Weekly note       (ten tydzień)  │
│  [o] Open note         (fuzzy)        │
│  [s] Search content    (ripgrep)      │
│  [n] New note                         │
│  [t] Topics            (5 aktywnych)  │
│  [b] Backlinks                        │
│  [q] Quit                             │
│                                       │
└───────────────────────────────────────┘
```

### Flow: Weekly Note (`w`)
1. If weekly note for current week exists → shows rendered preview (queries resolved from index)
2. If it doesn't exist → creates it by copying the SB template file, shows rendered preview
3. Press Enter → opens in `$EDITOR` (nvim)

### Flow: Open Note (`o`)
1. Fuzzy finder over all note titles (inline, bubbletea-based)
2. Select → opens in nvim

### Flow: Search (`s`)
1. Type query → live ripgrep results
2. Select result → opens in nvim at matching line

### Flow: New Note (`n`)
1. Prompts for path (with autocomplete on existing dirs)
2. Optionally select template and tags
3. Creates file, opens in nvim

### Flow: Topics (`t`)
1. Lists active topics (tag: topic, not archived) with last modified date
2. Select → opens in nvim

### Flow: Backlinks (`b`)
1. Prompts for note (fuzzy) or uses current context
2. Shows list of notes linking to it
3. Select → opens in nvim

### Direct subcommands (optional shortcuts)
Also support `vn w`, `vn o`, `vn s query` for power users who prefer CLI.

---

## Project Structure

```
vinote/                          (~/dev/vinote)
├── cmd/
│   └── main.go                  # entry point
├── internal/
│   ├── config/
│   │   └── config.go            # notes dir, editor, paths
│   ├── index/
│   │   ├── index.go             # scan ~/notes, build note list
│   │   └── frontmatter.go       # YAML frontmatter parser
│   ├── query/
│   │   └── query.go             # filter by tag, frontmatter fields, date, path
│   ├── weekly/
│   │   └── weekly.go            # weekly note creation (from SB template) + preview rendering
│   ├── wikilink/
│   │   └── wikilink.go          # parse [[links]], resolve paths, find backlinks
│   └── tui/
│       ├── app.go               # main bubbletea app model
│       ├── menu.go              # main menu view
│       ├── picker.go            # fuzzy note picker (reusable)
│       ├── search.go            # ripgrep search view
│       ├── weekly.go            # weekly preview + confirm view
│       └── styles.go            # lipgloss styles
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
- Cached as `.vinote/index.json` with mtime-based invalidation
- Rebuild on startup if any file is newer than cache
- **Performance:** uses `filepath.WalkDir` (no symlink stat overhead), parallel frontmatter parsing via goroutine pool, and incremental cache updates (only re-parse files with changed mtime). Tested target: <500ms for ~5000 notes

### Query (`internal/query/`)
Filter functions on `[]Note`:
- `ByTag(tag string)` - notes with given tag
- `ByPath(prefix string)` - notes under a path prefix
- `ByFrontmatter(key, value string)` - match frontmatter field
- `NotFrontmatter(key string)` - exclude notes with field set to true
- `ByDateRange(field string, from, to time.Time)` - date-based filtering
- Composable: `query.New(notes).ByTag("topic").NotFrontmatter("archived").Results()`

### Weekly (`internal/weekly/`)
Manages weekly notes using the existing SB template as single source of truth:

```go
func CreateFromTemplate(templatePath string, weekStart time.Time, targetDir string) (string, error)
func RenderPreview(idx *index.Index, filePath string) string
```

**Creating a new weekly note:**
1. Read the SB template file (path from config: `weekly_template`)
2. Replace date placeholders (week start, prev/next week links)
3. Write to `{weekly_dir}/{date}.md` — file keeps `${query[[...]]}` syntax intact

**Rendering a preview for TUI display:**
1. Read the weekly note file
2. Resolve `${query[[...]]}` expressions using the index:
   - Topics: `ByTag("topic").NotFrontmatter("archived")`
   - Daily notes: `ByPath("Journal/Day/").ByDateRange(...)`
   - Meetings: `ByTag("meeting").ByDateRange("date", ...)`
   - Tasks: `ByTag("task").ByFrontmatter("status", "completed").ByDateRange("doneDate", ...)`
3. Return rendered markdown for display (read-only preview, original file unchanged)

### Wikilink (`internal/wikilink/`)
- `Parse(content string) []string` - extract all wikilinks from markdown
- `Resolve(link string, notesDir string) string` - resolve to file path
- `Backlinks(idx *index.Index, notePath string) []Note` - find notes linking to given note

### TUI (`internal/tui/`)
Built with [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss) + [bubbles](https://github.com/charmbracelet/bubbles).

Main app model manages views:
- `menuView` - main menu with hotkeys
- `pickerView` - fuzzy note picker (reused by Open, Topics, Backlinks)
- `searchView` - live ripgrep search with results
- `weeklyView` - rendered preview of weekly note (queries resolved from index), Enter to open in nvim

After selection/creation, TUI exits and spawns `$EDITOR` with the selected file.

---

## Weekly Note Template

**Uses the existing Silverbullet template file** as the single source of truth. vinote reads this template (path configured via `weekly_template`), replaces date placeholders, and writes the new weekly note file with `${query[[...]]}` syntax preserved. This means:
- You only maintain one template (the SB one)
- SB renders queries dynamically in the browser as before
- vinote renders queries from its index for TUI preview only (read-only, never written back to file)

---

## Configuration

File: `~/.config/vinote/config.toml`

```toml
notes_dir = "~/notes"
editor = "nvim"

# paths relative to notes_dir
weekly_dir = "Allegro/Journal/Week"
daily_dir = "Journal/Day"
weekly_template = "templates/Weekly.md"  # SB template file, single source of truth

# directories to skip during indexing
skip_dirs = ["_plug", "Library", ".git", "archive"]
```

---

## Dependencies (Go)

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - TUI components (textinput, list, spinner)
- `github.com/charmbracelet/lipgloss` - TUI styling
- `github.com/spf13/cobra` - CLI subcommands (vn w, vn o, etc.)
- `gopkg.in/yaml.v3` - YAML frontmatter parsing
- `github.com/BurntSushi/toml` - config file
- `github.com/sahilm/fuzzy` - fuzzy matching

---

## Implementation Order

### Step 1: Project skeleton + config
- `go mod init`, cobra setup, config loading
- Binary name: `vn`

### Step 2: Note index
- Scan `~/notes`, parse frontmatter, extract wikilinks
- JSON cache with mtime invalidation

### Step 3: Query system
- Filter by tag, frontmatter, date range, path prefix
- Composable builder pattern

### Step 4: Weekly note support
- Read existing SB template, create weekly note file (preserving `${query[[...]]}` syntax)
- Render preview by resolving queries from index (TUI display only)
- Open in editor

### Step 5: TUI - main menu + weekly view
- bubbletea app with menu
- Weekly: preview + confirm + create + open editor

### Step 6: TUI - fuzzy picker (Open, Topics)
- Inline fuzzy note picker
- Filter by tag for Topics view

### Step 7: TUI - search
- Live ripgrep integration
- Select result → open in editor at line

### Step 8: Backlinks
- Find notes linking to a given note
- Show as pickable list in TUI

### Step 9 (later): NeoVim integration
- `~/.config/nvim/lua/plugins/vinote.lua`
- `gf` on wikilinks, `<leader>n` prefix for vinote commands

---

## Verification

1. `vn` → TUI opens with menu, counts are correct (topics count)
2. Press `w` → rendered preview shows current topics, daily notes, meetings (resolved from index)
3. If file doesn't exist → created from SB template with `${query[[...]]}` syntax preserved
4. Same file opens fine in Silverbullet (queries render dynamically as before)
5. Press `o` → fuzzy search finds "Pigeon", Enter opens in nvim
6. Press `s` → typing "rate limiting" shows matching files with preview
7. Press `t` → lists active topics (matches what Silverbullet shows on Allegro/Tematy.md)
8. Press `b` → shows backlinks for selected note
