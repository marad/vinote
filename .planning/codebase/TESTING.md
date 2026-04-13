# Testing Patterns

**Analysis Date:** 2026-04-13

## Test Framework

**Runner:**
- `go test` (standard Go testing)
- Test files: `*_test.go` using the `testing` package

**Assertion Library:**
- None; uses plain `if` statements and `t.Error*` methods

**Run Commands:**
```bash
go test ./...              # Run all tests
go test -v ./...           # Verbose output
go test -run TestName ./...  # Run specific test
go test -cover ./...       # Show coverage
go test -coverprofile=coverage.out ./...  # Generate coverage profile
```

## Test File Organization

**Location:**
- Co-located with source: test file in same package as code being tested
- Example: `internal/query/query_test.go` tests `internal/query/query.go`

**Naming:**
- Pattern: `[module]_test.go`
- Test functions: `func Test[FunctionName](t *testing.T)`

**Structure:**
```
internal/
├── query/
│   ├── query.go
│   └── query_test.go
├── index/
│   ├── index.go
│   ├── frontmatter.go
│   └── frontmatter_test.go
├── cli/
│   ├── init.go
│   └── init_test.go
└── wikilink/
    ├── wikilink.go
    └── wikilink_test.go
```

## Test Structure

**Suite Organization:**
Tests use plain functions, no suite setup. Global test data is defined at package level:

```go
// From internal/query/query_test.go
var testNotes = []index.Note{
    {Path: "topics/A", Title: "Topic A", Tags: []string{"topic"}, ...},
    {Path: "topics/B", Title: "Topic B", Tags: []string{"topic"}, ...},
    // ...
}

func TestByTag(t *testing.T) {
    got := ByTag(testNotes, "topic")
    if len(got) != 2 {
        t.Errorf("ByTag(topic) = %d notes, want 2", len(got))
    }
}
```

**Patterns:**

- **Table-driven tests:** Used for multiple input scenarios (see `internal/index/frontmatter_test.go`)
  ```go
  tests := []struct {
      name    string
      input   string
      wantFM  bool
      wantKey string
      wantVal string
  }{
      {
          name:    "standard frontmatter",
          input:   "---\ntitle: Hello\n---\n# Body",
          wantFM:  true,
          wantKey: "title",
          wantVal: "Hello",
      },
  }
  
  for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) {
          fm, _ := ParseFrontmatter(tt.input)
          // assertions
      })
  }
  ```

- **Setup/teardown:** Uses `t.TempDir()` for temporary directories (see `internal/cli/init_test.go`)
  ```go
  tmpDir := t.TempDir()
  t.Setenv("XDG_CONFIG_HOME", tmpDir)
  ```

- **Error checking:** Explicit comparison; early exit on critical failures
  ```go
  if len(got) != len(tt.want) {
      t.Fatalf("Parse(%q) = %v, want %v", tt.content, got, tt.want)
  }
  ```

## Mocking

**Framework:** None; uses real file I/O and environment variables

**Patterns:**
- Environment variables: set with `t.Setenv()` (see `internal/cli/init_test.go:16`)
- Temporary directories: `t.TempDir()` for isolated file operations
- No external service mocking; tests use in-memory data structures

**What to Mock (or avoid):**
```go
// From init_test.go: uses real filesystem
tmpDir := t.TempDir()
t.Setenv("XDG_CONFIG_HOME", tmpDir)
cmd.Execute()  // writes actual config file
content, _ := os.ReadFile(configPath)  // reads real file
```

**What NOT to Mock:**
- Local data structures (use real `index.Note`, `config.Config` instances)
- Standard library functions (os.Stat, os.ReadFile, etc.)
- Cobra commands: test end-to-end by calling `cmd.Execute()`

## Fixtures and Factories

**Test Data:**
Tests create minimal fixtures inline:

```go
// From internal/wikilink/wikilink_test.go
idx := &index.Index{
    Notes: []index.Note{
        {Path: "A", Wikilinks: []string{"B", "C"}},
        {Path: "B", Wikilinks: []string{"A"}},
        {Path: "C", Wikilinks: nil},
    },
}
```

**Location:**
- Defined at package level as package variables (e.g., `var testNotes = [...]`)
- Or inline within test functions
- No separate `fixtures.go` or factory files

**Pattern:**
Minimal required fields populated; omitted fields are zero-values:
```go
{Path: "topics/A", Title: "Topic A", Tags: []string{"topic"}, Frontmatter: map[string]any{"tags": "topic"}}
{Path: "meetings/M1", Title: "Meeting 1", Tags: nil, Frontmatter: nil}  // zero values
```

## Coverage

**Requirements:** None explicitly enforced (no `.coverprofile` minimum in config)

**View Coverage:**
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Current Coverage:**
- Query filters: fully tested (`query_test.go` covers all filter functions)
- Index parsing: fully tested (`frontmatter_test.go` covers frontmatter extraction, tag parsing)
- CLI commands: partially tested (`init_test.go` covers init command with various scenarios)
- Wikilinks: fully tested (`wikilink_test.go` covers parsing and backlink resolution)
- Weekly logic: not tested (no `weekly_test.go`)

## Test Types

**Unit Tests:**
- Scope: individual functions in isolation
- Approach: test one function with multiple input cases using table-driven tests
- Example: `TestByTag`, `TestParseDate`, `TestExtractTags`

**Integration Tests:**
- Scope: multi-function workflows (e.g., index load → query → filter)
- Approach: test composition of filters (e.g., `TestComposition` in `query_test.go`)
- Example: `TestInitCmd_CreatesConfig` tests config creation + file I/O + defaults validation

**E2E Tests:**
- Framework: Not used in current codebase
- CLI testing: uses integration-style testing with real file I/O (Cobra command execution)

## Common Patterns

**Async Testing:**
Not applicable (Go has goroutines, but no async/await testing in this codebase)

**Error Testing:**
Errors are validated implicitly via behavior; no explicit error assertions:

```go
// From init_test.go: tests success path, checks outputs
cmd := InitCmd()
if err := cmd.Execute(); err != nil {
    t.Fatal(err)
}

// Implicitly tests error handling by checking results
if !strings.Contains(out.String(), "created") {
    t.Error("output should mention config was created")
}
```

**Note on assertions:**
- `t.Fatalf()`: stop test immediately on critical failure
- `t.Errorf()`: record error and continue test
- `t.Error()`: alias for `t.Errorf()` with no format string
- No assertion library (plain comparisons)

```go
// From wikilink_test.go
if got[i] != tt.want[i] {
    t.Errorf("got %q, want %q", got[i], tt.want[i])
}

// From frontmatter_test.go
if fm == nil {
    t.Fatal("expected frontmatter, got nil")
}
```

**Test Input Validation:**
Table-driven tests check multiple scenarios in single test function:

```go
// From index/frontmatter_test.go
tests := []struct {
    name string
    fm   map[string]any
    want []string
}{
    {name: "comma-separated", fm: map[string]any{"tags": "topic, 2026"}, want: []string{"topic", "2026"}},
    {name: "yaml list", fm: map[string]any{"tags": []any{"meeting", "daily"}}, want: []string{"meeting", "daily"}},
    {name: "no tags", fm: map[string]any{"title": "Hello"}, want: nil},
    {name: "nil frontmatter", fm: nil, want: nil},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := ExtractTags(tt.fm)
        // assert
    })
}
```

---

*Testing analysis: 2026-04-13*
