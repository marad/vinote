# Coding Conventions

**Analysis Date:** 2026-04-13

## Naming Patterns

**Files:**
- Go source files: lowercase with underscores, e.g., `query.go`, `frontmatter.go`, `init_test.go`
- Tests: `[module]_test.go` pattern (e.g., `query_test.go` tests `query.go`)

**Functions:**
- Public functions: PascalCase (e.g., `ByTag`, `ParseFrontmatter`, `Build`)
- Private functions: camelCase (e.g., `parseDate`, `extractH1`, `splitKeyValue`)
- Cobra command functions: `[Command]Cmd()` pattern (e.g., `QueryCmd()`, `InitCmd()`)

**Variables:**
- Short-lived: single letters or abbreviations acceptable (e.g., `i`, `j`, `n`, `cfg`, `idx`)
- Package-level regexp: ALL_CAPS with `var` (e.g., `var wikilinkRe = regexp.MustCompile(...)`)
- Local variables: camelCase (e.g., `notesDir`, `skipSet`, `relPath`)

**Types:**
- Struct types: PascalCase (e.g., `Config`, `Note`, `Index`, `WeeklyData`)
- Field tags: snake_case matching TOML/JSON (e.g., `notes_dir`, `weekly_dir`, `mod_time`)

## Code Style

**Formatting:**
- Go standard: follows `gofmt` defaults (implicit, no .prettierrc or custom config)
- Tab indentation (default Go style)
- Max line length: appears to be ~100 characters, but no strict enforcement

**Linting:**
- No `.golangci.yml` or explicit linter config present
- Code follows idiomatic Go patterns (error handling, nil checks)

## Import Organization

**Order:**
1. Standard library imports (e.g., `"fmt"`, `"os"`, `"time"`)
2. External imports (e.g., `"github.com/spf13/cobra"`, `"gopkg.in/yaml.v3"`)
3. Internal imports (e.g., `"github.com/marad/vinote/internal/..."`)

**Path Aliases:**
- Full module path: `github.com/marad/vinote`
- Internal packages: `github.com/marad/vinote/internal/[package]`
- No aliases used in existing code

## Error Handling

**Patterns:**
- Explicit error returns: functions that can fail return `(T, error)`
- Error wrapping: use `fmt.Errorf("message: %w", err)` for context (see `internal/cli/init.go`)
- Graceful degradation: silently skip unparseable files/inaccessible entries in indexing (`internal/index/index.go:60-62`)
- Named return values: rarely used; mostly explicit returns

**Examples:**
```go
// Error wrapping pattern
if err := os.MkdirAll(configDir, 0o755); err != nil {
    return fmt.Errorf("failed to create config directory: %w", err)
}

// Graceful skip on error
note, err := parseNote(path, notesDir)
if err != nil {
    return nil  // skip unparseable files
}

// Error in return type
func Load(cfg config.Config) (*Index, error) { ... }
```

## Logging

**Framework:** Standard `fmt` package for CLI output

**Patterns:**
- `fmt.Fprintf(cmd.OutOrStdout(), ...)` for command output
- `fmt.Fprintf(cmd.ErrOrStderr(), ...)` for warnings/errors in CLI context
- No structured logging; plain text output

**Example:**
```go
// From internal/cli/init.go
fmt.Fprintf(cmd.OutOrStdout(), "Config already exists: %s (use --force to overwrite)\n", configFile)
fmt.Fprintf(cmd.ErrOrStderr(), "Warning: notes directory does not exist: %s\n", cfg.NotesAbsPath())
```

## Comments

**When to Comment:**
- Public functions: short comment explaining purpose (e.g., `// ByTag returns notes containing the given tag`)
- Complex logic: explain the "why" (e.g., fuzzy scoring bonuses in `internal/query/query.go:51-79`)
- Non-obvious behavior: clarify edge cases or parsing logic

**JSDoc/TSDoc:**
- Not applicable (Go project; no TypeScript)
- Go-style doc comments: start with function name (implicit in public functions)

**Examples:**
```go
// ByTag returns notes containing the given tag (case-insensitive).
func ByTag(notes []index.Note, tag string) []index.Note { ... }

// fuzzyScore returns a score for how well pattern matches text (0 = no match).
// Higher is better. Rewards consecutive matches and word-boundary alignment.
func fuzzyScore(text, pattern string) int { ... }

// ParseFrontmatter extracts YAML frontmatter from markdown content.
// Returns the parsed map and the remaining content after the frontmatter block.
func ParseFrontmatter(content string) (map[string]any, string) { ... }
```

## Function Design

**Size:** Functions are compact; most under 50 lines. Complex logic (e.g., `fuzzyScore`) includes inline comments explaining scoring.

**Parameters:**
- Typically pass config and data structures (e.g., `func ByTag(notes []index.Note, tag string)`)
- No variadic arguments; filter functions accept slices directly
- Flag parsing: via Cobra's `cmd.Flags().Get*` methods in CLI code

**Return Values:**
- Filtered slices: return `[]Type` (nil if empty, never create empty slice)
- Error-producing functions: return `(T, error)` tuple
- State queries: return `(value, bool)` for validity checks

## Module Design

**Exports:**
- Public structs and functions: PascalCase at package level
- Unexported helpers: camelCase (e.g., `parseDate`, `extractWikilinks`)
- Constants: no global constants (configuration via `config.Config` type)

**Barrel Files:**
- Not used; each package provides its own exports
- No `init.go` exporting multiple modules

**File Organization by Package:**
- `internal/index/`: indexing and caching (`index.go`, `frontmatter.go`, tests)
- `internal/query/`: filtering and search (`query.go`, tests)
- `internal/cli/`: command handlers (`init.go`, `query.go`, `backlinks.go`, etc.)
- `internal/config/`: configuration loading (`config.go`)
- `internal/wikilink/`: wikilink resolution (`wikilink.go`, tests)
- `internal/weekly/`: weekly note logic (`weekly.go`, no tests)

## Testing Conventions

**Test Files:** Co-located with source using `*_test.go` suffix (see TESTING.md for detailed patterns)

---

*Convention analysis: 2026-04-13*
