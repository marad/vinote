package index

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseFrontmatter extracts YAML frontmatter from markdown content.
// Returns the parsed map and the remaining content after the frontmatter block.
func ParseFrontmatter(content string) (map[string]any, string) {
	if !strings.HasPrefix(content, "---") {
		return nil, content
	}

	end := strings.Index(content[3:], "\n---")
	if end == -1 {
		return nil, content
	}

	yamlBlock := content[3 : end+3]
	rest := content[end+3+4:] // skip closing "---\n"

	var fm map[string]any
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return nil, content
	}

	return fm, rest
}

// ExtractTags pulls tags from frontmatter. Supports:
// - YAML list: ["tag1", "tag2"]
// - comma-separated string: "tag1, tag2"
func ExtractTags(fm map[string]any) []string {
	raw, ok := fm["tags"]
	if !ok {
		return nil
	}

	switch v := raw.(type) {
	case []any:
		tags := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				tags = append(tags, strings.TrimSpace(s))
			}
		}
		return tags
	case string:
		parts := strings.Split(v, ",")
		tags := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				tags = append(tags, t)
			}
		}
		return tags
	}

	return nil
}

// ExtractTitle returns the title from frontmatter "title" field, or empty string.
func ExtractTitle(fm map[string]any) string {
	if fm == nil {
		return ""
	}
	if t, ok := fm["title"]; ok {
		if s, ok := t.(string); ok {
			return s
		}
	}
	return ""
}
