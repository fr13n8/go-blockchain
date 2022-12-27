package block

import (
	"encoding/hex"
	"github.com/fr13n8/go-blockchain/utils"
	"math/big"
)

type Solver interface {
	Solve(*Block) bool
	Verify(Block) bool
}

const (
	MAX_NONCE         = ^uint64(0) // 2^32 - 1
	MINING_DIFFICULTY = "000000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"
)

type SHA256Solver struct {
	difficulty *big.Int
}

func NewSHA256Solver() Solver {
	targetBytes, _ := hex.DecodeString(MINING_DIFFICULTY)
	targetInt := new(big.Int).SetBytes(targetBytes)

	return &SHA256Solver{difficulty: targetInt}
}

func (s *SHA256Solver) Solve(b *Block) bool {
	for i := uint64(0); i <= MAX_NONCE; i++ {
		b.Header.Nonce = i
		hash := b.Hash()
		hashInt := utils.HashToBig(&hash)

		if hashInt.Cmp(s.difficulty) <= 0 {
			b.Header.Target = s.difficulty.Bytes()
			b.Header.Hash = hash
			return true
		}
	}

	return false
}

func (s *SHA256Solver) Verify(b Block) bool {
	hash := b.Hash()
	hashInt := utils.HashToBig(&hash)
	return hashInt.Cmp(s.difficulty) <= 0
}
