package cmd

import (
	"github.com/spf13/cobra"
)

var metaplexMintCmd = &cobra.Command{
	Use:   "mint",
	Short: "Metaplex related mint",
}

func init() {
	metaplexCmd.AddCommand(metaplexMintCmd)
}
