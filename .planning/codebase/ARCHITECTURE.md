# Architecture

**Analysis Date:** 2026-04-13

## Pattern Overview

**Overall:** Layered CLI application with separation between command handling, business logic, and data retrieval.

**Key Characteristics:**
- Modular domain packages (config, index, query, wikilink, weekly)
- Config-driven note discovery and indexing
- In-memory index with JSON file-based cache
- Composable query filters via function chaining
- Cobra CLI framework for command routing

## Layers

**Command Layer (CLI):**
- Purpose: Handle user input, parse flags, orchestrate business logic, output results
- Location: `internal/cli/`
- Contains: Cobra command definitions (`*Cmd()` functions), argument parsing
- Depends on: config, index, query, wikilink, weekly packages
- Used by: `cmd/vn/main.go` entry point

**Business Logic Layer:**
- Purpose: Core operations on notes (indexing, filtering, resolving links)
- Location: `internal/query/`, `internal/wikilink/`, `internal/weekly/`
- Contains: Query filters (ByTag, ByName, ByPath, ByDateRange), wikilink resolution, weekly note generation
- Depends on: index package (data structures)
- Used by: CLI layer

**Data Access Layer:**
- Purpose: Load, parse, and cache notes from filesystem
- Location: `internal/index/`
- Contains: Note parsing, frontmatter extraction, index building, cache management
- Depends on: config package (path configuration)
- Used by: CLI and business logic layers

**Configuration Layer:**
- Purpose: Load user settings and provide path resolution
- Location: `internal/config/`
- Contains: Config struct, TOML loading, path expansion, defaults
- Depends on: None (stdlib only)
- Used by: All other layers

## Data Flow

**Index Build Flow:**

1. User runs `vn index`
2. `internal/cli/index.go` → loads config
3. `internal/index/Build()` → walks notes directory
4. For each `.md` file: parse frontmatter, extract wikilinks, extract tags
5. Build in-memory `Index` struct with all `Note` records
6. `SaveCache()` writes JSON to `~/.config/vinote/index.json`
7. Output indexed count and timing

**Query Flow:**

1. User runs `vn query --tag foo --name bar`
2. `internal/cli/query.go` → loads config, loads index (cached or rebuilt)
3. Start with all notes from index: `notes := idx.Notes`
4. Apply filters sequentially via composition:
   - `ByTag(notes, "foo")` → filtered notes
   - `ByName(filtered, "bar")` → further filtered
5. Apply sorting (mtime, title, or path)
6. Output as JSON or tab-separated text

**Weekly View Flow:**

1. User runs `vn weekly-view --week 2026-W15`
2. `internal/cli/weekly.go` → loads config and index
3. `weekly.WeeklyView()` calculates week boundaries
4. Filters meetings and topics from index via `query.ByTag()` and `query.ByDateRange()`
5. Returns `WeeklyData` struct with JSON-serializable metadata
6. Output as JSON

**Wikilink Resolution Flow:**

1. User runs `vn resolve MyNote` or Neovim uses resolve in background
2. `internal/cli/resolve.go` → loads config and index
3. `wikilink.Resolve()` tries three strategies:
   - Exact match: `notesDir/MyNote.md`
   - Index file: `notesDir/MyNote/index.md`
   - Fuzzy match: search index by filename (case-insensitive)
4. Return absolute file path or error

**State Management:**
- **Config:** Loaded once per command from `~/.config/vinote/config.toml`
- **Index:** Cached in-memory during command execution; JSON cache checked for validity based on note mtime
- **Results:** Generated fresh each time; no state persistence between commands

## Key Abstractions

**Note:**
- Purpose: Represents a single markdown file with metadata
- Location: `internal/index/index.go`
- Fields: Path (relative), Title, Tags, Frontmatter (YAML), Wikilinks, ModTime
- Pattern: Value type, JSON-serializable, used throughout filtering/output

**Index:**
- Purpose: Complete collection of notes with metadata for fast querying
- Location: `internal/index/index.go`
- Contains: `[]Note` slice, Built timestamp
- Pattern: Immutable during command execution; cached to disk as JSON; validity checked via mtime

**Query Filters:**
- Purpose: Composable functions that filter `[]Note` slices
- Location: `internal/query/query.go`
- Examples: `ByTag`, `ByName`, `ByPath`, `ByFrontmatter`, `ByDateRange`, `NotFrontmatter`
- Pattern: Pure functions, chainable (output of one becomes input to next), case-insensitive where appropriate

**Config:**
- Purpose: User settings (notes directory, editor, weekly template)
- Location: `internal/config/config.go`
- Pattern: Loaded once per command; paths expanded (tilde → home); defaults applied if not in TOML

## Entry Points

**CLI Entry:**
- Location: `cmd/vn/main.go`
- Triggers: User runs `vn <command> [args]`
- Responsibilities: Register all subcommands (init, index, query, backlinks, resolve, weekly, weekly-view), execute via Cobra

**Subcommands (all in `internal/cli/`):**
- `InitCmd()` → initialize config file and check notes directory
- `IndexCmd()` → scan and cache notes
- `QueryCmd()` → filter notes with multiple flags, output JSON or text
- `BacklinksCmd()` → find notes linking to given note
- `ResolveCmd()` → resolve wikilink to file path
- `WeeklyCmd()` → get weekly note path (create from template if missing)
- `WeeklyViewCmd()` → get dynamic weekly data (meetings, topics, navigation)

## Error Handling

**Strategy:** Fail-fast with informative error messages; skip unparseable files during index build.

**Patterns:**
- Configuration errors bubble up immediately: "failed to load config"
- Index build skips unparseable files silently (no error), continues scanning
- Query operations return empty result slices on error, not errors themselves
- Wikilink resolution returns formatted error: "wikilink not found: X"
- File I/O errors wrapped with context: "failed to save cache: %w"

## Cross-Cutting Concerns

**Logging:** 
- CLI outputs to stdout (JSON, text) and stderr (warnings)
- No structured logging; simple `fmt.Fprintf()` calls
- Timing reported in IndexCmd: "Indexed N notes in Xms"

**Validation:**
- Tag extraction handles both YAML list and comma-separated string formats
- Date parsing tries multiple layouts (YYYY-MM-DD, RFC3339)
- Frontmatter parsing via YAML v3 library (gracefully returns nil on parse failure)
- Fuzzy matching uses custom scoring algorithm (consecutive matches, word boundaries, text length bonus)

**Cache Invalidation:**
- Index cache valid only if `cache.Built > max(note.ModTime)` for all notes
- Walk notes directory on every Load() to check validity
- No TTL; mtime-based only

---

*Architecture analysis: 2026-04-13*
