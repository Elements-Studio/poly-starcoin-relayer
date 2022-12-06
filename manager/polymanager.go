package manager

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/db"
	"github.com/elements-studio/poly-starcoin-relayer/log"
	"github.com/elements-studio/poly-starcoin-relayer/poly/msg"
	stcpoly "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	"github.com/novifinancial/serde-reflection/serde-generate/runtime/golang/serde"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology-crypto/signature"

	"github.com/polynetwork/bridge-common/base"
	"github.com/polynetwork/bridge-common/chains/bridge"
	"github.com/polynetwork/bridge-common/util"
	polysdk "github.com/polynetwork/poly-go-sdk"
	pcommon "github.com/polynetwork/poly-go-sdk/common"
	"github.com/polynetwork/poly/common"
	vconfig "github.com/polynetwork/poly/consensus/vbft/config"
	polytypes "github.com/polynetwork/poly/core/types"
	common2 "github.com/polynetwork/poly/native/service/cross_chain_manager/common"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	"github.com/starcoinorg/starcoin-go/types"
	diemtypes "github.com/starcoinorg/starcoin-go/types"
)

const (
	ChanLen                                = 64
	WAIT_STARCOIN_TRANSACTION_CONFIRM_TIME = time.Second * 120
	MAX_TIMEDOUT_TO_FAILED_WAITING_SECONDS = 120
)

var (
	GAS_SUBSIDY_ONLY_AFTER int64 = time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC).UnixNano() / 1000000
)

type PolyManager struct {
	config         *config.ServiceConfig
	polySdk        *polysdk.PolySdk
	currentHeight  uint32
	exitChan       chan int
	starcoinClient *stcclient.StarcoinClient
	senders        []*StarcoinSender
	db             db.DB
	bridgeSdk      *bridge.SDK
}

func NewPolyManager(servCfg *config.ServiceConfig, startblockHeight uint32, polySdk *polysdk.PolySdk, stcclient *stcclient.StarcoinClient, db db.DB) (*PolyManager, error) {
	//contractabi, err := abi.JSON(strings.NewReader(eccm_abi.EthCrossChainManagerABI))
	// if err != nil {
	// 	return nil, err
	// }
	// chainId, err := ethereumsdk.ChainID(context.Background())
	// if err != nil {
	// 	return nil, err
	// }
	// ks := tools.NewEthKeyStore(servCfg.ETHConfig, chainId)
	// accArr := ks.GetAccounts()
	// if len(servCfg.ETHConfig.KeyStorePwdSet) == 0 {
	// 	fmt.Println("please input the passwords for ethereum keystore: ")
	// 	for _, v := range accArr {
	// 		fmt.Printf("For address %s. ", v.Address.String())
	// 		raw, err := password.GetPassword()
	// 		if err != nil {
	// 			log.Fatalf("failed to input password: %v", err)
	// 			panic(err)
	// 		}
	// 		servCfg.ETHConfig.KeyStorePwdSet[strings.ToLower(v.Address.String())] = string(raw)
	// 	}
	// }
	// if err = ks.UnlockKeys(servCfg.ETHConfig); err != nil {
	// 	return nil, err
	// }
	accArr := servCfg.StarcoinConfig.PrivateKeys
	senders := make([]*StarcoinSender, len(accArr))
	for i, v := range senders {
		v = &StarcoinSender{}
		senderAddress, senderPrivateKey, err := getAccountAddressAndPrivateKey(accArr[i])
		if err != nil {
			log.Errorf("InitGenesis - Convert string to AccountAddress error:%s", err.Error())
			return nil, err
		}
		v.acc = tools.StarcoinAccount{
			Address: *senderAddress,
		}
		v.keyStore = tools.NewStarcoinKeyStore(senderPrivateKey, servCfg.StarcoinConfig.ChainId)
		v.starcoinClient = stcclient
		//v.keyStore = ks
		v.config = servCfg
		//v.polySdk = polySdk
		// v.contractAbi = &contractabi
		v.seqNumManager = tools.NewSeqNumManager(stcclient)
		v.cmap = make(map[string]chan *StarcoinTxInfo)
		v.db = db
		senders[i] = v
	}

	var bridgeSdk *bridge.SDK
	if servCfg.CheckFee {
		var err error
		bridgeSdk, err = bridge.WithOptions(0, servCfg.BridgeURLs, time.Minute, 10)
		if err != nil {
			log.Errorf("NewPolyManager - new bridge SDK error: %s\n", err.Error())
			return nil, err
		}
	}

	mgr := &PolyManager{
		exitChan:      make(chan int),
		config:        servCfg,
		polySdk:       polySdk,
		currentHeight: startblockHeight,
		//contractAbi:   &contractabi,
		db:             db,
		starcoinClient: stcclient,
		senders:        senders,
		bridgeSdk:      bridgeSdk,
	}

	ok, err := mgr.init()
	if !ok {
		log.Errorf("NewPolyManager - init failed\n")
		return nil, fmt.Errorf("NewPolyManager - init failed")
	} else if err != nil {
		log.Errorf("NewPolyManager - init ok, but something error: %v\n", err)
	}

	// //////////////////////////
	mgr.CheckSmtRoot()
	// //////////////////////////
	return mgr, nil
}

func (this *PolyManager) CheckSmtRoot() error {
	smtRootStr, err := this.getStarcoinCrossChainSmtRoot()
	if err != nil {
		log.Errorf("PolyManager.CheckSmtRoot - failed to getStarcoinCrossChainSmtRoot: %s", err.Error())
		return err
	}
	onChainSmtRoot, err := tools.HexToBytes(smtRootStr)
	if err != nil {
		log.Errorf("PolyManager.CheckSmtRoot - tools.HexToBytes error: %s", err.Error())
		return err
	}
	polyTx, err := this.db.GetLastProcessedPolyTx()
	if err != nil {
		log.Errorf("PolyManager.CheckSmtRoot - failed to db.GetLastProcessedPolyTx: %s", err.Error())
		return err
	}
	computedSmtRoot, err := polyTx.ComputePloyTxInclusionSmtRootHash()
	if err != nil {
		log.Errorf("PolyManager.CheckSmtRoot - failed to polyTx.ComputePloyTxInclusionSmtRootHash: %s", err.Error())
		return err
	}
	if bytes.Equal(onChainSmtRoot, computedSmtRoot) {
		log.Infof("PolyManager CheckSmtRoot - ok: %s", onChainSmtRoot)
		return nil
	} else {
		errCheck := fmt.Errorf("SMT root mismatched. On-chain SMT root: %s, off-chain computed: %s", onChainSmtRoot, computedSmtRoot)
		log.Errorf("PolyManager.CheckSmtRoot - error. %s", errCheck.Error())
		return errCheck
	}
}

func (this *PolyManager) init() (bool, error) {
	var err error
	if this.currentHeight > 0 {
		log.Infof("PolyManager init - start height from flag: %d", this.currentHeight)
		return true, nil
	}
	this.currentHeight, err = this.db.GetPolyHeight() // TODO: ignore DB error?
	curEpochStart := this.findCurEpochStartHeight()
	if curEpochStart > this.currentHeight {
		this.currentHeight = curEpochStart
		log.Infof("PolyManager init - latest height from CCD: %d", this.currentHeight)
		return true, err
	}
	log.Infof("PolyManager init - latest height from DB: %d", this.currentHeight)

	return true, err
}

func (this *PolyManager) MonitorChain() {
	monitorTicker := time.NewTicker(config.POLY_MONITOR_INTERVAL)
	var blockHandleResult bool
	for {
		select {
		case <-monitorTicker.C:
			latestheight, err := this.polySdk.GetCurrentBlockHeight()
			if err != nil {
				log.Errorf("PolyManager.MonitorChain - get poly chain block height error: %s", err.Error())
				continue
			}
			latestheight--
			if latestheight-this.currentHeight < config.ONT_USEFUL_BLOCK_NUM {
				continue
			}
			log.Infof("PolyManager.MonitorChain - poly chain current height: %d", latestheight)
			blockHandleResult = true
			for this.currentHeight <= latestheight-config.ONT_USEFUL_BLOCK_NUM {
				if this.currentHeight%10 == 0 {
					log.Infof("PolyManager.MonitorChain - handle confirmed poly Block height: %d", this.currentHeight)
				}
				blockHandleResult = this.handleDepositEvents(this.currentHeight)
				if blockHandleResult == false {
					//log.Debugf("PolyManager.MonitorChain - handleDepositEvents return false, height: %d", this.currentHeight)
					break
				}
				this.currentHeight++
			}
			//log.Debugf("PolyManager.MonitorChain - about to UpdatePolyHeight: %d", this.currentHeight-1)
			if err = this.db.UpdatePolyHeight(this.currentHeight - 1); err != nil {
				log.Errorf("PolyManager.MonitorChain - failed to save height of poly: %v", err)
			}
		case <-this.exitChan:
			return
		}
	}
}

func (this *PolyManager) MonitorFailedPolyTx() {
	monitorTicker := time.NewTicker(config.POLY_MONITOR_INTERVAL)
	for {
		select {
		case <-monitorTicker.C:
			// ////////////// Get failed or Processing-timed-out Poly(to Starcoin)Tx. ///////////////
			polyTx, err := this.db.GetFirstFailedPolyTx()
			if err != nil {
				log.Errorf("PolyManager.MonitorFailedPolyTx - failed to GetFirstFailedPolyTx: %s", err.Error())
				continue
			}
			this.handleFailedPolyTx(polyTx)
		case <-this.exitChan:
			return
		}
	}
}

func (this *PolyManager) handleFailedPolyTx(polyTx *db.PolyTx) error {
	if polyTx == nil {
		return nil
	}
	// //////////////// Maybe timed-out ///////////////////
	if db.STATUS_PROCESSING == polyTx.Status && polyTx.StarcoinTxHash != "" {
		log.Infof("handleFailedPolyTx - PolyTx status is '%s', but the StarcoinTxHash is '%s', maybe is just timed-out.", polyTx.Status, polyTx.StarcoinTxHash)
		updated, err := this.handleTimedOutPolyTx(polyTx)
		if err != nil {
			return err
		}
		if updated {
			return nil
		}
		log.Infof("handleFailedPolyTx - The PolyTx maybe really failed, txHash: %s, fromChainId: %d", polyTx.TxHash, polyTx.FromChainID)
	}
	sender := this.selectSender()
	//log.Debugf("Get failed poly Tx. hash: %s", polyTx.TxHash)
	ok := sender.sendPolyTxToStarcoin(polyTx)
	if !ok {
		err := fmt.Errorf("failed to sendPolyTxToStarcoin, txHash: %s, fromChainId: %d", polyTx.TxHash, polyTx.FromChainID)
		log.Errorf("PolyManager.handleFailedPolyTx - %s", err)
		return err
	}
	return nil
}

