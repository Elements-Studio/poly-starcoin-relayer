package manager

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/log"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	"github.com/ethereum/go-ethereum/common/hexutil"

	//"github.com/ethereum/go-ethereum/accounts/abi/bind" //todo remove this
	"github.com/ontio/ontology/smartcontract/service/native/cross_chain/cross_chain_manager"
	//"github.com/polynetwork/bridge-common/abi/eccm_abi" //todo remove this

	evttypes "github.com/elements-studio/poly-starcoin-relayer/bifrost/types"
	polysdk "github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly/common"
	common2 "github.com/polynetwork/poly/native/service/cross_chain_manager/common"
	"github.com/polynetwork/poly/native/service/cross_chain_manager/eth"
	scom "github.com/polynetwork/poly/native/service/header_sync/common"
	autils "github.com/polynetwork/poly/native/service/utils"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

type StarcoinManager struct {
	client        stcclient.StarcoinClient
	polySdk       polysdk.PolySdk
	polySigner    *polysdk.Account
	config        config.ServiceConfig
	header4sync   [][]byte
	currentHeight uint64
	forceHeight   uint64
	restClient    tools.RestClient
	exitChan      chan int
}

func (this *StarcoinManager) init() error {
	// get latest height
	latestHeight := this.findSyncedHeight()
	if latestHeight == 0 {
		return fmt.Errorf("init - the genesis block has not synced!")
	}
	if this.forceHeight > 0 && this.forceHeight < latestHeight {
		this.currentHeight = this.forceHeight
	} else {
		this.currentHeight = latestHeight
	}
	log.Infof("EthereumManager init - start height: %d", this.currentHeight)
	return nil
}

func (this *StarcoinManager) MonitorChain() {
	fetchBlockTicker := time.NewTicker(time.Duration(this.config.ETHConfig.MonitorInterval) * time.Second)
	var blockHandleResult bool
	for {
		select {
		case <-fetchBlockTicker.C:
			height, err := tools.GetStarcoinNodeHeight(this.config.StarcoinConfig.RestURL, this.restClient)
			if err != nil {
				log.Infof("StarcoinManager.MonitorChain - cannot get node height, err: %s", err)
				continue
			}
			if height-this.currentHeight <= config.ETH_USEFUL_BLOCK_NUM {
				continue
			}
			log.Infof("StarcoinManager.MonitorChain - eth height is %d", height)
			blockHandleResult = true
			for this.currentHeight < height-config.ETH_USEFUL_BLOCK_NUM {
				if this.currentHeight%10 == 0 {
					log.Infof("handle confirmed eth Block height: %d", this.currentHeight)
				}
				blockHandleResult = this.handleNewBlock(this.currentHeight + 1)
				if blockHandleResult == false {
					break
				}
				this.currentHeight++
				// try to commit header if more than 50 headers needed to be syned
				if len(this.header4sync) >= this.config.StarcoinConfig.HeadersPerBatch {
					if res := this.commitHeader(); res != 0 {
						blockHandleResult = false
						break
					}
				}
			}
			if blockHandleResult && len(this.header4sync) > 0 {
				this.commitHeader()
			}
		case <-this.exitChan:
			return
		}
	}
}

func (this *StarcoinManager) commitHeader() int {
	tx, err := this.polySdk.Native.Hs.SyncBlockHeader(
		this.config.ETHConfig.SideChainId,
		this.polySigner.Address,
		this.header4sync,
		this.polySigner,
	)
	if err != nil {
		errDesc := err.Error()
		if strings.Contains(errDesc, "get the parent block failed") || strings.Contains(errDesc, "missing required field") {
			log.Warnf("StarcoinManager.commitHeader - send transaction to poly chain err: %s", errDesc)
			this.rollBackToCommAncestor()
			return 0
		} else {
			log.Errorf("StarcoinManager.commitHeader - send transaction to poly chain err: %s", errDesc)
			return 1
		}
	}
	tick := time.NewTicker(100 * time.Millisecond)
	var h uint32
	for range tick.C {
		h, _ = this.polySdk.GetBlockHeightByTxHash(tx.ToHexString())
		curr, _ := this.polySdk.GetCurrentBlockHeight()
		if h > 0 && curr > h {
			break
		}
	}
	log.Infof("commitHeader - send transaction %s to poly chain and confirmed on height %d", tx.ToHexString(), h)
	this.header4sync = make([][]byte, 0)
	return 0
}

