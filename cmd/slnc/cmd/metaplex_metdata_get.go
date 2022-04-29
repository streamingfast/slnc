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
	"github.com/streamingfast/solana-go/rpc"

	"github.com/spf13/viper"

	"github.com/spf13/cobra"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/programs/metaplex"
)

var metaplexmetadataGetMintCmd = &cobra.Command{
	Use:   "get {mint_addr|metadata_addr}",
	Short: "Get Metaplex metadata for a given mint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		rpcClient := getClient()

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

		return getAndDisplayMetadata(metadataAddr, metadata)
	},
}

func init() {
	metaplexMetadataCmd.AddCommand(metaplexmetadataGetMintCmd)
}

func getMetaplexMetadata(ctx context.Context, rpcClient *rpc.Client, metaplexMetadataProgramID solana.PublicKey, addr solana.PublicKey) (solana.PublicKey, *metaplex.Metadata, error) {
	metadataAddr := addr
	var acc *rpc.GetAccountInfoResult
	var err error
	if acc, err = rpcClient.GetAccountInfo(metadataAddr); err != nil {
		return solana.PublicKey{}, nil, fmt.Errorf("unable to retrieve account: %w", err)
	}
	if acc.Value.Owner != metaplexMetadataProgramID {
		metadataAddr, err = metaplex.DeriveMetadataPublicKey(metaplexMetadataProgramID, addr)
		if err != nil {
			return solana.PublicKey{}, nil, fmt.Errorf("unable to derive metadata address: %w", err)
		}

		acc, err = rpcClient.GetAccountInfo(metadataAddr)
		if err != nil {
			return solana.PublicKey{}, nil, fmt.Errorf("unable to retrieve account: %w", err)
		}

		if acc.Value.Owner != metaplexMetadataProgramID {
			return solana.PublicKey{}, nil, fmt.Errorf("account provided is neither a mint or metadata addresses")
		}
	}

	metadata := &metaplex.Metadata{}
	if err := metadata.Decode(acc.Value.Data); err != nil {
		return solana.PublicKey{}, nil, fmt.Errorf("unable to decode metadata: %w", err)
	}

	return metadataAddr, metadata, nil
}

func getAndDisplayMetadata(metadataAddr solana.PublicKey, metadata *metaplex.Metadata) error {
	fmt.Println()
	fmt.Printf("Metadata Addr: %s\n", metadataAddr.String())
	fmt.Printf("Mint: %s\n", metadata.Mint.String())
	fmt.Printf("Udpate Authority: %s\n", metadata.UpdateAuthority.String())
	fmt.Printf("Primary Sale Happened: %t\n", metadata.PrimarySaleHappened)
	fmt.Printf("Is Mutable: %t\n", metadata.IsMutable)
	if metadata.EditionNonce != nil {
		fmt.Printf("Edition Nonce: %d\n", *metadata.EditionNonce)
	} else {
		fmt.Printf("No Edition Nonce\n")
	}
	if metadata.Collection != nil {
		fmt.Printf("Collection Key: %s\n", (*metadata.Collection).Key.String())
		fmt.Printf("Collection verified: %t\n", (*metadata.Collection).Verified)
	} else {
		fmt.Printf("No Collection\n")
	}

	if metadata.TokenStandard != nil {
		fmt.Printf("Token Standard: %d\n", (*metadata.TokenStandard))
	} else {
		fmt.Printf("No Token Standard\n")
	}

	fmt.Println("Data")
	fmt.Printf("> Name:%s \n", metadata.Data.Name)
	fmt.Printf("> Symbol: %s\n", metadata.Data.Symbol)
	fmt.Printf("> URI: %s\n", metadata.Data.URI)
	fmt.Printf("> Seller Basis Points: %d\n", metadata.Data.SellerFeeBasisPoints)
	if metadata.Data.Creators != nil && len(*metadata.Data.Creators) > 0 {
		fmt.Printf("> %d creators\n", len(*metadata.Data.Creators))
		for _, creator := range *metadata.Data.Creators {
			verified := "[ ]"
			if creator.Verified {
				verified = "[✅]"
			}

			fmt.Printf("> %s %d %s\n", creator.Address.String(), creator.Share, verified)
		}
	} else {
		fmt.Println("> No creators found")
	}

	if metadata.Data.URI != "" {
		fmt.Println()
		fetchAndPrintJSONFromURL("Metadata", metadata.Data.URI)
	}

	return nil
}