// handle timed-out transactions.
func (this *PolyManager) MonitorTimedOutPolyTx() {
	monitorTicker := time.NewTicker(config.POLY_MONITOR_INTERVAL)
	for {
		select {
		case <-monitorTicker.C:
			// //////////  setPolyTxProcessedIfOnChainSmtRootMatched ///////////
			polyTxList, err := this.db.GetTimedOutOrFailedPolyTxList()
			if err != nil {
				log.Errorf("PolyManager.MonitorTimedOutPolyTx - failed to GetTimedOutOrFailedPolyTxList: %s", err.Error())
			} else if polyTxList != nil {
				this.checkTimedOutOrFailedPolyTxListByOnChainSmtRoot(polyTxList)
			}
			// /////////////// remove PolyTx blocking the process //////////////
			polyTxToBeRemoved, err := this.db.GetFirstPolyTxToBeRemoved()
			if err != nil {
				log.Errorf("PolyManager.MonitorTimedOutPolyTx - failed to GetFirstPolyTxToBeRemoved: %s", err.Error())
			} else if polyTxToBeRemoved != nil {
				this.handlePolyTxToBeRemoved(polyTxToBeRemoved)
			}
			// ////////////////// handle First timed-out PolyTx ////////////////
			polyTx, err := this.db.GetFirstTimedOutPolyTx()
			if err != nil {
				log.Errorf("PolyManager.MonitorTimedOutPolyTx - failed to GetFirstTimedOutPolyTx: %s", err.Error())
				continue
			}
			if polyTx != nil {
				//log.Debugf("Get timed-out poly Tx. hash: %s", polyTx.TxHash)
				this.handleTimedOutPolyTx(polyTx)
			}
			// /////////////// handle RemovedPolyTxToBePushedBack //////////////
			removedPolyTx, err := this.db.GetFirstRemovedPolyTxToBePushedBack()
			if err != nil {
				log.Errorf("PolyManager.MonitorTimedOutPolyTx - failed to GetFirstRemovedPolyTxToBePushedBack: %s", err.Error())
				continue
			}
			if removedPolyTx != nil {
				this.handleRemovedPolyTxToBePushedBack(removedPolyTx)
			}
		case <-this.exitChan:
			return
		}
	}
}

func (this *PolyManager) MonitorGasSubsidy() {
	monitorTicker := time.NewTicker(config.GAS_SUBSIDY_MONITOR_INTERVAL)
	for {
		select {
		case <-monitorTicker.C:
			// ////////////// create gas subsidy ////////////////
			this.createGasSubsidies()
			// ////////////// handle not-sent gas subsidy ///////////////
			notSentGasSubsidy, err := this.db.GetFirstNotSentGasSubsidy()
			if err != nil {
				log.Errorf("PolyManager.MonitorGasSubsidy - failed to GetFirstNotSentGasSubsidy: %s", err.Error())
			}
			if notSentGasSubsidy != nil {
				this.handleNotSentGasSubsidy(notSentGasSubsidy) //continue
			}
			// ////////////////// handle First timed-out gas subsidy ////////////////
			timedOutGasSubsidy, err := this.db.GetFirstTimedOutGasSubsidy()
			if err != nil {
				log.Errorf("PolyManager.MonitorGasSubsidy - failed to GetFirstTimedOutPolyTx: %s", err.Error())
				continue
			}
			if timedOutGasSubsidy != nil {
				//log.Debugf("Get timed-out gas subsidy Tx. hash: %s", polyTx.TxHash)
				this.handleTimedOutGasSubsidy(timedOutGasSubsidy)
			}
			///////////////////////////////////////////////////////////
			failedGasSubsidy, err := this.db.GetFirstFailedGasSubsidy()
			if err != nil {
				log.Errorf("PolyManager.MonitorGasSubsidy - failed to GetFirstFailedGasSubsidy: %s", err.Error())
				continue
			}
			if failedGasSubsidy != nil {
				//log.Debugf("Get failed poly Tx. hash: %s", polyTx.TxHash)
				hfErr := this.handleFailedGasSubsidy(failedGasSubsidy)
				if hfErr != nil {
					log.Errorf("PolyManager.MonitorGasSubsidy - failed to handleFailedGasSubsidy: %s", hfErr.Error())
				}
			}
		case <-this.exitChan:
			return
		}
	}
}

func (this *PolyManager) createGasSubsidies() {
	for _, fromChainId := range this.config.StarcoinConfig.GasSubsidyConfig.FromChainIds {
		polyTxList, err := this.db.GetPolyTxListNotHaveGasSubsidy(uint64(fromChainId), GAS_SUBSIDY_ONLY_AFTER)
		if err != nil {
			log.Errorf("PolyManager.MonitorPolyTxNotHaveSubsidy - failed to GetPolyTxListNotHaveGasSubsidy: %s", err.Error())
			continue
		}
		for _, polyTx := range polyTxList {
			fromChainKey := strconv.FormatInt(int64(fromChainId), 10)
			gasSubsidy, err := PolyTxToGasSubsidy(polyTx, this.randomizeSubsidyAmount(fromChainKey))
			if err != nil {
				log.Errorf("PolyManager.MonitorPolyTxNotHaveSubsidy - failed to PolyTxToGasSubsidy: %s", err.Error())
				continue
			}
			// //////////////////////////////
			if this.isToAddressInGasSubsidyBlacklist(gasSubsidy.ToAddress) {
				gasSubsidy.SubsidyAmount = 0
			}
			// //////////////////////////////
			gtOrEqMinAmount, err := this.isGtOrEqSubsidizableMinAssetAmount(gasSubsidy.ToAssetHash, gasSubsidy.UnlockAssetAmount)
			if err != nil {
				log.Errorf("PolyManager.MonitorPolyTxNotHaveSubsidy - failed to invoke isGtOrEqSubsidizableMinAssetAmount: %s", err.Error())
				continue
			}
			if !gtOrEqMinAmount {
				gasSubsidy.SubsidyAmount = 0
			}
			// //////////////////////////////
			tooMuch, err := this.isToAddressTooMuchGasSubsidy(gasSubsidy.ToAddress)
			if err != nil {
				log.Errorf("PolyManager.MonitorPolyTxNotHaveSubsidy - failed to invoke isToAddressTooMuchGasSubsidy: %s", err.Error())
				continue
			}
			if tooMuch {
				gasSubsidy.SubsidyAmount = 0
			}
			// //////////////////////////////
			err = this.db.PutGasSubsidy(gasSubsidy)
			if err != nil {
				log.Errorf("PolyManager.MonitorPolyTxNotHaveSubsidy - failed to PutGasSubsidy: %s", err.Error())
				continue
			}
		}
	}
}

func (this *PolyManager) randomizeSubsidyAmount(fromChain string) uint64 {
	a := this.config.StarcoinConfig.GasSubsidyConfig.FromChains[fromChain].SubsidyAmount
	rand.Seed(time.Now().UnixNano())
	return uint64(rand.Int63n(int64(a)/10*9) + int64(a)/10)
}

func (this *PolyManager) isToAddressTooMuchGasSubsidy(addr string) (bool, error) {
	count, err := this.db.GetGasSubsidyCountByToAddress(addr)
	if err != nil {
		return false, err
	}
	return count >= 3, nil //this.config.StarcoinConfig.GasSubsidyConfig.MaxGasSubsidyCount, nil
}

func (this *PolyManager) isToAddressInGasSubsidyBlacklist(addr string) bool {
	a := strings.Split(this.config.StarcoinConfig.GasSubsidyConfig.ToAddressBlacklist, ",")
	for _, b := range a {
		if strings.EqualFold(b, addr) {
			return true
		}
	}
	return false
}

func (this *PolyManager) isGtOrEqSubsidizableMinAssetAmount(assetHash string, amount string) (bool, error) {
	a, err := strconv.ParseUint(amount, 10, 64)
	if err != nil {
		return false, err
	}
	return a >= this.config.StarcoinConfig.GasSubsidyConfig.SubsidizableMinAssetAmounts[assetHash], nil
}

func (this *PolyManager) handleFailedGasSubsidy(gasSubsidy *db.GasSubsidy) error {
	if gasSubsidy == nil {
		return nil
	}
	// //////////////// Maybe timed-out ///////////////////
	if db.STATUS_PROCESSING == gasSubsidy.Status && gasSubsidy.StarcoinTxHash != "" {
		log.Infof("handleFailedGasSubsidy - gas subsidy status is '%s', but the StarcoinTxHash is '%s', maybe is just timed-out.", gasSubsidy.Status, gasSubsidy.StarcoinTxHash)
		updated, err := this.handleTimedOutGasSubsidy(gasSubsidy)
		if err != nil {
			return err
		}
		if updated {
			return nil // status updated, just return!
		}
		log.Infof("handleFailedGasSubsidy - The gas subsidy maybe really failed, txHash: %s, fromChainId: %d", gasSubsidy.TxHash, gasSubsidy.FromChainID)
	}
	return this.sendGasSubsidyTxToStarcoin(gasSubsidy)
}

func (this *PolyManager) handleNotSentGasSubsidy(gasSubsidy *db.GasSubsidy) error {
	return this.sendGasSubsidyTxToStarcoin(gasSubsidy)
}

