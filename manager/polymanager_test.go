package manager

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/db"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	"github.com/joho/godotenv"
	"github.com/novifinancial/serde-reflection/serde-generate/runtime/golang/serde"
	polysdk "github.com/polynetwork/poly-go-sdk"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestLockSTC(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	from_asset_hash := []byte("0x00000000000000000000000000000001::STC::STC") // STC
	var to_chain_id uint64 = 318                                              // 318
	to_address, _ := tools.HexToBytes("0x416b32009fe49fcab1d5f2ba0153838f")
	amount := serde.Uint128{
		High: 0,
		Low:  10000000,
	}
	testLockStarcoinAsset(from_asset_hash, to_chain_id, to_address, amount, polyManager, t)
}

func TestLockSTCWithSTCFee(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	from_asset_hash := []byte("0x00000000000000000000000000000001::STC::STC") // STC
	var to_chain_id uint64 = 318                                              // 318
	to_address, _ := tools.HexToBytes("0x4afb6a3ED1e2ff212586fc6BcDb8DdAF")   //("0x416b32009fe49fcab1d5f2ba0153838f")
	amount := serde.Uint128{
		High: 0,
		Low:  10000000,
	}
	fee := serde.Uint128{
		High: 0,
		Low:  5000000,
	}
	id := serde.Uint128{
		High: 0,
		Low:  1,
	}
	testLockStarcoinAssetWithStcFee(from_asset_hash, to_chain_id, to_address, amount, fee, id, polyManager, t)
}

func TestLockSTC_to_eSTC_WithSTCFee(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	from_asset_hash := []byte("0x00000000000000000000000000000001::STC::STC")       // STC
	var to_chain_id uint64 = 2                                                      // 318
	to_address, _ := tools.HexToBytes("0x71DFDD2BF49E8Af5226E0078efA31ecf258bC44E") // an ethereum address
	amount := serde.Uint128{
		High: 0,
		Low:  1000000000,
	}
	fee := serde.Uint128{
		High: 0,
		Low:  1000000000,
	}
	id := serde.Uint128{
		High: 0,
		Low:  1,
	}
	testLockStarcoinAssetWithStcFee(from_asset_hash, to_chain_id, to_address, amount, fee, id, polyManager, t)
}

func TestLockXETHWithSTCFee(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	fmt.Println("================== polyManager ================")
	fmt.Println(polyManager)
	var from_asset_hash []byte
	from_asset_hash = []byte("0x416b32009fe49fcab1d5f2ba0153838f::XETH::XETH") // XETH asset hash(asset ID.) on Starcoin
	var to_chain_id uint64 = 2                                                 // to an ethereum network
	var to_address []byte
	to_address, _ = tools.HexToBytes("0x208D1Ae5bb7FD323ce6386C443473eD660825D46") // to an ethereum address
	amount := serde.Uint128{
		High: 0,
		Low:  115555000000,
	}
	fee := serde.Uint128{
		High: 0,
		Low:  5000000,
	}
	id := serde.Uint128{
		High: 0,
		Low:  1,
	}
	testLockStarcoinAssetWithStcFee(from_asset_hash, to_chain_id, to_address, amount, fee, id, polyManager, t)
}

func TestLockXETH(t *testing.T) {
	// //////////////////////////////////////////////////
	// note: bind the LockProxy hash first.
	// note: bind Asset Hash first!
	// //////////////////////////////////////////////////

	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	from_asset_hash := []byte("0x416b32009fe49fcab1d5f2ba0153838f::XETH::XETH")     // XETH asset hash(asset ID.) on Starcoin
	var to_chain_id uint64 = 2                                                      // to an ethereum network
	to_address, _ := tools.HexToBytes("0x208D1Ae5bb7FD323ce6386C443473eD660825D46") // to an ethereum address
	amount := serde.Uint128{
		High: 0,
		Low:  555555555000000,
	}
	testLockStarcoinAsset(from_asset_hash, to_chain_id, to_address, amount, polyManager, t)
}

