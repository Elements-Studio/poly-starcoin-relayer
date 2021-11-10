package tools

import (
	"fmt"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	"testing"
	"time"
)

func TestWaitTransactionConfirm(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	b, err := WaitTransactionConfirm(stcclient, "0x3f9b3d5c9a821327461a283e760afef055f0edd288d6826968bc4afd7620b0d5", time.Second*10)
	fmt.Println(b, err)
}