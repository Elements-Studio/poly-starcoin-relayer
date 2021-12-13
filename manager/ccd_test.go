package manager

import (
	"fmt"
	"testing"

	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestGetCurEpochStartHeight(t *testing.T) {
	client := stcclient.NewStarcoinClient("http://localhost:9850")
	ccd := NewCrossChainData(&client, "0x569AB535990a17Ac9Afd1bc57Faec683::CrossChainScript")
	h, err := ccd.getCurEpochStartHeight()
	fmt.Println(h)
	fmt.Println(err)
}

func TestGetCurEpochConPubKeyBytes(t *testing.T) {
	client := stcclient.NewStarcoinClient("http://localhost:9850")
	ccd := NewCrossChainData(&client, "0x569AB535990a17Ac9Afd1bc57Faec683::CrossChainScript")
	h, err := ccd.getCurEpochConPubKeyBytes()
	fmt.Println(h)
	fmt.Println(err)
}

func TestGetSparseMerkleRootHash(t *testing.T) {
	client := stcclient.NewStarcoinClient("https://halley-seed.starcoin.org")
	ccd := NewCrossChainData(&client, "0x57aa381a5d7c0141da3965393eed9958::CrossChainScript")
	h, err := ccd.getSparseMerkleRootHash()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(h)
}
