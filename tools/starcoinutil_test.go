package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestWaitTransactionConfirm(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://halley-seed.starcoin.org")
	b, err := WaitTransactionConfirm(stcclient, "0xb199fbbf9c7aeef9a0257f0d496ecd0f11ded014526965b4a294c66041272cae", time.Second*30)
	fmt.Println(b, err)
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
	fmt.Println(strings.EqualFold("\"Executed\"", string(txInfo.Status)))
	fmt.Println(txInfo.Status)
	fmt.Println(isKnownStarcoinTxFailureStatus(txInfo.Status))
}
