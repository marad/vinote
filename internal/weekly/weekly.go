package weekly

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/marad/vinote/internal/config"
	"github.com/marad/vinote/internal/index"
	"github.com/marad/vinote/internal/query"
)

// WeeklyData holds dynamic weekly view data built from the index.
type WeeklyData struct {
	Week       string       `json:"week"`
	PrevWeek   string       `json:"prev_week"`
	NextWeek   string       `json:"next_week"`
	DateRange  string       `json:"date_range"`
	FilePath   string       `json:"file_path"`
	FileExists bool         `json:"file_exists"`
	Meetings   []index.Note `json:"meetings"`
	Topics     []index.Note `json:"topics"`
}

// WeekStart returns the Monday of the given week, or current week if zero.
func WeekStart(t time.Time) time.Time {
	weekday := t.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	return t.AddDate(0, 0, -int(weekday-time.Monday))
}

// WeekFilePath returns the expected path for a weekly note file.
func WeekFilePath(cfg config.Config, weekStart time.Time) string {
	filename := weekStart.Format("2006-01-02") + ".md"
	return filepath.Join(cfg.WeeklyAbsDir(), filename)
}

// CreateFromTemplate creates a weekly note from the configured template.
// Supports both vinote placeholders ({{weekStart}} etc.) and a subset of
// Silverbullet template syntax. Live ${"$"}{query[[...]]} queries are
// unescaped but not executed — Silverbullet will run them when the page
// is opened there.
func CreateFromTemplate(cfg config.Config, weekStart time.Time, notes []index.Note) (string, error) {
	templatePath := cfg.WeeklyTemplateAbsPath()
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("cannot read weekly template: %w", err)
	}

	weekEnd := weekStart.AddDate(0, 0, 6)
	prevWeek := weekStart.AddDate(0, 0, -7)
	nextWeek := weekStart.AddDate(0, 0, 7)
	_, isoWeek := weekStart.ISOWeek()

	text := string(content)
	text = strings.ReplaceAll(text, "{{weekStart}}", weekStart.Format("2006-01-02"))
	text = strings.ReplaceAll(text, "{{weekEnd}}", weekEnd.Format("2006-01-02"))
	text = strings.ReplaceAll(text, "{{prevWeek}}", prevWeek.Format("2006-01-02"))
	text = strings.ReplaceAll(text, "{{nextWeek}}", nextWeek.Format("2006-01-02"))
	text = strings.ReplaceAll(text, "{{weekNumber}}", fmt.Sprintf("%d", isoWeek))
	text = expandSilverbullet(text, weekStart, notes)

	targetPath := WeekFilePath(cfg, weekStart)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(targetPath, []byte(text), 0o644); err != nil {
		return "", err
	}

	return targetPath, nil
}

var (
	metaTemplateRe = regexp.MustCompile(`(?m)^#meta/template/\S+\s*\n`)
	formatLinkRe   = regexp.MustCompile(`\$\{utils\.formatLink\(([^}]*)\)\}`)
	topicEachRe    = regexp.MustCompile(`(?s)\$\{template\.each\(\s*query\[\[\s*from\s+index\.tag\s+"topic"[^\]]*\]\]\s*,\s*templates\.pageItem\s*\)\s*or\s*"([^"]*)"\s*\}`)
)

