package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"fmt"
	"math/big"
)

type Signature struct {
	R *big.Int
	S *big.Int
}

func (s *Signature) String() string {
	return fmt.Sprintf("%064x%064x", s.R, s.S)
}

func SignatureFromString(s string) *Signature {
	x, y := String2BigIntTuple(s)
	return &Signature{
		R: x,
		S: y,
	}
}

func String2BigIntTuple(s string) (*big.Int, *big.Int) {
	bx, _ := hex.DecodeString(s[:64])
	by, _ := hex.DecodeString(s[64:])

	x := new(big.Int).SetBytes(bx)
	y := new(big.Int).SetBytes(by)

	return x, y
}

func PublicKeyFromString(s string) *ecdsa.PublicKey {
	x, y := String2BigIntTuple(s)
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}
}

func PrivateKeyFromString(s string, publicKey *ecdsa.PublicKey) *ecdsa.PrivateKey {
	b, err := hex.DecodeString(s[:])
	if err != nil {
		panic(err)
	}
	x := new(big.Int).SetBytes(b)
	return &ecdsa.PrivateKey{
		PublicKey: *publicKey,
		D:         x,
	}
}
