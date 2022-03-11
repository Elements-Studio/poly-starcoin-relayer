package manager

import (
	"fmt"
	"testing"
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/db"
	stcpoly "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	"github.com/joho/godotenv"
	"github.com/novifinancial/serde-reflection/serde-generate/runtime/golang/serde"
	polysdk "github.com/polynetwork/poly-go-sdk"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

// ///////////////////////// Test Init Starcoin Contracts START ///////////////////////////

func TestInitGenersis(t *testing.T) {
	// Poly devnet:
	// http://138.91.6.226:40336
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	err := polyManager.InitStarcoinGenesis(nil)
	// var height uint32 = 1319999
	// err := polyManager.InitGenesis(&height)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("Init poly genesis ok.")
}

// Test set or update ChainID on poly network.
func TestSetChainId(t *testing.T) {
	polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	//polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	chainType, _ := tools.ParseStructTypeTag("0x6c3bc3a6c651e88f5af8a570e661c6af::CrossChainGlobal::STARCOIN_CHAIN")
	chainId := uint64(318)
	txPayload := stcpoly.EncodeSetChainIdTxPayload(polyManager.config.StarcoinConfig.CCScriptModule, chainType, chainId)
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, polyManager.config.StarcoinConfig.PrivateKeys[0], &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*30)
	if err != nil {
		fmt.Print(err)
		t.FailNow()
	}
	fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
}

func TestInitFeeEventStore(t *testing.T) {
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	txPayload := stcpoly.EncodeEmptyArgsTxPaylaod(polyManager.config.StarcoinConfig.CCScriptModule, "init_fee_event_store")
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, polyManager.config.StarcoinConfig.PrivateKeys[0], &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*30)
	if err != nil {
		fmt.Print(err)
		t.FailNow()
	}
	fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
}

// Before cross from/to ethereum, bind the LockProxy hash first.
func TestBindEthereumProxyHash(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	starcoinClient := polyManager.starcoinClient
	chainId := uint64(2)                                                             // 2 is ethereum ropsten chain id on poly TestNet
	proxyHash, err := tools.HexToBytes("0xD8aE73e06552E270340b63A8bcAbf9277a1aac99") // LockProxy Contract Address
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	testBindProxyHash(starcoinClient, polyManager.config, chainId, proxyHash, t)
}

// Test bind or update starcoin LockProxy hash(contract ID).
func TestBindStarcoinProxyHash(t *testing.T) {
	polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	//polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	starcoinClient := polyManager.starcoinClient
	chainId := uint64(318) //318
	proxyHash := []byte("0x6c3bc3a6c651e88f5af8a570e661c6af::CrossChainScript")
	testBindProxyHash(starcoinClient, polyManager.config, chainId, proxyHash, t)
}

func testBindProxyHash(starcoinClient *stcclient.StarcoinClient, config *config.ServiceConfig, chainId uint64, proxyHash []byte, t *testing.T) {
	txPayload := stcpoly.EncodeBindProxyHashTxPayload(config.StarcoinConfig.CCScriptModule, chainId, proxyHash)
	txHash, err := submitStarcoinTransaction(starcoinClient, config.StarcoinConfig.PrivateKeys[0], &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	ok, err := tools.WaitTransactionConfirm(*starcoinClient, txHash, time.Second*30)
	if err != nil {
		fmt.Print(err)
		t.FailNow()
	}
	fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
}

func TestBindXETHAssetHash(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	fromAssetHash := []byte("0x18351d311d32201149a4df2a9fc2db8a::XETH::XETH")          // asset hash on Starcoin
	toChainId := uint64(2)                                                             // ethereum network
	toAssetHash, err := tools.HexToBytes("0x0000000000000000000000000000000000000000") // ETH Asset Hash on Ethereum Contract
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	testBindAssetHash(fromAssetHash, toChainId, toAssetHash, polyManager, t)
}

func TestBindSTCAssetHash(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	fromAssetHash := []byte("0x00000000000000000000000000000001::STC::STC") // asset hash on Starcoin
	toChainId := uint64(318)                                                // a starcoin network
	toAssetHash := []byte("0x00000000000000000000000000000001::STC::STC")   //support cross-to-self transfer
	testBindAssetHash(fromAssetHash, toChainId, toAssetHash, polyManager, t)
}

func TestBind_eSTC_AssetHash(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	fromAssetHash := []byte("0x00000000000000000000000000000001::STC::STC")          // asset hash on Starcoin
	toChainId := uint64(2)                                                           // a ethereum network
	toAssetHash, _ := tools.HexToBytes("0x6527BC0C4724B51c955E7A4654E2c15464C1851a") // ERC20 contract address on ethereum
	testBindAssetHash(fromAssetHash, toChainId, toAssetHash, polyManager, t)
}

// Test bind or update asset hash(asset ID).
func testBindAssetHash(fromAssetHash []byte, toChainId uint64, toAssetHash []byte, polyManager *PolyManager, t *testing.T) {
	txPayload := stcpoly.EncodeBindAssetHashTxPayload(polyManager.config.StarcoinConfig.CCScriptModule, fromAssetHash, toChainId, toAssetHash)
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, polyManager.config.StarcoinConfig.PrivateKeys[0], &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*30)
	if err != nil {
		fmt.Print(err)
		t.FailNow()
	}
	fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
}

func TestXEthInit(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	module := "0x18351d311d32201149a4df2a9fc2db8a::XETHScripts"
	txPayload := stcpoly.EncodeEmptyArgsTxPaylaod(module, "init")
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, polyManager.config.StarcoinConfig.PrivateKeys[0], &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(txHash)
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*60)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
}

func TestXUsdtInit(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	module := "0x18351d311d32201149a4df2a9fc2db8a::XUSDTScripts"
	txPayload := stcpoly.EncodeEmptyArgsTxPaylaod(module, "init")
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, polyManager.config.StarcoinConfig.PrivateKeys[0], &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(txHash)
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*60)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
}

// ///////////////////////// Test Init Starcoin Contracts END ///////////////////////////

func TestLockSTC(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	from_asset_hash := []byte("0x00000000000000000000000000000001::STC::STC") // STC
	var to_chain_id uint64 = 318                                              // 318
	to_address, _ := tools.HexToBytes("0x18351d311d32201149a4df2a9fc2db8a")
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
	to_address, _ := tools.HexToBytes("0x18351d311d32201149a4df2a9fc2db8a")
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
		Low:  110000000000,
	}
	fee := serde.Uint128{
		High: 0,
		Low:  2000000000,
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
	from_asset_hash = []byte("0x18351d311d32201149a4df2a9fc2db8a::XETH::XETH") // XETH asset hash(asset ID.) on Starcoin
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
	from_asset_hash := []byte("0x18351d311d32201149a4df2a9fc2db8a::XETH::XETH")     // XETH asset hash(asset ID.) on Starcoin
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
	fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
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
	fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
}

func TestHandleDepositEvents(t *testing.T) {
	//polyManager := getDevNetPolyManager(t)
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	var height uint32 = 24924223
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
