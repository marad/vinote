package cli

import (
	"fmt"

	"github.com/mradoszewski/vinote/internal/config"
	"github.com/mradoszewski/vinote/internal/index"
	"github.com/mradoszewski/vinote/internal/wikilink"
	"github.com/spf13/cobra"
)

// ResolveCmd returns the "resolve" subcommand.
func ResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <wikilink>",
		Short: "Resolve a wikilink to a file path",
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

			path, err := wikilink.Resolve(args[0], cfg.NotesAbsPath(), idx)
			if err != nil {
				return fmt.Errorf("not found: %s", args[0])
			}

			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}
