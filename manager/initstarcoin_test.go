package manager

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	stcpoly "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	"github.com/starcoinorg/starcoin-go/types"
)

// ///////////////////////// Test Init Starcoin Contracts START ///////////////////////////

func TestInitGenersis(t *testing.T) {
	// Poly devnet:
	// http://138.91.6.226:40336
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	//polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	//privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	polyManager := getMainNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	privateKeyConfig, _ := mainGenesisPrivateKeyConfig()
	fmt.Println(polyManager)
	err := polyManager.InitStarcoinGenesis(privateKeyConfig, nil)
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
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	chainType, _ := tools.ParseStructTypeTag("0x6c3bc3a6c651e88f5af8a570e661c6af::CrossChainGlobal::STARCOIN_CHAIN")
	chainId := uint64(318)
	txPayload := stcpoly.EncodeSetChainIdTxPayload(polyManager.config.StarcoinConfig.CCScriptModule, chainType, chainId)
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, privateKeyConfig, &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("Waiting Transaction Confirm, transaction hash: " + txHash)
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*120)
	if err != nil {
		fmt.Print(err)
		t.FailNow()
	}
	if !ok {
		fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
	} else {
		fmt.Println("WaitTransactionConfirm return OK.")
	}
}

func TestInitFeeEventStore(t *testing.T) {
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	txPayload := stcpoly.EncodeEmptyArgsTxPaylaod(polyManager.config.StarcoinConfig.CCScriptModule, "init_fee_event_store")
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, privateKeyConfig, &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("Waiting Transaction Confirm, transaction hash: " + txHash)
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*120)
	if err != nil {
		fmt.Print(err)
		t.FailNow()
	}
	if !ok {
		fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
	} else {
		fmt.Println("WaitTransactionConfirm return OK.")
	}
}

// Before cross from/to ethereum, bind the LockProxy hash first.
func TestBindEthereumProxyHash(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	starcoinClient := polyManager.starcoinClient
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	chainId := uint64(2) // 2 is ethereum ropsten chain id on poly TestNet
	//proxyHash, err := tools.HexToBytes("0xD8aE73e06552E270340b63A8bcAbf9277a1aac99") // LockProxy Contract Address
	proxyHash, err := tools.HexToBytes("0xfd40451429251a6dd535c4bb86a7d894409e900f") // LockProxy Contract Address
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	testBindProxyHash(starcoinClient, privateKeyConfig, polyManager.config, chainId, proxyHash, t)
}

// Test bind or update starcoin LockProxy hash(contract ID).
func TestBindStarcoinProxyHash(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	starcoinClient := polyManager.starcoinClient
	chainId := uint64(318) //318
	//proxyHash := []byte("0x6c3bc3a6c651e88f5af8a570e661c6af::CrossChainScript") // devnet
	proxyHash := []byte("0x416b32009fe49fcab1d5f2ba0153838f::CrossChainScript")
	testBindProxyHash(starcoinClient, privateKeyConfig, polyManager.config, chainId, proxyHash, t)
}

func TestPrintStarcoinProxyHashHex(t *testing.T) {
	// https://codebeautify.org/string-hex-converter
	proxyHash := []byte("0x416b32009fe49fcab1d5f2ba0153838f::CrossChainScript")
	fmt.Println(tools.EncodeToHex(proxyHash))
}

