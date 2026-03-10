package cli

import (
	"encoding/json"
	"fmt"

	"github.com/mradoszewski/vinote/internal/config"
	"github.com/mradoszewski/vinote/internal/index"
	"github.com/mradoszewski/vinote/internal/wikilink"
	"github.com/spf13/cobra"
)

// BacklinksCmd returns the "backlinks" subcommand.
func BacklinksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backlinks <note-path>",
		Short: "Find notes linking to the given note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			idx, err := index.Load(cfg)
			if err != nil {
				return err
			}

			notes := wikilink.Backlinks(idx, args[0])

			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			if notes == nil {
				notes = []index.Note{}
			}
			if err := enc.Encode(notes); err != nil {
				return fmt.Errorf("failed to encode backlinks: %w", err)
			}
			return nil
		},
	}
}
