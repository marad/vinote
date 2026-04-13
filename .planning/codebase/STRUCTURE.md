# Codebase Structure

**Analysis Date:** 2026-04-13

## Directory Layout

```
/home/marad/dev/vinote/
├── cmd/                    # CLI entry points
│   └── vn/
│       └── main.go         # Root Cobra command setup
├── internal/               # Private packages (not exported)
│   ├── cli/                # Command handlers
│   │   ├── init.go
│   │   ├── index.go
│   │   ├── query.go
│   │   ├── backlinks.go
│   │   ├── resolve.go
│   │   ├── weekly.go
│   │   └── *_test.go
│   ├── config/             # Configuration loading and paths
│   │   └── config.go
│   ├── index/              # Note indexing and parsing
│   │   ├── index.go
│   │   ├── frontmatter.go
│   │   └── *_test.go
│   ├── query/              # Query filters and composition
│   │   ├── query.go
│   │   └── *_test.go
│   ├── wikilink/           # Wikilink resolution and backlinks
│   │   ├── wikilink.go
│   │   └── *_test.go
│   └── weekly/             # Weekly note generation and view
│       └── weekly.go
├── lua/                    # Neovim plugin (Lua)
│   ├── vinote.lua          # Main plugin entry
│   └── blink/cmp/          # Wikilink completion source
├── feat-spec/              # Feature specifications (documentation)
│   ├── note-creation.md
│   └── meeting-note.md
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── README.md               # User documentation
├── SPEC.md                 # Technical specification
├── PLAN.md                 # Development plan
└── CLAUDE.md               # Project instructions (Seeds, Canopy)
```

## Directory Purposes

**`cmd/vn/`:**
- Purpose: CLI application entry point
- Contains: `main.go` with Cobra root command and subcommand registration
- Key files: `cmd/vn/main.go`

**`internal/cli/`:**
- Purpose: Command handlers for all subcommands
- Contains: Cobra command definitions, flag parsing, output formatting
- Key files: `init.go`, `index.go`, `query.go`, `backlinks.go`, `resolve.go`, `weekly.go`
- Pattern: Each file exports one `*Cmd()` function returning a `*cobra.Command`

**`internal/config/`:**
- Purpose: Configuration management and path resolution
- Contains: Config struct (TOML-backed), XDG_CONFIG_HOME support, path expansion
- Key files: `config.go`
- Pattern: Singleton-like config loaded once per command

**`internal/index/`:**
- Purpose: Note discovery, parsing, and caching
- Contains: Note struct, Index struct, filesystem walking, frontmatter parsing, cache I/O
- Key files: `index.go`, `frontmatter.go`
- Pattern: Build() scans directory tree, SaveCache/LoadCache use JSON, IsCacheValid checks mtime

**`internal/query/`:**
- Purpose: Filter and search operations on note collections
- Contains: Functions like ByTag, ByName, ByPath, ByDateRange that take `[]Note` and return filtered slice
- Key files: `query.go`
- Pattern: Composable functions (chainable filters); pure, no side effects

**`internal/wikilink/`:**
- Purpose: Wikilink parsing, resolution, and backlink discovery
- Contains: Wikilink regex parsing, three-tier resolution strategy, backlinks computation
- Key files: `wikilink.go`
- Pattern: Parse() extracts links from content, Resolve() finds target files, Backlinks() finds references

**`internal/weekly/`:**
- Purpose: Weekly note generation and dynamic view data
- Contains: Template-based creation, week navigation, query composition
- Key files: `weekly.go`
- Pattern: Generates structured WeeklyData for JSON output to Neovim

**`lua/`:**
- Purpose: Neovim plugin implementation
- Contains: Plugin initialization, keybindings, picker UI (via snacks.nvim)
- Key files: `vinote.lua` (main), `blink/cmp` (completion source)
- Note: Integrates with `vn` CLI via JSON output

## Key File Locations

**Entry Points:**
- `cmd/vn/main.go`: Binary entry point; registers all 7 subcommands via Cobra
- `internal/cli/*.go`: Each subcommand implemented as a separate file

**Configuration:**
- `internal/config/config.go`: Config struct, TOML loading, path helpers
- Config file location: `~/.config/vinote/config.toml` (XDG_CONFIG_HOME aware)
- Defaults: notes_dir=~/notes, editor=nvim, weekly_dir=Allegro/Journal/Week

