package block

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/fr13n8/go-blockchain/pkg/services/transaction"
	"github.com/fr13n8/go-blockchain/utils"
	"log"
	"time"
)

type Block struct {
	Header
	Transactions []*transaction.Transaction
}

type Header struct {
	PreviousHash   [32]byte
	MerkleRootHash []byte
	Timestamp      int64
	Nonce          uint64
	Target         []byte
	Hash           [32]byte
}

func (h *Header) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PreviousHash   string `json:"previous_hash"`
		MerkleRootHash []byte `json:"merkle_root_hash"`
		Timestamp      int64  `json:"timestamp"`
		Nonce          uint64 `json:"nonce"`
		Target         string `json:"target"`
	}{
		PreviousHash:   fmt.Sprintf("%x", h.PreviousHash),
		MerkleRootHash: h.MerkleRootHash,
		Timestamp:      h.Timestamp,
		Nonce:          h.Nonce,
		Target:         fmt.Sprintf("%x", h.Target),
	})
}

func New(nonce uint64, previousHash [32]byte, transactions []*transaction.Transaction) *Block {
	return &Block{
		Header: Header{
			Nonce:          nonce,
			PreviousHash:   previousHash,
			Timestamp:      time.Now().UnixNano(),
			MerkleRootHash: merkleRootHash(transactions),
		},
		Transactions: transactions,
	}
}

func NewGenesisBlock(transactions []*transaction.Transaction) *Block {
	return New(0, [32]byte{}, transactions)
}

func (b *Block) Print() {
	fmt.Printf("Timestamp: %d\n", b.Header.Timestamp)
	fmt.Printf("Nonce: %d\n", b.Header.Nonce)
	fmt.Printf("PreviousHash: %x\n", b.Header.PreviousHash)
	for _, t := range b.Transactions {
		t.Print()
	}
}

func (b *Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Header       *Header                    `json:"header"`
		Transactions []*transaction.Transaction `json:"transactions"`
	}{
		Header:       &b.Header,
		Transactions: b.Transactions,
	})
}

func (b *Block) Hash() [32]byte {
	m, err := b.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}
	return sha256.Sum256(m)
}

func (b *Block) HexHash() string {
	return fmt.Sprintf("%x", b.Hash())
}

func (b *Block) HexHashPrevBlock() string {
	return fmt.Sprintf("%x", b.Header.PreviousHash)
}

func merkleRootHash(transactions []*transaction.Transaction) []byte {
	var txHashes [][]byte

	for _, tx := range transactions {
		tm, err := tx.MarshalJSON()
		if err != nil {
			log.Fatal(err)
		}
		txHashes = append(txHashes, tm)
	}
	tree := utils.NewMerkleTree(txHashes)
	if tree == nil {
		return []byte{}
	}
	return tree.RootNode.Data
}
