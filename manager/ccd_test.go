package manager

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestGetCurEpochStartHeight(t *testing.T) {
	ccd := newCCD(t)
	h, err := ccd.getCurEpochStartHeight()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(h)
}

func TestGetCurEpochConPubKeyBytes(t *testing.T) {
	ccd := newCCD(t)
	h, err := ccd.getCurEpochConPubKeyBytes()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(h)
}

func TestGetNonMembershipSparseMerkleRootHashOnChain(t *testing.T) {
	ccd := newCCD(t)
	h, err := ccd.getSparseMerkleRootHash()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(hex.EncodeToString(h))
}

func newCCD(t *testing.T) *CrossChainData {
	servConfig := config.NewServiceConfig("../config-devnet.json")
	if servConfig == nil {
		t.FailNow()
	}
	client := stcclient.NewStarcoinClient("https://halley-seed.starcoin.org")
	return NewCrossChainData(&client, servConfig.StarcoinConfig.CCDModule)
}
