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
	"encoding/json"
	"fmt"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/rpc"
	"github.com/streamingfast/solana-go/rpc/confirm"
	"go.uber.org/zap"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/solana-go/programs/metaplex"
)

var metaplexMedatadaUpdateV1Cmd = &cobra.Command{
	Use:   "update-v1 {mint_addr/metadata_addr} {path_to_data_file} {update_authority}",
	Short: "Update Metaplex Metadata V1 for a given mint",
	Args:  cobra.ExactArgs(3),
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

		address, err := solana.PublicKeyFromBase58(args[0])
		if err != nil {
			return fmt.Errorf("unable to decode mint addr: %w", err)
		}

		metadataAddr, metadata, err := getMetaplexMetadata(ctx, rpcClient, programID, address)
		if err != nil {
			return fmt.Errorf("unable to rerieve metadata account: %w", err)
		}
		_ = metadata

		file, err := os.Open(args[1])
		if err != nil {
			return fmt.Errorf("unable to open data file: %w", err)
		}

		newMetadata := &metaplex.Data{}
		if err = json.NewDecoder(file).Decode(newMetadata); err != nil {
			return fmt.Errorf("unable to decode data: %w", err)
		}

		updateAuthority, err := solana.PublicKeyFromBase58(args[2])
		if err != nil {
			return fmt.Errorf("unable to decode mint addr: %w", err)
		}

		if !metadata.IsMutable {
			return fmt.Errorf("unable to update metadata, content is not mutable")
		}

		if metadata.UpdateAuthority != updateAuthority {
			return fmt.Errorf("current update authority in metatadata %q does not match update authority passed via command line %q", metadata.UpdateAuthority.String(), updateAuthority.String())
		}

		fmt.Printf("Updating Metadata @ %q from: \n", metadataAddr.String())
		cnt, _ := json.MarshalIndent(metadata, "", " ")
		fmt.Println(string(cnt))
		fmt.Println("to:")
		cnt, _ = json.MarshalIndent(newMetadata, "", " ")
		fmt.Println(string(cnt))

		if newMetadata.Creators != nil {
			for idx, creator := range *newMetadata.Creators {
				if creator.Address == updateAuthority {
					creator.Verified = true
					(*newMetadata.Creators)[idx] = creator
				}
			}
		}

		updateMetadataInstruction := metaplex.NewUpdateMetadataAccountV1Instruction(
			programID,
			newMetadata,
			&updateAuthority,
			&metadata.PrimarySaleHappened,
			metadataAddr,
			updateAuthority,
		)

		zlog.Debug("retrieving block hash")
		blockHashResult, err := rpcClient.GetLatestBlockhash(rpc.CommitmentFinalized)
		if err != nil {
			return fmt.Errorf("unable retrieve recent block hash: %w", err)
		}

		zlog.Debug("found block hash", zap.String("block_hash", blockHashResult.Value.Blockhash.String()))

		trx, err := solana.NewTransaction([]solana.Instruction{
			updateMetadataInstruction,
		}, blockHashResult.Value.Blockhash)
		if err != nil {
			return fmt.Errorf("unable to craft transaction: %w", err)
		}

		zlog.Debug("signing metaplex transaction")
		_, err = trx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
			// create account need to be signed by the private key of the new account
			// that is not in the vault and will be lost after the execution.
			for _, k := range vault.KeyBag {
				if k.PublicKey() == key {
					return &k
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("unable to sign transaction: %w", err)
		}

		zlog.Debug("sending transaction")

		trxHash, err := confirm.SendAndConfirmTransaction(ctx, rpcClient, wsClient, trx)
		if err != nil {
			return fmt.Errorf("unable to send transaction: %w", err)
		}

		fmt.Printf("Metaplex Metadata Updated, with transaction hash: %s\n", trxHash)
		fmt.Printf("  Metadata Address: %s\n", metadataAddr.String())
		fmt.Printf("Run `slnc metaplex get %s` to view metadata", metadataAddr.String())
		return nil
	},
}

func init() {
	metaplexMetadataCmd.AddCommand(metaplexMedatadaUpdateV1Cmd)
}
