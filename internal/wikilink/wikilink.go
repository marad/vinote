package wikilink

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mradoszewski/vinote/internal/index"
)

var wikilinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

// Parse extracts all wikilink targets from markdown content.
func Parse(content string) []string {
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

// Resolve resolves a wikilink to an absolute file path.
// Strategy:
// 1. Exact match: notesDir/link.md
// 2. Exact match: notesDir/link/index.md
// 3. Index lookup: match by filename
func Resolve(link string, notesDir string, idx *index.Index) (string, error) {
	// 1. Exact path match
	exact := filepath.Join(notesDir, link+".md")
	if _, err := os.Stat(exact); err == nil {
		return exact, nil
	}

	// 2. Index file
	indexFile := filepath.Join(notesDir, link, "index.md")
	if _, err := os.Stat(indexFile); err == nil {
		return indexFile, nil
	}

	// 3. Search index by filename
	linkBase := filepath.Base(link)
	for _, n := range idx.Notes {
		noteBase := filepath.Base(n.Path)
		if strings.EqualFold(noteBase, linkBase) {
			return filepath.Join(notesDir, n.Path+".md"), nil
		}
	}

	return "", fmt.Errorf("wikilink not found: %s", link)
}

// Backlinks returns notes that link to the given note path (relative, without .md).
func Backlinks(idx *index.Index, notePath string) []index.Note {
	// Normalize: the target could be matched by full path or just the filename
	targetBase := filepath.Base(notePath)

	var result []index.Note
	for _, n := range idx.Notes {
		if n.Path == notePath {
			continue // skip self
		}
		for _, link := range n.Wikilinks {
			linkBase := filepath.Base(link)
			if link == notePath || strings.EqualFold(linkBase, targetBase) {
				result = append(result, n)
				break
			}
		}
	}
	return result
}