func testBindProxyHash(starcoinClient *stcclient.StarcoinClient, privateKeyConfig map[string]string, config *config.ServiceConfig, chainId uint64, proxyHash []byte, t *testing.T) {
	txPayload := stcpoly.EncodeBindProxyHashTxPayload(config.StarcoinConfig.CCScriptModule, chainId, proxyHash)
	txHash, err := submitStarcoinTransaction(starcoinClient, privateKeyConfig, &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("Waiting Transaction Confirm, transaction hash: " + txHash)
	ok, err := tools.WaitTransactionConfirm(*starcoinClient, txHash, time.Second*120)
	if err != nil {
		fmt.Print(err)
		t.FailNow()
	}
	if !ok {
		fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
	} else {
		fmt.Println("WaitTransactionConfirm return OK.")
	}
}

func TestBindXETHAssetHash(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	fromAssetHash := []byte("0x416b32009fe49fcab1d5f2ba0153838f::XETH::XETH")          // asset hash on Starcoin
	toChainId := uint64(2)                                                             // ethereum network
	toAssetHash, err := tools.HexToBytes("0x0000000000000000000000000000000000000000") // ETH Asset Hash on Ethereum Contract
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	testBindAssetHash(fromAssetHash, toChainId, toAssetHash, polyManager, privateKeyConfig, t)
}

func TestPrintXETHAssetHashHex(t *testing.T) {
	assetHash := []byte("0x416b32009fe49fcab1d5f2ba0153838f::XETH::XETH")
	fmt.Println(tools.EncodeToHex(assetHash))
}

func TestBindXUSDTAssetHash(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	fromAssetHash := []byte("0x416b32009fe49fcab1d5f2ba0153838f::XUSDT::XUSDT") // asset hash on Starcoin
	toChainId := uint64(2)                                                      // ethereum network
	//toAssetHash, err := tools.HexToBytes("0xad3f96ae966ad60347f31845b7e4b333104c52fb") // USDT Asset Hash on Ethereum Contract
	//toAssetHash, err := tools.HexToBytes("0x74E9a2447De2e31C3D8c1f6BAeFBD09ed1162891") // USDT Asset Hash on Ethereum Contract(Ropsten)
	toAssetHash, err := tools.HexToBytes("0xdAC17F958D2ee523a2206206994597C13D831ec7") // USDT Asset Hash on Ethereum Contract(Mainnet)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	testBindAssetHash(fromAssetHash, toChainId, toAssetHash, polyManager, privateKeyConfig, t)
}

func TestPrintXUSDTAssetHashHex(t *testing.T) {
	assetHash := []byte("0x416b32009fe49fcab1d5f2ba0153838f::XUSDT::XUSDT")
	fmt.Println(tools.EncodeToHex(assetHash))
}

func TestBindSTCAssetHash(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	fromAssetHash := []byte("0x00000000000000000000000000000001::STC::STC") // asset hash on Starcoin
	toChainId := uint64(318)                                                // a starcoin network
	toAssetHash := []byte("0x00000000000000000000000000000001::STC::STC")   //support cross-to-self transfer
	testBindAssetHash(fromAssetHash, toChainId, toAssetHash, polyManager, privateKeyConfig, t)
}

func TestPrintSTCAssetHashHex(t *testing.T) {
	assetHash := []byte("0x00000000000000000000000000000001::STC::STC")
	fmt.Println(tools.EncodeToHex(assetHash))
}

func TestBindEthereumSTCAssetHash(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManagerIgnoreError() // Poly TestNet / Starcoin Barnard
	//toAssetHash, _ := tools.HexToBytes("0x6527BC0C4724B51c955E7A4654E2c15464C1851a") // OLD eSTC ERC20 contract address on ethereum
	//toAssetHash, _ := tools.HexToBytes("0x43e35ba290afe67c295321eeb539ce7756753823") // STC ERC20 contract address on ethereum
	fmt.Println(polyManager)
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	fromAssetHash := []byte("0x00000000000000000000000000000001::STC::STC") // asset hash on Starcoin
	toChainId := uint64(2)                                                  // a ethereum network
	//toAssetHash, _ := tools.HexToBytes("0x2e269dcdebdc5f2068dfb23972ed81ad1b0f9585") // STC ERC20 contract address on ethereum ropsten
	toAssetHash, _ := tools.HexToBytes("0xec8614B0a68786Dc7b452e088a75Cba4F68755b8") // STC ERC20 contract address on ethereum

	testBindAssetHash(fromAssetHash, toChainId, toAssetHash, polyManager, privateKeyConfig, t)
}

// Test bind or update asset hash(asset ID).
func testBindAssetHash(fromAssetHash []byte, toChainId uint64, toAssetHash []byte, polyManager *PolyManager, privateKeyConfig map[string]string, t *testing.T) {
	txPayload := stcpoly.EncodeBindAssetHashTxPayload(polyManager.config.StarcoinConfig.CCScriptModule, fromAssetHash, toChainId, toAssetHash)
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, privateKeyConfig, &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("Waiting Transaction Confirm, transaction hash: " + txHash)
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*120)
	if err != nil {
		fmt.Print(err)
		t.FailNow()
	}
	if !ok {
		fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
	} else {
		fmt.Println("WaitTransactionConfirm return OK.")
	}
}

func TestXEthInit(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	module := "0x416b32009fe49fcab1d5f2ba0153838f::XETHScripts"
	txPayload := stcpoly.EncodeEmptyArgsTxPaylaod(module, "init")
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, privateKeyConfig, &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("Waiting Transaction Confirm, transaction hash: " + txHash)
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*120)
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

