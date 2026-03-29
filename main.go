package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:           "lmtds",
		Short:         "laravel migrations to dbdiagram schema",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newLmtdsCmd())
	if err := fang.Execute(context.Background(), root); err != nil {
		os.Exit(1)
	}
}
