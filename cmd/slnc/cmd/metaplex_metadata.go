package cmd

import (
	"github.com/spf13/cobra"
)

var metaplexMetadataCmd = &cobra.Command{
	Use:   "metadata",
	Short: "Metaplex metadata related commands",
}

func init() {
	metaplexCmd.AddCommand(metaplexMetadataCmd)
}
