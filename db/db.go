package db

import (
	"encoding/hex"
	"hash"

	"golang.org/x/crypto/sha3"
)

const KEY_POLY_HEIGHT = "poly_height"

type DB interface {
	// Put Starcoin cross-chain Tx.(to poly) Check
	PutStarcoinTxCheck(txHash string, v []byte) error

	DeleteStarcoinTxCheck(txHash string) error

	// Put Starcoin cross-chain Tx.(to poly) Retry
	PutStarcoinTxRetry(k []byte) error

	DeleteStarcoinTxRetry(k []byte) error

	GetAllStarcoinTxCheck() (map[string][]byte, error)

	GetAllStarcoinTxRetry() ([][]byte, error)

	// Update poly height synced to Starcoin
	UpdatePolyHeight(h uint32) error

	GetPolyHeight() (uint32, error)

	GetPolyTx(txHash string) (*PolyTx, error)

	PutPolyTx(tx *PolyTx) (uint64, error)

	Close()
}

func Hash256Hex(v []byte) string {
	return hex.EncodeToString(Hash256(v))
}

func Hash256(v []byte) []byte {
	hasher := New256Hasher()
	hasher.Write(v)
	return hasher.Sum(nil)
}

func New256Hasher() hash.Hash {
	return sha3.New256() //sha256.New()
}
