package cli

import (
	"fmt"
	"time"

	"github.com/mradoszewski/vinote/internal/config"
	"github.com/mradoszewski/vinote/internal/index"
	"github.com/spf13/cobra"
)

// IndexCmd returns the "index" subcommand.
func IndexCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "index",
		Short: "Rebuild note index",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			start := time.Now()
			idx, err := index.Build(cfg)
			if err != nil {
				return err
			}

			if err := index.SaveCache(cfg, idx); err != nil {
				return fmt.Errorf("failed to save cache: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Indexed %d notes in %s\n", len(idx.Notes), time.Since(start).Round(time.Millisecond))
			return nil
		},
	}
}
