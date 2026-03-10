package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mradoszewski/vinote/internal/config"
	"github.com/mradoszewski/vinote/internal/index"
	"github.com/mradoszewski/vinote/internal/weekly"
	"github.com/spf13/cobra"
)

// WeeklyCmd returns the "weekly" subcommand.
func WeeklyCmd() *cobra.Command {
	var (
		create  bool
		weekStr string
	)

	cmd := &cobra.Command{
		Use:   "weekly",
		Short: "Manage weekly notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			ws := parseWeekFlag(weekStr)
			filePath := weekly.WeekFilePath(cfg, ws)

			if create {
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					created, err := weekly.CreateFromTemplate(cfg, ws)
					if err != nil {
						return err
					}
					filePath = created
				}
			}

			fmt.Fprintln(cmd.OutOrStdout(), filePath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&create, "create", false, "Create weekly note from template if missing")
	cmd.Flags().StringVar(&weekStr, "week", "", "Week in YYYY-Www format (default: current)")

	return cmd
}

// WeeklyViewCmd returns the "weekly-view" subcommand.
func WeeklyViewCmd() *cobra.Command {
	var weekStr string

	cmd := &cobra.Command{
		Use:   "weekly-view",
		Short: "Dynamic weekly view data (JSON)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			idx, err := index.Load(cfg)
			if err != nil {
				return err
			}

			ws := parseWeekFlag(weekStr)
			data := weekly.WeeklyView(cfg, idx.Notes, ws)

			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(data)
		},
	}

	cmd.Flags().StringVar(&weekStr, "week", "", "Week in YYYY-Www format (default: current)")

	return cmd
}

// parseWeekFlag parses "2026-W11" format, returns Monday of that week.
// Falls back to current week's Monday on parse error.
func parseWeekFlag(s string) time.Time {
	if s == "" {
		return weekly.WeekStart(time.Now())
	}

	// Expected format: YYYY-Www
	parts := strings.Split(s, "-W")
	if len(parts) != 2 {
		return weekly.WeekStart(time.Now())
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return weekly.WeekStart(time.Now())
	}

	week, err := strconv.Atoi(parts[1])
	if err != nil {
		return weekly.WeekStart(time.Now())
	}

	// ISO week to date: Jan 4 is always in week 1
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.Local)
	_, jan4Week := jan4.ISOWeek()
	return weekly.WeekStart(jan4.AddDate(0, 0, (week-jan4Week)*7))
}
