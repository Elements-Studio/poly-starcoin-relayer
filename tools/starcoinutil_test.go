package tools

import (
	"fmt"
	"testing"
	"time"

	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestWaitTransactionConfirm(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://halley-seed.starcoin.org")
	b, err := WaitTransactionConfirm(stcclient, "0x077fd430df5165a10caf5ebd175c58b14e05786d79d50d339e70a892c8aa68d0", time.Second*30)
	fmt.Println(b, err)
}

func TestGetStarcoinNodeHeight(t *testing.T) {
	restclient := NewRestClient()
	h, err := GetStarcoinNodeHeight("https://barnard-seed.starcoin.org", restclient)
	fmt.Println(h, err)
}

func TestGetTransactionProof(t *testing.T) {
	restclient := NewRestClient()
	var eventIndex uint64 = 1
	p, err := GetTransactionProof("https://halley-seed.starcoin.org", restclient,
		"0x815764e45c2f300cfd90e9a693207c0d4f8f2ad1e3e9774f489534f1ab74e3d5", 69937, &eventIndex)
	if err != nil {
		t.FailNow()
	}
	fmt.Println("--------------- transaction proof -----------------")
	fmt.Println(p)
}
