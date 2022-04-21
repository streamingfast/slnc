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
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/solana-go"
	associatedtokenaccount "github.com/streamingfast/solana-go/programs/associated-token-account"
	"github.com/streamingfast/solana-go/programs/metaplex"
	"github.com/streamingfast/solana-go/programs/system"
	"github.com/streamingfast/solana-go/programs/token"
	"github.com/streamingfast/solana-go/rpc"
	"github.com/streamingfast/solana-go/rpc/confirm"
	"github.com/streamingfast/solana-go/rpc/ws"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

var metaplexMedatadaMintEditionCmd = &cobra.Command{
	Use:   "mint-edition {recipient} {master_mint} {edition}",
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

		recipientAddr, err := solana.PublicKeyFromBase58(args[0])
		if err != nil {
			return fmt.Errorf("unable to decode mint addr: %w", err)
		}

		masterMintAddr, err := solana.PublicKeyFromBase58(args[1])
		if err != nil {
			return fmt.Errorf("unable to decode master mint addr: %w", err)
		}

		editionNum, err := strconv.ParseUint(args[2], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse edition number %q: %w", args[2], err)
		}

		rentLamports, err := rpcClient.GetMinimumBalanceForRentExemption(ctx, token.MINT_SIZE)
		if err != nil {
			return fmt.Errorf("unbale to get require rent exept for mint size: %w", err)
		}

		trxHash, err := mintEdition(ctx, rpcClient, wsClient, &MintEdition{
			RecipientAddr:  recipientAddr,
			MasterMintAddr: masterMintAddr,
			EditionNum:     editionNum,
		}, programID, adminKey, rentLamports)
		if err != nil {
			return err
		}

		fmt.Printf("Minted: %s Recipient: %s, Edition: %d\n", trxHash, recipientAddr.String(), editionNum)
		return nil
	},
}

type mintEditionAccounts struct {
	programID      solana.PublicKey
	recipientAddr  solana.PublicKey
	masterMintAddr solana.PublicKey
	adminKey       solana.PrivateKey
}

