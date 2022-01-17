package db

import (
	"encoding/hex"
	"hash"
	"time"

	"github.com/starcoinorg/starcoin-go/client"
	"golang.org/x/crypto/sha3"
)

const KEY_POLY_HEIGHT = "poly_height"

type DB interface {
	// Put Starcoin cross-chain Tx.(to poly) Check
	PutStarcoinTxCheck(txHash string, v []byte, e client.Event) error

	DeleteStarcoinTxCheck(txHash string) error

	// Put Starcoin cross-chain Tx.(to poly) Retry
	PutStarcoinTxRetry(k []byte, event client.Event) error

	DeleteStarcoinTxRetry(k []byte) error

	GetAllStarcoinTxCheck() (map[string]BytesAndEvent, error)

	GetAllStarcoinTxRetry() ([][]byte, []client.Event, error)

	// Update poly height synced to Starcoin
	UpdatePolyHeight(h uint32) error

	GetPolyHeight() (uint32, error)

	GetPolyTxRetry(txHash string, fromChainID uint64) (*PolyTxRetry, error)

	PutPolyTxRetry(tx *PolyTxRetry) error

	GetPolyTx(txHash string, fromChainID uint64) (*PolyTx, error)

	PutPolyTx(tx *PolyTx) (uint64, error)

	SetPolyTxStatus(txHash string, fromChainID uint64, status string) error

	SetPolyTxStatusProcessing(txHash string, fromChainID uint64, starcoinTxHash string) error

	SetPolyTxStatusProcessed(txHash string, fromChainID uint64, starcoinTxHash string) error

	GetFirstFailedPolyTx() (*PolyTx, error)

	Close()
}

type BytesAndEvent struct {
	Bytes []byte
	Event client.Event
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

func currentTimeMillis() int64 {
	return time.Now().UnixNano() / 1000000
}
