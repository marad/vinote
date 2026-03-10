package weekly

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mradoszewski/vinote/internal/config"
	"github.com/mradoszewski/vinote/internal/index"
	"github.com/mradoszewski/vinote/internal/query"
)

// WeeklyData holds dynamic weekly view data built from the index.
type WeeklyData struct {
	Week       string       `json:"week"`
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

// CreateFromTemplate creates a weekly note from the Silverbullet template.
func CreateFromTemplate(cfg config.Config, weekStart time.Time) (string, error) {
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

	targetPath := WeekFilePath(cfg, weekStart)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(targetPath, []byte(text), 0o644); err != nil {
		return "", err
	}

	return targetPath, nil
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

	return WeeklyData{
		Week:       fmt.Sprintf("%d-W%02d", isoYear, isoWeek),
		DateRange:  fmt.Sprintf("%s – %s", weekStart.Format("Jan 2"), weekEnd.Format("Jan 2, 2006")),
		FilePath:   relPath,
		FileExists: fileExists,
		Meetings:   meetings,
		Topics:     topics,
	}
}
