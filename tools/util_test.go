package tools

import (
	"fmt"
	"math/big"
	"testing"
)

func TestEncodeBigInt(t *testing.T) {
	hex := "0x02000000000000000000000000000000"
	bytes, _ := HexWithPrefixToBytes(hex)
	fmt.Println(bytes)
	bigInt := big.NewInt(0)
	bigInt.SetBytes(bytes)
	fmt.Println(bigInt)
	fmt.Println(bigInt.Bytes())
	str := EncodeBigInt(bigInt)
	fmt.Println(bigInt.Uint64())
	fmt.Println(str)
}
