package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/rpc"
	"go.uber.org/zap"
)

var metaplexMintListEditionCmd = &cobra.Command{
	Use:   "list-edition {master_edition_addr}",
	Short: "Retrieves Mint hash for a given edition drop.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rpcClient := getClient(rpc.WithDebug())

		masterEditionAccount, err := solana.PublicKeyFromBase58(args[0])
		if err != nil {
			return fmt.Errorf("unable to decode mint addr: %w", err)
		}

		validAccountLenght := viper.GetUint64("metaplex-mint-list-edition-cmd-valid-account-length")
		onlyHashes := viper.GetBool("metaplex-mint-list-edition-cmd-only-hashes")

		fmt.Println("")
		fmt.Println("Retrieving Transactions:")
		fmt.Println("")

		limit := uint64(1000)
		lastSignature := ""
		hasMore := true
		totalCount := 0
		failedTrxCount := 0
		invalidTrxCount := 0
		processedTrxCount := 0

		for hasMore {
			signatures, err := rpcClient.GetSignaturesForAddress(masterEditionAccount, &rpc.GetSignaturesForAddressOpts{
				Limit:  limit,
				Before: lastSignature,
			})
			if err != nil {
				return fmt.Errorf("failed to retrieve signatures: %w", err)
			}
			hasMore = len(signatures) != 0

			for _, trxSignature := range signatures {
				lastSignature = trxSignature.Signature
				totalCount++
				if trxSignature.Err != nil {
					failedTrxCount++
					zlog.Info("skipping failed transactions", zap.String("signature", trxSignature.Signature))
					continue
				}
				transaction, err := rpcClient.GetConfirmedTransaction(trxSignature.Signature)
				if err != nil {
					return fmt.Errorf("failed to fetch trx %s: %w", trxSignature.Signature, err)
				}

				if transaction.Transaction == nil {
					return fmt.Errorf("no trx on resp %s", trxSignature.Signature)
				}

				if len(transaction.Transaction.Message.AccountKeys) != int(validAccountLenght) {
					invalidTrxCount++
					zlog.Info("skipping failed transactions", zap.String("signature", trxSignature.Signature))
					continue
				}
				processedTrxCount++
				if onlyHashes {
					fmt.Printf("%s\n", transaction.Transaction.Message.AccountKeys[1].String())
				} else {
					fmt.Printf("%s,%s\n", trxSignature.Signature, transaction.Transaction.Message.AccountKeys[1].String())
				}
			}
		}

		fmt.Println("")
		fmt.Println("Summary")
		fmt.Println("Master Edition > ", masterEditionAccount.String())
		fmt.Println("Total TRX > ", totalCount)
		fmt.Println("Failed TRX > ", failedTrxCount)
		fmt.Println("Invalid TRX > ", invalidTrxCount)
		fmt.Println("Processed TRX > ", processedTrxCount)
		return nil
	},
}

func init() {
	metaplexMintCmd.AddCommand(metaplexMintListEditionCmd)
	metaplexMintListEditionCmd.Flags().Uint64("valid-account-length", 15, "the expected account length on the mind edition transaction")
	metaplexMintListEditionCmd.Flags().Bool("only-hashes", false, "display only the hashes")
}
