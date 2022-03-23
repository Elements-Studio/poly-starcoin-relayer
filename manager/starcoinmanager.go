package manager

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/db"
	"github.com/elements-studio/poly-starcoin-relayer/log"
	"github.com/elements-studio/poly-starcoin-relayer/tools"

	"github.com/ontio/ontology/smartcontract/service/native/cross_chain/cross_chain_manager"

	stcpolyevts "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly/events"
	polysdk "github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly/common"
	common2 "github.com/polynetwork/poly/native/service/cross_chain_manager/common"
	scom "github.com/polynetwork/poly/native/service/header_sync/common"
	autils "github.com/polynetwork/poly/native/service/utils"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

const (
	ERROR_DESC_GET_PARENT_BLOCK_FAILED = "get the parent block failed"
	ERROR_DESC_MISSING_REQUIRED_FIELD  = "missing required field"
)

type StarcoinManager struct {
	client        *stcclient.StarcoinClient
	polySdk       *polysdk.PolySdk
	polySigner    *polysdk.Account
	config        *config.ServiceConfig
	header4sync   [][]byte
	currentHeight uint64
	forceHeight   uint64
	restClient    *tools.RestClient
	exitChan      chan int
	db            db.DB
}

func NewStarcoinManager(servconfig *config.ServiceConfig, startheight uint64, startforceheight uint64, ontsdk *polysdk.PolySdk, client *stcclient.StarcoinClient, db db.DB) (*StarcoinManager, error) {
	var wallet *polysdk.Wallet
	var err error
	if !common.FileExisted(servconfig.PolyConfig.WalletFile) {
		wallet, err = ontsdk.CreateWallet(servconfig.PolyConfig.WalletFile)
		if err != nil {
			return nil, err
		}
	} else {
		wallet, err = ontsdk.OpenWallet(servconfig.PolyConfig.WalletFile)
		if err != nil {
			log.Errorf("NewStarcoinManager - wallet open error: %s", err.Error())
			return nil, err
		}
	}
	signer, err := wallet.GetDefaultAccount([]byte(servconfig.PolyConfig.WalletPwd))
	if err != nil || signer == nil {
		signer, err = wallet.NewDefaultSettingAccount([]byte(servconfig.PolyConfig.WalletPwd))
		if err != nil {
			log.Errorf("NewStarcoinManager - wallet password error")
			return nil, err
		}

		err = wallet.Save()
		if err != nil {
			return nil, err
		}
	}
	log.Infof("NewStarcoinManager - poly address: %s", signer.Address.ToBase58())

	mgr := &StarcoinManager{
		config:        servconfig,
		exitChan:      make(chan int),
		currentHeight: startheight,
		forceHeight:   startforceheight,
		restClient:    tools.NewRestClient(),
		client:        client,
		polySdk:       ontsdk,
		polySigner:    signer,
		header4sync:   make([][]byte, 0),
		//crosstx4sync:  make([]*CrossTransfer, 0),
		db: db,
	}
	err = mgr.init()
	if err != nil {
		return nil, err
	} else {
		return mgr, nil
	}
}

func (this *StarcoinManager) init() error {
	// get latest height
	latestHeight, _ := this.findSyncedHeight()
	if latestHeight == 0 {
		return fmt.Errorf("init - the genesis block has not synced!")
	}
	if this.forceHeight > 0 && this.forceHeight < latestHeight {
		this.currentHeight = this.forceHeight
	} else {
		this.currentHeight = latestHeight
	}
	log.Infof("StarcoinManager init - start height: %d", this.currentHeight)
	return nil
}

