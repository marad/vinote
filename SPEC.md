# vinote — Specyfikacja Techniczna

## 1. Przegląd systemu

vinote to terminalowy system zarządzania notatkami składający się z dwóch komponentów:

- **`vn`** — CLI napisane w Go, odpowiedzialne za indeksowanie, filtrowanie i tworzenie notatek
- **`vinote.lua`** — plugin Neovim (Lua), dostarczający UI oparty na snacks.nvim

System działa na katalogu `~/notes` współdzielonym z Silverbullet. vinote nigdy nie modyfikuje istniejących plików — jedynie je czyta oraz tworzy nowe (np. weekly notes).

---

## 2. Go CLI (`vn`)

### 2.1 Struktura projektu

```
vinote/
├── cmd/
│   └── main.go                  # cobra root command, wiring
├── internal/
│   ├── config/config.go         # ładowanie ~/.config/vinote/config.toml
│   ├── index/
│   │   ├── index.go             # skanowanie ~/notes, budowanie indeksu
│   │   └── frontmatter.go       # parser YAML frontmatter
│   ├── query/query.go           # filtry na []Note
│   ├── weekly/weekly.go         # tworzenie weekly notes + weekly-view
│   ├── wikilink/wikilink.go     # parsowanie [[linków]], resolve, backlinks
│   └── cli/
│       ├── index.go
│       ├── query.go
│       ├── weekly.go
│       ├── backlinks.go
│       └── resolve.go
├── go.mod
└── go.sum
```

### 2.2 Zależności

| Pakiet | Wersja | Cel |
|---|---|---|
| `github.com/spf13/cobra` | latest | CLI subcommands |
| `gopkg.in/yaml.v3` | latest | parsowanie YAML frontmatter |
| `github.com/BurntSushi/toml` | latest | parsowanie config.toml |

### 2.3 Konfiguracja

Plik: `~/.config/vinote/config.toml`

```go
type Config struct {
    NotesDir       string   `toml:"notes_dir"`        // default: "~/notes"
    Editor         string   `toml:"editor"`            // default: "nvim"
    WeeklyDir      string   `toml:"weekly_dir"`        // relative to notes_dir
    WeeklyTemplate string   `toml:"weekly_template"`   // relative to notes_dir
    SkipDirs       []string `toml:"skip_dirs"`         // dirs to skip during indexing
}
```

Ładowanie:
1. Czytaj `~/.config/vinote/config.toml`
2. Jeśli plik nie istnieje — użyj wartości domyślnych
3. Rozwiń `~` w ścieżkach do `$HOME`

### 2.4 Model danych

```go
// internal/index/index.go

type Note struct {
    Path        string            `json:"path"`         // względna do notes_dir, np. "Allegro/Tematy/Pigeon"
    Title       string            `json:"title"`        // z H1, frontmatter "title", lub nazwa pliku
    Tags        []string          `json:"tags"`         // z frontmatter "tags"
    Frontmatter map[string]any    `json:"frontmatter"`  // wszystkie pola frontmatter
    Wikilinks   []string          `json:"wikilinks"`    // wychodzące [[linki]]
    ModTime     time.Time         `json:"mod_time"`
}

type Index struct {
    Notes   []Note    `json:"notes"`
    Built   time.Time `json:"built"`
}
```

### 2.5 Indeksowanie (`internal/index/`)

**Skanowanie:**
- `filepath.WalkDir` po `notes_dir` rekursywnie
- Pomija katalogi z `skip_dirs` (domyślnie: `_plug`, `Library`, `.git`, `archive`)
- Przetwarza tylko pliki `*.md`

**Parsowanie frontmatter (`frontmatter.go`):**
- Szuka bloków `---` na początku pliku
- Parsuje YAML między ogranicznikami za pomocą `gopkg.in/yaml.v3`
- Wyciąga `tags` — obsługuje format lista YAML i comma-separated string
- Wyciąga `title` jeśli obecny

**Wyciąganie tytułu (priorytet):**
1. Pole `title` z frontmatter
2. Pierwszy nagłówek H1 (`# Tytuł`) z treści
3. Nazwa pliku bez `.md`

**Wyciąganie wikilinków:**
- Regex: `\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`
- Zwraca listę targetów (bez aliasów po `|`)