func mintEdition(
	ctx context.Context,
	rpcClient *rpc.Client,
	wsClient *ws.Client,
	mintEdition *MintEdition,
	programID solana.PublicKey,
	adminKey solana.PrivateKey,
	rentLamports int,
) (string, error) {
	mintPublicKey, mintPrivateKey, err := solana.NewRandomPrivateKey()
	if err != nil {
		return "", fmt.Errorf("unable to generate mint private key: %w", err)
	}

	newMetadataAddr, err := metaplex.DeriveMetadataPublicKey(programID, mintPublicKey)
	if err != nil {
		return "", fmt.Errorf("unable to dereive new metadata key: %w", err)
	}

	newEditionAddr, err := metaplex.DeriveMetadataEditionPublicKey(programID, mintPublicKey)
	if err != nil {
		return "", fmt.Errorf("unable to dervie new edition key: %w", err)
	}

	masterEditionAddr, err := metaplex.DeriveMetadataEditionPublicKey(programID, mintEdition.MasterMintAddr)
	if err != nil {
		return "", fmt.Errorf("unable to derive master edition key: %w", err)
	}

	masterMetadataAddr, err := metaplex.DeriveMetadataPublicKey(programID, mintEdition.MasterMintAddr)
	if err != nil {
		return "", fmt.Errorf("unable to derive master edition key: %w", err)
	}

	recipientSPLTokenAccountAddr := associatedtokenaccount.MustGetAssociatedTokenAddress(
		mintPublicKey,
		token.PROGRAM_ID,
		mintEdition.RecipientAddr,
	)

	edition := mintEdition.EditionNum / 248
	editionPdaAddr, err := metaplex.DeriveMetadataEditionCreationMarkPublicKey(programID, mintEdition.MasterMintAddr, fmt.Sprintf("%d", edition))
	if err != nil {
		return "", fmt.Errorf("unable to derive edition creation account: %w", err)
	}

	masterSPLTokenAccountAddr := associatedtokenaccount.MustGetAssociatedTokenAddress(
		mintEdition.MasterMintAddr,
		token.PROGRAM_ID,
		adminKey.PublicKey(),
	)

	zlog.Info("attempting to mint",
		zap.String("recipient_addr", mintEdition.RecipientAddr.String()),
		zap.String("mint_addr", mintPublicKey.String()),
		zap.String("master_mint_addr", mintEdition.MasterMintAddr.String()),
		zap.String("recipient_spl_token_addr", recipientSPLTokenAccountAddr.String()),
		zap.String("admin_key_addr", adminKey.PublicKey().String()),
		zap.String("new_metadata_addr", newMetadataAddr.String()),
		zap.String("new_edition_addr", newEditionAddr.String()),
		zap.String("master_edition_addr", masterEditionAddr.String()),
		zap.String("master_metadata_addr", masterMetadataAddr.String()),
		zap.String("edition_pda_addr", editionPdaAddr.String()),
		zap.String("master_spl_token_account_addr", masterSPLTokenAccountAddr.String()),
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
		associatedtokenaccount.NewCreateInstruction(
			adminKey.PublicKey(),
			recipientSPLTokenAccountAddr,
			mintEdition.RecipientAddr,
			mintPublicKey,
			token.PROGRAM_ID,
		),
		token.NewMintTo(
			1,
			mintPublicKey,
			recipientSPLTokenAccountAddr,
			adminKey.PublicKey(),
		),
		metaplex.NewMintNewEditionFromMasterEditionViaToken(
			programID,
			mintEdition.EditionNum,
			newMetadataAddr,
			newEditionAddr,
			masterEditionAddr,
			mintPublicKey,
			editionPdaAddr,
			adminKey.PublicKey(),
			adminKey.PublicKey(),
			adminKey.PublicKey(),
			masterSPLTokenAccountAddr,
			adminKey.PublicKey(),
			masterMetadataAddr,
		),
	}

	trxHash := ""
	i := 0
	for i < RETRY_COUNT {
		i++
		trxHash, err = sendMintEditionTrx(ctx, rpcClient, wsClient, instructions, func(key solana.PublicKey) *solana.PrivateKey {
			// create account need to be signed by the private key of the new account
			// that is not in the vault and will be lost after the execution.
			if mintPublicKey == key {
				return &mintPrivateKey
			}

			if adminKey.PublicKey() == key {
				return &adminKey
			}
			return nil
		})
		if err != nil {
			zlog.Info("error minting will retry", zap.Error(err))
			time.Sleep(50 * time.Millisecond)
			continue
		}
		break
	}
	if trxHash == "" {
		return "", fmt.Errorf("exceed retry count could not resolve: %w", err)
	}

	return trxHash, nil
}

var RETRY_COUNT = 5

type getterFunc = func(key solana.PublicKey) *solana.PrivateKey

func sendMintEditionTrx(ctx context.Context, rpcClient *rpc.Client, wsClient *ws.Client, instructions []solana.Instruction, getter getterFunc) (string, error) {
	blockHashResult, err := rpcClient.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return "", fmt.Errorf("unable retrieve recent block hash: %w", err)
	}

	trx, err := solana.NewTransaction(instructions, blockHashResult.Value.Blockhash)
	if err != nil {
		return "", fmt.Errorf("unable to create transaction: %w", err)
	}

	_, err = trx.Sign(getter)
	if err != nil {
		return "", fmt.Errorf("unable to sign transaction: %w", err)
	}

	zlog.Info("transction signed, sending to chain", zap.String("trx_sign", trx.Signatures[0].String()))
	trxHash, err := confirm.SendAndConfirmTransaction(ctx, rpcClient, wsClient, trx)
	if err != nil {
		if strings.Contains(err.Error(), "Instruction 4: custom program error: 0x3") {
			zlog.Info("mint edition failed, most likely mint exisit, skipping")
			return "lost", nil
		}
		return "", fmt.Errorf("unable to send transaction: %w", err)
	}
	return trxHash, nil
}
func init() {
	metaplexMetadataCmd.AddCommand(metaplexMedatadaMintEditionCmd)
}