func TestXUsdtInit(t *testing.T) {
	//polyManager := getDevNetPolyManager(t) // Poly DevNet / Starcoin Halley
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	module := "0x416b32009fe49fcab1d5f2ba0153838f::XUSDTScripts"
	txPayload := stcpoly.EncodeEmptyArgsTxPaylaod(module, "init")
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, privateKeyConfig, &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("Waiting Transaction Confirm, transaction hash: " + txHash)
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*120)
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

func TestSetAdminAccount(t *testing.T) {
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	accountAddress, err := types.ToAccountAddress("0xb6D69DD935EDf7f2054acF12eb884df8")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	txPayload := stcpoly.EncodeAccountAddressTxPaylaod(polyManager.config.StarcoinConfig.CCScriptModule, "set_admin_account", *accountAddress)
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, privateKeyConfig, &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("Waiting Transaction Confirm, transaction hash: " + txHash)
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*120)
	if err != nil {
		fmt.Print(err)
		t.FailNow()
	}
	if !ok {
		fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
	} else {
		fmt.Println("WaitTransactionConfirm return OK.")
	}
}

func TestSetFeeCollectionAccount(t *testing.T) {
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	accountAddress, err := types.ToAccountAddress("0x7F7C0C04E447CaFfc7a526Ef1bF8D549")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	txPayload := stcpoly.EncodeAccountAddressTxPaylaod(polyManager.config.StarcoinConfig.CCScriptModule, "set_fee_collection_account", *accountAddress)
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, privateKeyConfig, &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("Waiting Transaction Confirm, transaction hash: " + txHash)
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*120)
	if err != nil {
		fmt.Print(err)
		t.FailNow()
	}
	if !ok {
		fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
	} else {
		fmt.Println("WaitTransactionConfirm return OK.")
	}
}

func TestSetFreeze(t *testing.T) {
	polyManager := getTestNetPolyManager(t) // Poly TestNet / Starcoin Barnard
	fmt.Println(polyManager)
	privateKeyConfig, _ := barnardGenesisPrivateKeyConfig()
	freeze := false
	txPayload := stcpoly.EncodeBoolTxPaylaod(polyManager.config.StarcoinConfig.CCScriptModule, "set_freeze", freeze)
	txHash, err := submitStarcoinTransaction(polyManager.starcoinClient, privateKeyConfig, &txPayload)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("Waiting Transaction Confirm, transaction hash: " + txHash)
	ok, err := tools.WaitTransactionConfirm(*polyManager.starcoinClient, txHash, time.Second*120)
	if err != nil {
		fmt.Print(err)
		t.FailNow()
	}
	if !ok {
		fmt.Printf("WaitTransactionConfirm return, isAllOK?: %v, or else got error?: %v\n", ok, err)
	} else {
		fmt.Println("WaitTransactionConfirm return OK.")
	}
}

// ///////////////////////// Test Init Starcoin Contracts END ///////////////////////////

func barnardGenesisPrivateKeyConfig() (map[string]string, error) {
	privateKeyConfig := make(map[string]string)
	account, privateKey, err := barnardGenesisAccountAddressAndPrivateKey()
	if err != nil {
		return nil, err
	}
	privateKeyConfig[account] = privateKey
	return privateKeyConfig, nil
}

func barnardGenesisAccountAddressAndPrivateKey() (string, string, error) {
	account := "0x416b32009fe49fcab1d5f2ba0153838f"
	if account == "" {
		return "", "", errors.New("Plz. provide account address.")
	}
	privateKey := os.Getenv("PRIVATE_KEY_416b320")
	if privateKey == "" {
		return "", "", errors.New("Plz. privide private key.")
	}
	return account, privateKey, nil
}

func mainGenesisPrivateKeyConfig() (map[string]string, error) {
	privateKeyConfig := make(map[string]string)
	account, privateKey, err := mainGenesisAccountAddressAndPrivateKey()
	if err != nil {
		return nil, err
	}
	privateKeyConfig[account] = privateKey
	return privateKeyConfig, nil
}

func mainGenesisAccountAddressAndPrivateKey() (string, string, error) {
	account := "0xe52552637c5897a2d499fbf08216f73e"
	if account == "" {
		return "", "", errors.New("Plz. provide account address.")
	}
	privateKey := os.Getenv("PRIVATE_KEY_e525526")
	if privateKey == "" {
		return "", "", errors.New("Plz. privide private key.")
	}
	return account, privateKey, nil
}
