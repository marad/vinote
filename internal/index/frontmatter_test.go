package index

import (
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantFM  bool
		wantKey string
		wantVal string
	}{
		{
			name:    "standard frontmatter",
			input:   "---\ntitle: Hello\ntags: topic\n---\n# Body",
			wantFM:  true,
			wantKey: "title",
			wantVal: "Hello",
		},
		{
			name:   "no frontmatter",
			input:  "# Just a heading\nSome content",
			wantFM: false,
		},
		{
			name:   "no closing delimiter",
			input:  "---\ntitle: Hello\n# Body",
			wantFM: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, _ := ParseFrontmatter(tt.input)
			if tt.wantFM {
				if fm == nil {
					t.Fatal("expected frontmatter, got nil")
				}
				if fm[tt.wantKey] != tt.wantVal {
					t.Errorf("got %v, want %v", fm[tt.wantKey], tt.wantVal)
				}
			} else {
				if fm != nil {
					t.Errorf("expected nil frontmatter, got %v", fm)
				}
			}
		})
	}
}

func TestExtractTags(t *testing.T) {
	tests := []struct {
		name string
		fm   map[string]any
		want []string
	}{
		{
			name: "comma-separated",
			fm:   map[string]any{"tags": "topic, 2026"},
			want: []string{"topic", "2026"},
		},
		{
			name: "yaml list",
			fm:   map[string]any{"tags": []any{"meeting", "daily"}},
			want: []string{"meeting", "daily"},
		},
		{
			name: "no tags",
			fm:   map[string]any{"title": "Hello"},
			want: nil,
		},
		{
			name: "nil frontmatter",
			fm:   nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTags(tt.fm)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tag[%d]: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractH1(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{"# Hello World\nSome text", "Hello World"},
		{"Some text\n# Title\nMore", "Title"},
		{"No heading here", ""},
	}

	for _, tt := range tests {
		got := extractH1(tt.body)
		if got != tt.want {
			t.Errorf("extractH1(%q) = %q, want %q", tt.body, got, tt.want)
		}
	}
}

func TestExtractWikilinks(t *testing.T) {
	tests := []struct {
		content string
		want    []string
	}{
		{"See [[Pigeon]] and [[Nowe Allegro]]", []string{"Pigeon", "Nowe Allegro"}},
		{"Link with [[alias|display text]]", []string{"alias"}},
		{"No links here", nil},
		{"Duplicate [[A]] and [[A]]", []string{"A"}},
	}

	for _, tt := range tests {
		got := extractWikilinks(tt.content)
		if len(got) != len(tt.want) {
			t.Fatalf("extractWikilinks(%q) = %v, want %v", tt.content, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("link[%d]: got %q, want %q", i, got[i], tt.want[i])
			}
		}
	}
}