func (this *StarcoinManager) rollBackToCommAncestor() {
	for ; ; this.currentHeight-- {
		raw, err := this.polySdk.GetStorage(autils.HeaderSyncContractAddress.ToHexString(),
			append(append([]byte(scom.MAIN_CHAIN), autils.GetUint64Bytes(this.config.ETHConfig.SideChainId)...), autils.GetUint64Bytes(this.currentHeight)...))
		if len(raw) == 0 || err != nil {
			continue
		}
		hdr, err := this.client.HeaderByNumber(context.Background(), big.NewInt(int64(this.currentHeight))) //todo starcoin get headerByNumber method...
		if err != nil {
			log.Errorf("rollBackToCommAncestor - failed to get header by number, so we wait for one second to retry: %v", err)
			time.Sleep(time.Second)
			this.currentHeight++
		}
		if bytes.Equal(hdr.Hash().Bytes(), raw) {
			log.Infof("rollBackToCommAncestor - find the common ancestor: %s(number: %d)", hdr.Hash().String(), this.currentHeight)
			break
		}
	}
	this.header4sync = make([][]byte, 0)
}

func (this *StarcoinManager) handleNewBlock(height uint64) bool {
	ret := this.handleBlockHeader(height)
	if !ret {
		log.Errorf("StarcoinManager.handleNewBlock - handleBlockHeader on height :%d failed", height)
		return false
	}
	ret = this.fetchLockDepositEvents(height, this.client)
	if !ret {
		log.Errorf("StarcoinManager.handleNewBlock - fetchLockDepositEvents on height :%d failed", height)
	}
	return true
}

func (this *StarcoinManager) handleBlockHeader(height uint64) bool {
	// hdr, err := this.client.HeaderByNumber(context.Background(), big.NewInt(int64(height)))
	// if err != nil {
	// 	log.Errorf("handleBlockHeader - GetNodeHeader on height :%d failed", height)
	// 	return false
	// }
	// rawHdr, _ := hdr.MarshalJSON()
	block, err := this.client.GetBlockByNumber(int(height))
	if err != nil {
		log.Errorf("handleBlockHeader - GetBlockByNumber on height :%d failed", height)
		return false
	}
	hdr := block.BlockHeader
	rawHdr, _ := json.Marshal(hdr)
	var hdrHashBytes []byte //todo hdr.Hash().Bytes()
	raw, _ := this.polySdk.GetStorage(autils.HeaderSyncContractAddress.ToHexString(),
		append(append([]byte(scom.MAIN_CHAIN), autils.GetUint64Bytes(this.config.StarcoinConfig.SideChainId)...), autils.GetUint64Bytes(height)...))
	if len(raw) == 0 || !bytes.Equal(raw, hdrHashBytes) {
		this.header4sync = append(this.header4sync, rawHdr)
	}
	return true
}

