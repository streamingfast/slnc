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
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/slnc/vault"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/programs/metaplex"
	"github.com/streamingfast/solana-go/programs/system"
	"github.com/streamingfast/solana-go/programs/token"
	"github.com/streamingfast/solana-go/rpc"
	"github.com/streamingfast/solana-go/rpc/confirm"
	"go.uber.org/zap"
	"os"
)

var metaplexMedatadaCreateCmd = &cobra.Command{
	Use:   "create {path_to_data_file}",
	Short: "Create Metaplex metadata V2",
	Args:  cobra.ExactArgs(1),
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

		file, err := os.Open(args[0])
		if err != nil {
			return fmt.Errorf("unable to open data file: %w", err)
		}

		data := &metaplex.DataV2{}
		if err = json.NewDecoder(file).Decode(data); err != nil {
			return fmt.Errorf("unable to decode data: %w", err)
		}

		rentLamports, err := rpcClient.GetMinimumBalanceForRentExemption(ctx, token.MINT_SIZE)
		if err != nil {
			return fmt.Errorf("unbale to get require rent exept for mint size: %w", err)
		}

		mintPublicKey, mintPrivateKey, err := solana.NewRandomPrivateKey()
		if err != nil {
			return fmt.Errorf("unable to generate mint private key: %w", err)
		}

		adminKey, err := selectAccountFromVault(vault)
		if err != nil {
			return fmt.Errorf("unable to select admin key: %w", err)
		}

		metadataAccount, err := metaplex.DeriveMetadataPublicKey(programID, mintPublicKey)
		if err != nil {
			return fmt.Errorf("unable to derive metdata account: %w", err)
		}

		zlog.Info("creating token",
			zap.String("mint_addr", mintPublicKey.String()),
			zap.String("metadata_addr", metadataAccount.String()),
			zap.Reflect("data", data),
		)

		instructions := []solana.Instruction{
			system.NewCreateAccountInstruction(
				uint64(rentLamports),
				token.MINT_SIZE,
				token.PROGRAM_ID,
				adminKey.PublicKey(),
				mintPublicKey,
			),
			token.NewInitializeMintInstruction(
				0,
				mintPublicKey,
				adminKey.PublicKey(),
				nil,
				system.SYSVAR_RENT,
			),
			metaplex.NewCreateMetadataAccountV2Instruction(
				programID,
				*data,
				true,
				metadataAccount,
				mintPublicKey,
				adminKey.PublicKey(),
				adminKey.PublicKey(),
				adminKey.PublicKey(),
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
			if adminKey.PublicKey() == key {
				return &adminKey
			}

			if mintPrivateKey.PublicKey() == key {
				return &mintPrivateKey
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
		fmt.Printf("  Mint Address: %s\n", mintPublicKey.String())
		fmt.Printf("  Owner Address: %s\n", adminKey.PublicKey().String())
		fmt.Printf("Run `slnc metaplex metadata get %s -u %q` to view metadataAddress", mintPublicKey.String(), getRPCURL())
		return nil
	},
}

func init() {
	metaplexMetadataCmd.AddCommand(metaplexMedatadaCreateCmd)
}

func selectAccountFromVault(vault *vault.Vault) (solana.PrivateKey, error) {
	defaultKeyStr := viper.GetString("global-default-vault-key")
	var defaultKey *solana.PublicKey
	var err error
	if defaultKeyStr != "" {
		v, err := solana.PublicKeyFromBase58(defaultKeyStr)
		if err != nil {
			return nil, fmt.Errorf("invalid default key")
		}
		defaultKey = &v
	}
	prompt := promptui.Select{
		Label: "Select an administrator account",
		Items: []string{},
	}

	keys := []string{}
	for _, pkey := range vault.KeyBag {
		if defaultKey != nil && defaultKey.String() == pkey.PublicKey().String() {
			return pkey, nil
		}
		keys = append(keys, pkey.PublicKey().String())
	}

	prompt.Items = keys
	_, result, err := prompt.Run()

	if err != nil {
		return nil, fmt.Errorf("must requretn a key")
	}

	for _, pkey := range vault.KeyBag {
		if pkey.PublicKey().String() == result {
			return pkey, nil
		}
	}

	return nil, fmt.Errorf("unknown key: %q", result)
}
