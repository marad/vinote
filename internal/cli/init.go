package cli

import (
	"fmt"
	"os"

	"github.com/marad/vinote/internal/config"
	"github.com/spf13/cobra"
)

// InitCmd returns the "init" subcommand.
func InitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize vinote configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")

			configDir := config.ConfigDir()
			configFile := config.ConfigFilePath()

			if err := os.MkdirAll(configDir, 0o755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			configCreated := false
			if _, err := os.Stat(configFile); err == nil && !force {
				fmt.Fprintf(cmd.OutOrStdout(), "Config already exists: %s (use --force to overwrite)\n", configFile)
			} else {
				if err := os.WriteFile(configFile, []byte(config.DefaultConfigTOML()), 0o644); err != nil {
					return fmt.Errorf("failed to write config: %w", err)
				}
				configCreated = true
			}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			notesExist := true
			if _, err := os.Stat(cfg.NotesAbsPath()); os.IsNotExist(err) {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: notes directory does not exist: %s\n", cfg.NotesAbsPath())
				notesExist = false
			}

			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "vinote initialized:")
			if configCreated {
				fmt.Fprintf(cmd.OutOrStdout(), "  Config:  %s (created)\n", configFile)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  Config:  %s (already exists)\n", configFile)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  Notes:   %s\n", cfg.NotesAbsPath())
			if !notesExist {
				fmt.Fprintf(cmd.OutOrStdout(), "           ^ directory does not exist\n")
			}
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "Edit the config file to customize your setup.")
			fmt.Fprintln(cmd.OutOrStdout(), "Run 'vn index' to build the note index.")

			return nil
		},
	}

	cmd.Flags().Bool("force", false, "Overwrite existing config file")
	return cmd
}
