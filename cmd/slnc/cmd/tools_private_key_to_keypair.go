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
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/streamingfast/solana-go"
)

var toKeypairPrivateKeyToolsCmd = &cobra.Command{
	Use:   "to-keypair",
	Short: "Converts base58 private key to a byte array",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		privateKey := args[0]
		keypairPath := args[1]
		fmt.Println("Converting: ", privateKey)
		pkey, err := solana.PrivateKeyFromBase58(privateKey)
		if err != nil {
			return fmt.Errorf("unable to deo decode private key")
		}

		values := []string{}
		for _, b := range pkey {
			values = append(values, fmt.Sprintf("%d", b))
		}

		cnt := fmt.Sprintf("[%s]", strings.Join(values, ","))
		fmt.Println("Writing keypair: ", keypairPath)
		if err := ioutil.WriteFile(keypairPath, []byte(cnt), os.ModePerm); err != nil {
			return fmt.Errorf("unable to write file")
		}
		return nil
	},
}

func init() {
	privateKeytoolsCmd.AddCommand(toKeypairPrivateKeyToolsCmd)
}