func (this *StarcoinManager) MonitorChain() {
	//fetchBlockTicker := time.NewTicker(200 * time.Millisecond) // for test
	fetchBlockTicker := time.NewTicker(time.Duration(this.config.StarcoinConfig.MonitorInterval) * time.Second)
	var blockHandleResult bool
	for {
		select {
		case <-fetchBlockTicker.C:
			height, err := tools.GetStarcoinNodeHeight(this.config.StarcoinConfig.RestURL, this.restClient)
			if err != nil {
				log.Infof("StarcoinManager.MonitorChain - cannot get node height, err: %s", err.Error())
				continue
			}
			if height-this.currentHeight <= config.STARCOIN_USEFUL_BLOCK_NUM {
				continue
			}
			log.Infof("StarcoinManager.MonitorChain - starcoin height is %d", height)
			blockHandleResult = true
			for this.currentHeight < height-config.STARCOIN_USEFUL_BLOCK_NUM {
				if this.currentHeight%10 == 0 {
					log.Infof("StarcoinManager.MonitorChain - handle confirmed starcoin block height: %d", this.currentHeight)
				}
				// ///////////////////////////////////////////////////////////////////////////
				// TODO: if len(this.header4sync) is too large, should not continue???
				// ///////////////////////////////////////////////////////////////////////////
				max_header4sync_size := this.config.StarcoinConfig.HeadersPerBatch * 3
				if len(this.header4sync) > max_header4sync_size {
					log.Errorf("StarcoinManager.MonitorChain - length of this.header4sync is too large(%d > %d). Starcoin block height: %d. Reset!", len(this.header4sync), max_header4sync_size, this.currentHeight)
					time.Sleep(time.Second * 10)
					// ////////////////////////////////////////////////////////////////////
					// reset???
					//if this.currentHeight > uint64(len(this.header4sync)) {
					this.currentHeight = this.currentHeight - uint64(len(this.header4sync))
					this.header4sync = make([][]byte, 0)
					//}
					blockHandleResult = false
					// ////////////////////////////////////////////////////////////////////
					break
				}

				blockHandleResult = this.handleNewBlock(this.currentHeight + 1)
				// fmt.Println("----------------- this.handleNewBlock() result -----------------")
				// fmt.Println(blockHandleResult)
				if blockHandleResult == false {
					break
				}
				this.currentHeight++
				// fmt.Println("--------------- this.currentHeight++ ----------------")
				// fmt.Println(this.currentHeight)
				// fmt.Println("----------------- len(header4sync) -----------------")
				// fmt.Println(len(this.header4sync))
				// try to commit header if more than 50 headers needed to be syned
				if len(this.header4sync) >= this.config.StarcoinConfig.HeadersPerBatch {
					// fmt.Println("--------- len(this.header4sync) >= this.config.StarcoinConfig.HeadersPerBatch --------")
					// fmt.Println("----------------- len(header4sync) -----------------")
					// fmt.Println(len(this.header4sync))
					// fmt.Println("--------------- this.header4sync ----------------")
					// printHeader4sync(this.header4sync)
					// fmt.Println("--------------- this.currentHeight ----------------")
					// fmt.Println(this.currentHeight)
					// fmt.Println("---------------- commit headers... ------------------")
					if res := this.commitHeader(); res != 0 {
						log.Errorf("StarcoinManager.MonitorChain - commitHeader error, current height: %d", this.currentHeight)
						blockHandleResult = false
						break
					}
				}
			}
			// fmt.Println("----------------- blockHandleResult -----------------")
			// fmt.Println(blockHandleResult)
			// fmt.Println("----------------- len(header4sync) -----------------")
			// fmt.Println(len(this.header4sync))
			// fmt.Println("--------------- this.currentHeight ----------------")
			// fmt.Println(this.currentHeight)
			if blockHandleResult && len(this.header4sync) > 0 {
				// fmt.Println("--------------- this.header4sync ----------------")
				// printHeader4sync(this.header4sync)
				// fmt.Println("---------------- commit headers... ------------------")
				res := this.commitHeader()
				if res != 0 {
					log.Errorf("StarcoinManager.MonitorChain - commitHeader error, current height: %d", this.currentHeight)
				}
			}
		case <-this.exitChan:
			return
		}
	}
}

// this is a test method
// func printHeader4sync(headers [][]byte) {
// 	fmt.Println("[")
// 	for i, h := range headers {
// 		if i != 0 {
// 			fmt.Println(",")
// 		}
// 		fmt.Println(string(h))
// 	}
// 	fmt.Println("]")
// }

