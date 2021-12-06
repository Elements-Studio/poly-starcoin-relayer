package db

import (
	"crypto/sha256"
	"encoding/hex"
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

func Sha256HashHex(v []byte) string {
	hasher := sha256.New()
	hasher.Write(v)
	h := hasher.Sum(nil)
	return hex.EncodeToString(h)
}
