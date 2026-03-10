package index

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mradoszewski/vinote/internal/config"
)

var wikilinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

// Note represents a single markdown note.
type Note struct {
	Path        string         `json:"path"`
	Title       string         `json:"title"`
	Tags        []string       `json:"tags"`
	Frontmatter map[string]any `json:"frontmatter"`
	Wikilinks   []string       `json:"wikilinks"`
	ModTime     time.Time      `json:"mod_time"`
}

// Index holds all indexed notes and metadata.
type Index struct {
	Notes []Note    `json:"notes"`
	Built time.Time `json:"built"`
}

// Build scans the notes directory and builds a fresh index.
func Build(cfg config.Config) (*Index, error) {
	notesDir := cfg.NotesAbsPath()
	skipSet := make(map[string]bool, len(cfg.SkipDirs))
	for _, d := range cfg.SkipDirs {
		skipSet[d] = true
	}

	var notes []Note

	err := filepath.WalkDir(notesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}

		if d.IsDir() {
			if skipSet[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		note, err := parseNote(path, notesDir)
		if err != nil {
			return nil // skip unparseable files
		}

		notes = append(notes, note)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Index{Notes: notes, Built: time.Now()}, nil
}

func parseNote(absPath, notesDir string) (Note, error) {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return Note{}, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return Note{}, err
	}

	relPath, _ := filepath.Rel(notesDir, absPath)
	// Remove .md extension from path
	relPath = strings.TrimSuffix(relPath, ".md")

	text := string(content)
	fm, body := ParseFrontmatter(text)

	title := ExtractTitle(fm)
	if title == "" {
		title = extractH1(body)
	}
	if title == "" {
		title = filepath.Base(relPath)
	}

	tags := ExtractTags(fm)
	wikilinks := extractWikilinks(text)

	return Note{
		Path:        relPath,
		Title:       title,
		Tags:        tags,
		Frontmatter: fm,
		Wikilinks:   wikilinks,
		ModTime:     info.ModTime(),
	}, nil
}

func extractH1(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return ""
}

func extractWikilinks(content string) []string {
	matches := wikilinkRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var links []string
	for _, m := range matches {
		link := strings.TrimSpace(m[1])
		if !seen[link] {
			seen[link] = true
			links = append(links, link)
		}
	}
	return links
}

// CachePath returns the path where the index cache is stored.
func CachePath(cfg config.Config) string {
	return filepath.Join(config.ConfigDir(), "index.json")
}

// LoadCache reads the cached index from disk.
func LoadCache(cfg config.Config) (*Index, error) {
	data, err := os.ReadFile(CachePath(cfg))
	if err != nil {
		return nil, err
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

// SaveCache writes the index to disk.
func SaveCache(cfg config.Config, idx *Index) error {
	cachePath := CachePath(cfg)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath, data, 0o644)
}

// IsCacheValid checks whether the cache is newer than all note files.
func IsCacheValid(cfg config.Config, idx *Index) bool {
	notesDir := cfg.NotesAbsPath()
	skipSet := make(map[string]bool, len(cfg.SkipDirs))
	for _, d := range cfg.SkipDirs {
		skipSet[d] = true
	}

	valid := true
	filepath.WalkDir(notesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipSet[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(idx.Built) {
			valid = false
			return filepath.SkipAll
		}
		return nil
	})
	return valid
}

// Load returns the cached index if valid, otherwise rebuilds and caches.
func Load(cfg config.Config) (*Index, error) {
	cached, err := LoadCache(cfg)
	if err == nil && IsCacheValid(cfg, cached) {
		return cached, nil
	}

	idx, err := Build(cfg)
	if err != nil {
		return nil, err
	}

	_ = SaveCache(cfg, idx)
	return idx, nil
}
