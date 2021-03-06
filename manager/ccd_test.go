package manager

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestGetCurEpochStartHeight(t *testing.T) {
	ccd, err := newDevNetCCD(t)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	h, err := ccd.getCurEpochStartHeight()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(h)
}

func TestGetCurEpochConPubKeyBytes(t *testing.T) {
	ccd, err := newDevNetCCD(t)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	h, err := ccd.getCurEpochConPubKeyBytes()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(h)
}

func TestGetOnChainTxSparseMerkleTreeRootHash(t *testing.T) {
	ccd, err := newDevNetCCD(t)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	h, err := ccd.getOnChainTxSparseMerkleTreeRootHash()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(hex.EncodeToString(h)) //6c6e4784a4692516afaf129656f58dd40770f93aba116647afdd80ddc69b206f
	fmt.Println(h)
}

func newDevNetCCD(t *testing.T) (*CrossChainData, error) {
	servConfig := config.NewServiceConfig("../config-devnet.json")
	if servConfig == nil {
		t.FailNow()
	}
	//client := stcclient.NewStarcoinClient("https://halley-seed.starcoin.org")
	client := stcclient.NewStarcoinClient(servConfig.StarcoinConfig.RestURL)
	return NewCrossChainData(&client, servConfig.StarcoinConfig.CCDModule)
}