func (this *PolyManager) sendGasSubsidyTxToStarcoin(gasSubsidy *db.GasSubsidy) error {
	// //////////////////////////////////////////////////
	if gasSubsidy.SubsidyAmount <= 0 {
		log.Info("PolyManager.handleNotSentGasSubsidy - not need to send gas subsidy, because amount is zero.")
		this.db.SetGasSubsidyStatusProcessed(gasSubsidy.TxHash, gasSubsidy.FromChainID, gasSubsidy.Status)
		return nil
	}
	// //////////////////////////////////////////////////
	if len(this.config.StarcoinConfig.GasSubsidyConfig.SenderPrivateKeys) == 0 {
		log.Info("PolyManager.handleNotSentGasSubsidy - failed to get sender PrivateKey")
		return nil
	}
	senderAndPK := this.config.StarcoinConfig.GasSubsidyConfig.SenderPrivateKeys[0]
	senderAddress, _, err := getAccountAddressAndPrivateKey(senderAndPK)
	payee, err := types.ToAccountAddress(gasSubsidy.ToAddress)
	if err != nil {
		log.Error("PolyManager.handleNotSentGasSubsidy - failed to get sender PrivateKey")
		return err
	}
	amount := serde.Uint128{
		Low:  gasSubsidy.SubsidyAmount,
		High: 0,
	}
	signedTx, seqNumber, err := EncodeAndSignTransferStcTransaction(this.starcoinClient, senderAndPK, *payee, amount)
	if err != nil {
		log.Error("PolyManager.handleNotSentGasSubsidy - failed to EncodeAndSignTransferStcTransaction")
		return err
	}
	offChainTxHash, err := stcclient.GetSignedUserTransactionHash(signedTx)
	if err != nil {
		log.Error("PolyManager.handleNotSentGasSubsidy - failed to GetSignedUserTransactionHash")
		return err
	}
	// //////////////////////////////////////////////////////////////////
	// Set gas subsidy's Starcoin Txn. info.(and  status to PROCESSING)
	err = this.db.SetGasSubsidyStarcoinTxInfo(gasSubsidy.TxHash, gasSubsidy.FromChainID, gasSubsidy.Status, offChainTxHash, senderAddress[:], seqNumber)
	if err != nil {
		log.Error("PolyManager.handleNotSentGasSubsidy - failed to SetGasSubsidyStarcoinTxInfo")
		return err
	}
	gasSubsidy.Status = db.STATUS_PROCESSING // now status is processing
	// /////////////////////////////////////////////////////////////////
	txhash, err := this.starcoinClient.SubmitSignedTransaction(context.Background(), signedTx)
	if err != nil {
		log.Error("PolyManager.handleNotSentGasSubsidy - failed to SubmitSignedTransaction")
		return err
	}
	if !strings.EqualFold(txhash, tools.EncodeToHex(offChainTxHash)) {
		errMsg := fmt.Sprintf("Returned Tx. Hash(%s) != signed Tx. Hash(%s)", txhash, tools.EncodeToHex(offChainTxHash))
		log.Error("PolyManager.handleNotSentGasSubsidy - %s", errMsg)
		return fmt.Errorf(errMsg)
	}
	// ///////////// wait Transaction Confirm ///////////////
	isSuccess, err := tools.WaitTransactionConfirm(*this.starcoinClient, txhash, WAIT_STARCOIN_TRANSACTION_CONFIRM_TIME)
	if isSuccess {
		log.Infof("successful to handle Gas Subsidy: (starcoin_hash: %s, seq_number: %d, starcoin_explorer: %s)",
			txhash, seqNumber, tools.GetExplorerUrl(this.config.StarcoinConfig.ChainId)+txhash)
		this.db.SetGasSubsidyStatusProcessed(gasSubsidy.TxHash, gasSubsidy.FromChainID, db.STATUS_PROCESSING)
	} else {
		if err == nil {
			log.Infof("failed to handle Gas Subsidy, error is nil. Maybe timed out or cannot get transaction info.")
			dbErr := this.db.SetGasSubsidyStatus(gasSubsidy.TxHash, gasSubsidy.FromChainID, db.STATUS_PROCESSING, db.STATUS_TIMEDOUT) // set status to TIMED-OUT!
			if dbErr != nil {
				log.Errorf("failed to Set Gas Subsidy to timed-out. Error: %v", dbErr)
				//return dbErr
			}
		} else {
			dbErr := this.db.SetGasSubsidyStatus(gasSubsidy.TxHash, gasSubsidy.FromChainID, db.STATUS_PROCESSING, db.STATUS_FAILED) // set status to FAILED.
			if dbErr != nil {
				log.Errorf("failed to Set Gas Subsidy to failed. Error: %v", dbErr)
				//return dbErr
			}
		}
		log.Errorf("failed to handle Gas Subsidy: (starcoin_hash: %s, seq_number: %d, starcoin_explorer: %s), error: %v",
			txhash, seqNumber, tools.GetExplorerUrl(this.config.StarcoinConfig.ChainId)+txhash, err)
		return err // this err maybe nil
	}
	return nil
}

func (this *PolyManager) handleRemovedPolyTxToBePushedBack(removedPolyTx *db.RemovedPolyTx) error {
	if removedPolyTx.Status != db.STATTUS_TO_BE_PUSHED_BACK {
		return nil // just ignore
	}
	err := this.db.PushBackRemovePolyTx(removedPolyTx.ID)
	return err
}

func (this *PolyManager) handlePolyTxToBeRemoved(polyTx *db.PolyTx) error {
	smtRootStr, err := this.getStarcoinCrossChainSmtRoot()
	if err != nil {
		log.Errorf("PolyManager.handlePolyTxToBeRemoved - (poly to starcoin)failed to getStarcoinCrossChainSmtRoot: %s", err.Error())
		return err
	}
	onChainSmtRoot, err := tools.HexToBytes(smtRootStr)
	if err != nil {
		log.Errorf("PolyManager.handlePolyTxToBeRemoved - (poly to starcoin)failed to tools.HexToBytes: %s", err.Error())
		return err
	}
	nonMemberSmtRoot, err := polyTx.GetSmtNonMembershipRootHash()
	if err != nil {
		log.Errorf("PolyManager.handlePolyTxToBeRemoved - (poly to starcoin)failed to PolyTx.GetSmtNonMembershipRootHash: %s", err.Error())
		return err
	}
	if bytes.Equal(onChainSmtRoot, nonMemberSmtRoot) {
		err = this.db.RemovePolyTx(polyTx)
		if err != nil {
			log.Errorf("PolyManager.handlePolyTxToBeRemoved - (poly to starcoin)failed to db.RemovePolyTx: %s", err.Error())
			return err
		}
	} else {
		err = fmt.Errorf("PolyTx cannot be removed because it's non-membership SMT root is not matched with current on-chain SMT Root")
		log.Errorf("PolyManager.handlePolyTxToBeRemoved - (poly to starcoin)%s, TxIndex: %d", err.Error(), polyTx.TxIndex)
		return err
	}
	return nil
}

func (this *PolyManager) checkTimedOutOrFailedPolyTxListByOnChainSmtRoot(list []*db.PolyTx) error {
	starcoinHeight, err := tools.GetStarcoinNodeHeight(this.config.StarcoinConfig.RestURL, tools.NewRestClient())
	if err != nil {
		log.Errorf("PolyManager.checkTimedOutOrFailedPolyTxListByOnChainSmtRoot - (poly to starcoin)failed to GetStarcoinNodeHeight: %s", err.Error())
		return err
	}
	smtRootStr, err := this.getStarcoinCrossChainSmtRoot()
	if err != nil {
		log.Errorf("PolyManager.checkTimedOutOrFailedPolyTxListByOnChainSmtRoot - (poly to starcoin)failed to getStarcoinCrossChainSmtRoot: %s", err.Error())
		return err
	}
	onChainSmtRoot, err := tools.HexToBytes(smtRootStr)
	if err != nil {
		log.Errorf("PolyManager.checkTimedOutOrFailedPolyTxListByOnChainSmtRoot - (poly to starcoin)failed to tools.HexToBytes: %s", err.Error())
		return err
	}
	for _, polyTx := range list {
		s, err := this.setPolyTxProcessedIfOnChainSmtRootMatched(polyTx, onChainSmtRoot, starcoinHeight)
		if err != nil {
			log.Errorf("PolyManager.checkTimedOutOrFailedPolyTxListByOnChainSmtRoot - (poly to starcoin)setPolyTxProcessedIfOnChainSmtRootMatched error: %s", err.Error())
			continue
		}
		if s != "" {
			log.Infof("PolyManager.checkTimedOutOrFailedPolyTxListByOnChainSmtRoot - set Poly(to starcoin)Tx status to PROCESSED because of matched SMT root. Starcoin hash: %s, SMT root: %s", polyTx.StarcoinTxHash, hex.EncodeToString(onChainSmtRoot))
			break
		} else {
			continue
		}
	}
	return nil
}

// Set PolyTx's status to PROCESSED if the SMT Root which included it matchs On-Chain SMT Root.
// Return Starcoin tx. hash if matched, or else return empty string.
func (this *PolyManager) setPolyTxProcessedIfOnChainSmtRootMatched(polyTx *db.PolyTx, onChainSmtRoot []byte, starcoinHeight uint64) (string, error) {
	if polyTx.StarcoinTxHash == "" {
		return "", nil
	}
	txInfo, err := this.starcoinClient.GetTransactionInfoByHash(context.Background(), polyTx.StarcoinTxHash)
	if err != nil {
		return "", nil
	}
	if txInfo == nil || txInfo.BlockNumber == "" {
		return "", nil // Cannot find Transaction info on-chain
	}
	txHeight, err := strconv.ParseUint(txInfo.BlockNumber, 10, 64)
	if err != nil {
		return "", err
	}
	if txHeight > starcoinHeight-this.config.StarcoinConfig.BlockConfirmations {
		return "", nil
	}
	computedSmtRoot, err := polyTx.ComputePloyTxInclusionSmtRootHash()
	if bytes.Equal(onChainSmtRoot, computedSmtRoot) {
		this.db.SetPolyTxStatusProcessed(polyTx.TxHash, polyTx.FromChainID, polyTx.Status, polyTx.StarcoinTxHash)
		return polyTx.StarcoinTxHash, nil
	} else {
		return "", nil
	}
}

func (this *PolyManager) getStarcoinCrossChainSmtRoot() (string, error) {
	genesisAccountAddress := this.config.StarcoinConfig.GenesisAccountAddress
	resType := this.config.StarcoinConfig.CCSMTRootResourceType
	smtRoot, err := GetStarcoinCrossChainSmtRoot(this.starcoinClient, genesisAccountAddress, resType)
	return smtRoot, err
}

