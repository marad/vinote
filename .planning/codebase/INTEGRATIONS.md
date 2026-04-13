# External Integrations

**Analysis Date:** 2026-04-13

## APIs & External Services

**None** - No external API integrations detected. vinote is a standalone, offline-first system.

## Data Storage

**Databases:**
- None - No database backend. Uses local filesystem only.

**File Storage:**
- Local filesystem - Notes directory stored at `~/notes` (configurable via `~/.config/vinote/config.toml`)
- File format: Markdown (.md) with optional YAML frontmatter
- Configuration: `notesDir` field in config

**Caching:**
- Local JSON cache - Index cached at `~/.config/vinote/index.json` for performance
  - Client: Custom Go implementation in `internal/index/` (LoadCache, SaveCache, IsCacheValid functions)
  - Invalidation: mtime-based cache invalidation (checks modification time of notes files)
  - Cache path: `CachePath()` in `internal/index/index.go`

## Authentication & Identity

**Auth Provider:**
- None - Local filesystem access only, no authentication required

**Implementation:**
- Config file-based (reads from user's `~/.config/` directory)
- File ownership security relies on OS file permissions

## Monitoring & Observability

**Error Tracking:**
- None - No error tracking service integrated

**Logs:**
- stdout/stderr only - Commands output JSON results to stdout, errors to stderr
- No persistent logging configured
- Neovim plugin logs to Neovim notification system via `vim.notify()`

## CI/CD & Deployment

**Hosting:**
- GitHub repository - github.com/marad/vinote
- Distributed via `go install github.com/marad/vinote/cmd/vn@latest`

**CI Pipeline:**
- None detected - No GitHub Actions or CI configuration files present

## Environment Configuration

**Required env vars:**
- None - All configuration via TOML file at `~/.config/vinote/config.toml`

**Secrets location:**
- No secrets managed - No external authentication or API keys needed

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None

## CLI-to-Editor Communication

**Neovim Plugin Integration (`lua/vinote.lua`):**
- Invokes `vn` CLI commands asynchronously via `vim.fn.jobstart()`
- Reads `~/.config/vinote/config.toml` directly from Neovim to get notes directory
- Parses JSON output from `vn query`, `vn backlinks`, `vn resolve` commands
- Launches external editor (`nvim` by default) for note creation/editing

## Completion Framework Integration

**blink.cmp Source (`lua/blink/cmp/sources/vinote.lua`):**
- Implements blink.cmp completion source for wikilink autocomplete
- Triggers on `[[` character sequence
- Invokes `vn query --all --json` to fetch available notes
- Returns completion items with note paths and metadata

## Shared Resources

**Configuration:**
- Single source of truth: `~/.config/vinote/config.toml`
- Read by both CLI and Neovim plugin
- Specifies: notes_dir, editor, weekly_dir, weekly_template, skip_dirs

**Index/Cache:**
- Shared between CLI and Neovim plugin
- Located at `~/.config/vinote/index.json`
- Created by `vn index` command, consumed by `vn query` and other commands

---

*Integration audit: 2026-04-13*
