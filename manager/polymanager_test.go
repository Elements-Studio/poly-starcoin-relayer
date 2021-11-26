package manager

import (
	"fmt"
	"testing"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/db"
	polysdk "github.com/polynetwork/poly-go-sdk"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestInitGenersis(t *testing.T) {
	// Poly devnet:
	// http://138.91.6.226:40336
	polyManager := getTestPolyManager(t)
	fmt.Println(polyManager)
	err := polyManager.InitGenesis(nil)
	// var height uint32 = 1319999
	// err := polyManager.InitGenesis(&height)
	fmt.Println(err)
}

func TestHandleDepositEvents(t *testing.T) {
	polyManager := getTestPolyManager(t)
	fmt.Println(polyManager)
	var height uint32 = 1088
	err := polyManager.handleDepositEvents(height)
	fmt.Println(err)
}

func TestGetPolyCurrentBlockHeight(t *testing.T) {
	polyManager := getTestPolyManager(t)
	fmt.Println(polyManager)
	h, err := polyManager.polySdk.GetCurrentBlockHeight()
	fmt.Println(h, err)
}

func TestGetPolyLastConfigBlockNumAtHeight(t *testing.T) {
	polyManager := getTestPolyManager(t)
	fmt.Println(polyManager)
	polyManager.getPolyLastConfigBlockNumAtHeight(1319999)
}

func getTestPolyManager(t *testing.T) *PolyManager {
	config := config.NewServiceConfig("../config-devnet.json")
	fmt.Println(config)
	polySdk := polysdk.NewPolySdk()
	setUpPoly(polySdk, config.PolyConfig.RestURL)
	db, err := db.NewMySqlDB(config.MySqlDSN)
	if err != nil {
		t.FailNow()
	}
	starcoinClient := stcclient.NewStarcoinClient(config.StarcoinConfig.RestURL)
	polyManager, err := NewPolyManager(config, 0, polySdk, &starcoinClient, db)
	if err != nil {
		t.FailNow()
	}
	return polyManager
}

func setUpPoly(poly *polysdk.PolySdk, RpcAddr string) error {
	poly.NewRpcClient().SetAddress(RpcAddr)
	hdr, err := poly.GetHeaderByHeight(0)
	if err != nil {
		return err
	}
	poly.SetChainId(hdr.ChainID)
	return nil
}