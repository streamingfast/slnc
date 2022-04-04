package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/streamingfast/slnc/arweave"
	"github.com/streamingfast/solana-go/programs/metaplex"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
)

var metaplexmetadataUploadCmd = &cobra.Command{
	Use:   "upload {arweave_wallet_path} {image_path} {json_path}",
	Short: "Uploads image and metadata json to arweave",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		arweaveWalletPath := args[0]
		imagePath := args[1]
		jsonPath := args[2]

		zlog.Info("uplaoding to arweave",
			zap.String("arweave_wallet_path", arweaveWalletPath),
			zap.String("image_path", imagePath),
			zap.String("json_path", jsonPath),
		)

		arweave, err := arweave.NewArweave("arweave.net", arweave.WithLogger(zlog), arweave.WithWalletFilepath(arweaveWalletPath))
		if err != nil {
			return fmt.Errorf("unable to setup arweave: %w", err)
		}

		fmt.Println("uploading image and metadata from:", arweave.Wallet.Address())

		imageCnt, err := ioutil.ReadFile(imagePath)
		if err != nil {
			return fmt.Errorf("failed to read image %q: %w", imagePath, err)
		}

		tx, err := arweave.UploadAndConfirm(ctx, imageCnt)
		if err != nil {
			return fmt.Errorf("failed to send data: %w", err)
		}
		imageUrl := fmt.Sprintf("https://arweave.net/%s", tx.ID())
		fmt.Println("Image uploaded ULR: ", imageUrl)

		f, err := os.Open(jsonPath)
		if err != nil {
			return fmt.Errorf("unable to open file %q: %w", err)
		}
		defer f.Close()

		data := &metaplex.DataV2{}
		if err := json.NewDecoder(f).Decode(data); err != nil {
			return fmt.Errorf("unable to decode metadate: %w", err)
		}

		data.URI = imageUrl

		return nil
	},
}

func init() {
	metaplexMetadataCmd.AddCommand(metaplexmetadataUploadCmd)
}
