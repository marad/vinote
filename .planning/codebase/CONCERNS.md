# Codebase Concerns

**Analysis Date:** 2026-04-13

## Error Handling Gaps

**Silent index building failures:**
- Issue: In `internal/index/index.go`, the `Build()` function silently skips unparseable notes and inaccessible entries with `return nil` comments instead of logging warnings
- Files: `internal/index/index.go` (lines 45, 61)
- Impact: Users won't be notified if notes fail to parse due to YAML errors, encoding issues, or permission problems. Index corruption can go undetected.
- Fix approach: Log warnings or collect error summaries when files fail to parse. Consider adding a `--verbose` flag to report problematic files during indexing.

**Unchecked UserHomeDir errors:**
- Issue: `os.UserHomeDir()` errors are ignored with bare `_` in `internal/config/config.go` (lines 56, 95)
- Files: `internal/config/config.go` (lines 56, 95)
- Impact: On systems where HOME cannot be determined, the function silently returns empty string joined with paths, resulting in invalid config directory paths like `/.config/vinote`
- Fix approach: Return error instead of silently ignoring, or provide a fallback (e.g., current working directory)

**Query date parsing failures:**
- Issue: In `internal/cli/query.go`, date parsing errors are silently ignored (lines 68-69)
- Files: `internal/cli/query.go` (lines 68-69)
- Impact: Invalid date arguments like `--from invalid-date` silently become zero time, which then converts to year 2000 fallback (line 74), potentially returning unintended note ranges
- Fix approach: Return error or warn user when date parsing fails instead of silent fallback behavior

## Testing Gaps

**Missing coverage for multiple packages:**
- What's not tested: `internal/config`, `internal/weekly`, `cmd/vn/main.go`
- Files: `internal/config/config.go`, `internal/weekly/weekly.go`, `cmd/vn/main.go`
- Risk: Config path expansion (especially tilde expansion), weekly template creation, and main entry point are untested and could break silently
- Priority: High

**No integration tests:**
- What's not tested: Full end-to-end workflows (index → query → resolve), config loading with real files
- Files: All CLI commands
- Risk: Individual unit tests pass but commands may fail when used together or with real notes directory structures
- Priority: Medium

**Incomplete edge case coverage in query parsing:**
- What's not tested: Fuzzy search behavior with special characters, multi-value frontmatter fields, complex date formats
- Files: `internal/query/query.go` (fuzzyScore function)
- Risk: Unexpected scoring behavior or crashes with malformed input data
- Priority: Medium

## Fragile Areas

**Path normalization inconsistency:**
- Files: `internal/index/index.go`, `internal/wikilink/wikilink.go`
- Why fragile: Note paths are stored without `.md` extension (line 87 in index.go), but resolved with `.md` appended (line 56 in wikilink.go). Case sensitivity handling is inconsistent (case-insensitive in Backlinks but case-sensitive in path construction elsewhere)
- Safe modification: Create a `NotePath` type or utility functions that encapsulate these rules, ensuring all path operations go through a single place
- Test coverage: Gaps in path handling with mixed case, symbolic links, and directory hierarchies

**Frontmatter parsing brittleness:**
- Files: `internal/index/frontmatter.go` (lines 12-29)
- Why fragile: Hardcoded delimiter matching with `strings.Index()` won't handle YAML documents with dashes in content or unusual formatting
- Safe modification: Use a proper YAML frontmatter parser or add stricter delimiter detection (e.g., require `---` at start of line, not embedded in content)
- Test coverage: No tests for edge cases like `---` appearing in frontmatter content or YAML with special characters

**Wikilink resolution ambiguity:**
- Files: `internal/wikilink/wikilink.go` (lines 38-61)
- Why fragile: Resolve strategy searches by exact path, then index file, then filename match. A file named `foo.md` in different directories could match multiple times, but only first match is returned
- Safe modification: Add clear disambiguation rules or return all matches instead of first match
- Test coverage: No tests for ambiguous cases where multiple files could match the same wikilink

## Dependencies at Risk