// if success, return 0
func (this *StarcoinManager) commitHeader() int {
	tx, err := this.polySdk.Native.Hs.SyncBlockHeader(
		this.config.StarcoinConfig.SideChainId,
		this.polySigner.Address,
		this.header4sync,
		this.polySigner,
	)
	if err != nil {
		errDesc := err.Error()
		if strings.Contains(errDesc, ERROR_DESC_GET_PARENT_BLOCK_FAILED) || strings.Contains(errDesc, ERROR_DESC_MISSING_REQUIRED_FIELD) {
			log.Warnf("StarcoinManager.commitHeader - send transaction to poly chain err: %s, about to rollBackToCommAncestor...", errDesc)
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
		h, err = this.polySdk.GetBlockHeightByTxHash(tx.ToHexString())
		if err != nil {
			_ = err //log.Debugf("StarcoinManager.commitHeader - GetBlockHeightByTxHash: %d", h)
		}
		curr, err := this.polySdk.GetCurrentBlockHeight()
		if err != nil {
			_ = err //log.Debugf("StarcoinManager.commitHeader - GetCurrentBlockHeight: %d", curr)
		}
		//
		// TODO: ignore errors? like this:
		// h, _ = this.polySdk.GetBlockHeightByTxHash(tx.ToHexString())
		// curr, _ := this.polySdk.GetCurrentBlockHeight()
		//
		if h > 0 && curr > h {
			log.Debugf("Break! GetCurrentBlockHeight() > GetBlockHeightByTxHash(tx.ToHexString()). Tx height: %d, current block height: %d", h, curr)
			break
		}
		//log.Debugf("Waiting GetCurrentBlockHeight() > GetBlockHeightByTxHash(tx.ToHexString()). Tx height: %d, current block height: %d", h, curr)
	}

	log.Infof("commitHeader - send transaction %s to poly chain and confirmed on height %d", tx.ToHexString(), h)
	this.header4sync = make([][]byte, 0)
	return 0
}

func (this *StarcoinManager) rollBackToCommAncestor() {
	for ; ; this.currentHeight-- {
		//log.Debugf("this.currentHeight: %d", this.currentHeight)
		raw, err := this.polySdk.GetStorage(autils.HeaderSyncContractAddress.ToHexString(),
			append(append([]byte(scom.MAIN_CHAIN), autils.GetUint64Bytes(this.config.StarcoinConfig.SideChainId)...), autils.GetUint64Bytes(this.currentHeight)...))
		if len(raw) == 0 || err != nil {
			//log.Debugf("rollBackToCommAncestor - len(raw) == 0 || err != nil, len(raw): %d, err: %v", len(raw), err)
			continue
		}

		hdr, err := this.client.HeaderByNumber(context.Background(), this.currentHeight)
		//hdr, err := this.client.HeaderWithDifficutyInfoByNumber(context.Background(), this.currentHeight)
		if err != nil {
			log.Errorf("rollBackToCommAncestor - failed to get header by number, so we wait for one second to retry: %v", err)
			time.Sleep(time.Second)
			this.currentHeight++
			continue
		}
		hdrhash, err := hdr.Hash()
		if err != nil {
			log.Errorf("rollBackToCommAncestor - failed to get header hash, so we wait for one second to retry: %v", err)
			time.Sleep(time.Second)
			this.currentHeight++
			continue // TODO: is this ok??
		}
		if bytes.Equal(hdrhash, raw) {
			// // ---------------- debug code start... -------------------
			// hdrInPoly, err := this.polySdk.GetStorage(autils.HeaderSyncContractAddress.ToHexString(),
			// 	append(append([]byte(scom.HEADER_INDEX), autils.GetUint64Bytes(this.config.StarcoinConfig.SideChainId)...), raw...))
			// if err != nil {
			// 	log.Debugf("rollBackToCommAncestor - GetStorage error: %s", err.Error())
			// }
			// log.Debugf("rollBackToCommAncestor - header in poly: %s", hex.EncodeToString(hdrInPoly))
			// // ---------------- debug code end -------------------
			log.Infof("rollBackToCommAncestor - find the common ancestor: %s(block hash) %s(header hash) (number: %d)", hdr.BlockHash, hex.EncodeToString(hdrhash), this.currentHeight)
			break
		} else {
			if this.currentHeight%10 == 0 {
				log.Infof("rollBackToCommAncestor - rolling back..., current height: %d", this.currentHeight)
			}
			// // ---------------- debug code start... -------------------
			// log.Debugf("rollBackToCommAncestor - hdr hash: %s, raw hash from poly: %s, current height: %d", hex.EncodeToString(hdrhash), hex.EncodeToString(raw), this.currentHeight)
			// // ---------------- debug code end -------------------
		}
	}
	this.header4sync = make([][]byte, 0)
	// fmt.Println("---------------- rollBackToCommAncestor, len(this.header4sync) --------------")
	// fmt.Println(this.header4sync)
	// fmt.Println("---------------- rollBackToCommAncestor,  this.currentHeight --------------")
	// fmt.Println(this.currentHeight)
}

func (this *StarcoinManager) handleNewBlock(height uint64) bool {
	ret := this.handleBlockHeader(height)
	if !ret {
		log.Errorf("StarcoinManager.handleNewBlock - handleBlockHeader on height :%d failed", height)
		return false
	}
	ret, err := this.fetchLockDepositEvents(height)
	if err != nil {
		log.Errorf("StarcoinManager.handleNewBlock - fetchLockDepositEvents on height :%d failed", height)
		return false //not ignore event fetch error here! Is this ok?
	} else if !ret {
		log.Debugf("No events need to handle on height :%d", height)
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
	// block, err := this.client.GetBlockByNumber(context.Background(), int(height))
	// if err != nil {
	// 	log.Errorf("handleBlockHeader - GetBlockByNumber on height :%d failed", height)
	// 	return false
	// }

	hdr, err := this.client.HeaderWithDifficultyInfoByNumber(context.Background(), height) //block.BlockHeader
	if err != nil {
		log.Errorf("handleBlockHeader - HeaderWithDifficutyInfoByNumber on height :%d failed", height)
		return false
	}
	// // ------------------------------------ debug code start ---------------------------------
	// log.Infof("----------------- height: %d -----------------", height)
	// headerhash, err := hdr.BlockHeader.Hash()
	// blockhash, err := tools.HexToBytes(hdr.BlockHeader.BlockHash)
	// if bytes.Compare(headerhash, blockhash) != 0 {
	// 	log.Warnf("hdr.Hash(): %s <> hdr.BlockHash: %s", hex.EncodeToString(headerhash), hex.EncodeToString(blockhash))
	// 	panic(1)
	// } else {
	// 	log.Infof("hdr.Hash(): %s == hdr.BlockHash: %s", hex.EncodeToString(headerhash), hex.EncodeToString(blockhash))
	// }
	// return true
	// // ------------------------------------ debug code end ------------------------------------

	rawHdr, marshalErr := json.Marshal(hdr)
	if marshalErr != nil {
		log.Errorf("handleBlockHeader - json.Marshal error on height: %d", height)
		return false
	}
	raw, _ := this.polySdk.GetStorage(autils.HeaderSyncContractAddress.ToHexString(),
		append(append([]byte(scom.MAIN_CHAIN), autils.GetUint64Bytes(this.config.StarcoinConfig.SideChainId)...), autils.GetUint64Bytes(height)...))
	// this var `raw` is block(header) hash saved in poly
	hdrhash, err := hdr.BlockHeader.Hash() //calculated hash
	if err != nil {
		log.Errorf("handleBlockHeader - get header hash on height :%d failed", height)
		return false
	}
	blockhash, err := tools.HexToBytes(hdr.BlockHeader.BlockHash) // hash returned by JSON RPC
	if err != nil {
		log.Errorf("handleBlockHeader - get hdr.BlockHeader.BlockHash on height :%d failed", height)
		return false
	}
	if !bytes.Equal(hdrhash, blockhash) {
		log.Errorf("handleBlockHeader - hdr.Hash(): %s <> hdr.BlockHeader.BlockHash: %s", hex.EncodeToString(hdrhash), hex.EncodeToString(blockhash))
		return false //panic(1)
	}
	// //////////// forcely rewrite now!!! //////////////
	// _ = raw
	// this.header4sync = append(this.header4sync, rawHdr)
	// //////////////////////////////////////////////////
	// Or shoud use this normally???
	if len(raw) == 0 || !bytes.Equal(raw, hdrhash) {
		this.header4sync = append(this.header4sync, rawHdr)
	}
	return true
}

func (this *StarcoinManager) fetchLockDepositEvents(height uint64) (bool, error) {
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
		Address:   []string{this.config.StarcoinConfig.CrossChainEventAddress},
		TypeTags:  []string{this.config.StarcoinConfig.CrossChainEventTypeTag},
		FromBlock: height,
		ToBlock:   &height,
	}

	//events, err :=] lockContract.FilterCrossChainEvent(opt, nil)
	events, err := this.client.GetEvents(context.Background(), eventFilter)
	if err != nil {
		log.Errorf("fetchLockDepositEvents - FilterCrossChainEvent error :%s", err.Error())
		return false, err
	}
	if events == nil {
		log.Infof("fetchLockDepositEvents - no events found on FilterCrossChainEvent")
		return false, nil
	}

	for _, evt := range events {
		//evt := events.Event
		//fmt.Println(evt)
		evtData, err := tools.HexToBytes(evt.Data)
		if err != nil {
			log.Errorf("fetchLockDepositEvents - hex.DecodeString error :%s", err.Error())
			return false, err
		}
		ccEvent, err := stcpolyevts.BcsDeserializeCrossChainEvent(evtData)
		// // Fire the cross chain event denoting there is a cross chain request from Ethereum network to other public chains through Poly chain network
		// emit CrossChainEvent(tx.origin, paramTxHash, msg.sender, toChainId, toContract, rawParam);
		// // event CrossChainEvent(
		// //     address indexed sender,
		// //     bytes txId,                      // transaction hash
		// //     address proxyOrAssetContract,    // msg.sender
		// //     uint64 toChainId,
		// //     bytes toContract,
		// //     bytes rawdata
		// // );
		if err != nil {
			log.Errorf("fetchLockDepositEvents - BcsDeserializeCrossChainDepositEvent error :%s", err.Error())
			return false, err
		}
		var isTarget bool
		if len(this.config.ProxyOrAssetContracts) > 0 {
			// // fmt.Println(tools.EncodeToHex(ccEvent.Sender))
			// fmt.Println("---------------- height -----------------")
			// fmt.Println(height)
			// fmt.Println("---------------- ProxyOrAssetContract -----------------")
			// fmt.Println(ccEvent.ProxyOrAssetContract)
			// fmt.Println(string(ccEvent.ProxyOrAssetContract)) //tools.EncodeToHex(ccEvent.ProxyOrAssetContract)
			// fmt.Println("---------------- TxId(CrossChainEvent.TxId) -----------------")
			// fmt.Println(tools.EncodeToHex(ccEvent.TxId))
			// fmt.Println("---------------- ToChainId -----------------")
			// fmt.Println(ccEvent.ToChainId)
			// fmt.Println("---------------- ToContract -----------------")
			// fmt.Println(string(ccEvent.ToContract))
			// fmt.Println("---------------- RawData -----------------")
			// fmt.Println(tools.EncodeToHex(ccEvent.RawData))
			//var proxyOrAssetContract string
			proxyOrAssetContract := string(ccEvent.ProxyOrAssetContract) // for 'source' proxy contract, filter is outbound chain Id.
			for _, v := range this.config.ProxyOrAssetContracts {        // renamed TargetContracts
				chainIdArrMap, ok := v[proxyOrAssetContract]
				if ok {
					if len(chainIdArrMap["outbound"]) == 0 {
						isTarget = true
						break
					}
					for _, id := range chainIdArrMap["outbound"] {
						if id == ccEvent.ToChainId {
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
		//_ = param.Deserialization(common.NewZeroCopySource([]byte(ccEvent.RawData)))
		raw, _ := this.polySdk.GetStorage(autils.CrossChainManagerContractAddress.ToHexString(),
			append(append([]byte(cross_chain_manager.DONE_TX), autils.GetUint64Bytes(this.config.StarcoinConfig.SideChainId)...), param.CrossChainID...))
		if len(raw) != 0 {
			log.Debugf("fetchLockDepositEvents - ccid %s (tx_hash: %s) already on poly",
				hex.EncodeToString(param.CrossChainID), evt.TransactionHash)
			continue
		}
		index := big.NewInt(0)
		index.SetBytes(ccEvent.TxId)
		txHash, err := tools.HexWithPrefixToBytes(evt.TransactionHash)
		if err != nil {
			log.Errorf("fetchLockDepositEvents - tools.HexWithPrefixToBytes error: %s", err.Error())
			return false, err
		}
		// fmt.Println("---------------- Starcoin Transaction Hash -----------------")
		// fmt.Println(tools.EncodeToHex(txHash))
		crossTx := &CrossTransfer{
			txIndex: tools.EncodeBigInt(index), // tools.EncodeBigInt(ccEvent.TxId to big.Int),
			txId:    txHash,                    // starcoin tx hash
			toChain: uint32(ccEvent.ToChainId),
			value:   ccEvent.RawData,
			height:  height,
		}
		sink := common.NewZeroCopySink(nil)
		crossTx.Serialization(sink)
		err = this.db.PutStarcoinTxRetry(sink.Bytes(), evt)
		if err != nil {
			log.Errorf("fetchLockDepositEvents - this.db.PutStarcoinTxRetry error: %s", err.Error())
			return false, err
		}
		log.Infof("fetchLockDepositEvent -  height: %d", height)
	}
	return true, nil
}

func (this *StarcoinManager) MonitorDeposit() {
	monitorTicker := time.NewTicker(time.Duration(this.config.StarcoinConfig.MonitorInterval) * time.Second)
	for {
		select {
		case <-monitorTicker.C:
			height, err := tools.GetStarcoinNodeHeight(this.config.StarcoinConfig.RestURL, this.restClient)
			if err != nil {
				log.Infof("MonitorDeposit - cannot get starcoin node height, err: %s", err.Error())
				continue
			}
			snycheight, err := this.findSyncedHeight()
			if err != nil {
				log.Errorf("MonitorDeposit - Failed to findSyncedHeight %v %v", snycheight, err)
			}
			if snycheight > height-config.STARCOIN_PROOF_USERFUL_BLOCK {
				log.Info("MonitorDeposit from starcoin - snyced starcoin height", snycheight, "starcoin height", height, "diff", height-snycheight)
				// try to handle deposit event when we are at latest height
				err := this.handleLockDepositEvents(snycheight)
				if err != nil {
					log.Errorf("MonitorDeposit - Failed to handleLockDepositEvents %v %v", snycheight, err)
				}
			}

		case <-this.exitChan:
			return
		}

	}
}

func (this *StarcoinManager) handleLockDepositEvents(refHeight uint64) error {
	retryList, eventList, err := this.db.GetAllStarcoinTxRetry()
	if err != nil {
		return fmt.Errorf("handleLockDepositEvents - this.db.GetAllStarcoinTxRetry error: %s", err.Error())
	}
	for i, v := range retryList {
		// fmt.Println("------------------------ event from retry list ----------------------------")
		// fmt.Println(hex.EncodeToString(v))
		time.Sleep(time.Second * 1)
		crosstx := new(CrossTransfer)
		err := crosstx.Deserialization(common.NewZeroCopySource(v))
		// fmt.Println("------------------------ crosstx deserialized ----------------------------")
		// fmt.Printf("crosstx.height: %d\n", crosstx.height)
		// fmt.Printf("crosstx.toChain: %d\n", crosstx.toChain)
		// fmt.Println("crosstx.txIndex: " + crosstx.txIndex + " // tools.EncodeBigInt(CrossChainEvent.TxId to big.Int)")
		// fmt.Println("crosstx.txId: " + hex.EncodeToString(crosstx.txId) + " // starcoin tx hash")
		// fmt.Println("crosstx.value: " + hex.EncodeToString(crosstx.value) + " // CrossChainEvent.RawData")

		// //////////////////////
		evt := eventList[i]
		// //////////////////////
		if err != nil {
			log.Errorf("handleLockDepositEvents - retry.Deserialization error: %s", err.Error())
			continue
		}
		//1. decode events
		//key := crosstx.txIndex
		//fmt.Println(key)
		//keyBytes, err := eth.MappingKeyAt(key, "01") //starcoin version...
		// if err != nil {
		// 	log.Errorf("handleLockDepositEvents - MappingKeyAt error:%s\n", err.Error())
		// 	continue
		// }
		if refHeight <= crosstx.height+this.config.StarcoinConfig.BlockConfirmations {
			log.Debugf("handleLockDepositEvents - this height(%d) is not confirmed yet.", crosstx.height)
			continue
		}
		//height := int64(refHeight - this.config.StarcoinConfig.BlockConfirmations)
		//heightHex := hexutil.EncodeBig(big.NewInt(height))
		//proofKey := hexutil.Encode(keyBytes)

		//2. get proof
		// starcoin GetProof...
		eventIdx := evt.EventIndex
		txGlobalIdx, err := strconv.ParseUint(evt.TransactionGlobalIndex, 10, 64)
		if err != nil {
			log.Errorf("handleLockDepositEvents - ParseUint error :%s\n", err.Error())
			return err // TODO: or continue???
		}
		proof, err := tools.GetTransactionProof(this.config.StarcoinConfig.RestURL, this.restClient, evt.BlockHash, txGlobalIdx, &eventIdx)
		if err != nil {
			log.Errorf("handleLockDepositEvents - GetTransactionProof error :%s\n", err.Error())
			return err
		}
		// proof, err := tools.GetProof(this.config.StarcoinConfig.RestURL, this.config.StarcoinConfig.CCDContractAddress, proofKey, heightHex, this.restClient)
		// if err != nil {
		// 	log.Errorf("handleLockDepositEvents - error :%s\n", err.Error())
		// 	continue
		// }
		//3. commit proof to poly
		height, err := strconv.ParseUint(evt.BlockNumber, 10, 64)
		if err != nil {
			log.Errorf("handleLockDepositEvents - ParseUint error :%s\n", err.Error())
			return err
		}
		// evtMsg, err := json.Marshal(evt)
		// if err != nil {
		// 	log.Errorf("handleLockDepositEvents - json.Marshal(evt) error :%s\n", err.Error())
		// 	return err
		// }
		evtMsg := StarcoinToPolyHeaderOrCrossChainMsg{
			EventIndex: &eventIdx,
			AccessPath: nil,
		}
		evtMsgBS, err := json.Marshal(evtMsg) // like this: {"event_index":1}
		if err != nil {
			log.Errorf("handleLockDepositEvents - json.Marshal(evtMsg) error :%s\n", err.Error())
			return err
		}
		//txHash, err := this.commitProof(uint32(height), []byte(proof), crosstx.value, crosstx.txId, evtMsgBS)
		txHash, err := this.commitProof(uint32(height), []byte(proof), []byte{}, crosstx.txId, evtMsgBS)
		if err != nil {
			if strings.Contains(err.Error(), "chooseUtxos, current utxo is not enough") {
				log.Infof("handleLockDepositEvents - invokeNativeContract error: %s", err.Error())
				continue
			} else {
				// if err := this.db.DeleteStarcoinTxRetry(v); err != nil {
				// 	log.Errorf("handleLockDepositEvents - this.db.DeleteStarcoinTxRetry error: %s", err.Error())
				// }
				if strings.Contains(err.Error(), "tx already done") {
					log.Debugf("handleLockDepositEvents - starcoin_tx %s already on poly", tools.EncodeToHex(crosstx.txId))
					if err := this.db.DeleteStarcoinTxRetry(v); err != nil {
						log.Errorf("handleLockDepositEvents - this.db.DeleteStarcoinTxRetry error: %s", err.Error())
					}
				} else {
					log.Errorf("handleLockDepositEvents - invokeNativeContract error for starcoin_tx %s: %s", tools.EncodeToHex(crosstx.txId), err)
				}
				continue
			}
		}
		//4. put to check db for checking
		err = this.db.PutStarcoinTxCheck(txHash, v, evt)
		if err != nil {
			log.Errorf("handleLockDepositEvents - this.db.PutStarcoinTxCheck error: %s", err.Error())
		}
		err = this.db.DeleteStarcoinTxRetry(v)
		if err != nil {
			log.Errorf("handleLockDepositEvents - this.db.DeleteStarconTxRetry error: %s", err.Error())
		}
		log.Infof("handleLockDepositEvents - syncProofToAlia txHash is %s", txHash)
	}
	return nil
}

func (this *StarcoinManager) commitProof(height uint32, proof []byte, value []byte, txhash []byte, eventMsg []byte) (string, error) {
	log.Debugf("commit proof, height: %d, proof: %s, value: %s, txhash: %s", height, string(proof), hex.EncodeToString(value), hex.EncodeToString(txhash))
	relayAddr, err := tools.HexToBytes(this.polySigner.Address.ToHexString())
	if err != nil {
		return "", err
	}
	// fmt.Println("--------- parameters of polySdk.Native.Ccm.ImportOuterTransfer ----------")
	// fmt.Println("--------- sourceChainId ---------")
	// fmt.Println(this.config.StarcoinConfig.SideChainId) // sourceChainId uint64,
	// fmt.Println("--------- txData ---------")
	// fmt.Println(hex.EncodeToString(value)) // txData []byte, // CrossChainEvent.RawData
	// fmt.Println("--------- height ---------")
	// fmt.Println(height) // height
	// fmt.Println("--------- proof ---------")
	// fmt.Println(string(proof)) // proof []byte
	// fmt.Println("--------- relayAddr ---------")
	// fmt.Println(hex.EncodeToString(relayAddr)) // relayAddr
	// fmt.Println("--------- headerOrCrossChainMsg ---------")
	// fmt.Println(string(eventMsg)) // headerOrCrossChainMsg
	// fmt.Println("--------- signer ---------")
	// fmt.Println(this.polySigner.Address.ToHexString()) // polySigner
	// fmt.Println("------- end of params of polySdk.Native.Ccm.ImportOuterTransfer ---------")

	tx, err := this.polySdk.Native.Ccm.ImportOuterTransfer(
		this.config.StarcoinConfig.SideChainId, //sourceChainId uint64,
		value,                                  //txData []byte, 					// CrossChainEvent.RawData
		height,                                 //height uint32,
		proof,                                  //proof []byte,						// Starcoin event proof
		relayAddr,                              //relayerAddress []byte,
		eventMsg,                               //HeaderOrCrossChainMsg []byte, 	// Starcoin event json
		this.polySigner)                        //signer *Account)

	if err != nil {
		return "", err
	} else {
		log.Infof("commitProof - send transaction to poly chain: ( poly_txhash: %s, starcoin_txhash: %s, height: %d )",
			tx.ToHexString(), tools.EncodeToHex(txhash), height)
		return tx.ToHexString(), nil
	}
}

func (this *StarcoinManager) CheckDeposit() {
	checkTicker := time.NewTicker(time.Duration(this.config.StarcoinConfig.MonitorInterval) * time.Second)
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
		return fmt.Errorf("checkLockDepositEvents - this.db.GetAllStarcoinTxCheck error: %s", err.Error())
	}
	for k, v := range checkMap {
		event, err := this.polySdk.GetSmartContractEvent(k)
		if err != nil {
			log.Errorf("checkLockDepositEvents - this.polySdk.GetSmartContractEvent error: %s", err.Error())
			continue
		}
		if event == nil {
			continue
		}
		if event.State != 1 {
			log.Infof("checkLockDepositEvents - state of poly tx %s is not success", k)
			err := this.db.PutStarcoinTxRetry(v.Bytes, v.Event)
			if err != nil {
				log.Errorf("checkLockDepositEvents - this.db.PutStarcoinTxRetry error:%s", err.Error())
			}
		}
		err = this.db.DeleteStarcoinTxCheck(k)
		if err != nil {
			log.Errorf("checkLockDepositEvents - this.db.DeleteStarcoinTxCheck error:%s", err.Error())
		}
	}
	return nil
}

// Find starcoin synced height from poly storage. Return 0 if error.
func (this *StarcoinManager) findSyncedHeight() (uint64, error) {
	// try to get key
	var sideChainIdBytes [8]byte
	binary.LittleEndian.PutUint64(sideChainIdBytes[:], this.config.StarcoinConfig.SideChainId)
	contractAddress := autils.HeaderSyncContractAddress
	key := append([]byte(scom.CURRENT_HEADER_HEIGHT), sideChainIdBytes[:]...)
	// try to get storage
	result, err := this.polySdk.GetStorage(contractAddress.ToHexString(), key)
	if err != nil {
		return 0, err
	}
	if result == nil || len(result) == 0 {
		return 0, nil
	} else {
		return binary.LittleEndian.Uint64(result), nil
	}
}

type CrossTransfer struct {
	txIndex string
	// starcoin tx hash
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

type StarcoinToPolyHeaderOrCrossChainMsg struct {
	EventIndex *int    `json:"event_index,omitempty"`
	AccessPath *string `json:"access_path,omitempty"`
}
