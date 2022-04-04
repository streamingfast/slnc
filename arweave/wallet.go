package arweave

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mendsley/gojwk"
)

type Wallet struct {
	key       *gojwk.Key
	pubKey    *rsa.PublicKey
	address   string
	publicKey string
}

var opts = &rsa.PSSOptions{
	SaltLength: rsa.PSSSaltLengthAuto,
	Hash:       crypto.SHA256,
}

func NewWalletFromFile(cnt []byte) (*Wallet, error) {
	key := &gojwk.Key{}
	if err := json.Unmarshal(cnt, &key); err != nil {
		return nil, fmt.Errorf("unable to unmarshall key: %w", err)
	}

	publicKey, err := key.DecodePublicKey()
	if err != nil {
		return nil, fmt.Errorf("unable to decode public key: %w", err)
	}
	pubKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("could not typecast key to %T", rsa.PublicKey{})
	}

	// Take the "n", in bytes and hash it using SHA256

	h := sha256.New()
	h.Write(pubKey.N.Bytes())

	// Finally base64url encode it to have the resulting address
	return &Wallet{
		address:   base64.RawURLEncoding.EncodeToString(h.Sum(nil)),
		publicKey: base64.RawURLEncoding.EncodeToString(pubKey.N.Bytes()),
		pubKey:    pubKey,
		key:       key,
	}, nil
}
func (w *Wallet) Address() string {
	return w.address
}

func (w *Wallet) Verify(msg []byte, sig []byte) error {
	pub, err := w.key.DecodePublicKey()
	if err != nil {
		return err
	}
	pubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("could not typecast key to %T", rsa.PublicKey{})
	}

	err = rsa.VerifyPSS(pubKey, crypto.SHA256, msg, sig, opts)
	if err != nil {
		return err
	}
	return nil
}

// Sign signs a message using the RSA-PSS scheme with an MGF SHA256 masking function
func (w *Wallet) Sign(msg []byte) ([]byte, error) {
	priv, err := w.key.DecodePrivateKey()
	if err != nil {
		return nil, err
	}
	rng := rand.Reader
	privRsa, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("could not typecast key to %T", rsa.PrivateKey{})
	}

	sig, err := rsa.SignPSS(rng, privRsa, crypto.SHA256, msg, opts)
	if err != nil {
		return nil, err
	}
	return sig, nil
}
