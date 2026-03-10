package main

import (
	"fmt"
	"os"

	"github.com/mradoszewski/vinote/internal/cli"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "vn",
		Short: "vinote — terminal note-taking system",
	}

	root.AddCommand(
		cli.IndexCmd(),
		cli.QueryCmd(),
		cli.BacklinksCmd(),
		cli.ResolveCmd(),
		cli.WeeklyCmd(),
		cli.WeeklyViewCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
