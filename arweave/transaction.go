package arweave

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
)

// NewTransaction creates a brand new transaction struct
func NewTransaction(lastTx string, owner *big.Int, quantity string, target string, data []byte, reward string) *Transaction {
	return &Transaction{
		lastTx:   lastTx,
		owner:    owner,
		quantity: quantity,
		target:   target,
		data:     data,
		reward:   reward,
		tags:     make([]Tag, 0),
	}
}

func (t *Transaction) ID() string {
	return base64.RawURLEncoding.EncodeToString(t.id)
}

func (t *Transaction) Data() string {
	return base64.RawURLEncoding.EncodeToString(t.data)
}

func (t *Transaction) Sign(w Wallet) (*Transaction, error) {
	// format the message
	payload, err := t.FormatMsgBytes()
	if err != nil {
		return nil, err
	}
	msg := sha256.Sum256(payload)

	sig, err := w.Sign(msg[:])
	if err != nil {
		return nil, err
	}

	err = w.Verify(msg[:], sig)
	if err != nil {
		return nil, err
	}

	id := sha256.Sum256((sig))

	idB := make([]byte, len(id))
	copy(idB, id[:])
	t.id = idB

	// we copy t into tx
	tx := Transaction(*t)
	// add the signature and ID to our new signature struct
	tx.signature = sig

	return &tx, nil
}

// MarshalJSON marshals as JSON
func (t *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.format())
}

// UnmarshalJSON unmarshals as JSON
func (t *Transaction) UnmarshalJSON(input []byte) error {
	txn := transactionJSON{}
	err := json.Unmarshal(input, &txn)
	if err != nil {
		return err
	}
	id, err := base64.RawURLEncoding.DecodeString(txn.ID)
	if err != nil {
		return err
	}
	t.id = id
	t.lastTx = txn.LastTx

	// gives me byte representation of the big num
	owner, err := base64.RawURLEncoding.DecodeString(txn.Owner)
	if err != nil {
		return err
	}
	n := new(big.Int)
	t.owner = n.SetBytes(owner)

	t.tags = txn.Tags
	t.target = txn.Target
	t.quantity = txn.Quantity

	data, err := base64.RawURLEncoding.DecodeString(txn.Data)
	if err != nil {
		return err
	}
	t.data = data
	t.reward = txn.Reward

	sig, err := base64.RawURLEncoding.DecodeString(txn.Signature)
	if err != nil {
		return err
	}
	t.signature = sig

	return nil
}

func (t *Transaction) Tags() ([]Tag, error) {
	tags := []Tag{}
	for _, tag := range t.tags {
		// access name
		tagName, err := base64.RawURLEncoding.DecodeString(tag.Name)
		if err != nil {
			return nil, err
		}
		tagValue, err := base64.RawURLEncoding.DecodeString(tag.Value)
		if err != nil {
			return nil, err
		}
		tags = append(tags, Tag{Name: string(tagName), Value: string(tagValue)})
	}
	return tags, nil
}

// FormatMsgBytes formats the message that needs to be signed. All fields
// need to be an array of bytes originating from the necessary data (not base64url encoded).
// The signing message is the SHA256 of the concatenation of the byte arrays
// of the owner public Wallet, target address, data, quantity, reward and last transaction
func (t *Transaction) FormatMsgBytes() ([]byte, error) {
	var msg []byte
	lastTx, err := base64.RawURLEncoding.DecodeString(t.lastTx)
	if err != nil {
		return nil, err
	}
	target, err := base64.RawURLEncoding.DecodeString(t.target)
	if err != nil {
		return nil, err
	}

	tags, err := t.encodeTagData()
	if err != nil {
		return nil, err
	}

	msg = append(msg, t.owner.Bytes()...)
	msg = append(msg, target...)
	msg = append(msg, t.data...)
	msg = append(msg, t.quantity...)
	msg = append(msg, t.reward...)
	msg = append(msg, lastTx...)
	msg = append(msg, tags...)

	return msg, nil
}

// We need to encode the tag data properly for the signature. This means having the unencoded
// value of the Name field concatenated with the unencoded value of the Value field
func (t *Transaction) encodeTagData() (string, error) {
	tagString := ""
	unencodedTags, err := t.Tags()
	if err != nil {
		return "", err
	}
	for _, tag := range unencodedTags {
		tagString += tag.Name + tag.Value
	}

	return tagString, nil
}

// Format formats the transactions to a JSONTransaction that can be sent out to an arweave node
func (t *Transaction) format() *transactionJSON {
	return &transactionJSON{
		ID:        base64.RawURLEncoding.EncodeToString(t.id),
		LastTx:    t.lastTx,
		Owner:     base64.RawURLEncoding.EncodeToString(t.owner.Bytes()),
		Tags:      t.tags,
		Target:    t.target,
		Quantity:  t.quantity,
		Data:      base64.RawURLEncoding.EncodeToString(t.data),
		Reward:    t.reward,
		Signature: base64.RawURLEncoding.EncodeToString(t.signature),
	}
}