// expandSilverbullet substitutes a subset of Silverbullet template expressions:
//   - Strips leading #meta/template/... marker (template-identity tag)
//   - ${journal.getFirstDayOfWeek|previousWeek|nextWeek()} → date (YYYY-MM-DD)
//   - ${utils.formatLink(path, label)} → [[path|label]] (supports Lua ".." concat)
//   - ${template.each(query[[from index.tag "topic" ...]], templates.pageItem) or "fallback"}
//     → bulleted list of non-archived topic notes
//   - ${"$"}{query[[...]]} → ${query[[...]]} (unescapes SB live-query marker)
func expandSilverbullet(text string, weekStart time.Time, notes []index.Note) string {
	prevWeek := weekStart.AddDate(0, 0, -7)
	nextWeek := weekStart.AddDate(0, 0, 7)

	weekStartStr := weekStart.Format("2006-01-02")
	prevWeekStr := prevWeek.Format("2006-01-02")
	nextWeekStr := nextWeek.Format("2006-01-02")

	// Drop template-identity marker so the generated note isn't treated as a template.
	text = metaTemplateRe.ReplaceAllString(text, "")

	// Bare ${journal.*()} calls → unquoted date.
	text = strings.ReplaceAll(text, "${journal.getFirstDayOfWeek()}", weekStartStr)
	text = strings.ReplaceAll(text, "${journal.previousWeek()}", prevWeekStr)
	text = strings.ReplaceAll(text, "${journal.nextWeek()}", nextWeekStr)

	// Embedded journal.*() calls (inside utils.formatLink args etc.) → quoted date
	// so they behave as Lua string literals during downstream parsing.
	text = strings.ReplaceAll(text, "journal.getFirstDayOfWeek()", fmt.Sprintf("%q", weekStartStr))
	text = strings.ReplaceAll(text, "journal.previousWeek()", fmt.Sprintf("%q", prevWeekStr))
	text = strings.ReplaceAll(text, "journal.nextWeek()", fmt.Sprintf("%q", nextWeekStr))

	// ${utils.formatLink(<path expr>, <label expr>)} → [[path|label]]
	text = formatLinkRe.ReplaceAllStringFunc(text, func(match string) string {
		inner := formatLinkRe.FindStringSubmatch(match)[1]
		args, ok := splitTopLevelArgs(inner, 2)
		if !ok {
			return match
		}
		path, ok := evalLuaStringConcat(args[0])
		if !ok {
			return match
		}
		label, ok := evalLuaStringConcat(args[1])
		if !ok {
			return match
		}
		return fmt.Sprintf("[[%s|%s]]", path, label)
	})

	// ${template.each(query[[from index.tag "topic" ...]], templates.pageItem) or "fallback"}
	text = topicEachRe.ReplaceAllStringFunc(text, func(match string) string {
		fallback := ""
		if sm := topicEachRe.FindStringSubmatch(match); len(sm) >= 2 {
			fallback = sm[1]
		}
		active := query.ByTag(query.NotFrontmatter(notes, "archived"), "topic")
		var lines []string
		for _, n := range active {
			// SB query excludes `string.startsWith(name, "Archiwum")` — same here.
			if strings.HasPrefix(n.Path, "Archiwum") {
				continue
			}
			lines = append(lines, fmt.Sprintf("* [[%s]]", n.Path))
		}
		if len(lines) == 0 {
			return fallback
		}
		return strings.Join(lines, "\n")
	})

	// Unescape SB live-query marker: ${"$"}{query[[...]]} → ${query[[...]]}
	text = strings.ReplaceAll(text, `${"$"}`, "$")

	return text
}

// splitTopLevelArgs splits s on commas that are NOT inside double-quoted strings.
// Returns nil,false if the count doesn't match.
func splitTopLevelArgs(s string, want int) ([]string, bool) {
	var args []string
	var cur strings.Builder
	inStr := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' && (i == 0 || s[i-1] != '\\'):
			inStr = !inStr
			cur.WriteByte(c)
		case c == ',' && !inStr:
			args = append(args, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		args = append(args, strings.TrimSpace(cur.String()))
	}
	if len(args) != want {
		return nil, false
	}
	return args, true
}

// evalLuaStringConcat evaluates a Lua-style string expression like
// `"a" .. "b" .. "c"` → "abc". Returns false if the expression isn't a
// pure concatenation of quoted literals.
func evalLuaStringConcat(expr string) (string, bool) {
	parts := strings.Split(expr, "..")
	var out strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) < 2 || p[0] != '"' || p[len(p)-1] != '"' {
			return "", false
		}
		out.WriteString(p[1 : len(p)-1])
	}
	return out.String(), true
}

// WeeklyView builds dynamic weekly data from the index.
func WeeklyView(cfg config.Config, notes []index.Note, weekStart time.Time) WeeklyData {
	weekEnd := weekStart.AddDate(0, 0, 6)
	isoYear, isoWeek := weekStart.ISOWeek()

	filePath := WeekFilePath(cfg, weekStart)
	_, err := os.Stat(filePath)
	fileExists := err == nil

	// Relative path for JSON output
	relPath, _ := filepath.Rel(cfg.NotesAbsPath(), filePath)

	meetings := query.ByDateRange(query.ByTag(notes, "meeting"), "date", weekStart, weekEnd)
	topics := query.ByTag(query.NotFrontmatter(notes, "archived"), "topic")

	prevStart := weekStart.AddDate(0, 0, -7)
	nextStart := weekStart.AddDate(0, 0, 7)
	prevYear, prevW := prevStart.ISOWeek()
	nextYear, nextW := nextStart.ISOWeek()

	return WeeklyData{
		Week:       fmt.Sprintf("%d-W%02d", isoYear, isoWeek),
		PrevWeek:   fmt.Sprintf("%d-W%02d", prevYear, prevW),
		NextWeek:   fmt.Sprintf("%d-W%02d", nextYear, nextW),
		DateRange:  fmt.Sprintf("%s – %s", weekStart.Format("Jan 2"), weekEnd.Format("Jan 2, 2006")),
		FilePath:   relPath,
		FileExists: fileExists,
		Meetings:   meetings,
		Topics:     topics,
	}
}