**Cache:**
- Zapisywany jako `~/.config/vinote/index.json`
- Walidacja: porównanie `Built` z najnowszym `mtime` plików w `notes_dir`
- Jeśli jakikolwiek plik jest nowszy niż cache — pełny rebuild
- Rebuild wykonywany synchronicznie przy każdym wywołaniu `vn` (jeśli cache nieaktualny)

### 2.6 System zapytań (`internal/query/`)

Czyste funkcje filtrujące operujące na `[]Note`:

```go
func ByTag(notes []Note, tag string) []Note
func ByPath(notes []Note, prefix string) []Note
func ByFrontmatter(notes []Note, key, value string) []Note
func NotFrontmatter(notes []Note, key string) []Note
func ByDateRange(notes []Note, field string, from, to time.Time) []Note
```

- `ByTag` — filtruje notatki zawierające dany tag (case-insensitive)
- `ByPath` — filtruje po prefixie ścieżki (np. `"Allegro/Tematy"`)
- `ByFrontmatter` — filtruje po wartości pola frontmatter
- `NotFrontmatter` — wyklucza notatki z polem ustawionym na `true` / niepustym
- `ByDateRange` — filtruje po polu daty w frontmatter (parsuje string → `time.Time`)

Kompozycja przez łańcuchowanie: `ByTag(NotFrontmatter(notes, "archived"), "topic")`

### 2.7 Wikilinki (`internal/wikilink/`)

```go
func Parse(content string) []string
func Resolve(link string, notesDir string) (string, error)
func Backlinks(idx *Index, notePath string) []Note
```

**Resolve** — strategia rozwiązywania `[[link]]` → ścieżka:
1. Dokładne dopasowanie: `{notesDir}/{link}.md`
2. Dokładne dopasowanie: `{notesDir}/{link}/index.md`
3. Przeszukanie indeksu — dopasowanie po nazwie pliku (bez ścieżki katalogu)
4. Jeśli brak — zwróć błąd

**Backlinks** — iteracja po indeksie, zwraca notatki których `Wikilinks` zawierają target pasujący do `notePath`.

### 2.8 Weekly Notes (`internal/weekly/`)

**Tworzenie:**
```go
func CreateFromTemplate(templatePath string, weekStart time.Time, targetDir string) (string, error)
```

1. Czytaj plik szablonu SB (`weekly_template` z configu)
2. Zamień placeholdery dat:
   - `{{weekStart}}` → data poniedziałku (format: `2006-01-02`)
   - `{{weekEnd}}` → data niedzieli
   - `{{prevWeek}}` / `{{nextWeek}}` → linki do poprzedniego/następnego tygodnia
   - `{{weekNumber}}` → numer tygodnia ISO
3. **Zachowaj** `${query[[...]]}` bez zmian (syntax SB, nie parsowany)
4. Zapisz jako `{weekly_dir}/{YYYY-MM-DD}.md` (data poniedziałku)
5. Zwróć ścieżkę do pliku

**Weekly View (dane dynamiczne):**
```go
func WeeklyView(notes []Note, weekStart time.Time) WeeklyData

type WeeklyData struct {
    Week       string `json:"week"`        // "2026-W11"
    DateRange  string `json:"date_range"`  // "Mar 9–15, 2026"
    FilePath   string `json:"file_path"`   // ścieżka do pliku weekly (lub "")
    FileExists bool   `json:"file_exists"`
    Meetings   []Note `json:"meetings"`    // tag:meeting + data w zakresie tygodnia
    Topics     []Note `json:"topics"`      // tag:topic, nie archived
}
```

Budowane z indeksu (nie z parsowania pliku weekly note):
- Meetings: `ByDateRange(ByTag(notes, "meeting"), "date", weekStart, weekEnd)`
- Topics: `ByTag(NotFrontmatter(notes, "archived"), "topic")`

### 2.9 Komendy CLI (`internal/cli/`)

Wszystkie komendy zarejestrowane jako cobra subcommands w `cmd/main.go`.

#### `vn index`

- Wymusza pełny rebuild indeksu (ignoruje cache)
- Wyświetla: liczbę notatek, czas budowania
- Exit code 0 przy sukcesie

#### `vn query`