func testLockStarcoinAsset(from_asset_hash []byte, to_chain_id uint64, to_address []byte, amount serde.Uint128, polyManager *PolyManager, t *testing.T) {
	txHash, err := LockStarcoinAsset(polyManager.starcoinClient, polyManager.config.StarcoinConfig.PrivateKeys[0], polyManager.config.StarcoinConfig.CCScriptModule, from_asset_hash, to_chain_id, to_address, amount)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("LockStarcoinAsset return hash: " + txHash)
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*30)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	if !ok {
		fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
	} else {
		fmt.Println("WaitTransactionConfirm return OK.")
	}
}

func testLockStarcoinAssetWithStcFee(from_asset_hash []byte, to_chain_id uint64, to_address []byte, amount serde.Uint128, fee serde.Uint128, id serde.Uint128, polyManager *PolyManager, t *testing.T) {
	txHash, err := LockStarcoinAssetWithStcFee(polyManager.starcoinClient, polyManager.config.StarcoinConfig.PrivateKeys[0], polyManager.config.StarcoinConfig.CCScriptModule, from_asset_hash, to_chain_id, to_address, amount, fee, id)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("LockStarcoinAssetWithFee return hash: " + txHash)
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*30)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	if !ok {
		fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
	} else {
		fmt.Println("WaitTransactionConfirm return OK.")
	}
}

func TestHandleDepositEvents(t *testing.T) {
	//polyManager := getDevNetPolyManager(t)
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	var height uint32 = 25764514 //25298673 //24924223
	ok := polyManager.handleDepositEvents(height)
	fmt.Println("---------------- handleDepositEvents result -----------------")
	fmt.Println(ok)
	if !ok {
		t.FailNow()
	}
}

func TestGetPolyHeightByTxHash(t *testing.T) {
	polyManager := getDevNetPolyManager(t)
	//polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard

	polyTxHash := "61341c16ec50ec4b2c364ee3dfc3ccdb5af540eba38c89160de75afd3322052d"
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
	polyManager := getDevNetPolyManager(t)
	//polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard

	//fmt.Println(polyManager)
	polyTxHash := "0a2a6502415f878d8866ae3b7d646327ce28fe3c592f7f08091c6ed6db4e55ac"
	fromChainId := uint64(218)
	polyTx, err := polyManager.db.GetPolyTx(polyTxHash, fromChainId)
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
	polyManager := getDevNetPolyManager(t)
	//polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard

	fmt.Println(polyManager)
	h, err := polyManager.polySdk.GetCurrentBlockHeight()
	fmt.Println(h, err)
}

func TestGetPolyLastConfigBlockNumAtHeight(t *testing.T) {
	polyManager := getDevNetPolyManager(t)
	//polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard

	fmt.Println(polyManager)
	polyManager.getPolyLastConfigBlockNumAtHeight(1319999)
}

func TestCheckStarcoinStatusByProof(t *testing.T) {
	p, err := tools.HexToBytes("0xfd230120ab2c4dea41a96f2ac5becbc2ad8775db1416742bd597c99de6015f2b5e2f811b0200000000000000200000000000000000000000000000000000000000000000000000000000002da1200177d9fc54ec34995ac699485135a8ca3ba73c5e21341ecca08db78145b272ee14d8ae73e06552e270340b63a8bcabf9277a1aac993e0100000000000034307831383335316433313164333232303131343961346466326139666332646238613a3a43726f7373436861696e53637269707406756e6c6f636b5e2c307830303030303030303030303030303030303030303030303030303030303030313a3a5354433a3a5354431066a75557fc3f687eb849d9199498f4aa00ab904100000000000000000000000000000000000000000000000000000000")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	polyManager := getTestNetPolyManager(t)
	b, s, m := polyManager.checkStarcoinStatusByProof(p)
	fmt.Println(b, s, m)
}

