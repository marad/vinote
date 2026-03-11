package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/mradoszewski/vinote/internal/index"
)

// ByTag returns notes containing the given tag (case-insensitive).
func ByTag(notes []index.Note, tag string) []index.Note {
	tag = strings.ToLower(tag)
	var result []index.Note
	for _, n := range notes {
		for _, t := range n.Tags {
			if strings.ToLower(t) == tag {
				result = append(result, n)
				break
			}
		}
	}
	return result
}

// ByPath returns notes whose path starts with the given prefix.
func ByPath(notes []index.Note, prefix string) []index.Note {
	var result []index.Note
	for _, n := range notes {
		if strings.HasPrefix(n.Path, prefix) {
			result = append(result, n)
		}
	}
	return result
}

// ByFrontmatter returns notes where the frontmatter field matches the value.
func ByFrontmatter(notes []index.Note, key, value string) []index.Note {
	var result []index.Note
	for _, n := range notes {
		if n.Frontmatter == nil {
			continue
		}
		if v, ok := n.Frontmatter[key]; ok {
			if fmt.Sprintf("%v", v) == value {
				result = append(result, n)
			}
		}
	}
	return result
}

// NotFrontmatter excludes notes where the frontmatter field is set to true or non-empty.
func NotFrontmatter(notes []index.Note, key string) []index.Note {
	var result []index.Note
	for _, n := range notes {
		if n.Frontmatter == nil {
			result = append(result, n)
			continue
		}
		v, ok := n.Frontmatter[key]
		if !ok {
			result = append(result, n)
			continue
		}
		switch val := v.(type) {
		case bool:
			if !val {
				result = append(result, n)
			}
		case string:
			if val == "" {
				result = append(result, n)
			}
		default:
			// field exists with non-empty value — exclude
		}
	}
	return result
}

// ByDateRange returns notes where the frontmatter date field falls within [from, to].
func ByDateRange(notes []index.Note, field string, from, to time.Time) []index.Note {
	from = truncateToDate(from)
	to = truncateToDate(to)

	var result []index.Note
	for _, n := range notes {
		if n.Frontmatter == nil {
			continue
		}
		raw, ok := n.Frontmatter[field]
		if !ok {
			continue
		}
		t, ok := parseDate(raw)
		if !ok {
			continue
		}
		t = truncateToDate(t)
		if (t.Equal(from) || t.After(from)) && (t.Equal(to) || t.Before(to)) {
			result = append(result, n)
		}
	}
	return result
}

// parseDate extracts a time.Time from a frontmatter value.
// Handles: time.Time (YAML v3), and various string formats.
func parseDate(v any) (time.Time, bool) {
	switch val := v.(type) {
	case time.Time:
		return val, true
	case string:
		for _, layout := range []string{
			"2006-01-02",
			time.RFC3339,
			"2006-01-02T15:04:05Z",
		} {
			if t, err := time.Parse(layout, val); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
