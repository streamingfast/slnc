package cmd

import (
	"github.com/spf13/cobra"
)

var metaplexCandymachineCmd = &cobra.Command{
	Use:   "candymachine",
	Short: "Metaplex Candy Machine related commands",
}

func init() {
	metaplexCmd.AddCommand(metaplexCandymachineCmd)
}
