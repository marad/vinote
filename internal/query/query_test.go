package query

import (
	"testing"
	"time"

	"github.com/mradoszewski/vinote/internal/index"
)

var testNotes = []index.Note{
	{Path: "topics/A", Title: "Topic A", Tags: []string{"topic"}, Frontmatter: map[string]any{"tags": "topic"}},
	{Path: "topics/B", Title: "Topic B", Tags: []string{"topic"}, Frontmatter: map[string]any{"tags": "topic", "archived": true}},
	{Path: "meetings/M1", Title: "Meeting 1", Tags: []string{"meeting"}, Frontmatter: map[string]any{"tags": "meeting", "date": "2026-03-10"}},
	{Path: "meetings/M2", Title: "Meeting 2", Tags: []string{"meeting"}, Frontmatter: map[string]any{"tags": "meeting", "date": "2026-03-15"}},
	{Path: "other/X", Title: "Other", Tags: nil, Frontmatter: nil},
}

func TestByTag(t *testing.T) {
	got := ByTag(testNotes, "topic")
	if len(got) != 2 {
		t.Errorf("ByTag(topic) = %d notes, want 2", len(got))
	}
	got = ByTag(testNotes, "Meeting")
	if len(got) != 2 {
		t.Errorf("ByTag(Meeting) case-insensitive = %d notes, want 2", len(got))
	}
}

func TestByPath(t *testing.T) {
	got := ByPath(testNotes, "topics/")
	if len(got) != 2 {
		t.Errorf("ByPath(topics/) = %d notes, want 2", len(got))
	}
}

func TestNotFrontmatter(t *testing.T) {
	got := NotFrontmatter(testNotes, "archived")
	if len(got) != 4 {
		t.Errorf("NotFrontmatter(archived) = %d notes, want 4", len(got))
	}
}

func TestByDateRange(t *testing.T) {
	from := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	got := ByDateRange(testNotes, "date", from, to)
	if len(got) != 1 {
		t.Errorf("ByDateRange = %d notes, want 1", len(got))
	}
	if len(got) > 0 && got[0].Title != "Meeting 1" {
		t.Errorf("got %q, want Meeting 1", got[0].Title)
	}
}

func TestComposition(t *testing.T) {
	// Active topics: tag:topic, not archived
	got := ByTag(NotFrontmatter(testNotes, "archived"), "topic")
	if len(got) != 1 {
		t.Errorf("composition = %d notes, want 1", len(got))
	}
	if len(got) > 0 && got[0].Title != "Topic A" {
		t.Errorf("got %q, want Topic A", got[0].Title)
	}
}
