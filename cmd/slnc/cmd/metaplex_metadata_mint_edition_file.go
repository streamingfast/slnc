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
	"encoding/csv"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/programs/token"
	"io"
	"os"
	"strconv"
	"strings"
)

var metaplexMedatadaMintEditionFromFileCmd = &cobra.Command{
	Use:   "mint-edition-from-file {file-path}",
	Short: "Mint an edition",
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

		adminKey, err := selectAccountFromVault(vault)
		if err != nil {
			return fmt.Errorf("unable to select admin key: %w", err)
		}

		rentLamports, err := rpcClient.GetMinimumBalanceForRentExemption(ctx, token.MINT_SIZE)
		if err != nil {
			return fmt.Errorf("unbale to get require rent exept for mint size: %w", err)
		}

		filePath := args[0]
		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("unable to open %q: %w", filePath, err)
		}
		defer f.Close()
		csvReader := csv.NewReader(f)
		seenHeader := false
		idx := 0
		for {
			idx++
			rec, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read line: %w", err)
			}

			if !seenHeader {
				if err = extractHeader(rec); err != nil {
					return fmt.Errorf("failed to validate header: %w", err)
				}
				seenHeader = true
				continue
			}

			recipientAddr, err := solana.PublicKeyFromBase58(rec[0])
			if err != nil {
				return fmt.Errorf("unable to decode mint addr: %w", err)
			}

			masterMintAddr, err := solana.PublicKeyFromBase58(rec[1])
			if err != nil {
				return fmt.Errorf("unable to decode master mint addr: %w", err)
			}

			editionNum, err := strconv.ParseUint(rec[2], 10, 64)
			if err != nil {
				return fmt.Errorf("unable to parse edition number %q: %w", rec[2], err)
			}

			trxHash, err := mintEdition(ctx, rpcClient, wsClient, &mintEditionAccounts{
				programID:      programID,
				recipientAddr:  recipientAddr,
				masterMintAddr: masterMintAddr,
				adminKey:       adminKey,
			}, editionNum, rentLamports)
			if err != nil {
				return err
			}

			fmt.Printf("Minted: %s Recipient: %s, Edition: %d\n", trxHash, recipientAddr.String(), editionNum)
		}
		return nil
	},
}

var expectedHeaders = []string{"recipient", "master mint", "edition"}

func extractHeader(records []string) error {
	if len(records) < len(expectedHeaders) {
		return fmt.Errorf("execpged at0least 10 columns")
	}

	for idx, eh := range expectedHeaders {
		if strings.ToLower(eh) != strings.ToLower(records[idx]) {
			return fmt.Errorf("Expected column %d to be %s not %s", idx, eh, records[idx])
		}
	}
	return nil
}

func init() {
	metaplexMetadataCmd.AddCommand(metaplexMedatadaMintEditionFromFileCmd)
}
