package query

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/marad/vinote/internal/index"
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

// ByName returns notes whose title fuzzy-matches the pattern, sorted by score (best first).
func ByName(notes []index.Note, pattern string) []index.Note {
	pattern = strings.ToLower(pattern)
	type scored struct {
		note  index.Note
		score int
	}
	var matches []scored
	for _, n := range notes {
		if s := fuzzyScore(strings.ToLower(n.Title), pattern); s > 0 {
			matches = append(matches, scored{n, s})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})
	result := make([]index.Note, len(matches))
	for i, m := range matches {
		result[i] = m.note
	}
	return result
}

// fuzzyScore returns a score for how well pattern matches text (0 = no match).
// Higher is better. Rewards consecutive matches and word-boundary alignment.
func fuzzyScore(text, pattern string) int {
	if len(pattern) == 0 {
		return 1
	}
	score := 0
	pi := 0
	prevMatchPos := -2
	for i := 0; i < len(text) && pi < len(pattern); i++ {
		if text[i] != pattern[pi] {
			continue
		}
		score++
		if i == prevMatchPos+1 {
			score += 3
		}
		if i == 0 || text[i-1] == ' ' || text[i-1] == '/' || text[i-1] == '-' || text[i-1] == '_' {
			score += 5
		}
		prevMatchPos = i
		pi++
	}
	if pi < len(pattern) {
		return 0
	}
	if bonus := 50 - len(text); bonus > 0 {
		score += bonus
	}
	return score
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
