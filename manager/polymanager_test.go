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
	var height uint32 = 6003
	ok := polyManager.handleDepositEvents(height)
	fmt.Println("---------------- handleDepositEvents result -----------------")
	fmt.Println(ok)
	if !ok {
		t.FailNow()
	}
}

func TestGetPolyHeightByTxHash(t *testing.T) {
	//commitProof - send transaction to poly chain: ( poly_txhash: 61341c16ec50ec4b2c364ee3dfc3ccdb5af540eba38c89160de75afd3322052d,
	//starcoin_txhash: 0x9cfb0f1a47fa8a3c7cf1a60b652434adc78adf147205fea34b3e74fa6fff9bff, height: 219972 )
	polyTxHash := "61341c16ec50ec4b2c364ee3dfc3ccdb5af540eba38c89160de75afd3322052d"
	polyManager := getTestPolyManager(t)
	h, err := polyManager.polySdk.GetBlockHeightByTxHash(polyTxHash)
	if err != nil {
		t.FailNow()
	}
	fmt.Println(h)

	e, err := polyManager.polySdk.GetSmartContractEvent(polyTxHash)
	fmt.Println(e)
	fmt.Println("------------- event.State -------------")
	fmt.Println(e.State)
	fmt.Println("------------- event.Notify -------------")
	fmt.Println(e.Notify)
	fmt.Println(err)
}

func TestSendPolyTxToStarcoin(t *testing.T) {
	polyManager := getTestPolyManager(t)
	//fmt.Println(polyManager)
	polyTxHash := "0a2a6502415f878d8866ae3b7d646327ce28fe3c592f7f08091c6ed6db4e55ac"
	polyTx, err := polyManager.db.GetPolyTx(polyTxHash)
	if err != nil {
		t.FailNow()
	}
	fmt.Println(polyTx)

	sender := polyManager.senders[0]
	stcTxInfo, err := sender.polyTxToStarcoinTxInfo(polyTx)
	if err != nil {
		t.FailNow()
	}
	fmt.Println(stcTxInfo)
	err = sender.sendTxToStarcoin(stcTxInfo)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
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
