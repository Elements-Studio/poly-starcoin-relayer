package manager

import (
	"fmt"
	"testing"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/db"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	polysdk "github.com/polynetwork/poly-go-sdk"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestCommitHeader(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	fmt.Println(starcoinManager)
	ok := starcoinManager.handleBlockHeader(1)
	fmt.Println(ok)
	fmt.Println(starcoinManager.header4sync)
	//starcoinManager.commitHeader()
	//fmt.Println("todo...")
}

func TestFetchLockDepositEvents(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	fmt.Println(starcoinManager)
	ok := starcoinManager.fetchLockDepositEvents(777)
	fmt.Println(ok)
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
	// starcoinManager, err := NewStarcoinManager(config, 0, 0, polySdk, &starcoinClient, db)
	// if err != nil {
	// 	fmt.Println("NewStarcoinManager() error:" + err.Error())
	// 	t.FailNow()
	// }
	// ---------------------------------------------------------------
	starcoinManager := &StarcoinManager{
		config:        config,
		exitChan:      make(chan int),
		currentHeight: 1,
		forceHeight:   1,
		restClient:    tools.NewRestClient(),
		client:        &starcoinClient,
		polySdk:       polySdk,
		//polySigner:    signer,
		header4sync: make([][]byte, 0),
		//crosstx4sync:  make([]*CrossTransfer, 0),
		db: db,
	}
	//ignore this error:init - the genesis block has not synced!
	starcoinManager.init()
	// ---------------------------------------------------------------
	return starcoinManager
}
