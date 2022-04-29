package cmd

import (
	"fmt"
	associatedtokenaccount "github.com/streamingfast/solana-go/programs/associated-token-account"
	"github.com/streamingfast/solana-go/rpc/confirm"
	"go.uber.org/zap"
	"strconv"

	"github.com/streamingfast/solana-go/rpc"

	"github.com/spf13/cobra"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/programs/token"
)

var tokenMintToCmd = &cobra.Command{
	Use:   "mint-to {mint} {recipient} {amount}",
	Short: "Mints",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		rpcCli := getClient(rpc.WithDebug())
		wsCli, err := getWsClient(ctx)
		if err != nil {
			return fmt.Errorf("unable to setup websocket client: %w", err)
		}
		vault := mustGetWallet()
		mintAddr, err := solana.PublicKeyFromBase58(args[0])
		if err != nil {
			return fmt.Errorf("decoding account key: %w", err)
		}

		recipientAddr, err := solana.PublicKeyFromBase58(args[1])
		if err != nil {
			return fmt.Errorf("decoding account key: %w", err)
		}

		amount, err := strconv.ParseUint(args[2], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse amount %q: %w", args[2], err)
		}

		mint, err := token.FetchMint(rpcCli, mintAddr)
		if err != nil {
			return fmt.Errorf("unable to get mint: %w", err)
		}

		var signer *solana.Account
		for _, privateKey := range vault.KeyBag {
			if privateKey.PublicKey() == mint.MintAuthority {
				signer = &solana.Account{PrivateKey: privateKey}
			}
		}

		if signer == nil {
			return fmt.Errorf("spl token account owner %q must be present in the vault to sign the send transaction", mint.MintAuthority.String())
		}

		recipientSPLTokenAccount := associatedtokenaccount.MustGetAssociatedTokenAddress(
			mintAddr,
			token.PROGRAM_ID,
			recipientAddr,
		)

		instructions := []solana.Instruction{}
		_, err = rpcCli.GetAccountInfo(recipientSPLTokenAccount)
		if err != nil && err != rpc.ErrNotFound {
			return fmt.Errorf("failed to look up recipient spl token account %q: %w", recipientSPLTokenAccount.String(), err)
		}
		if err == rpc.ErrNotFound {
			instructions = append(instructions, associatedtokenaccount.NewCreateInstruction(
				signer.PublicKey(),
				recipientSPLTokenAccount,
				recipientAddr,
				mintAddr,
				token.PROGRAM_ID,
			))
		}

		instructions = append(instructions, token.NewMintTo(
			amount,
			mintAddr,
			recipientSPLTokenAccount,
			signer.PublicKey(),
		))

		blockHashResult, err := rpcCli.GetLatestBlockhash(rpc.CommitmentFinalized)
		if err != nil {
			return fmt.Errorf("unable retrieve recent block hash: %w", err)
		}

		trx, err := solana.NewTransaction(instructions, blockHashResult.Value.Blockhash)
		if err != nil {
			return fmt.Errorf("unable to create transaction: %w", err)
		}

		zlog.Debug("issuing whitelist token",
			zap.String("mint_addr", mintAddr.String()),
			zap.String("recipient_addr", recipientAddr.String()),
			zap.String("spl_toke_account", recipientSPLTokenAccount.String()),
		)

		zlog.Debug("signing transactions")
		_, err = trx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
			// create account need to be signed by the private key of the new account
			// that is not in the vault and will be lost after the execution.
			if signer.PublicKey() == key {
				return &signer.PrivateKey
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("unable to sign transaction: %w", err)
		}

		fmt.Printf("Minting %s to %s\n", mintAddr.String(), recipientAddr.String())
		trxHash, err := confirm.SendAndConfirmTransaction(ctx, rpcCli, wsCli, trx)
		if err != nil {
			return fmt.Errorf("unable to send transaction: %w", err)
		}
		fmt.Printf("Mint To successful, with transaction hash: %s\n", trxHash)
		return nil
	},
}

func init() {
	tokenCmd.AddCommand(tokenMintToCmd)
}