Flagi:
- `--tag <tag>` — filtruj po tagu (wielokrotne użycie = OR)
- `--not <field>` — wyklucz po polu frontmatter
- `--path <prefix>` — filtruj po prefixie ścieżki
- `--field <key=value>` — filtruj po polu frontmatter
- `--from`, `--to` — zakres dat (format `YYYY-MM-DD`)
- `--date-field <field>` — pole do filtrowania dat (default: `date`)
- `--json` — output JSON (domyślny gdy stdout nie jest terminalem)
- `--all` — zwróć wszystkie notatki (bez filtrów)

Output JSON: tablica obiektów `Note`.
Output tekstowy: `path<TAB>title` — jedna linia na notatkę.

#### `vn weekly`

Flagi:
- `--create` — utwórz plik jeśli nie istnieje
- `--week <YYYY-Www>` — tydzień (default: bieżący)

Bez `--create`: wypisz ścieżkę do pliku weekly note (istnieje lub nie).
Z `--create`: utwórz z szablonu jeśli nie istnieje, wypisz ścieżkę.

#### `vn weekly-view`

Flagi:
- `--week <YYYY-Www>` — tydzień (default: bieżący)

Output: JSON `WeeklyData`.

#### `vn backlinks <note-path>`

- `note-path` — ścieżka względna do notatki (argument pozycyjny)
- Output JSON: tablica `Note` linkujących do podanej notatki
- Output tekstowy: `path<TAB>title` na linię

#### `vn resolve <wikilink>`

- `wikilink` — tekst linku bez `[[]]` (argument pozycyjny)
- Output: absolutna ścieżka do pliku
- Exit code 1 jeśli nie znaleziono

---

## 3. Plugin Neovim (`plugin/vinote.lua`)

### 3.1 Wymagania

- Neovim >= 0.10
- AstroNvim v5 z snacks.nvim
- `vn` binary w `$PATH`

### 3.2 Konfiguracja pluginu

Plugin ładowany jako moduł Lua. Konfiguracja minimalna — korzysta z configu `vn` (notes_dir itd.).

```lua
-- Opcjonalna konfiguracja w setup()
{
  binary = "vn",           -- ścieżka do binary
  notes_dir = "~/notes",   -- nadpisuje config vn (opcjonalne)
}
```

### 3.3 Keybindings

Prefix: `<leader>v`

| Key | Funkcja | Opis |
|---|---|---|
| `<leader>vw` | `vinote.weekly()` | Weekly view |
| `<leader>vo` | `vinote.open()` | Open note (picker) |
| `<leader>vs` | `vinote.search()` | Search content (live grep) |
| `<leader>vn` | `vinote.new()` | New note |
| `<leader>vt` | `vinote.topics()` | Topics picker |
| `<leader>vb` | `vinote.backlinks()` | Backlinks picker |
| `gf` | `vinote.follow_link()` | Follow [[wikilink]] |

Keybindings rejestrowane dla `ft = "markdown"` (buforów markdown).

### 3.4 Wzorzec komunikacji z CLI

Wszystkie wywołania `vn` asynchroniczne przez `vim.fn.jobstart`:

```lua
local function vn_async(args, on_result)
  local stdout = {}
  vim.fn.jobstart({"vn", unpack(args)}, {
    stdout_buffered = true,
    on_stdout = function(_, data) stdout = data end,
    on_exit = function(_, code)
      if code == 0 then
        on_result(table.concat(stdout, "\n"))
      end
    end,
  })
end
```

Parsowanie JSON response: `vim.json.decode()`.

### 3.5 Implementacja funkcji

#### `vinote.open()` — Open Note

1. `vn_async({"query", "--all", "--json"}, callback)`
2. Parsuj JSON → lista notatek
3. Otwórz `snacks.picker` z items = notatki, display = title, preview = treść pliku
4. Na select → `vim.cmd.edit(notes_dir .. "/" .. note.path .. ".md")`

#### `vinote.search()` — Search Content

1. Otwórz `snacks.picker.grep({ dirs = { notes_dir } })`
2. snacks obsługuje live grep, preview, selekcję natywnie

#### `vinote.topics()` — Topics Picker

1. `vn_async({"query", "--tag=topic", "--not=archived", "--json"}, callback)`
2. Otwórz `snacks.picker` z wynikami
3. Display: title + mod_time
4. Na select → otwórz notatkę

#### `vinote.backlinks()` — Backlinks Picker

