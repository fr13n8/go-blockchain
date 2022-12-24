package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"github.com/fr13n8/go-blockchain/utils"
)

func NewTransaction(senderPrivateKey *ecdsa.PrivateKey, senderPublicKey *ecdsa.PublicKey, senderAddress string, recipientAddress string, amount float32) *Transaction {
	return &Transaction{
		senderPrivateKey: senderPrivateKey,
		senderPublicKey:  senderPublicKey,
		senderAddress:    senderAddress,
		recipientAddress: recipientAddress,
		amount:           amount,
	}
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		SenderAddress    string  `json:"sender_address"`
		RecipientAddress string  `json:"recipient_address"`
		Amount           float32 `json:"amount"`
	}{
		SenderAddress:    t.senderAddress,
		RecipientAddress: t.recipientAddress,
		Amount:           t.amount,
	})
}

func (t *Transaction) GenerateSignature() *utils.Signature {
	m, err := t.MarshalJSON()
	if err != nil {
		panic(err)
	}
	h := sha256.Sum256(m)
	r, s, err := ecdsa.Sign(rand.Reader, t.senderPrivateKey, h[:])
	if err != nil {
		panic(err)
	}
	return &utils.Signature{
		R: r,
		S: s,
	}
}