**Core Logic:**
- `internal/index/index.go`: Note and Index structs, Build(), Load(), cache management
- `internal/index/frontmatter.go`: YAML frontmatter parsing, tag extraction
- `internal/query/query.go`: All filter functions (ByTag, ByName, ByPath, ByFrontmatter, ByDateRange, NotFrontmatter)
- `internal/wikilink/wikilink.go`: Parse(), Resolve(), Backlinks()
- `internal/weekly/weekly.go`: WeeklyData struct, WeeklyView(), CreateFromTemplate()

**Testing:**
- `internal/cli/init_test.go`: CLI init command tests
- `internal/index/frontmatter_test.go`: Frontmatter parsing tests
- `internal/query/query_test.go`: Query filter composition tests
- `internal/wikilink/wikilink_test.go`: Wikilink parsing and resolution tests
- Pattern: Unit tests co-located with implementation (`*_test.go` files)

## Naming Conventions

**Files:**
- Go: lowercase with underscores for multi-word (`init_test.go`, `frontmatter.go`)
- Lua: lowercase with underscores (`vinote.lua`)
- Markdown docs: UPPERCASE (`README.md`, `SPEC.md`)

**Functions:**
- Public (exported): PascalCase (`Load`, `Build`, `ByTag`, `Resolve`, `WeeklyView`)
- Private (unexported): camelCase (`parseNote`, `extractWikilinks`, `fuzzyScore`, `parseWeekFlag`)
- Command builders: PascalCase ending in "Cmd" (`InitCmd`, `IndexCmd`, `QueryCmd`)

**Variables:**
- Receiver names: short, often single letter (`idx`, `cfg`, `cmd`)
- Loop variables: `i`, `j`, `n` for slices
- Config/index: `cfg`, `idx`
- Local short-lived: camelCase

**Types:**
- Struct: PascalCase (`Note`, `Index`, `Config`, `WeeklyData`)
- Interfaces: Go idiom, rarely used here

## Where to Add New Code

**New Subcommand:**
1. Create `internal/cli/newcmd.go` with `func NewCmdCmd() *cobra.Command`
2. Register in `cmd/vn/main.go` via `root.AddCommand(cli.NewCmdCmd())`
3. Import necessary packages (config, index, query, etc.)

**New Query Filter:**
1. Add function in `internal/query/query.go`: `func By<Criterion>(notes []index.Note, param string) []index.Note`
2. Follow pattern: iterate notes slice, filter based on criterion, return matching subset
3. Add tests in `internal/query/query_test.go` using testNotes fixture

**New Index Field (Note Property):**
1. Add field to `Note` struct in `internal/index/index.go`
2. Update `parseNote()` function to populate field
3. Add JSON tag for cache serialization
4. Update any query functions that reference Note fields

**New Configuration Option:**
1. Add field to `Config` struct in `internal/config/config.go` with TOML tag
2. Add to `DefaultConfig()` return
3. Update `DefaultConfigTOML()` comment string
4. Use via `cfg.FieldName` (already path-expanded if applicable)

**New CLI Output:**
- Use `cmd.OutOrStdout()` for normal output (respects test double)
- Use `cmd.ErrOrStderr()` for warnings/errors
- JSON: use `json.NewEncoder(cmd.OutOrStdout()).SetIndent("", "  ").Encode(data)`
- Text: use `fmt.Fprintf(cmd.OutOrStdout(), ...)`

**Utilities (Shared Helpers):**
- File format helpers: `internal/index/` (frontmatter, YAML)
- Date/time helpers: `internal/weekly/` or `internal/query/` (already exist)
- String utilities: Add to relevant package (no dedicated utilities package)

## Special Directories

**`internal/`:**
- Purpose: Go convention for private packages (not importable from outside module)
- Generated: No
- Committed: Yes
- All business logic lives here; `cmd/vn` is minimal

**`lua/`:**
- Purpose: Neovim plugin source code
- Generated: No
- Committed: Yes
- Loaded by Lazy.nvim, calls out to `vn` CLI via JSON

**`.planning/codebase/`:**
- Purpose: Codebase documentation (architecture, structure, conventions, testing, concerns)
- Generated: Yes (by mapper tool)
- Committed: Yes
- Used by plan/execute orchestrators

**`feat-spec/`:**
- Purpose: High-level feature specifications and requirements
- Generated: No
- Committed: Yes
- Reference for future implementation work

---

*Structure analysis: 2026-04-13*
