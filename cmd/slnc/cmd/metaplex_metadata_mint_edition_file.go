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
	"context"
	"encoding/csv"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dhammer"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/programs/token"
	"github.com/streamingfast/solana-go/rpc"
	"github.com/streamingfast/solana-go/rpc/ws"
	"go.uber.org/zap"
	"io"
	"os"
	"strconv"
	"strings"
)

const PARALLE_MINT_EDITION = 10

var metaplexMedatadaMintEditionFromFileCmd = &cobra.Command{
	Use:   "mint-edition-from-file {file-path} {outfile-path}",
	Short: "Mint an edition",
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

		adminKey, err := selectAccountFromVault(vault)
		if err != nil {
			return fmt.Errorf("unable to select admin key: %w", err)
		}

		rentLamports, err := rpcClient.GetMinimumBalanceForRentExemption(ctx, token.MINT_SIZE)
		if err != nil {
			return fmt.Errorf("unbale to get require rent exept for mint size: %w", err)
		}

		filePath := args[0]
		outPath := args[1]
		zlog.Info("processing file mint edition",
			zap.String("file_path", filePath),
			zap.String("out_path", outPath),
		)
		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("unable to open %q: %w", filePath, err)
		}
		defer f.Close()

		mintEditionCompleted := []*MintEdition{}
		mintEditionToProcess := []*MintEdition{}

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

			me, err := fromRecord(rec)
			if err != nil {
				return fmt.Errorf("unable to parse rec at line %d: %w", idx, err)
			}
			if me.TrxID != "" {
				zlog.Info("skipping already minted edition", zap.Reflect("mint_edition", me), zap.Int("row", idx))
				mintEditionCompleted = append(mintEditionCompleted, me)
				continue
			}
			mintEditionToProcess = append(mintEditionToProcess, me)
		}

		queue := dhammer.NewNailer(PARALLE_MINT_EDITION, mintEditionJob(rpcClient, wsClient, programID, adminKey, rentLamports))
		queue.Start(ctx)
		zlog.Info("found mint edition to process", zap.Int("count", len(mintEditionToProcess)))
		// producer async
		go func() {
			for _, p := range mintEditionToProcess {
				queue.Push(ctx, p)
			}
			queue.Close()
		}()

		csvfile, err := os.Create(outPath)
		if err != nil {
			return err
		}

		csvwriter := csv.NewWriter(csvfile)
		if err := csvwriter.Write(expectedHeaders); err != nil {
			return err
		}

		for _, me := range mintEditionCompleted {
			if err := csvwriter.Write([]string{
				me.RecipientAddr.String(),
				me.MasterMintAddr.String(),
				fmt.Sprintf("%d", me.EditionNum),
				me.TrxID,
			}); err != nil {
				return err
			}
		}

		for queueOutput := range queue.Out {
			me := queueOutput.(*MintEdition)

			if err := csvwriter.Write([]string{
				me.RecipientAddr.String(),
				me.MasterMintAddr.String(),
				fmt.Sprintf("%d", me.EditionNum),
				me.TrxID,
			}); err != nil {
				return err
			}
		}

		if queue.Err() != nil {
			return fmt.Errorf("mint edition from file failed: %w", queue.Err())
		}

		csvwriter.Flush()

		csvfile.Close()

		return nil
	},
}

type MintEdition struct {
	RecipientAddr  solana.PublicKey
	MasterMintAddr solana.PublicKey
	EditionNum     uint64
	TrxID          string
}

func fromRecord(rec []string) (*MintEdition, error) {
	recipientAddr, err := solana.PublicKeyFromBase58(rec[0])
	if err != nil {
		return nil, fmt.Errorf("unable to decode mint addr: %w", err)
	}

	masterMintAddr, err := solana.PublicKeyFromBase58(rec[1])
	if err != nil {
		return nil, fmt.Errorf("unable to decode master mint addr: %w", err)
	}

	editionNum, err := strconv.ParseUint(rec[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse edition number %q: %w", rec[2], err)
	}

	me := &MintEdition{
		RecipientAddr:  recipientAddr,
		MasterMintAddr: masterMintAddr,
		EditionNum:     editionNum,
		TrxID:          rec[3],
	}
	return me, nil

}

var expectedHeaders = []string{"recipient", "master mint", "edition", "transaction"}

func extractHeader(records []string) error {
	if len(records) < len(expectedHeaders) {
		return fmt.Errorf("execpged at-least %d columns", len(expectedHeaders))
	}

	for idx, eh := range expectedHeaders {
		if strings.ToLower(eh) != strings.ToLower(records[idx]) {
			return fmt.Errorf("expected column %d to be %s not %s", idx, eh, records[idx])
		}
	}
	return nil
}

func mintEditionJob(rpcClient *rpc.Client, wsClient *ws.Client, programID solana.PublicKey, adminKey solana.PrivateKey, rentLamports int) dhammer.NailerFunc {
	return func(ctx context.Context, in interface{}) (interface{}, error) {
		me := in.(*MintEdition)
		trxHash, err := mintEdition(ctx, rpcClient, wsClient, me, programID, adminKey, rentLamports)
		if err != nil {
			zlog.Error("unable to mint edition", zap.Reflect("mint_edition", me), zap.Error(err))
			return me, nil
		}
		me.TrxID = trxHash
		return me, nil
	}
}

func init() {
	metaplexMetadataCmd.AddCommand(metaplexMedatadaMintEditionFromFileCmd)
}
