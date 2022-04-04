package arweave

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"time"
)

type Arweave struct {
	host     string
	port     uint64
	protocol string
	logger   *zap.Logger
	Wallet   *Wallet
	cli      *Client
}

func NewArweave(host string, opts ...Option) (*Arweave, error) {
	var err error
	a := &Arweave{
		host:     host,
		port:     443,
		protocol: "https",
		logger:   zap.NewNop(),
	}
	for _, opt := range opts {
		a, err = opt(a)
		if err != nil {
			return nil, fmt.Errorf("failed to applu options: %w", err)
		}
	}

	a.cli = NewClient(fmt.Sprintf("%s://%s:%d", a.protocol, a.host, a.port))
	return a, nil
}

type Option = func(a *Arweave) (*Arweave, error)

func WithLogger(zlog *zap.Logger) Option {
	return func(a *Arweave) (*Arweave, error) {
		a.logger = zlog
		return a, nil
	}
}

func WithWalletFilepath(walletFilepath string) Option {
	return func(a *Arweave) (*Arweave, error) {
		cnt, err := ioutil.ReadFile(walletFilepath)
		if err != nil {
			return nil, fmt.Errorf("unable to read Wallet file %q: %w", walletFilepath, err)
		}

		w, err := NewWalletFromFile(cnt)
		if err != nil {
			return nil, fmt.Errorf("unable to create wallet from file content: %w", err)
		}
		a.Wallet = w
		return a, nil
	}
}

func Insecure() Option {
	return func(a *Arweave) (*Arweave, error) {
		a.protocol = "http"
		a.port = 80
		return a, nil
	}
}

func (a *Arweave) UploadAndConfirm(ctx context.Context, data []byte) (*Transaction, error) {
	tx, err := a.Upload(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("unable to create transction: %w", err)
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		receipt, err := a.cli.GetTransaction(ctx, base64.StdEncoding.EncodeToString(tx.id))
		if receipt != nil {
			return receipt, nil
		}
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve transaction: %w", err)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}

}

func (a *Arweave) Upload(ctx context.Context, data []byte) (*Transaction, error) {
	if a.Wallet == nil {
		return nil, fmt.Errorf("unabel to upload content without a wallet with 'WithWalletFilepath' option when creating Arweave")
	}
	a.logger.Debug("uploading content to arweave",
		zap.String("wallet", a.Wallet.address),
		zap.Int("content_size", len(data)),
	)

	lastTx, err := a.cli.TxAnchor(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve last transaction: %w", err)
	}

	a.logger.Debug("found last transaction", zap.String("last_trx", lastTx))

	price, err := a.cli.GetPrice(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("unable to get reward: %w", err)
	}

	a.logger.Debug("retrieve price for content", zap.String("price", price))

	// Non encoded transaction fields
	tx := NewTransaction(
		lastTx,
		a.Wallet.pubKey.N,
		"0",
		"",
		data,
		price,
	)

	tx, err = tx.Sign(*a.Wallet)
	if err != nil {
		return nil, fmt.Errorf("unable to sign transaction: %w", err)
	}

	serialized, err := json.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}
	_, err = a.cli.Commit(ctx, serialized)
	if err != nil {
		return nil, fmt.Errorf("unable ot commit transaction: %w", err)
	}

	return tx, nil
}
