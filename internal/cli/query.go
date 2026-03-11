package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/mradoszewski/vinote/internal/config"
	"github.com/mradoszewski/vinote/internal/index"
	"github.com/mradoszewski/vinote/internal/query"
	"github.com/spf13/cobra"
)

// QueryCmd returns the "query" subcommand.
func QueryCmd() *cobra.Command {
	var (
		tags      []string
		notFields []string
		pathPfx   string
		fields    []string
		dateFrom  string
		dateTo    string
		dateField string
		sortBy    string
		jsonOut   bool
		all       bool
	)

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Filter and list notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			idx, err := index.Load(cfg)
			if err != nil {
				return err
			}

			notes := idx.Notes

			if !all {
				for _, tag := range tags {
					notes = query.ByTag(notes, tag)
				}
				for _, nf := range notFields {
					notes = query.NotFrontmatter(notes, nf)
				}
				if pathPfx != "" {
					notes = query.ByPath(notes, pathPfx)
				}
				for _, f := range fields {
					k, v := splitKeyValue(f)
					if k != "" {
						notes = query.ByFrontmatter(notes, k, v)
					}
				}
				if dateFrom != "" || dateTo != "" {
					from, _ := time.Parse("2006-01-02", dateFrom)
					to, _ := time.Parse("2006-01-02", dateTo)
					if to.IsZero() {
						to = time.Now().AddDate(10, 0, 0)
					}
					if from.IsZero() {
						from = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
					}
					field := dateField
					if field == "" {
						field = "date"
					}
					notes = query.ByDateRange(notes, field, from, to)
				}
			}

			// Sort results
			switch sortBy {
			case "mtime":
				sort.Slice(notes, func(i, j int) bool {
					return notes[i].ModTime.After(notes[j].ModTime)
				})
			case "title":
				sort.Slice(notes, func(i, j int) bool {
					return notes[i].Title < notes[j].Title
				})
			case "path":
				sort.Slice(notes, func(i, j int) bool {
					return notes[i].Path < notes[j].Path
				})
			}

			// Default to JSON if stdout is not a terminal
			useJSON := jsonOut
			if !useJSON {
				if fi, err := os.Stdout.Stat(); err == nil {
					if fi.Mode()&os.ModeCharDevice == 0 {
						useJSON = true
					}
				}
			}

			if useJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(notes)
			}

			for _, n := range notes {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", n.Path, n.Title)
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&tags, "tag", nil, "Filter by tag (repeatable)")
	cmd.Flags().StringSliceVar(&notFields, "not", nil, "Exclude by frontmatter field (repeatable)")
	cmd.Flags().StringVar(&pathPfx, "path", "", "Filter by path prefix")
	cmd.Flags().StringSliceVar(&fields, "field", nil, "Filter by frontmatter key=value (repeatable)")
	cmd.Flags().StringVar(&dateFrom, "from", "", "Date range start (YYYY-MM-DD)")
	cmd.Flags().StringVar(&dateTo, "to", "", "Date range end (YYYY-MM-DD)")
	cmd.Flags().StringVar(&dateField, "date-field", "date", "Frontmatter field for date filtering")
	cmd.Flags().StringVar(&sortBy, "sort", "mtime", "Sort by: mtime (newest first), title, path")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Force JSON output")
	cmd.Flags().BoolVar(&all, "all", false, "Return all notes (no filters)")

	return cmd
}

func splitKeyValue(s string) (string, string) {
	for i, c := range s {
		if c == '=' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}
