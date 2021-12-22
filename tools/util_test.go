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

// func TestMisc(t *testing.T) {
// 	one := make([]byte, 2)
// 	two := make([]byte, 2)
// 	one[0] = 0x00
// 	one[1] = 0x01
// 	two[0] = 0x02
// 	two[1] = 0x03
// 	r := append(one, two...)
// 	fmt.Println(r)
// 	ss := make([][]byte, 0, 2)
// 	ss = append(ss, one, two)
// 	fmt.Println(ss)
// 	for _, s := range ss {
// 		r = append(r, s...)
// 	}
// 	fmt.Println(r)
// }
