package main

import (
	"fmt"

	"github.com/guigolab/bamstats"
	"github.com/spf13/cobra"
)

func versionString(cmd *cobra.Command) string {
	return fmt.Sprintf("%s version %s\n", cmd.Name(), bamstats.Version())
}

func hasVersionFlag(cmd *cobra.Command) (version bool) {
	version, err := cmd.Flags().GetBool("version")
	if err == nil && version {
		fmt.Print(versionString(cmd))
	}
	return
}
