package weekly

import (
	"strings"
	"testing"
	"time"

	"github.com/marad/vinote/internal/index"
)

func TestExpandSilverbullet_StripsMetaTemplate(t *testing.T) {
	in := "#meta/template/slash\n# Rest\n"
	got := expandSilverbullet(in, monday("2026-04-06"), nil)
	if strings.Contains(got, "#meta/template") {
		t.Fatalf("meta/template marker should be stripped, got: %q", got)
	}
	if !strings.HasPrefix(got, "# Rest\n") {
		t.Fatalf("expected body to start with '# Rest', got: %q", got)
	}
}

func TestExpandSilverbullet_JournalFunctions(t *testing.T) {
	in := "# ${journal.getFirstDayOfWeek()}\nprev: ${journal.previousWeek()}\nnext: ${journal.nextWeek()}\n"
	got := expandSilverbullet(in, monday("2026-04-06"), nil)
	want := "# 2026-04-06\nprev: 2026-03-30\nnext: 2026-04-13\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestExpandSilverbullet_FormatLink(t *testing.T) {
	in := `${utils.formatLink("Journal/Week/" .. journal.previousWeek(), "Poprzedni tydzień")}`
	got := expandSilverbullet(in, monday("2026-04-06"), nil)
	want := "[[Journal/Week/2026-03-30|Poprzedni tydzień]]"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestExpandSilverbullet_FormatLink_SimpleArgs(t *testing.T) {
	in := `${utils.formatLink("Foo/Bar", "Label")}`
	got := expandSilverbullet(in, monday("2026-04-06"), nil)
	want := "[[Foo/Bar|Label]]"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestExpandSilverbullet_TopicEach_WithTopics(t *testing.T) {
	notes := []index.Note{
		{Path: "Allegro/Tematy/Pigeon", Tags: []string{"topic"}},
		{Path: "Allegro/Tematy/Subsidy", Tags: []string{"topic"}, Frontmatter: map[string]any{"archived": true}},
		{Path: "Archiwum/Old", Tags: []string{"topic"}},
		{Path: "Allegro/Tematy/Fresh", Tags: []string{"topic"}},
	}
	in := `${template.each(query[[from index.tag "topic" where not archived and not string.startsWith(name, "Archiwum")]], templates.pageItem) or "Nic się nie dzieje?!"}`
	got := expandSilverbullet(in, monday("2026-04-06"), notes)
	want := "* [[Allegro/Tematy/Pigeon]]\n* [[Allegro/Tematy/Fresh]]"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestExpandSilverbullet_TopicEach_Empty(t *testing.T) {
	in := `${template.each(query[[from index.tag "topic" where not archived]], templates.pageItem) or "Nothing!"}`
	got := expandSilverbullet(in, monday("2026-04-06"), nil)
	if got != "Nothing!" {
		t.Fatalf("want fallback 'Nothing!', got %q", got)
	}
}

func TestExpandSilverbullet_UnescapesLiveQuery(t *testing.T) {
	in := `${"$"}{query[[from index.tag "meeting"]]}`
	got := expandSilverbullet(in, monday("2026-04-06"), nil)
	want := `${query[[from index.tag "meeting"]]}`
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestExpandSilverbullet_FullTemplate(t *testing.T) {
	// Emulates the user's Silverbullet template end-to-end.
	in := `#meta/template/slash
# ${journal.getFirstDayOfWeek()}
${utils.formatLink("Journal/Week/" .. journal.previousWeek(), "Poprzedni tydzień")} | ${utils.formatLink("Journal/Week/" .. journal.nextWeek(), "Następny tydzień")}

## Cele
${template.each(query[[from index.tag "topic" where not archived and not string.startsWith(name, "Archiwum")]], templates.pageItem) or "Nic się nie dzieje?!"}

## Daily Notes
${"$"}{query[[from index.tag "page"]]}
`
	notes := []index.Note{
		{Path: "Allegro/Tematy/Pigeon", Tags: []string{"topic"}},
	}
	got := expandSilverbullet(in, monday("2026-04-06"), notes)
	want := `# 2026-04-06
[[Journal/Week/2026-03-30|Poprzedni tydzień]] | [[Journal/Week/2026-04-13|Następny tydzień]]

## Cele
* [[Allegro/Tematy/Pigeon]]

## Daily Notes
${query[[from index.tag "page"]]}
`
	if got != want {
		t.Fatalf("full template mismatch\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func TestSplitTopLevelArgs(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`"a", "b"`, []string{`"a"`, `"b"`}},
		{`"a, x", "b"`, []string{`"a, x"`, `"b"`}},
		{`"a" .. "b", "c"`, []string{`"a" .. "b"`, `"c"`}},
	}
	for _, c := range cases {
		got, ok := splitTopLevelArgs(c.in, 2)
		if !ok {
			t.Errorf("splitTopLevelArgs(%q) not ok", c.in)
			continue
		}
		if len(got) != 2 || got[0] != c.want[0] || got[1] != c.want[1] {
			t.Errorf("splitTopLevelArgs(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestEvalLuaStringConcat(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{`"abc"`, "abc", true},
		{`"a" .. "b"`, "ab", true},
		{`"a" .. "b" .. "c"`, "abc", true},
		{`"a" .. x`, "", false},
		{`notquoted`, "", false},
	}
	for _, c := range cases {
		got, ok := evalLuaStringConcat(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("evalLuaStringConcat(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// monday returns Monday 00:00 local time for the given YYYY-MM-DD string.
func monday(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		panic(err)
	}
	return t
}
