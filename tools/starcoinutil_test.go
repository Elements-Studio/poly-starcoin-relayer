package tools

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/novifinancial/serde-reflection/serde-generate/runtime/golang/serde"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestWaitTransactionConfirm(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://halley-seed.starcoin.org")
	b, err := WaitTransactionConfirm(stcclient, "0xb199fbbf9c7aeef9a0257f0d496ecd0f11ded014526965b4a294c66041272cae", time.Second*30)
	fmt.Println(b, err)
}

func TestGetTokenScalingFactor(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	tokenType := "0x416b32009fe49fcab1d5f2ba0153838f::XUSDT::XUSDT"
	sf, err := GetTokenScalingFactor(&stcclient, tokenType)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Printf("Token(%s) scaling factor: %d\n", tokenType, sf)
}

func TestGetStarcoinNodeHeight(t *testing.T) {
	restclient := NewRestClient()
	h, err := GetStarcoinNodeHeight("https://barnard-seed.starcoin.org", restclient)
	fmt.Println(h, err)
}

func TestGetTransactionProof(t *testing.T) {
	restclient := NewRestClient()
	var eventIndex int = 1
	p, err := GetTransactionProof("https://halley-seed.starcoin.org", restclient,
		"0x815764e45c2f300cfd90e9a693207c0d4f8f2ad1e3e9774f489534f1ab74e3d5", 69937, &eventIndex)
	if err != nil {
		t.FailNow()
	}
	fmt.Println("--------------- transaction proof -----------------")
	fmt.Println(p)
}

func TestGetTransactionInfoByHash(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://halley-seed.starcoin.org")
	txInfo, err := stcclient.GetTransactionInfoByHash(context.Background(), "0xb199fbbf9c7aeef9a0257f0d496ecd0f11ded014526965b4a294c66041272cae")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(txInfo)
	fmt.Println(strings.EqualFold("\"Executed\"", string(txInfo.Status)))
	fmt.Println(txInfo.Status)
	fmt.Println(isKnownStarcoinTxFailureStatus(txInfo.Status))
}

func TestIsAcceptToken(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	accountAddr := "0xdb9d6f70922c8deb4c9c6500633f425d"
	// accountAddr := "0xd117638e105403784bf6A92AA1276Ec1"
	tokenType := "0x00000000000000000000000000000001::STC::STC"
	// tokenType := "0x416b32009fe49fcab1d5f2ba0153838f::XETH::XETH"
	a, err := IsAcceptToken(&stcclient, accountAddr, tokenType)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Printf("Is account '%s' accept token '%s': %v\n", accountAddr, tokenType, a)
}

func TestAccountExistsAt(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	//accountAddr := "0xdb9d6f70922c8deb4c9c6500633f425d"
	accountAddr := "0x4afb6a3ED1e2ff212586fc6BcDb8DdAF"
	a, err := AccountExistsAt(&stcclient, accountAddr)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Printf("Account Exists At '%s': %v\n", accountAddr, a)
}

func TestUint128AndBigIntConvert(t *testing.T) {
	u1 := serde.Uint128{
		High: 0,
		Low:  2423531242,
	}
	b1 := Uint128ToBigInt(u1)
	if u1.Low != b1.Uint64() {
		t.FailNow()
	}
	fmt.Println(b1.String())
	u1_c := BigIntToUint128(b1)
	fmt.Println(u1_c.Low)
	if u1.High != u1_c.High || u1.Low != u1_c.Low {
		t.FailNow()
	}

	// /////////////////////
	u2 := serde.Uint128{
		High: 1124,
		Low:  2423531242,
	}
	b2 := Uint128ToBigInt(u2)
	fmt.Println(b2.String())
	b2_expected := new(big.Int).Add(new(big.Int).Mul(new(big.Int).Add(new(big.Int).SetUint64(math.MaxUint64), new(big.Int).SetUint64(1)), new(big.Int).SetUint64(u2.High)), new(big.Int).SetUint64(u2.Low))
	fmt.Println(b2_expected.String())
	if b2.String() != b2_expected.String() {
		t.FailNow()
	}
	// ////////////////////
	u2_c := BigIntToUint128(b2)
	if u2.High != u2_c.High || u2.Low != u2_c.Low {
		t.FailNow()
	}
}
