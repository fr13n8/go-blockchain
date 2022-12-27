package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/fr13n8/go-blockchain/utils"
	"log"
)

type Transaction struct {
	senderPrivateKey *ecdsa.PrivateKey
	senderPublicKey  *ecdsa.PublicKey
	senderAddress    string
	recipientAddress string
	amount           float32
	Id               [32]byte
}

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
		Id               string  `json:"id"`
		SenderAddress    string  `json:"sender_address"`
		RecipientAddress string  `json:"recipient_address"`
		Amount           float32 `json:"amount"`
	}{
		Id:               fmt.Sprintf("%x", t.Id),
		SenderAddress:    t.senderAddress,
		RecipientAddress: t.recipientAddress,
		Amount:           t.amount,
	})
}

func (t *Transaction) HexHash() string {
	return fmt.Sprintf("%x", t.Id)
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

func (t *Transaction) GenerateSignature() *utils.Signature {
	h, err := t.Hash()
	if err != nil {
		log.Fatal(err)
	}

	r, s, err := ecdsa.Sign(rand.Reader, t.senderPrivateKey, h[:])
	if err != nil {
		panic(err)
	}
	return &utils.Signature{
		R: r,
		S: s,
	}
}
