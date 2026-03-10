package wikilink

import (
	"testing"

	"github.com/mradoszewski/vinote/internal/index"
)

func TestParse(t *testing.T) {
	tests := []struct {
		content string
		want    []string
	}{
		{"See [[Pigeon]] and [[Nowe Allegro]]", []string{"Pigeon", "Nowe Allegro"}},
		{"Link with [[alias|display text]]", []string{"alias"}},
		{"No links here", nil},
	}

	for _, tt := range tests {
		got := Parse(tt.content)
		if len(got) != len(tt.want) {
			t.Fatalf("Parse(%q) = %v, want %v", tt.content, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("got %q, want %q", got[i], tt.want[i])
			}
		}
	}
}

func TestBacklinks(t *testing.T) {
	idx := &index.Index{
		Notes: []index.Note{
			{Path: "A", Wikilinks: []string{"B", "C"}},
			{Path: "B", Wikilinks: []string{"A"}},
			{Path: "C", Wikilinks: nil},
		},
	}

	got := Backlinks(idx, "B")
	if len(got) != 1 {
		t.Fatalf("Backlinks(B) = %d, want 1", len(got))
	}
	if got[0].Path != "A" {
		t.Errorf("got %q, want A", got[0].Path)
	}

	got = Backlinks(idx, "A")
	if len(got) != 1 {
		t.Fatalf("Backlinks(A) = %d, want 1", len(got))
	}
	if got[0].Path != "B" {
		t.Errorf("got %q, want B", got[0].Path)
	}

	got = Backlinks(idx, "C")
	if len(got) != 1 {
		t.Fatalf("Backlinks(C) = %d, want 1", len(got))
	}
}
