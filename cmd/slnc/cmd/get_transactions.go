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
	"github.com/spf13/cobra"
	"github.com/streamingfast/solana-go"
	_ "github.com/streamingfast/solana-go/programs/serum"
	_ "github.com/streamingfast/solana-go/programs/system"
	_ "github.com/streamingfast/solana-go/programs/token"
	_ "github.com/streamingfast/solana-go/programs/tokenregistry"
	"github.com/streamingfast/solana-go/rpc"
	"github.com/streamingfast/solana-go/text"
)

var getTransactionsCmd = &cobra.Command{
	Use:   "transactions {account}",
	Short: "Retrieve transaction for a specific account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getClient()

		address := args[0]
		pubKey, err := solana.PublicKeyFromBase58(address)
		if err != nil {
			return fmt.Errorf("invalid account address %q: %w", address, err)
		}

		csList, err := client.GetSignaturesForAddress(pubKey, &rpc.GetSignaturesForAddressOpts{
			Limit:  1,
			Before: "",
			Until:  "",
		})
		if err != nil {
			return fmt.Errorf("unable to retrieve confirmed transaction signatures for account: %w", err)
		}

		for _, cs := range csList {
			fmt.Println("-----------------------------------------------------------------------------------------------")
			text.EncoderColorCyan.Print("Transaction: ")
			fmt.Println(cs.Signature)

			text.EncoderColorGreen.Print("Slot: ")
			fmt.Println(cs.Slot)
			text.EncoderColorGreen.Print("Memo: ")
			fmt.Println(cs.Memo)

			ct, err := client.GetConfirmedTransaction(cs.Signature)
			if err != nil {
				return fmt.Errorf("unable to get confirmed transaction with signature %q: %w", cs.Signature, err)
			}

			if ct.Meta.Err != nil {
				cnt, _ := json.Marshal(ct.Meta.Err.Raw)
				return fmt.Errorf("unable to get confirmed transaction with signature %q: %s", cs.Signature, string(cnt))
			}

			fmt.Print("\nInstructions:\n-------------\n\n")
			for _, i := range ct.Transaction.Message.Instructions {

				id := ct.Transaction.Message.AccountKeys[i.ProgramIdIndex]
				decoder := solana.InstructionDecoderRegistry[id.String()]
				if decoder == nil {
					fmt.Println("raw instruction:")
					fmt.Printf("Program: %s Data: %s\n", id.String(), i.Data)
					fmt.Println("Accounts:")
					for _, accIndex := range i.Accounts {
						key := ct.Transaction.Message.AccountKeys[accIndex]
						fmt.Printf("%s\n", key.String())
						//fmt.Printf("%s Is Writable: %t Is Signer: %t\n", key.String(), ct.Transaction.IsWritable(key), ct.Transaction.IsSigner(key))
					}
					fmt.Printf("\n\n")
					continue
				}

				//decoded, err := decoder(ct.Transaction.AccountMetaList(), i.Data)
				//if err != nil {
				//	return fmt.Errorf("unable to decode instruction: %w", err)
				//}
				//
				//err = text.NewEncoder(os.Stdout).Encode(decoded, nil)
				//if err != nil {
				//	return fmt.Errorf("unable to text encoder instruction: %w", err)
				//}
			}
			text.EncoderColorCyan.Print("\n\nEnd of transaction\n\n")
		}

		return nil
	},
}

func init() {
	getCmd.AddCommand(getTransactionsCmd)
}
