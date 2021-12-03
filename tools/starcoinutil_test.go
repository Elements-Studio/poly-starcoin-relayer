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
