package manager

import (
	"fmt"
	"testing"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/db"
	polysdk "github.com/polynetwork/poly-go-sdk"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestFindSyncedHeight(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	fmt.Println(starcoinManager)
	h := starcoinManager.findSyncedHeight()
	fmt.Println(h) // 66856
}

func TestCommitHeader(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	fmt.Println(starcoinManager)
	ok := starcoinManager.handleBlockHeader(66857)
	fmt.Println(ok)
	fmt.Println(starcoinManager.header4sync)
	r := starcoinManager.commitHeader()
	// 0 for ok
	fmt.Println(r)
	if r != 0 {
		t.FailNow()
	}
}

func TestFetchLockDepositEvents(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	fmt.Println(starcoinManager)
	ok := starcoinManager.fetchLockDepositEvents(777)
	fmt.Println(ok)

	rl, err := starcoinManager.db.GetAllStarcoinTxRetry()
	fmt.Println(len(rl))
	fmt.Println(rl)
	fmt.Println(err)
}

func TestMisc(t *testing.T) {
	var proof []byte = []byte("{}")[:]
	fmt.Print(proof)
}

func getTestStarcoinManager(t *testing.T) *StarcoinManager {
	config := config.NewServiceConfig("../config-devnet.json")
	fmt.Println(config)
	polySdk := polysdk.NewPolySdk()
	setUpPoly(polySdk, config.PolyConfig.RestURL)
	db, err := db.NewMySqlDB(config.MySqlDSN)
	if err != nil {
		fmt.Println("new DB error:" + err.Error())
		t.FailNow()
	}
	starcoinClient := stcclient.NewStarcoinClient(config.StarcoinConfig.RestURL)
	config.PolyConfig.WalletFile = "../../../polynetwork/poly/wallet.dat"
	starcoinManager, err := NewStarcoinManager(config, 0, 0, polySdk, &starcoinClient, db)
	if err != nil {
		fmt.Println("NewStarcoinManager() error:" + err.Error())
		t.FailNow()
	}
	// ---------------------------------------------------------------
	// starcoinManager := &StarcoinManager{
	// 	config:        config,
	// 	exitChan:      make(chan int),
	// 	currentHeight: 1,
	// 	forceHeight:   1,
	// 	restClient:    tools.NewRestClient(),
	// 	client:        &starcoinClient,
	// 	polySdk:       polySdk,
	// 	//polySigner:    signer,
	// 	header4sync: make([][]byte, 0),
	// 	//crosstx4sync:  make([]*CrossTransfer, 0),
	// 	db: db,
	// }
	// //ignore this error:init - the genesis block has not synced!
	// starcoinManager.init()
	// ---------------------------------------------------------------
	return starcoinManager
}
