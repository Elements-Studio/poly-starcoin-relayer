package tools

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

// encode big int to hex string
func EncodeBigInt(b *big.Int) string {
	if b.Uint64() == 0 {
		return "00"
	}
	return hex.EncodeToString(b.Bytes())
}

// encode bytes to hex with prefix
func EncodeToHex(b []byte) string {
	return "0x" + hex.EncodeToString(b)
}

func HexWithPrefixToBytes(str string) ([]byte, error) {
	if !strings.HasPrefix(str, "0x") {
		return nil, fmt.Errorf("it does not have 0x prefix")
	}
	return hex.DecodeString(str[2:])
}

func HexToBytes(str string) ([]byte, error) {
	if !strings.HasPrefix(str, "0x") {
		return hex.DecodeString(str[:])
	}
	return hex.DecodeString(str[2:])
}
