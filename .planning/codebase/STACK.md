# Technology Stack

**Analysis Date:** 2026-04-13

## Languages

**Primary:**
- Go 1.25.5 - CLI (`vn` binary) for note indexing, querying, and management

**Secondary:**
- Lua - Neovim plugin integration for UI and terminal interface
- YAML - Frontmatter parsing in markdown notes

## Runtime

**Environment:**
- Go 1.25.5

**Package Manager:**
- Go modules (`go.mod`, `go.sum`)
- Lockfile: Present (`go.sum`)

## Frameworks

**Core:**
- Cobra 1.10.2 - CLI framework for subcommand structure (`cmd/vn/main.go`)

**Neovim Integration:**
- Lua - Vim plugin scripting
- snacks.nvim - Picker UI for note selection and navigation (required dependency for Lua plugin)
- blink.cmp - Completion framework for wikilink autocomplete integration (`lua/blink/cmp/sources/vinote.lua`)

**Configuration:**
- TOML - Configuration file parsing via BurntSushi/toml

## Key Dependencies

**Critical:**
- github.com/spf13/cobra v1.10.2 - CLI subcommand routing and flag parsing
- gopkg.in/yaml.v3 v3.0.1 - YAML frontmatter parsing in markdown files
- github.com/BurntSushi/toml v1.6.0 - TOML configuration file parsing

**Infrastructure:**
- github.com/spf13/pflag v1.0.9 - Flag parsing (indirect, required by Cobra)
- github.com/inconshreveable/mousetrap v1.1.0 - Windows console support (indirect, required by Cobra)

## Configuration

**Environment:**
- Configuration file: `~/.config/vinote/config.toml` (TOML format)
- Looks for XDG_CONFIG_HOME or defaults to `~/.config/vinote/`

**Build:**
- Standard Go build via `go build` and `go install`
- Installed as CLI binary named `vn`

## Platform Requirements

**Development:**
- Go 1.25.5 or later
- Standard Unix build tools (make, git)

**Runtime (CLI):**
- Any OS with Go runtime support (Linux, macOS, Windows, etc.)
- No external binaries required

**Runtime (Neovim Plugin):**
- Neovim with Lua support
- snacks.nvim plugin (required for picker UI)
- blink.cmp plugin (optional but required for wikilink completion features)
- lazy.nvim package manager (for plugin installation)

## Output Format

- JSON - All CLI commands output JSON to stdout for machine parsing
- TOML - Configuration format

---

*Stack analysis: 2026-04-13*
