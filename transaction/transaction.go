package transaction

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

type Transaction struct {
	SenderAddress    string
	RecipientAddress string
	Amount           float32
}

func NewTransaction(senderAddress string, recipientAddress string, value float32) *Transaction {
	return &Transaction{SenderAddress: senderAddress, RecipientAddress: recipientAddress, Amount: value}
}

func (t *Transaction) Print() {
	fmt.Printf("%s\n", strings.Repeat("-", 25))
	fmt.Printf("SenderAddress: %s\n", t.SenderAddress)
	fmt.Printf("RecipientAddress: %s\n", t.RecipientAddress)
	fmt.Printf("Amount: %.1f\n", t.Amount)
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		SenderAddress    string  `json:"sender_address"`
		RecipientAddress string  `json:"recipient_address"`
		Amount           float32 `json:"amount"`
	}{
		SenderAddress:    t.SenderAddress,
		RecipientAddress: t.RecipientAddress,
		Amount:           t.Amount,
	})
}

func (t *Transaction) Hash() ([32]byte, error) {
	m, err := t.MarshalJSON()
	if err != nil {
		return [32]byte{}, err
	}

	first := sha256.Sum256(m)
	second := sha256.Sum256(first[:])
	return second, nil
}

type Request struct {
	RecipientAddress string  `json:"recipient_address"`
	Amount           float32 `json:"amount"`
	SenderAddress    string  `json:"sender_address"`
	SenderPublicKey  string  `json:"sender_public_key"`
	Signature        string  `json:"signature"`
}

func (t *Request) Validate() bool {
	if t.RecipientAddress == "" || t.SenderAddress == "" || t.SenderPublicKey == "" || t.Signature == "" {
		return false
	}
	return true
}