**Missing error wrapping in weekly module:**
- Risk: `CreateFromTemplate()` in `internal/weekly/weekly.go` returns errors from template reading and file writing without context
- Impact: Users cannot distinguish between template-not-found vs permission-denied vs disk-full errors
- Migration plan: Wrap errors with `fmt.Errorf()` to add context messages, making debugging easier

**TOML parsing resilience:**
- Risk: Config loading in `internal/config/config.go` (line 40) uses `toml.DecodeFile()` which fails hard if config is malformed
- Impact: Single typo in config file makes entire app unusable with cryptic parser error
- Mitigation: Add validation function that checks for required fields and provides user-friendly error messages
- Migration plan: Validate config after loading and provide specific guidance on fixes

## Performance Considerations

**Cache invalidation strategy:**
- Current: Checks if any `.md` file has `ModTime > idx.Built` (expensive walk for large note directories)
- Files: `internal/index/index.go` (IsCacheValid function, lines 169-201)
- Concern: Walking entire notes directory on every Load() call defeats caching benefits for operations that run frequently (e.g., every keystroke in editor integration)
- Improvement path: Track directory modification time instead of individual file mtimes, or use inotify-based invalidation (on systems that support it)

**Fuzzy score calculation:**
- Files: `internal/query/query.go` (fuzzyScore function, lines 52-80)
- Concern: O(n*m) algorithm where n=text length, m=pattern length. Bonuses for consecutive matches and word boundaries can produce counterintuitive rankings
- Improvement path: Add configurable weighting, benchmark against real note collections, or add score caching for repeated queries

## Scaling Limits

**In-memory index:**
- Current capacity: Tested with ~1000 notes (~100ms build time), but full index is loaded into memory on every command
- Limit: Systems with 100,000+ notes or notes > 10MB will experience slow queries and high memory usage
- Scaling path: Stream-based queries without full index load, or implement lazy loading of note metadata

**Directory walking performance:**
- Current: `filepath.WalkDir()` is called multiple times (Build, IsCacheValid, and again in query CLI) without parallelization
- Limit: Note directories on network filesystems or with thousands of entries will be slow
- Scaling path: Cache walk results, parallelize using goroutines, or implement incremental indexing

## Security Considerations

**Path traversal in config:**
- Risk: `notes_dir` in config can be any path; no validation that it's within expected boundaries
- Files: `internal/config/config.go`
- Current mitigation: None — application assumes config is trusted
- Recommendations: Document that config is trusted input, add safety validation for unusual paths if app is ever exposed to untrusted config sources

**Frontmatter injection via wikilinks:**
- Risk: Wikilink targets come from note content, not validated. Could theoretically be exploited if wikilinks are rendered unsafely in a web interface
- Files: `internal/wikilink/wikilink.go`
- Current mitigation: None — application is CLI-only, not exposed to web
- Recommendations: If adding web UI, properly escape wikilink targets and validate against index

**Config file permissions:**
- Risk: Config is written with `0o644` (world-readable) and may contain sensitive paths or future credential fields
- Files: `internal/cli/init.go` (line 30), `internal/weekly/weekly.go` (line 67)
- Current mitigation: None
- Recommendations: Use `0o600` for config files to restrict to owner only, add security note to DefaultConfigTOML

## Limitations and Known Issues

**No transaction semantics:**
- Problem: Index cache and config files can be partially written if write fails mid-operation, leaving corrupted state
- Impact: Recovery requires manual deletion of `~/.config/vinote/index.json`
- Blocks: Reliable distributed note syncing

**Limited YAML support:**
- Problem: Complex YAML (anchors, references, nested structures) is not fully tested
- Blocks: Power users with sophisticated frontmatter schemes

**Case sensitivity in cross-platform usage:**
- Problem: Wikilink resolution uses case-insensitive matching on filename (line 55 in wikilink.go) but case-sensitive for paths
- Impact: Links break inconsistently between Linux (case-sensitive filesystem) and macOS/Windows (case-insensitive)
- Blocks: Reliable cross-platform note sync