// Handle timed-out Poly(to Starcoin)Tx., return true if status in DB updated.
func (this *PolyManager) handleTimedOutPolyTx(polyTx *db.PolyTx) (bool, error) {
	var err error
	var updated bool = false
	if polyTx.StarcoinTxHash == "" {
		err = this.db.SetPolyTxStatus(polyTx.TxHash, polyTx.FromChainID, polyTx.Status, db.STATUS_UNKNOWN_ERROR)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	errorCallback := func(e error) {
		log.Errorf("PolyManager.handleTimedOutPolyTx GetTransactionInfoByHash - failed to GetTransactionInfoByHash: %s", err.Error())
		err = e
	}
	executedCallback := func() {
		err = this.db.SetPolyTxStatusProcessed(polyTx.TxHash, polyTx.FromChainID, polyTx.Status, polyTx.StarcoinTxHash)
		if err == nil {
			updated = true
		}
		log.Infof("PolyManager.handleTimedOutPolyTx set timed-out Poly(to starcoin)Tx status to EXECUTED. FromChainId: %d, TxHash: %s, Starcoin hash: %s", polyTx.FromChainID, polyTx.TxHash, polyTx.StarcoinTxHash)
	}
	knownFailureCallback := func() {
		err = this.db.SetPolyTxStatus(polyTx.TxHash, polyTx.FromChainID, polyTx.Status, db.STATUS_FAILED)
		if err == nil {
			updated = true
		}
		log.Infof("PolyManager.handleTimedOutPolyTx set timed-out Poly(to starcoin)Tx status to FAILED. FromChainId: %d, TxHash: %s, Starcoin hash: %s", polyTx.FromChainID, polyTx.TxHash, polyTx.StarcoinTxHash)
	}
	unknownFailureCallback := func() {
		if polyTx.UpdatedAt < db.CurrentTimeMillis()-MAX_TIMEDOUT_TO_FAILED_WAITING_SECONDS*1000 {
			err = this.db.SetPolyTxStatus(polyTx.TxHash, polyTx.FromChainID, polyTx.Status, db.STATUS_FAILED)
			if err == nil {
				updated = true
			}
			log.Infof("PolyManager.handleTimedOutPolyTx set timed-out Poly(to starcoin)Tx status to FAILED because exceeded MAX_TIMEDOUT_TO_FAILED_WAITING_SECONDS. FromChainId: %d, TxHash: %s, Starcoin hash: %s", polyTx.FromChainID, polyTx.TxHash, polyTx.StarcoinTxHash)
		}
	}
	checkStarcoinTransaction(this.starcoinClient, polyTx.StarcoinTxHash, errorCallback, executedCallback, knownFailureCallback, unknownFailureCallback)
	return updated, err
}

// Handle timed-out gas subsidy (Starcoin)Tx., return true if status updated.
func (this *PolyManager) handleTimedOutGasSubsidy(gasSubsidy *db.GasSubsidy) (bool, error) {
	var err error
	var updated bool = false
	if gasSubsidy.StarcoinTxHash == "" {
		err = this.db.SetGasSubsidyStatus(gasSubsidy.TxHash, gasSubsidy.FromChainID, gasSubsidy.Status, db.STATUS_UNKNOWN_ERROR)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	errorCallback := func(e error) {
		err = e
		log.Errorf("PolyManager.handleTimedOutGasSubsidy GetTransactionInfoByHash - failed to GetTransactionInfoByHash: %s", err.Error())
	}
	executedCallback := func() {
		err = this.db.SetGasSubsidyStatusProcessed(gasSubsidy.TxHash, gasSubsidy.FromChainID, gasSubsidy.Status)
		if err == nil {
			updated = true
		}
		log.Infof("PolyManager.handleTimedOutGasSubsidy set timed-out gas subsidy(to starcoin)Tx status to EXECUTED. FromChainId: %d, TxHash: %s, Starcoin hash: %s", gasSubsidy.FromChainID, gasSubsidy.TxHash, gasSubsidy.StarcoinTxHash)
	}
	knownFailureCallback := func() {
		err = this.db.SetGasSubsidyStatus(gasSubsidy.TxHash, gasSubsidy.FromChainID, gasSubsidy.Status, db.STATUS_FAILED)
		if err == nil {
			updated = true
		}
		log.Infof("PolyManager.handleTimedOutGasSubsidy set timed-out gas subsidy(to starcoin)Tx status to FAILED. FromChainId: %d, TxHash: %s, Starcoin hash: %s", gasSubsidy.FromChainID, gasSubsidy.TxHash, gasSubsidy.StarcoinTxHash)
	}
	unknownFailureCallback := func() {
		if gasSubsidy.UpdatedAt < db.CurrentTimeMillis()-MAX_TIMEDOUT_TO_FAILED_WAITING_SECONDS*1000 {
			err = this.db.SetGasSubsidyStatus(gasSubsidy.TxHash, gasSubsidy.FromChainID, gasSubsidy.Status, db.STATUS_FAILED)
			if err == nil {
				updated = true
			}
			log.Infof("PolyManager.handleTimedOutGasSubsidy set timed-out gas subsidy(to starcoin)Tx status to FAILED because exceeded MAX_TIMEDOUT_TO_FAILED_WAITING_SECONDS. FromChainId: %d, TxHash: %s, Starcoin hash: %s", gasSubsidy.FromChainID, gasSubsidy.TxHash, gasSubsidy.StarcoinTxHash)
		}
	}
	checkStarcoinTransaction(this.starcoinClient, gasSubsidy.StarcoinTxHash, errorCallback, executedCallback, knownFailureCallback, unknownFailureCallback)
	return updated, err
}

func (this *PolyManager) MonitorDeposit() {
	monitorTicker := time.NewTicker(config.POLY_MONITOR_INTERVAL)
	for {
		select {
		case <-monitorTicker.C:
			//this.handleLockDepositEvents()
			list, err := this.db.GetAllPolyTxRetry()
			if err != nil {
				log.Errorf("MonitorDeposit - db.GetAllPolyTxRetry error: %s", err.Error())
				continue
			}
			for _, r := range list {
				this.handlePolyTxRetry(r)
			}
		case <-this.exitChan:
			return
		}
	}
}

func (this *PolyManager) handlePolyTxRetry(r *db.PolyTxRetry) error {
	bridgeTransactionBS, err := hex.DecodeString(r.BridgeTransaction)
	if err != nil {
		log.Errorf("handlePolyTxRetry - bridgeTransaction.Deserialization error: %s", err.Error())
		return err
	}
	bridgeTransaction := new(BridgeTransaction)
	err = bridgeTransaction.Deserialization(common.NewZeroCopySource(bridgeTransactionBS))
	if err != nil {
		log.Errorf("handlePolyTxRetry - bridgeTransaction.Deserialization error: %s", err.Error())
		return err
	}
	starcoinOk, starcoinStatus, starcoinMsg := this.checkStarcoinStatusByProof(bridgeTransaction.rawAuditPath)
	if !starcoinOk {
		dbErr := this.db.UpdatePolyTxStarcoinStatus(r.TxHash, r.FromChainID, starcoinStatus, starcoinMsg)
		if dbErr != nil {
			log.Errorf("handlePolyTxRetry - UpdatePolyTxStarcoinStatus() error: %s", dbErr.Error())
		}
		return fmt.Errorf(starcoinStatus + " " + starcoinMsg)
	}

	// ///////////////////////
	err = this.db.IncreasePolyTxRetryCheckFeeCount(r.TxHash, r.FromChainID, r.CheckFeeCount)
	if err != nil {
		log.Errorf("handlePolyTxRetry - IncreasePolyTxRetryCheckFeeCount() error: %s", err.Error())
		return err
	}
	e, err := r.GetPolyEvent()
	if err != nil {
		log.Errorf("handlePolyTxRetry - GetPolyEvent() error: %s", err.Error())
		return err
	}
	s, err := CheckFee(this.bridgeSdk, r.FromChainID, e.TxId, e.PolyHash)
	if err != nil {
		log.Errorf("handlePolyTxRetry - CheckFee() error: %s", err.Error())
		return err
	}
	if s.Pass() {
		px, err := polyTxRetryToPolyTx(r)
		if err != nil {
			return err
		}
		sender := this.selectSender()
		log.Infof("sender %s is handling poly tx ( PolyTxHash: %s, TxHash: %s, FromChainId: %d )", tools.EncodeToHex(sender.acc.Address[:]), px.PolyTxHash, px.TxHash, px.FromChainID)
		sent, saved := sender.putPolyTxThenSend(px)
		if !sent {
			log.Errorf("handlePolyTxRetry - failed to putPolyTxThenSend, not sent. PolyTxHash: %s, TxHash: %s, FromChainId: %d", px.PolyTxHash, px.TxHash, px.FromChainID)
		}
		if !saved {
			err := fmt.Errorf("Failed to putPolyTxThenSend, not saved. PolyTxHash: %s, TxHash: %s, FromChainId: %d", px.PolyTxHash, px.TxHash, px.FromChainID)
			log.Errorf("handlePolyTxRetry - error: %s", err.Error())
			return err
		} else {
			//return this.db.SetPolyTxRetryFeeStatus(r.TxHash, r.FromChainID, strconv.Itoa(int(bridge.PAID)))
			return this.db.DeletePolyTxRetry(r.TxHash, r.FromChainID)
		}
	} else {
		err := this.db.SetPolyTxRetryFeeStatus(r.TxHash, r.FromChainID, strconv.Itoa(int(s.Status)))
		if err != nil {
			log.Errorf("handlePolyTxRetry - SetPolyTxRetryFeeStatus() error: %s", err.Error())
			return err
		}
	}
	return nil
}

func (this *PolyManager) ReHandlePolyHeight(height uint64) bool {
	return this.handleDepositEvents(uint32(height))
}

func (this *PolyManager) handleDepositEvents(height uint32) bool {
	lastEpoch := this.findCurEpochStartHeight()
	hdr, err := this.polySdk.GetHeaderByHeight(height + 1)
	if err != nil {
		log.Errorf("handleDepositEvents - GetNodeHeader on height :%d failed", height)
		return false
	}
	isCurr := lastEpoch < height+1
	isEpoch, pubkList, err := this.IsEpoch(hdr)
	if err != nil {
		log.Errorf("falied to check isEpoch: %v", err)
		return false
	}
	var (
		anchor *polytypes.Header
		hp     string
	)
	if !isCurr { //lastEpoch >= height+1
		anchor, err = this.polySdk.GetHeaderByHeight(lastEpoch + 1)
		if err != nil {
			log.Errorf("handleDepositEvents - polySdk.GetHeaderByHeight, height: %d, error: %v", lastEpoch+1, err)
		}
		proof, err := this.polySdk.GetMerkleProof(height+1, lastEpoch+1)
		if err != nil {
			log.Errorf("handleDepositEvents - polySdk.GetMerkleProof, block height: %d, root height: %d, error: %v", height+1, lastEpoch+1, err)
		}
		hp = proof.AuditPath
	} else if isEpoch {
		anchor, err = this.polySdk.GetHeaderByHeight(height + 2)
		if err != nil {
			log.Errorf("handleDepositEvents - polySdk.GetHeaderByHeight, height: %d, error: %v", height+2, err)
		}
		proof, err := this.polySdk.GetMerkleProof(height+1, height+2)
		if err != nil {
			log.Errorf("handleDepositEvents - polySdk.GetMerkleProof, block height: %d, root height: %d, error: %v", height+1, height+2, err)
		}
		hp = proof.AuditPath
	}

	cnt := 0
	events, err := this.polySdk.GetSmartContractEventByBlock(height)
	for err != nil {
		log.Errorf("handleDepositEvents - get block event at height:%d error: %s", height, err.Error())
		return false
	}

	heightProcessed := true
	for _, event := range events {
		for _, notify := range event.Notify {
			if notify.ContractAddress == this.config.PolyConfig.EntranceContractAddress {
				states := notify.States.([]interface{})
				method, _ := states[0].(string) // ignore is safe
				if method != "makeProof" {
					//log.Debug("It is not a 'makeProof' method!")
					continue
				}
				if uint64(states[2].(float64)) != this.config.StarcoinConfig.SideChainId {
					//log.Debug("uint64(states[2].(float64)) != this.config.StarcoinConfig.SideChainId")
					continue
				}
				proof, err := this.polySdk.GetCrossStatesProof(hdr.Height-1, states[5].(string))
				if err != nil {
					log.Errorf("handleDepositEvents - failed to get proof for key %s: %v", states[5].(string), err)
					continue
				}
				auditpath, err := hex.DecodeString(proof.AuditPath)
				if err != nil {
					log.Errorf("handleDepositEvents - failed to hex.DecodeString(proof.AuditPath): %v", err)
				}
				value, _, _, err := tools.ParseAuditpath(auditpath)
				if err != nil {
					log.Errorf("handleDepositEvents - failed to tools.ParseAuditpath(auditpath): %v", err)
				}
				param := &common2.ToMerkleValue{}
				if err := param.Deserialization(common.NewZeroCopySource(value)); err != nil {
					log.Errorf("handleDepositEvents - failed to deserialize MakeTxParam (value: %x, err: %v)", value, err)
					continue
				}
				var isTarget bool
				if len(this.config.ProxyOrAssetContracts) > 0 {
					//proxyOrAssetContract := ethcommon.BytesToAddress(param.MakeTxParam.ToContractAddress).String()
					proxyOrAssetContract := string(param.MakeTxParam.ToContractAddress) // starcoin module(address and name)...
					for _, v := range this.config.ProxyOrAssetContracts {
						chainIdArrMap, ok := v[proxyOrAssetContract]
						if ok {
							if len(chainIdArrMap["inbound"]) == 0 { // for 'target' proxy contract, filter is inbound chain Id.
								isTarget = true
								break
							}
							for _, id := range chainIdArrMap["inbound"] {
								if id == param.FromChainID {
									isTarget = true
									break
								}
							}
							if isTarget {
								break
							}
						}
					}
					if !isTarget {
						//fmt.Printf("!isTarget, IGNORED! proxyOrAssetContract: %s\n", proxyOrAssetContract)
						continue
					}
				}
				cnt++
				//fmt.Println(cnt)
				//fmt.Println(states[0].(string))

				putToRetry := this.config.CheckFee
				if !putToRetry {
					b, _, _ := this.checkStarcoinStatusByProof(auditpath) // ignore check-status and message
					putToRetry = !b
				}
				if putToRetry {
					ok := this.putPolyTxRetry(height, event, notify, hdr, param, hp, anchor, auditpath)
					if !ok {
						heightProcessed = false // break! re-process this height!
						break
					} else {
						//continue
					}
					// /////////////////////////////////////////////////////////
					// then check fee and starcoin status for PolyTxRetry ...
					// /////////////////////////////////////////////////////////
				} else {
					sender := this.selectSender()
					log.Infof("sender %s is handling poly tx ( hash: %s, height: %d )", tools.EncodeToHex(sender.acc.Address[:]), event.TxHash, height)
					// //////////////////////////
					sent, saved := sender.commitDepositEventsWithHeader(hdr, param, hp, anchor, event.TxHash, auditpath)
					if !sent { // temporarily ignore the error for tx?
						log.Errorf("handleDepositEvents - failed to commitDepositEventsWithHeader, not sent. Poly tx hash: %s", event.TxHash)
					}
					if !saved {
						log.Errorf("handleDepositEvents - failed to commitDepositEventsWithHeader, not saved. Poly tx hash: %s", event.TxHash)
						heightProcessed = false // break! re-process this height!
						break
					} else {
						// continue
					}
				} // end if
			} //end if notify.ContractAddress == this.config.PolyConfig.EntranceContractAddress {
			//----------- break ------------
			if !heightProcessed {
				break
			}
			//----------- break ------------
		} // end for _, notify := range event.Notify {
	} //for _, event := range events {

	if !heightProcessed {
		return false
	}
	if cnt == 0 && isEpoch && isCurr {
		sender := this.selectSender()
		return sender.changeBookKeeper(hdr, pubkList) // commitHeader
	}

	return true
}

// Check Starcoin on-chain status for the PolyTx, see if it is suitable to commit on chain.
// If suitable, return true, or else return false and check-status and message.
func (this *PolyManager) checkStarcoinStatusByProof(proof []byte) (bool, string, string) {
	_, unlockArgs, err := ParseCrossChainUnlockParamsFromProof(proof)
	if err != nil {
		log.Errorf("PolyManager.checkStarcoinStatusByProof - failed to ParseCrossChainUnlockParamsFromProof, error: %s", err.Error())
		return false, db.STARCOIN_STATUS_PARSE_TX_ERROR, "ParseCrossChainUnlockParamsFromProof error"
	}
	addr := tools.EncodeToHex(unlockArgs.ToAddress)
	tokenType := string(unlockArgs.ToAssetHash)
	accountExists, existsErr := tools.AccountExistsAt(this.starcoinClient, addr)
	if existsErr != nil {
		log.Infof("PolyManager.checkStarcoinStatusByProof - call AccountExistsAt() error: %s", existsErr.Error())
		return false, "", ""
	}
	if accountExists {
		isAcceptToken, isAccErr := tools.IsAcceptToken(this.starcoinClient, addr, tokenType)
		if isAccErr != nil {
			log.Infof("PolyManager.checkStarcoinStatusByProof - call IsAcceptToken() error: %s", isAccErr.Error())
			return false, "", ""
		}
		if !isAcceptToken {
			msg := fmt.Sprintf("'%s' not accept token '%s'", addr, tokenType)
			log.Debug("PolyManager.checkStarcoinStatusByProof - " + msg)
			return false, db.STARCOIN_STATUS_NOT_ACCEPT_TOKEN, msg
		} else {
			return true, "", ""
		}
	} else {
		return true, "", "" // treated as Is-Accept-Token, cause account will be auto-created.
	}
}

// Put PolyTx in DB, return true if saved in DB.
func (this *PolyManager) putPolyTxRetry(height uint32, event *pcommon.SmartContactEvent, notify *pcommon.NotifyEventInfo, hdr *polytypes.Header, param *common2.ToMerkleValue, hp string, anchor *polytypes.Header, auditpath []byte) bool {
	polyMsgTx, err := newPolyMsgTx(height, event, notify)
	if err != nil {
		log.Errorf("putPolyTxRetry - failed to newPolyMsgTx, height: %d, poly hash(event.TxHash): %s", height, event.TxHash)
		return false
	}
	//log.Debugf("putPolyTxRetry - newPolyMsgTx, height: %d, poly hash(event.TxHash): %s", height, event.TxHash)
	btx := &BridgeTransaction{
		header:       hdr,
		param:        param,
		headerProof:  hp,
		anchorHeader: anchor,
		//polyTxHash:   event.TxHash,
		rawAuditPath: auditpath,
		//hasPay:       ...,
		//fee:          "0",
	}
	btxSink := common.NewZeroCopySink(nil)
	btx.Serialization(btxSink)
	txRetry, err := db.NewPolyTxRetry(param.TxHash, param.FromChainID, btxSink.Bytes(), polyMsgTx)
	if err != nil {
		log.Errorf("putPolyTxRetry - failed to NewPolyTxRetry, not saved. Poly tx hash: %s", event.TxHash)
		return false
	}
	err = this.db.PutPolyTxRetry(txRetry)
	if err != nil {
		duplicate, checkErr := db.IsDuplicatePolyTxRetryError(this.db, txRetry, err)
		if checkErr != nil {
			log.Errorf("putPolyTxRetry - call db.IsDuplicatePolyTxError() error: %s", checkErr.Error())
			return false // treated as not saved
		}
		if duplicate {
			log.Warnf("putPolyTxRetry - duplicate PolyTxRetry. Poly tx hash: %s", event.TxHash)
			// ignore, treated as saved
		} else {
			log.Errorf("putPolyTxRetry - failed to PutPolyTxRetry, not saved. Poly tx hash: %s", event.TxHash)
			return false
		}
	}
	return true
}

func newPolyMsgTx(height uint32, event *pcommon.SmartContactEvent, notify *pcommon.NotifyEventInfo) (*msg.Tx, error) {
	// if notify.ContractAddress == this.config.PolyConfig.EntranceContractAddress {
	// 	return nil, fmt.Errorf("notify.ContractAddress == this.config.PolyConfig.EntranceContractAddress")
	// }
	states := notify.States.([]interface{})
	if len(states) < 6 {
		return nil, fmt.Errorf("len(states) < 6")
	}
	method, _ := states[0].(string) // ignore is safe
	if method != "makeProof" {
		return nil, fmt.Errorf("method != \"makeProof\"")
	}

	dstChain := uint64(states[2].(float64))
	if dstChain == 0 {
		return nil, fmt.Errorf("Invalid dst chain id in poly tx, hash: %s", event.TxHash)
	}

	tx := new(msg.Tx)
	tx.DstChainId = dstChain
	tx.PolyKey = states[5].(string)
	tx.PolyHeight = uint32(height)
	tx.PolyHash = event.TxHash
	//tx.TxType = msg.POLY
	tx.TxId = states[3].(string)
	tx.SrcChainId = uint64(states[1].(float64))
	switch tx.SrcChainId {
	case base.NEO, base.NEO3, base.ONT:
		tx.TxId = util.ReverseHex(tx.TxId)
	}

	return tx, nil
}

func (this *PolyManager) IsEpoch(hdr *polytypes.Header) (bool, []byte, error) {
	blkInfo := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(hdr.ConsensusPayload, blkInfo); err != nil {
		return false, nil, fmt.Errorf("IsEpoch - unmarshal blockInfo error: %s", err.Error())
	}
	if hdr.NextBookkeeper == common.ADDRESS_EMPTY || blkInfo.NewChainConfig == nil {
		return false, nil, nil
	}

	//eccdAddr := ethcommon.HexToAddress(this.config.StarcoinConfig.CCDContractAddress)
	//eccd, err := eccd_abi.NewEthCrossChainData(eccdAddr, this.ethClient)
	ccd, err := NewCrossChainData(this.starcoinClient, this.config.StarcoinConfig.CCDModule)
	if err != nil {
		return false, nil, fmt.Errorf("failed to new CCD: %v", err)
	}
	rawKeepers, err := ccd.getCurEpochConPubKeyBytes()
	if err != nil {
		return false, nil, fmt.Errorf("failed to get current epoch keepers: %v", err)
	}

	bookkeepers := readBookKeeperPublicKeys(blkInfo)
	publickeys := make([]byte, 0)
	sink := common.NewZeroCopySink(nil)
	sink.WriteUint64(uint64(len(bookkeepers)))
	for _, key := range bookkeepers {
		raw := tools.GetNoCompresskey(key)
		publickeys = append(publickeys, raw...)
		sink.WriteVarBytes(crypto.Keccak256(tools.GetEthNoCompressKey(key)[1:])[12:])
	}
	if bytes.Equal(rawKeepers, sink.Bytes()) {
		return false, nil, nil
	}
	return true, publickeys, nil
}

func (this *PolyManager) InitStarcoinGenesis(privateKeyConfig map[string]string, height *uint32) error {
	var (
		cfgBlockNum uint32
		err         error
	)
	if height == nil {
		cfgBlockNum, err = this.getPolyLastConfigBlockNum()
	} else {
		cfgBlockNum, err = this.getPolyLastConfigBlockNumAtHeight(*height)
	}
	if err != nil {
		log.Errorf("InitGenesis - getPolyLastConfigBlockNum error")
		return err
	}
	hdr, err := this.polySdk.GetHeaderByHeight(cfgBlockNum)
	if err != nil {
		return err
	}
	publickeys, err := readBookKeeperPublicKeyBytes(hdr)
	if err != nil {
		log.Errorf("InitGenesis - readBookKeeperPublicKeyBytes error")
		return err
	}
	//fmt.Println(publickeys)
	//privateKeyConfig := this.config.StarcoinConfig.PrivateKeys[0]
	senderAddress, senderPrivateKey, err := getAccountAddressAndPrivateKey(privateKeyConfig)
	if err != nil {
		log.Errorf("InitGenesis - Convert string to AccountAddress error:%s", err.Error())
		return err
	}
	seqNum, err := this.starcoinClient.GetAccountSequenceNumber(context.Background(), tools.EncodeToHex(senderAddress[:]))
	if err != nil {
		log.Errorf("InitGenesis - GetAccountSequenceNumber error:%s", err.Error())
		return err
	}
	gasPrice, err := this.starcoinClient.GetGasUnitPrice(context.Background())
	if err != nil {
		log.Errorf("InitGenesis - GetAccountSequenceNumber error:%s", err.Error())
		return err
	}
	rawHdr := hdr.GetMessage()
	// fmt.Println("---------------------- raw_header ----------------------")
	// fmt.Println(hex.EncodeToString(rawHdr))
	// fmt.Println("------------------ public_keys -------------------")
	// fmt.Println(hex.EncodeToString(publickeys))
	// fmt.Println("--------------------------------------------------")
	txPayload := stcpoly.EncodeInitGenesisTxPayload(this.config.StarcoinConfig.CCMModule, rawHdr, publickeys)

	userTx, err := this.starcoinClient.BuildRawUserTransaction(context.Background(), *senderAddress, txPayload, gasPrice, stcclient.DEFAULT_MAX_GAS_AMOUNT*4, seqNum)
	if err != nil {
		log.Errorf("InitGenesis - BuildRawUserTransaction error:%s", err.Error())
		return err
	}
	txHash, err := this.starcoinClient.SubmitTransaction(context.Background(), senderPrivateKey, userTx)
	if err != nil {
		log.Errorf("InitGenesis - SubmitTransaction error:%s", err.Error())
		fmt.Printf("InitGenesis - SubmitTransaction error:%s\n", err.Error())
		return err
	}
	log.Debugf("InitGenesis - SubmitTransaction, get hash: %s", txHash)
	fmt.Printf("InitGenesis - SubmitTransaction, get hash: %s\n", txHash)
	// wait transaction confirmed?
	ok, err := tools.WaitTransactionConfirm(*this.starcoinClient, txHash, WAIT_STARCOIN_TRANSACTION_CONFIRM_TIME)
	if err != nil {
		log.Errorf("InitGenesis - WaitTransactionConfirm(%s) error: %s", txHash, err.Error())
		return err
	} else if !ok {
		log.Errorf("InitGenesis - WaitTransactionConfirm failed(unknown error).")
		return fmt.Errorf("WaitTransactionConfirm failed(unknown error)")
	}
	err = this.db.UpdatePolyHeight(cfgBlockNum)
	if err != nil {
		log.Errorf("InitGenesis - UpdatePolyHeight error: %s", err.Error())
		return err
	}
	return nil
}

func (this *PolyManager) getPolyLastConfigBlockNum() (uint32, error) {
	height, err := this.polySdk.GetCurrentBlockHeight()
	if err != nil {
		return 0, err
	}
	return this.getPolyLastConfigBlockNumAtHeight(height)
}

func (this *PolyManager) getPolyLastConfigBlockNumAtHeight(height uint32) (uint32, error) {
	hdr, err := this.polySdk.GetHeaderByHeight(height)
	if err != nil {
		return 0, err
	}
	blkInfo := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(hdr.ConsensusPayload, blkInfo); err != nil {
		return 0, fmt.Errorf("readBookKeeperPublicKeyBytes - unmarshal blockInfo error: %s", err.Error())
	}
	fmt.Printf("---------------- LastConfigBlockNum at %d -----------------\n", height)
	fmt.Println(blkInfo.LastConfigBlockNum)
	return blkInfo.LastConfigBlockNum, nil
}

// Get current poly epoch start height on Starcoin. Return 0 if error.
func (this *PolyManager) findCurEpochStartHeight() uint32 {
	//ethcommon.HexToAddress(this.config.StarcoinConfig.ECCDContractAddress)
	//instance, err := eccd_abi.NewEthCrossChainData(address, this.ethClient)
	// if err != nil {
	// 	log.Errorf("findCurEpochStartHeight - new eth cross chain failed: %s", err.Error())
	// 	return 0
	// }
	instance, err := NewCrossChainData(this.starcoinClient, this.config.StarcoinConfig.CCDModule)
	if err != nil {
		log.Errorf("findCurEpochStartHeight - NewCrossChainData failed: %s", err.Error())
		return 0
	}
	height, err := instance.getCurEpochStartHeight()
	if err != nil {
		log.Errorf("findCurEpochStartHeight - GetCurEpochStartHeight failed: %s", err.Error())
		return 0
	}
	return uint32(height)
}

func (this *PolyManager) Stop() {
	this.exitChan <- 1
	close(this.exitChan)
	log.Infof("poly chain manager exit.")
}

func (this *PolyManager) selectSender() *StarcoinSender {
	sum := big.NewInt(0)
	balArr := make([]*big.Int, len(this.senders))
	for i, v := range this.senders {
	RETRY:
		bal, err := v.Balance()
		if err != nil {
			log.Errorf("failed to get balance for %s: %v", tools.EncodeToHex(v.acc.Address[:]), err)
			time.Sleep(time.Second)
			goto RETRY
		}
		sum.Add(sum, bal)
		balArr[i] = big.NewInt(sum.Int64())
	}
	sum.Rand(rand.New(rand.NewSource(time.Now().Unix())), sum)
	for i, v := range balArr {
		res := v.Cmp(sum)
		if res == 1 || res == 0 {
			return this.senders[i]
		}
	}
	return this.senders[0]
}

type StarcoinSender struct {
	starcoinClient *stcclient.StarcoinClient
	seqNumManager  *tools.SeqNumManager
	keyStore       *tools.StarcoinKeyStore
	acc            tools.StarcoinAccount
	config         *config.ServiceConfig
	cmap           map[string]chan *StarcoinTxInfo
	db             db.DB
}

// type EthSender struct {
// 	acc          accounts.Account
// 	keyStore     *tools.EthKeyStore
// 	cmap         map[string]chan *EthTxInfo
// 	nonceManager *tools.NonceManager
// 	ethClient    *ethclient.Client
// 	polySdk      *sdk.PolySdk
// 	config       *config.ServiceConfig
// 	contractAbi  *abi.ABI
// }

// return two bool value, first indicate if Starcoin transaction has been sent, second indicate if trasaction has been saved in DB
func (this *StarcoinSender) commitDepositEventsWithHeader(header *polytypes.Header, param *common2.ToMerkleValue, headerProof string, anchorHeader *polytypes.Header, polyTxHash string, rawAuditPath []byte) (bool, bool) {
	headerData, rawProof, rawAnchor, sigs, err := getRawHeaderAndHeaderProofAndSig(header, param, headerProof, anchorHeader)
	polyTx, err := db.NewPolyTx(param.TxHash, param.FromChainID, rawAuditPath, headerData, rawProof, rawAnchor, sigs, polyTxHash)
	if err != nil {
		log.Errorf("commitDepositEventsWithHeader - db.NewPolyTx error: %s", err.Error())
		return false, false
	}
	return this.putPolyTxThenSend(polyTx)
}

func (this *StarcoinSender) putPolyTxThenSend(polyTx *db.PolyTx) (bool, bool) {
	_, err := this.db.PutPolyTx(polyTx)
	if err != nil {
		log.Errorf("putPolyTxThenSend - db.PutPolyTx error: %s", err.Error())
		duplicate, checkErr := db.IsDuplicatePolyTxError(this.db, polyTx, err)
		if checkErr != nil {
			return false, false // not sent, treated as not saved
		}
		if duplicate {
			log.Warnf("putPolyTxThenSend - db.PutPolyTx found duplicate poly tx., FromChainID: %d, Txhash: %s", polyTx.FromChainID, polyTx.TxHash)
			return false, true // not sent, but treated as saved
		}
		return false, false
	}

	return this.sendPolyTxToStarcoin(polyTx), true
}

func polyTxRetryToPolyTx(r *db.PolyTxRetry) (*db.PolyTx, error) {
	bs, err := hex.DecodeString(r.BridgeTransaction)
	if err != nil {
		log.Errorf("polyTxRetryToPolyTx - hex.DecodeString(r.BridgeTransaction) error: %s", err.Error())
		return nil, err
	}
	bridgeTransaction := new(BridgeTransaction)
	err = bridgeTransaction.Deserialization(common.NewZeroCopySource(bs))
	if err != nil {
		log.Errorf("polyTxRetryToPolyTx - bridgeTransaction.Deserialization error: %s", err.Error())
		return nil, err
	}
	txHash, err := hex.DecodeString(r.TxHash)
	if err != nil {
		log.Errorf("polyTxRetryToPolyTx - hex.DecodeString(r.TxHash) error: %s", err.Error())
		return nil, err
	}
	e, err := r.GetPolyEvent()
	if err != nil {
		log.Errorf("polyTxRetryToPolyTx - GetPolyEvent error: %s", err.Error())
		return nil, err
	}
	headerData, rawProof, rawAnchor, sigs, err := getRawHeaderAndHeaderProofAndSig(bridgeTransaction.header, bridgeTransaction.param, bridgeTransaction.headerProof, bridgeTransaction.anchorHeader)
	polyTx, err := db.NewPolyTx(txHash, r.FromChainID, bridgeTransaction.rawAuditPath, headerData, rawProof, rawAnchor, sigs, e.PolyHash)
	if err != nil {
		log.Errorf("polyTxRetryToPolyTx - db.NewPolyTx error: %s", err.Error())
		return nil, err
	}
	return polyTx, nil
}

func getRawHeaderAndHeaderProofAndSig(header *polytypes.Header, param *common2.ToMerkleValue, headerProof string, anchorHeader *polytypes.Header) ([]byte, []byte, []byte, []byte, error) {
	var (
		sigs       []byte
		headerData []byte
	)
	if anchorHeader != nil && headerProof != "" {
		for _, sig := range anchorHeader.SigData {
			temp := make([]byte, len(sig))
			copy(temp, sig)
			newsig, err := signature.ConvertToEthCompatible(temp)
			if err != nil {
				log.Errorf("getRawHeaderAndHeaderProofAndSig - signature.ConvertToEthCompatible error: %v", err)
			}
			sigs = append(sigs, newsig...)
		}
	} else {
		for _, sig := range header.SigData {
			temp := make([]byte, len(sig))
			copy(temp, sig)
			newsig, err := signature.ConvertToEthCompatible(temp)
			if err != nil {
				log.Errorf("getRawHeaderAndHeaderProofAndSig - signature.ConvertToEthCompatible error: %v", err)
			}
			sigs = append(sigs, newsig...)
		}
	}

	// // ///////////////////////////////////
	// // eccdAddr := ethcommon.HexToAddress(this.config.ETHConfig.ECCDContractAddress)
	// // eccd, err := eccd_abi.NewEthCrossChainData(eccdAddr, this.ethClient)
	// ccd := NewCrossChainData(this.starcoinClient, this.config.StarcoinConfig.CCDModule)
	// // if err != nil {
	// // 	panic(fmt.Errorf("failed to new CCM: %v", err))
	// // }
	// fromTx := [32]byte{}
	// copy(fromTx[:], param.TxHash[:32])
	// res, _ := ccd.checkIfFromChainTxExist(param.FromChainID, fromTx[:])
	// if res {
	// 	log.Debugf("already relayed to starcoin: ( from_chain_id: %d, from_txhash: %x,  param.Txhash: %x)",
	// 		param.FromChainID, param.TxHash, param.MakeTxParam.TxHash)
	// 	return true
	// }
	// //log.Infof("poly proof with header, height: %d, key: %s, proof: %s", header.Height-1, string(key), proof.AuditPath)
	// // ///////////////////////////////////

	rawProof, err := hex.DecodeString(headerProof)
	if err != nil {
		log.Errorf("getRawHeaderAndHeaderProofAndSig - hex.DecodeString(headerProof) error: %v", err)
	}
	var rawAnchor []byte
	if anchorHeader != nil {
		rawAnchor = anchorHeader.GetMessage()
	}
	headerData = header.GetMessage()

	// Solidity code:
	//
	// function verifyHeaderAndExecuteTx(
	// 	bytes memory proof,
	// 	bytes memory rawHeader,
	// 	bytes memory headerProof,
	// 	bytes memory curRawHeader,
	// 	bytes memory headerSig //The coverted signature veriable for solidity derived from Poly chain consensus nodes' signature
	// ) whenNotPaused public returns (bool){

	// txData, err := this.contractAbi.Pack("verifyHeaderAndExecuteTx",
	// 	rawAuditPath, // Poly chain tx merkle proof
	// 	headerData,   // The header containing crossStateRoot to verify the above tx merkle proof
	// 	rawProof,     // The header merkle proof used to verify rawHeader
	// 	rawAnchor,    // Any header in current epoch consensus of Poly chain
	// 	sigs)

	return headerData, rawProof, rawAnchor, sigs, nil
}

func (this *StarcoinSender) sendPolyTxToStarcoin(polyTx *db.PolyTx) bool {
	// ////////////////////////////////////////////////////////////
	// Update PolyTx status to processing(sending to Starcoin),
	// set transactio hash to empty first.
	err := this.db.SetPolyTxStatusProcessing(polyTx.TxHash, polyTx.FromChainID, polyTx.Status)
	if err != nil {
		log.Errorf("failed to SetPolyTxStatusProcessing. Error: %v, txIndex: %d", err, polyTx.TxIndex)
		return false
	}
	polyTx.Status = db.STATUS_PROCESSING
	// /////////////////////////////////////////////////////////////
	if polyTx.SmtNonMembershipRootHash == "" {
		err = this.db.UpdatePolyTxNonMembershipProofByIndex(polyTx.TxIndex)
		if err != nil {
			log.Errorf("failed to db.UpdatePolyTxNonMembershipProofByIndex. Error: %v, txIndex: %d", err, polyTx.TxIndex)
			return false
		}
		polyTx, err = this.db.GetPolyTx(polyTx.TxHash, polyTx.FromChainID)
		if err != nil {
			log.Errorf("failed to re-get PolyTx, error: %v, txIndex: %d", err, polyTx.TxIndex)
			return false
		}
	}
	// ///////////////////////////////////////////////////////////
	stcTxInfo, err := this.polyTxToStarcoinTxInfo(polyTx)
	if err != nil {
		return false
	}
	// contractaddr := ethcommon.HexToAddress(this.config.ETHConfig.ECCMContractAddress)
	// callMsg := ethereum.CallMsg{
	// 	From: this.acc.Address, To: &contractaddr, Gas: 0, GasPrice: gasPrice,
	// 	Value: big.NewInt(0), Data: txData,
	// }
	// this.starcoinClient.BuildRawUserTransaction(context.Background(), this.acc.Address, txPayload, gasPrice, stcclient.DEFAULT_MAX_GAS_AMOUNT)
	// gasLimit, err := this.starcoinClient.EstimateGasByDryRunRaw(context.Background(), callMsg)
	// if err != nil {
	// 	log.Errorf("commitDepositEventsWithHeader - estimate gas limit error: %s", err.Error())
	// 	return false
	// }

	k := this.getRouter()
	// One sender is responsible for multi-router(channel),
	// create channel if not exists.
	c, ok := this.cmap[k]
	if !ok {
		c = make(chan *StarcoinTxInfo, ChanLen)
		this.cmap[k] = c
		go func() {
			for v := range c {
				if err := this.sendTxToStarcoin(v); err != nil {
					txBytes, seErr := v.txPayload.BcsSerialize()
					_ = seErr // ignore is safe
					log.Errorf("failed to send tx to starcoin: error: %v, txData: %s", err, hex.EncodeToString(txBytes))
				}
			}
		}()
	}
	// TODO:: could be blocked
	c <- stcTxInfo // put Tx. into channel
	return true
}

func (this *StarcoinSender) polyTxToStarcoinTxInfo(polyTx *db.PolyTx) (*StarcoinTxInfo, error) {
	//polyTxHash := polyTx.TxHash
	p, err := polyTx.GetPolyTxProof()
	if err != nil {
		return nil, err
	}
	rawAuditPath, err := hex.DecodeString(p.Proof)
	if err != nil {
		log.Errorf("polyTxToStarcoinTxInfo - hex.DecodeString error: %s", err.Error())
		return nil, err
	}
	headerData, err := hex.DecodeString(p.Header)
	if err != nil {
		log.Errorf("polyTxToStarcoinTxInfo - hex.DecodeString error: %s", err.Error())
		return nil, err
	}
	rawProof, err := hex.DecodeString(p.HeaderProof)
	if err != nil {
		log.Errorf("polyTxToStarcoinTxInfo - hex.DecodeString error: %s", err.Error())
		return nil, err
	}
	rawAnchor, err := hex.DecodeString(p.AnchorHeader)
	if err != nil {
		log.Errorf("polyTxToStarcoinTxInfo - hex.DecodeString error: %s", err.Error())
		return nil, err
	}
	sigs, err := hex.DecodeString(p.HeaderSig)
	if err != nil {
		log.Errorf("polyTxToStarcoinTxInfo - hex.DecodeString error: %s", err.Error())
		return nil, err
	}

	leafData, err := polyTx.GetSmtProofNonMembershipLeafData()
	if err != nil {
		log.Errorf("polyTxToStarcoinTxInfo - GetSmtProofNonMembershipLeafData error: %s", err.Error())
		return nil, err
	}
	sideNodes, err := polyTx.GetSmtProofSideNodes()
	if err != nil {
		log.Errorf("polyTxToStarcoinTxInfo - GetSmtProofSideNodes error: %s", err.Error())
		return nil, err
	}
	rootHash, err := polyTx.GetSmtNonMembershipRootHash()
	if err != nil {
		log.Errorf("polyTxToStarcoinTxInfo - GetSmtNonMembershipRootHash error: %s", err.Error())
		return nil, err
	}

	txPayload := stcpoly.EncodeCCMVerifyHeaderAndExecuteTxPayload(this.config.StarcoinConfig.CCMModule,
		rawAuditPath,
		headerData,
		rawProof,
		rawAnchor,
		sigs,
		rootHash,
		leafData,
		concatByteSlices(sideNodes),
	)

	return &StarcoinTxInfo{
		txPayload: txPayload,
		//contractAddr: contractaddr,
		//gasPrice:   gasPrice,
		//gasLimit:   stcclient.DEFAULT_MAX_GAS_AMOUNT, //gasLimit,
		polyTxHash:      polyTx.TxHash,
		polyFromChainID: polyTx.FromChainID,
	}, nil
}

func concatByteSlices(ss [][]byte) []byte {
	r := make([]byte, 0, len(ss)*32)
	for _, s := range ss {
		r = append(r, s...)
	}
	return r
}

// commitHeader
func (this *StarcoinSender) changeBookKeeper(header *polytypes.Header, pubkList []byte) bool {
	headerdata := header.GetMessage()
	//var (
	// txData []byte
	// txErr  error
	//sigs []byte
	//)
	sigs := encodeHeaderSigData(header)

	gasPrice, err := this.starcoinClient.GetGasUnitPrice(context.Background()) //this.ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		log.Errorf("changeBookKeeper - get suggest sas price failed error: %s", err.Error())
		return false
	}

	// txData, txErr = this.contractAbi.Pack("changeBookKeeper", headerdata, pubkList, sigs)
	// if txErr != nil {
	// 	log.Errorf("changeBookKeeper - err:" + err.Error())
	// 	return false
	// }
	// contractaddr := ethcommon.HexToAddress(this.config.StarcoinConfig.CCMContractAddress)
	// callMsg := ethereum.CallMsg{
	// 	From: this.acc.Address, To: &contractaddr, Gas: 0, GasPrice: gasPrice,
	// 	Value: big.NewInt(0), Data: txData,
	// }
	// gasLimit, err := this.ethClient.EstimateGas(context.Background(), callMsg) // TODO: gasLimit...
	// if err != nil {
	// 	log.Errorf("changeBookKeeper - estimate gas limit error: %s", err.Error())
	// 	return false
	// }
	txPayload := stcpoly.EncodeCCMChangeBookKeeperTxPayload(this.config.StarcoinConfig.CCMModule, headerdata, pubkList, sigs)

	nonce := this.seqNumManager.GetAccountSeqNum(this.acc.Address)
	// tx := types.NewTransaction(nonce, contractaddr, big.NewInt(0), gasLimit, gasPrice, txData)
	// signedtx, err := this.keyStore.SignTransaction(tx, this.acc)

	// if err != nil {
	// 	log.Errorf("changeBookKeeper - sign raw tx error: %s", err.Error())
	// 	return false
	// }

	rawUserTx, err := this.starcoinClient.BuildRawUserTransaction(context.Background(), this.acc.Address, txPayload, gasPrice, this.config.StarcoinConfig.MaxGasAmount, nonce)
	// TODO: use max gas???
	if err != nil {
		log.Errorf("changeBookKeeper - BuildRawUserTransaction error: %s", err.Error())
		return false
	}
	var txhash string
	//txhash := signedtx.Hash() // TODO: cal txhash self???
	if txhash, err = this.starcoinClient.SubmitTransaction(context.Background(), this.keyStore.GetPrivateKey(), rawUserTx); err != nil {
		log.Errorf("changeBookKeeper - send transaction error:%s\n", err.Error())
		return false
	}

	hdrhash := header.Hash()
	isSuccess, err := tools.WaitTransactionConfirm(*this.starcoinClient, txhash, WAIT_STARCOIN_TRANSACTION_CONFIRM_TIME)
	if isSuccess {
		log.Infof("successful to relay poly header to starcoin: (header_hash: %s, height: %d, starcoin_txhash: %s, nonce: %d, starcoin_explorer: %s)",
			hdrhash.ToHexString(), header.Height, txhash, nonce, tools.GetExplorerUrl(this.keyStore.GetChainId())+txhash)
	} else {
		if err == nil {
			log.Infof("failed to relay poly header to starcoin, error is nil.  Maybe timed out or cannot get transaction info.")
		}
		log.Errorf("failed to relay poly header to starcoin: (header_hash: %s, height: %d, starcoin_txhash: %s, nonce: %d, starcoin_explorer: %s), error: %v",
			hdrhash.ToHexString(), header.Height, txhash, nonce, tools.GetExplorerUrl(this.keyStore.GetChainId())+txhash, err)
	}
	return true
}

func (this *StarcoinSender) sendTxToStarcoin(txInfo *StarcoinTxInfo) error {
	nonce := this.seqNumManager.GetAccountSeqNum(this.acc.Address)
	// tx := types.NewTransaction(nonce, txInfo.contractAddr, big.NewInt(0), txInfo.gasLimit, txInfo.gasPrice, txInfo.txData)
	// signedtx, err := this.keyStore.SignTransaction(tx, this.acc)
	// if err != nil {
	// 	this.seqNumManager.ReturnSeqNum(this.acc.Address, nonce)
	// 	return fmt.Errorf("commitDepositEventsWithHeader - sign raw tx error and return nonce %d: %v", nonce, err)
	// }
	// err = this.starcoinClient.SendTransaction(context.Background(), signedtx)
	// if err != nil {
	// 	this.seqNumManager.ReturnSeqNum(this.acc.Address, nonce)
	// 	return fmt.Errorf("commitDepositEventsWithHeader - send transaction error and return nonce %d: %v", nonce, err)
	// }
	// hash := signedtx.Hash()

	gasPrice, err := this.starcoinClient.GetGasUnitPrice(context.Background())
	if err != nil {
		log.Errorf("commitDepositEventsWithHeader - get suggest sas price failed error: %s", err.Error())
		return err
	}

	rawUserTx, err := this.starcoinClient.BuildRawUserTransaction(context.Background(), this.acc.Address,
		txInfo.txPayload, gasPrice, this.config.StarcoinConfig.MaxGasAmount, nonce)
	// TODO: use max gas???
	if err != nil {
		log.Errorf("sendTxToStarcoin - BuildRawUserTransaction error: %s", err.Error())
		return err
	}
	var txhash string
	if txhash, err = this.starcoinClient.SubmitTransaction(context.Background(), this.keyStore.GetPrivateKey(), rawUserTx); err != nil {
		log.Errorf("sendTxToStarcoin - submit transaction error:%s\n", err.Error())
		return err
	}
	// TODO: cal txhash self???

	// ///////////// update Starcoin transaction hash in DB ///////////////
	dbErr := this.db.SetProcessingPolyTxStarcoinTxHash(txInfo.polyTxHash, txInfo.polyFromChainID, txhash)
	if dbErr != nil {
		log.Errorf("failed to SetProcessingPolyTxStarcoinTxHash. Error: %v, polyTxHash: %s", dbErr, txInfo.polyTxHash)
		return err
	}
	// now PolyTx status is PROCESSING
	// /////////////////////////////////////////////

	//isSuccess := this.waitTransactionConfirm(txInfo.polyTxHash, hash)
	isSuccess, err := tools.WaitTransactionConfirm(*this.starcoinClient, txhash, WAIT_STARCOIN_TRANSACTION_CONFIRM_TIME)
	if isSuccess {
		log.Infof("successful to relay tx to starcoin: (starcoin_hash: %s, nonce: %d, poly_hash: %s, starcoin_explorer: %s)",
			txhash, nonce, txInfo.polyTxHash, tools.GetExplorerUrl(this.keyStore.GetChainId())+txhash)
		this.db.SetPolyTxStatusProcessed(txInfo.polyTxHash, txInfo.polyFromChainID, db.STATUS_PROCESSING, txhash)
	} else {
		if err == nil {
			log.Infof("failed to relay tx to starcoin, error is nil. Maybe timed out or cannot get transaction info.")
			dbErr := this.db.SetPolyTxStatus(txInfo.polyTxHash, txInfo.polyFromChainID, db.STATUS_PROCESSING, db.STATUS_TIMEDOUT) // set relay-to-starcoin status to TIMED-OUT!
			if dbErr != nil {
				log.Errorf("failed to SetPolyTxStatus to timed-out. Error: %v, polyTxHash: %s", dbErr, txInfo.polyTxHash)
				//return dbErr
			}
		} else {
			dbErr := this.db.SetPolyTxStatus(txInfo.polyTxHash, txInfo.polyFromChainID, db.STATUS_PROCESSING, db.STATUS_FAILED) // set relay-to-starcoin status to FAILED.
			if dbErr != nil {
				log.Errorf("failed to SetPolyTxStatus to failed. Error: %v, polyTxHash: %s", dbErr, txInfo.polyTxHash)
				//return dbErr
			}
		}
		log.Errorf("failed to relay tx to starcoin: (starcoin_hash: %s, nonce: %d, poly_hash: %s, starcoin_explorer: %s), error: %v",
			txhash, nonce, txInfo.polyTxHash, tools.GetExplorerUrl(this.keyStore.GetChainId())+txhash, err)

		return err // this err maybe nil
	}
	return nil
}

func (this *StarcoinSender) getRouter() string {
	return strconv.FormatInt(rand.Int63n(this.config.RoutineNum), 10)
}

func (this *StarcoinSender) Balance() (*big.Int, error) {
	//balance, err := this.ethClient.BalanceAt(context.Background(), this.acc.Address, nil)
	balance, err := this.starcoinClient.GetBalanceOfStc(context.Background(), tools.EncodeToHex(this.acc.Address[:]))
	if err != nil {
		return nil, err
	}
	return balance, nil
}

type StarcoinTxInfo struct {
	//txData   []byte
	txPayload diemtypes.TransactionPayload
	//gasLimit  uint64
	//gasPrice int
	//contractAddr ethcommon.Address
	polyTxHash      string
	polyFromChainID uint64
}

func readBookKeeperPublicKeyBytes(hdr *polytypes.Header) ([]byte, error) {
	blkInfo := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(hdr.ConsensusPayload, blkInfo); err != nil {
		return nil, fmt.Errorf("readBookKeeperPublicKeyBytes - unmarshal blockInfo error: %s", err.Error())
	}
	if hdr.NextBookkeeper == common.ADDRESS_EMPTY || blkInfo.NewChainConfig == nil {
		return nil, fmt.Errorf("readBookKeeperPublicKeyBytes - blkInfo.NewChainConfig == nil")
	}
	bookkeepers := readBookKeeperPublicKeys(blkInfo)
	publickeys := make([]byte, 0)
	for _, key := range bookkeepers {
		raw := tools.GetNoCompresskey(key)
		publickeys = append(publickeys, raw...)
	}
	return publickeys, nil
}

func readBookKeeperPublicKeys(blkInfo *vconfig.VbftBlockInfo) []keypair.PublicKey {
	var bookkeepers []keypair.PublicKey
	for _, peer := range blkInfo.NewChainConfig.Peers {
		keystr, err := hex.DecodeString(peer.ID)
		if err != nil {
			log.Errorf("readBookKeeperPublicKeys - hex.DecodeString(peer.ID) error: %v", err)
		}
		key, err := keypair.DeserializePublicKey(keystr)
		if err != nil {
			log.Errorf("readBookKeeperPublicKeys - keypair.DeserializePublicKey(keystr) error: %v", err)
		}
		bookkeepers = append(bookkeepers, key)
	}
	bookkeepers = keypair.SortPublicKeys(bookkeepers)
	return bookkeepers
}

func encodeHeaderSigData(header *polytypes.Header) []byte {
	var sigs []byte
	for _, sig := range header.SigData {
		temp := make([]byte, len(sig))
		copy(temp, sig)
		newsig, err := signature.ConvertToEthCompatible(temp)
		if err != nil {
			log.Errorf("encodeHeaderSigData - signature.ConvertToEthCompatible error: %v", err)
		}
		sigs = append(sigs, newsig...)
	}
	return sigs
}

// func (this *StarcoinSender) waitTransactionConfirm(polyTxHash string, hash ethcommon.Hash) bool {
// 	for {
// 		time.Sleep(time.Second * 1)
// 		_, ispending, err := this.ethClient.TransactionByHash(context.Background(), hash)
// 		if err != nil {
// 			continue
// 		}
// 		log.Debugf("( starcoin_transaction %s, poly_tx %s ) is pending: %v", hash.String(), polyTxHash, ispending)
// 		if ispending == true {
// 			continue
// 		} else {
// 			receipt, err := this.ethClient.TransactionReceipt(context.Background(), hash)
// 			if err != nil {
// 				continue
// 			}
// 			return receipt.Status == types.ReceiptStatusSuccessful
// 		}
// 	}
// }

// ///////////////// BridgeTransaction //////////////////

type BridgeTransaction struct {
	header       *polytypes.Header
	param        *common2.ToMerkleValue
	headerProof  string
	anchorHeader *polytypes.Header
	//polyTxHash   string
	rawAuditPath []byte
	//hasPay       uint8
	//fee          string
}

// func (this *BridgeTransaction) PolyHash() string {
// 	return this.polyTxHash
// }

func (this *BridgeTransaction) Serialization(sink *common.ZeroCopySink) {
	this.header.Serialization(sink)
	this.param.Serialization(sink)
	if this.headerProof != "" && this.anchorHeader != nil {
		sink.WriteUint8(1)
		sink.WriteString(this.headerProof)
		this.anchorHeader.Serialization(sink)
	} else {
		sink.WriteUint8(0)
	}
	//sink.WriteString(this.polyTxHash)
	sink.WriteVarBytes(this.rawAuditPath)
	//sink.WriteUint8(this.hasPay)
	//sink.WriteString(this.fee)
}

func (this *BridgeTransaction) Deserialization(source *common.ZeroCopySource) error {
	this.header = new(polytypes.Header)
	err := this.header.Deserialization(source)
	if err != nil {
		return err
	}
	this.param = new(common2.ToMerkleValue)
	err = this.param.Deserialization(source)
	if err != nil {
		return err
	}
	anchor, eof := source.NextUint8()
	if eof {
		return fmt.Errorf("Waiting deserialize anchor error")
	}
	if anchor == 1 {
		this.headerProof, eof = source.NextString()
		if eof {
			return fmt.Errorf("Waiting deserialize header proof error")
		}
		this.anchorHeader = new(polytypes.Header)
		this.anchorHeader.Deserialization(source)
	}
	// this.polyTxHash, eof = source.NextString()
	// if eof {
	// 	return fmt.Errorf("Waiting deserialize poly tx hash error")
	// }
	this.rawAuditPath, eof = source.NextVarBytes()
	if eof {
		return fmt.Errorf("Waiting deserialize poly tx hash error")
	}
	// this.hasPay, eof = source.NextUint8()
	// if eof {
	// 	return fmt.Errorf("Waiting deserialize has pay error")
	// }
	// this.fee, eof = source.NextString()
	// if eof {
	// 	return fmt.Errorf("Waiting deserialize fee error")
	// }
	return nil
}