1. Pobierz ścieżkę bieżącego bufora, oblicz ścieżkę względną do `notes_dir`
2. `vn_async({"backlinks", relative_path}, callback)`
3. Otwórz `snacks.picker` z wynikami
4. Na select → otwórz notatkę

#### `vinote.follow_link()` — Follow Wikilink (gf)

1. Pobierz słowo pod kursorem, wyciągnij `[[link]]` pattern
2. `vn_async({"resolve", link}, callback)`
3. Na sukces → `vim.cmd.edit(resolved_path)`
4. Na błąd → `vim.notify("Wikilink not found: " .. link, vim.log.levels.WARN)`

#### `vinote.weekly()` — Weekly View

1. `vn_async({"weekly-view"}, callback)`
2. Parsuj `WeeklyData` JSON
3. Jeśli `file_exists == false`:
   - `vn_async({"weekly", "--create"}, callback)` → potem ponów weekly-view
4. Renderuj float window:
   - Nagłówek: `Week` + `DateRange`
   - Sekcja "Meetings" — lista z tytułami
   - Sekcja "Topics" — lista z tytułami
5. Keybindings w floacie:
   - `<CR>` na elemencie → otwórz notatkę
   - `e` → otwórz plik weekly note do edycji
   - `q` / `<Esc>` → zamknij float

#### `vinote.new()` — New Note

1. `vim.ui.input({ prompt = "Note path: " }, callback)` — z completion na katalogach
2. Opcjonalnie: prompt na tagi
3. Utwórz plik z frontmatter:
   ```yaml
   ---
   title: <tytuł z nazwy pliku>
   tags:
   - <wybrane tagi>
   ---
   ```
4. `vim.cmd.edit(path)`

---

## 4. Kolejność implementacji

| Krok | Zakres | Weryfikacja |
|---|---|---|
| 1 | Skeleton: `go mod init`, cobra root, config loading | `vn --help` wyświetla pomoc |
| 2 | Index: skanowanie, frontmatter, wikilinks, cache | `vn index` raportuje liczbę notatek |
| 3 | Query: filtry + komenda `vn query` | `vn query --tag=topic --not=archived --json` zwraca JSON |
| 4 | Wikilinks: resolve + backlinks + komendy | `vn resolve Pigeon` → ścieżka; `vn backlinks path` → JSON |
| 5 | Plugin core: open, search, topics, backlinks, gf | Keybindings działają w nvim |
| 6 | Weekly creation: template → file | `vn weekly --create` → plik z `${query}` intact |
| 7 | Weekly view: dynamic data + float | `<leader>vw` → float z meetings/topics |
| 8 | New note: prompt + create | `<leader>vn` → nowy plik z frontmatter |

Każdy krok jest samodzielnie testowalny i deployowalny. Kroki 1–4 (Go CLI) mogą być implementowane niezależnie od kroków 5–8 (plugin).

---

## 5. Obsługa błędów

| Sytuacja | Zachowanie |
|---|---|
| Brak `config.toml` | Użyj wartości domyślnych |
| `notes_dir` nie istnieje | Błąd z komunikatem, exit code 1 |
| Brak frontmatter w pliku | `Note` z pustymi `Tags`, `Frontmatter` |
| Niepoprawny YAML | Pomiń frontmatter, loguj warning na stderr |
| Wikilink nie resolves | `vn resolve` → exit 1; plugin → `vim.notify` warning |
| Szablon weekly nie istnieje | `vn weekly --create` → błąd z komunikatem |
| `vn` binary nie w PATH | Plugin → `vim.notify` error przy pierwszym wywołaniu |
| Cache JSON uszkodzony | Pełny rebuild indeksu |

---

## 6. Konwencje kodu

### Go
- Standardowy layout Go: `cmd/`, `internal/`
- Exported types i funkcje z komentarzami godoc
- Błędy zwracane jako `error`, nie panikuj
- Testy: `*_test.go` obok kodu, table-driven tests
- JSON output na stdout, błędy/logi na stderr

### Lua
- Jeden plik modułu `vinote.lua` (ewentualnie split na podmoduły jeśli >300 linii)
- Async calls przez `jobstart`, nigdy `vim.fn.system` (blokujące) w hot path
- Korzystaj z `vim.json.decode`, `vim.notify`, `vim.ui.input` — standardowe API nvim
