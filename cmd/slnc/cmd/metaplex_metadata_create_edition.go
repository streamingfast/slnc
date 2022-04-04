// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/programs/metaplex"
	"github.com/streamingfast/solana-go/programs/token"
	"github.com/streamingfast/solana-go/rpc"
	"github.com/streamingfast/solana-go/rpc/confirm"
	"go.uber.org/zap"
	"strconv"
)

var metaplexMedatadaCreateEditionCmd = &cobra.Command{
	Use:   "create-edition {mint} {mas_supply",
	Short: "Create Metaplex Master Edition v3",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		vault := mustGetWallet()
		rpcClient := getClient()
		wsClient, err := getWsClient(ctx)
		if err != nil {
			return fmt.Errorf("unable to retrieve ws client: %w", err)
		}

		metaplexMetaProgramId := viper.GetString("metaplex-global-meta-program-id")
		programID, err := solana.PublicKeyFromBase58(metaplexMetaProgramId)
		if err != nil {
			return fmt.Errorf("unable to decode metaplex metadata programId %q: %w", metaplexMetaProgramId, err)
		}

		mintAddr, err := solana.PublicKeyFromBase58(args[0])
		if err != nil {
			return fmt.Errorf("unable to decode mint addr: %w", err)
		}

		maxSupply, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse max supply: %w", err)
		}

		mint, err := token.FetchMint(ctx, rpcClient, mintAddr)
		if err != nil {
			return fmt.Errorf("unable to get mint: %w", err)
		}

		metadataAddr, err := metaplex.DeriveMetadataPublicKey(programID, mintAddr)
		if err != nil {
			return fmt.Errorf("unable to decode metadata address: %w", err)
		}

		var acc *rpc.GetAccountInfoResult
		acc, err = rpcClient.GetAccountInfo(ctx, metadataAddr)
		if err != nil {
			return fmt.Errorf("unable to retrieve account: %w", err)
		}

		metadata := &metaplex.Metadata{}
		if err := metadata.Decode(acc.Value.Data); err != nil {
			return fmt.Errorf("unable to decode metadata: %w", err)
		}

		if uint64(mint.Supply) != 1 {
			return fmt.Errorf("unable to crearte master edition without a mint supply set to 1. You must mint first")
		}

		var mintAutority *solana.Account
		var metadataUpdateAthority *solana.Account
		for _, privateKey := range vault.KeyBag {
			if privateKey.PublicKey() == mint.MintAuthority {
				mintAutority = &solana.Account{PrivateKey: privateKey}
			}
			if privateKey.PublicKey() == metadata.UpdateAuthority {
				metadataUpdateAthority = &solana.Account{PrivateKey: privateKey}
			}
		}

		if mintAutority == nil {
			return fmt.Errorf("spl token account owner %q must be present in the vault to sign the send transaction", mint.MintAuthority.String())
		}

		if metadataUpdateAthority == nil {
			return fmt.Errorf("metaplex metadata update authority account %q must be present in the vault to sign the send transaction", metadata.UpdateAuthority.String())
		}

		payerKey, err := selectAccountFromVault(vault)
		if err != nil {
			return fmt.Errorf("unable to select admin key: %w", err)
		}

		metadataAccount, err := metaplex.DeriveMetadataPublicKey(programID, mintAddr)
		if err != nil {
			return fmt.Errorf("unable to derive metdata account: %w", err)
		}

		editionAccount, err := metaplex.DeriveMetadataEditionPublicKey(programID, mintAddr)
		if err != nil {
			return fmt.Errorf("unable to derive metdata account: %w", err)
		}

		zlog.Info("creating master token",
			zap.String("mint_addr", mintAddr.String()),
			zap.String("metadata_addr", metadataAccount.String()),
			zap.String("edition_addr", editionAccount.String()),
			zap.String("payer_addr", payerKey.PublicKey().String()),
			zap.Reflect("max_supply", maxSupply),
		)

		instructions := []solana.Instruction{
			metaplex.NewCreateMetadataMasterEditionV3Instruction(
				programID,
				&maxSupply,
				editionAccount,
				mintAddr,
				metadataUpdateAthority.PublicKey(),
				mintAutority.PublicKey(),
				payerKey.PublicKey(),
				metadataAccount,
			),
		}

		blockHashResult, err := rpcClient.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
		if err != nil {
			return fmt.Errorf("unable retrieve recent block hash: %w", err)
		}

		trx, err := solana.NewTransaction(instructions, blockHashResult.Value.Blockhash)
		if err != nil {
			return fmt.Errorf("unable to create transaction: %w", err)
		}

		zlog.Info("signing transactions")
		_, err = trx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
			// create account need to be signed by the private key of the new account
			// that is not in the vault and will be lost after the execution.
			if payerKey.PublicKey() == key {
				return &payerKey
			}

			if mintAutority.PublicKey() == key {
				return &mintAutority.PrivateKey
			}
			if metadataUpdateAthority.PublicKey() == key {
				return &metadataUpdateAthority.PrivateKey
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("unable to sign transaction: %w", err)
		}

		zlog.Info("sending transaction")
		trxHash, err := confirm.SendAndConfirmTransaction(ctx, rpcClient, wsClient, trx)
		if err != nil {
			return fmt.Errorf("unable to send transaction: %w", err)
		}

		fmt.Printf("Creating whitelist token, with transaction hash: %s\n", trxHash)
		fmt.Printf("  Mint Address: %s\n", mintAddr.String())
		fmt.Printf("  Metada Address: %s\n", metadataAddr.String())
		fmt.Printf("  Edition Address: %s\n", editionAccount.String())
		return nil
	},
}

func init() {
	metaplexMetadataCmd.AddCommand(metaplexMedatadaCreateEditionCmd)
}