func TestHandleNotSentGasSubsidy(t *testing.T) {
	polyManager := getTestNetPolyManager(t)
	gasSubsidy, err := polyManager.db.GetFirstNotSentGasSubsidy()
	if err != nil {
		fmt.Printf("failed to GetFirstNotSentGasSubsidy: %s", err.Error())
		t.FailNow()
	}
	if gasSubsidy == nil {
		return
	}
	err = polyManager.handleNotSentGasSubsidy(gasSubsidy)
	if err != nil {
		fmt.Printf("failed to handleNotSentGasSubsidy: %s", err.Error())
		t.FailNow()
	}
}

func TestHandleFailedPolyTx(t *testing.T) {
	polyManager := getTestNetPolyManager(t)
	polyTx, err := polyManager.db.GetFirstFailedPolyTx()
	if err != nil {
		fmt.Printf("failed to GetFirstFailedPolyTx: %s", err.Error())
		t.FailNow()
	}
	if polyTx == nil {
		fmt.Println("polyTx == nil")
		return
	}
	err = polyManager.handleFailedPolyTx(polyTx)
	if err != nil {
		fmt.Printf("failed to HandleFailedPolyTx: %s", err.Error())
		t.FailNow()
	}
}

func TestPrintConfig(t *testing.T) {
	config := config.NewServiceConfig("../config-testnet.json")
	//fmt.Println(string(config.TreasuriesConfig.Treasuries["Starcoin"].Tokens["USDT"].OpeningBalance))
	//fmt.Println(config)
	//fmt.Println(config.TreasuriesConfig.AlertDiscordWebhookUrl)
	j, _ := json.Marshal(config.TreasuriesConfig)
	fmt.Println(string(j))
}

func getDevNetPolyManager(t *testing.T) *PolyManager {
	config := config.NewServiceConfig("../config-devnet.json")
	p, err := getPolyManager(config, false)
	if err != nil {
		t.FailNow()
	}
	return p
}

func getTestNetPolyManager(t *testing.T) *PolyManager {
	config := config.NewServiceConfig("../config-testnet.json")
	p, err := getPolyManager(config, false)
	if err != nil {
		t.FailNow()
	}
	return p
}

func getTestNetPolyManagerIgnoreError() *PolyManager {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Load .env file failed...")
	}
	config := config.NewServiceConfig("../config-testnet.json")
	//fmt.Println(config.StarcoinConfig.PrivateKeys)
	p, err := getPolyManager(config, true)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("============= Ignored above errors ===============")
	}
	return p
}

func getMainNetPolyManager(t *testing.T) *PolyManager {
	config := config.NewServiceConfig("../config-mainnet.json")
	p, err := getPolyManager(config, false)
	if err != nil {
		t.FailNow()
	}
	return p
}

func getMainNetPolyManagerIgnoreError() *PolyManager {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Load .env file failed...")
	}
	config := config.NewServiceConfig("../config-mainnet.json")
	//fmt.Println(config.StarcoinConfig.PrivateKeys)
	p, err := getPolyManager(config, true)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("============= Ignored above errors ===============")
	}
	return p
}

func getPolyManager(config *config.ServiceConfig, ignoreErr bool) (*PolyManager, error) {
	fmt.Println(config)
	starcoinClient := stcclient.NewStarcoinClient(config.StarcoinConfig.RestURL)
	polySdk := polysdk.NewPolySdk()
	err := setUpPoly(polySdk, config.PolyConfig.RestURL)
	if err != nil && !ignoreErr {
		return nil, err //t.FailNow()
	}
	db, err := db.NewMySqlDB(config.MySqlDSN)
	if err != nil && !ignoreErr {
		return nil, err //t.FailNow()
	}
	polyManager, err := NewPolyManager(config, 0, polySdk, &starcoinClient, db)
	if err != nil && !ignoreErr {
		return nil, err //t.FailNow()
	}
	if ignoreErr && polyManager == nil {
		polyManager = new(PolyManager)
		polyManager.config = config
		polyManager.starcoinClient = &starcoinClient
		polyManager.polySdk = polySdk
	}
	return polyManager, nil
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

// func TestMisc3(t *testing.T) {
// 	var i interface{} = big.NewInt(11111)
// 	s, ok := i.(string)
// 	fmt.Println(s)
// 	fmt.Println(ok)
// }