func (this *StarcoinManager) fetchLockDepositEvents(height uint64, client *stcclient.StarcoinClient) bool {
	// lockAddress := ethcommon.HexToAddress(this.config.StarcoinConfig.ECCMContractAddress)
	// lockContract, err := eccm_abi.NewEthCrossChainManager(lockAddress, client)
	// if err != nil {
	// 	return false
	// }
	// opt := &bind.FilterOpts{
	// 	Start:   height,
	// 	End:     &height,
	// 	Context: context.Background(),
	// }
	eventFilter := &stcclient.EventFilter{
		Address:   this.config.StarcoinConfig.ECCMContractAddress, //todo filter this address???
		FromBlock: height,
		ToBlock:   height,
	}

	events, err := this.client.GetEvents(eventFilter)
	//events, err := lockContract.FilterCrossChainEvent(opt, nil) // todo get events from starcoin
	if err != nil {
		log.Errorf("fetchLockDepositEvents - FilterCrossChainEvent error :%s", err.Error())
		return false
	}
	if events == nil {
		log.Infof("fetchLockDepositEvents - no events found on FilterCrossChainEvent")
		return false
	}

	for _, evt := range events {
		//evt := events.Event
		evtData, err := hex.DecodeString(evt.Data)
		if err != nil {
			log.Errorf("fetchLockDepositEvents - hex.DecodeString error :%s", err.Error())
			return false
		}
		ccDepositEvt, err := evttypes.BcsDeserializeCrossChainDepositEvent(evtData)
		if err != nil {
			log.Errorf("fetchLockDepositEvents - BcsDeserializeCrossChainDepositEvent error :%s", err.Error())
			return false
		}
		var isTarget bool
		if len(this.config.TargetContracts) > 0 {
			var toContractStr string //todo toContractStr := evt.ProxyOrAssetContract.String()
			for _, v := range this.config.TargetContracts {
				toChainIdArr, ok := v[toContractStr]
				if ok {
					if len(toChainIdArr["outbound"]) == 0 {
						isTarget = true
						break
					}
					for _, id := range toChainIdArr["outbound"] {
						if id == ccDepositEvt.ToChainId { // todo is this OK??
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
				continue
			}
		}
		param := &common2.MakeTxParam{}
		_ = param.Deserialization(common.NewZeroCopySource([]byte(evt.Rawdata)))
		raw, _ := this.polySdk.GetStorage(autils.CrossChainManagerContractAddress.ToHexString(),
			append(append([]byte(cross_chain_manager.DONE_TX), autils.GetUint64Bytes(this.config.StarcoinConfig.SideChainId)...), param.CrossChainID...))
		if len(raw) != 0 {
			log.Debugf("fetchLockDepositEvents - ccid %s (tx_hash: %s) already on poly",
				hex.EncodeToString(param.CrossChainID), evt.Raw.TxHash.Hex())
			continue
		}
		index := big.NewInt(0)
		index.SetBytes(evt.TxId)
		crossTx := &CrossTransfer{
			txIndex: tools.EncodeBigInt(index),
			txId:    evt.Raw.TxHash.Bytes(),
			toChain: uint32(evt.ToChainId),
			value:   []byte(evt.Rawdata),
			height:  height,
		}
		sink := common.NewZeroCopySink(nil)
		crossTx.Serialization(sink)
		err = this.db.PutStarcoinTxRetry(sink.Bytes()) //todo db...
		if err != nil {
			log.Errorf("fetchLockDepositEvents - this.db.PutStarcoinTxRetry error: %s", err)
		}
		log.Infof("fetchLockDepositEvent -  height: %d", height)
	}
	return true
}

func (this *StarcoinManager) MonitorDeposit() {
	monitorTicker := time.NewTicker(time.Duration(this.config.StarcoinConfig.MonitorInterval) * time.Second)
	for {
		select {
		case <-monitorTicker.C:
			height, err := tools.GetStarcoinNodeHeight(this.config.StarcoinConfig.RestURL, this.restClient)
			if err != nil {
				log.Infof("MonitorDeposit - cannot get eth node height, err: %s", err)
				continue
			}
			snycheight := this.findSyncedHeight()
			log.Log.Info("MonitorDeposit from eth - snyced eth height", snycheight, "eth height", height, "diff", height-snycheight)
			this.handleLockDepositEvents(snycheight) //todo handle ...
		case <-this.exitChan:
			return
		}
	}
}

func (this *StarcoinManager) handleLockDepositEvents(refHeight uint64) error {
	retryList, err := this.db.GetAllStarcoinTxRetry()
	if err != nil {
		return fmt.Errorf("handleLockDepositEvents - this.db.GetAllStarcoinTxRetry error: %s", err)
	}
	for _, v := range retryList {
		time.Sleep(time.Second * 1)
		crosstx := new(CrossTransfer)
		err := crosstx.Deserialization(common.NewZeroCopySource(v))
		if err != nil {
			log.Errorf("handleLockDepositEvents - retry.Deserialization error: %s", err)
			continue
		}
		//1. decode events
		key := crosstx.txIndex
		keyBytes, err := eth.MappingKeyAt(key, "01")
		if err != nil {
			log.Errorf("handleLockDepositEvents - MappingKeyAt error:%s\n", err.Error())
			continue
		}
		if refHeight <= crosstx.height+this.config.ETHConfig.BlockConfig {
			continue
		}
		height := int64(refHeight - this.config.StarcoinConfig.BlockConfig)
		heightHex := hexutil.EncodeBig(big.NewInt(height))
		proofKey := hexutil.Encode(keyBytes)
		//2. get proof
		proof, err := tools.GetProof(this.config.ETHConfig.RestURL, this.config.ETHConfig.ECCDContractAddress, proofKey, heightHex, this.restClient)
		if err != nil {
			log.Errorf("handleLockDepositEvents - error :%s\n", err.Error())
			continue
		}
		//3. commit proof to poly
		txHash, err := this.commitProof(uint32(height), proof, crosstx.value, crosstx.txId)
		if err != nil {
			if strings.Contains(err.Error(), "chooseUtxos, current utxo is not enough") {
				log.Infof("handleLockDepositEvents - invokeNativeContract error: %s", err)
				continue
			} else {
				if err := this.db.DeleteStarcoinTxRetry(v); err != nil {
					log.Errorf("handleLockDepositEvents - this.db.DeleteStarcoinTxRetry error: %s", err)
				}
				if strings.Contains(err.Error(), "tx already done") {
					log.Debugf("handleLockDepositEvents - eth_tx %s already on poly", ethcommon.BytesToHash(crosstx.txId).String())
				} else {
					log.Errorf("handleLockDepositEvents - invokeNativeContract error for eth_tx %s: %s", ethcommon.BytesToHash(crosstx.txId).String(), err)
				}
				continue
			}
		}
		//4. put to check db for checking
		err = this.db.PutStarcoinTxCheck(txHash, v)
		if err != nil {
			log.Errorf("handleLockDepositEvents - this.db.PutStarcoinTxCheck error: %s", err)
		}
		err = this.db.DeleteStarconTxRetry(v)
		if err != nil {
			log.Errorf("handleLockDepositEvents - this.db.DeleteStarconTxRetry error: %s", err)
		}
		log.Infof("handleLockDepositEvents - syncProofToAlia txHash is %s", txHash)
	}
	return nil
}

func (this *StarcoinManager) commitProof(height uint32, proof []byte, value []byte, txhash []byte) (string, error) {
	log.Debugf("commit proof, height: %d, proof: %s, value: %s, txhash: %s", height, string(proof), hex.EncodeToString(value), hex.EncodeToString(txhash))
	tx, err := this.polySdk.Native.Ccm.ImportOuterTransfer(
		this.config.ETHConfig.SideChainId,
		value,
		height,
		proof,
		ethcommon.Hex2Bytes(this.polySigner.Address.ToHexString()), //todo starcoin...
		[]byte{},
		this.polySigner)
	if err != nil {
		return "", err
	} else {
		log.Infof("commitProof - send transaction to poly chain: ( poly_txhash: %s, eth_txhash: %s, height: %d )",
			tx.ToHexString(), ethcommon.BytesToHash(txhash).String(), height)
		return tx.ToHexString(), nil
	}
}

func (this *StarcoinManager) CheckDeposit() {
	checkTicker := time.NewTicker(time.Duration(this.config.ETHConfig.MonitorInterval) * time.Second)
	for {
		select {
		case <-checkTicker.C:
			// try to check deposit
			this.checkLockDepositEvents()
		case <-this.exitChan:
			return
		}
	}
}
func (this *StarcoinManager) checkLockDepositEvents() error {
	checkMap, err := this.db.GetAllStarcoinTxCheck()
	if err != nil {
		return fmt.Errorf("checkLockDepositEvents - this.db.GetAllStarcoinTxCheck error: %s", err)
	}
	for k, v := range checkMap {
		event, err := this.polySdk.GetSmartContractEvent(k)
		if err != nil {
			log.Errorf("checkLockDepositEvents - this.polySdk.GetSmartContractEvent error: %s", err)
			continue
		}
		if event == nil {
			continue
		}
		if event.State != 1 {
			log.Infof("checkLockDepositEvents - state of poly tx %s is not success", k)
			err := this.db.PutStarcoinTxRetry(v)
			if err != nil {
				log.Errorf("checkLockDepositEvents - this.db.PutStarcoinTxRetry error:%s", err)
			}
		}
		err = this.db.DeleteStarcoinTxCheck(k)
		if err != nil {
			log.Errorf("checkLockDepositEvents - this.db.DeleteStarcoinTxCheck error:%s", err)
		}
	}
	return nil
}

func (this *StarcoinManager) findSyncedHeight() uint64 {
	// try to get key
	var sideChainIdBytes [8]byte
	binary.LittleEndian.PutUint64(sideChainIdBytes[:], this.config.StarcoinConfig.SideChainId)
	contractAddress := autils.HeaderSyncContractAddress
	key := append([]byte(scom.CURRENT_HEADER_HEIGHT), sideChainIdBytes[:]...)
	// try to get storage
	result, err := this.polySdk.GetStorage(contractAddress.ToHexString(), key)
	if err != nil {
		return 0
	}
	if result == nil || len(result) == 0 {
		return 0
	} else {
		return binary.LittleEndian.Uint64(result)
	}
}

type CrossTransfer struct {
	txIndex string
	txId    []byte
	value   []byte
	toChain uint32
	height  uint64
}

func (this *CrossTransfer) Serialization(sink *common.ZeroCopySink) {
	sink.WriteString(this.txIndex)
	sink.WriteVarBytes(this.txId)
	sink.WriteVarBytes(this.value)
	sink.WriteUint32(this.toChain)
	sink.WriteUint64(this.height)
}

func (this *CrossTransfer) Deserialization(source *common.ZeroCopySource) error {
	txIndex, eof := source.NextString()
	if eof {
		return fmt.Errorf("Waiting deserialize txIndex error")
	}
	txId, eof := source.NextVarBytes()
	if eof {
		return fmt.Errorf("Waiting deserialize txId error")
	}
	value, eof := source.NextVarBytes()
	if eof {
		return fmt.Errorf("Waiting deserialize value error")
	}
	toChain, eof := source.NextUint32()
	if eof {
		return fmt.Errorf("Waiting deserialize toChain error")
	}
	height, eof := source.NextUint64()
	if eof {
		return fmt.Errorf("Waiting deserialize height error")
	}
	this.txIndex = txIndex
	this.txId = txId
	this.value = value
	this.toChain = toChain
	this.height = height
	return nil
}
