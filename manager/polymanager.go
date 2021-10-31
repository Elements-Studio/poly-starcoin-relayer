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
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/log"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	"github.com/ethereum/go-ethereum"

	//ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology-crypto/signature"
	"github.com/polynetwork/bridge-common/abi/eccd_abi" // remove this
	polysdk "github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly/common"
	vconfig "github.com/polynetwork/poly/consensus/vbft/config"
	polytypes "github.com/polynetwork/poly/core/types"
	common2 "github.com/polynetwork/poly/native/service/cross_chain_manager/common"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	stctypes "github.com/starcoinorg/starcoin-go/types"
)

const (
	ChanLen = 64
)

type PolyManager struct {
	config         *config.ServiceConfig
	polySdk        *polysdk.PolySdk
	currentHeight  uint32
	exitChan       chan int
	starcoinClient *stcclient.StarcoinClient
	senders        []*StarcoinSender
}

func (this *PolyManager) init() bool {
	if this.currentHeight > 0 {
		log.Infof("PolyManager init - start height from flag: %d", this.currentHeight)
		return true
	}
	this.currentHeight = this.db.GetPolyHeight() // todo db module...
	curEpochStart := this.findCurEpochStartHeight()
	if curEpochStart > this.currentHeight {
		this.currentHeight = curEpochStart
		log.Infof("PolyManager init - latest height from ECCM: %d", this.currentHeight)
		return true
	}
	log.Infof("PolyManager init - latest height from DB: %d", this.currentHeight)

	return true
}

func (this *PolyManager) MonitorChain() {
	ret := this.init()
	if ret == false {
		log.Errorf("MonitorChain - init failed\n")
	}
	monitorTicker := time.NewTicker(config.POLY_MONITOR_INTERVAL)
	var blockHandleResult bool
	for {
		select {
		case <-monitorTicker.C:
			latestheight, err := this.polySdk.GetCurrentBlockHeight()
			if err != nil {
				log.Errorf("PolyManager.MonitorChain - get poly chain block height error: %s", err)
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
					log.Infof("handle confirmed poly Block height: %d", this.currentHeight)
				}
				blockHandleResult = this.handleDepositEvents(this.currentHeight)
				if blockHandleResult == false {
					break
				}
				this.currentHeight++
			}
			if err = this.db.UpdatePolyHeight(this.currentHeight - 1); err != nil {
				log.Errorf("PolyManager.MonitorChain - failed to save height of poly: %v", err)
			}
		case <-this.exitChan:
			return
		}
	}
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
		anchor, _ = this.polySdk.GetHeaderByHeight(lastEpoch + 1)
		proof, _ := this.polySdk.GetMerkleProof(height+1, lastEpoch+1)
		hp = proof.AuditPath
	} else if isEpoch {
		anchor, _ = this.polySdk.GetHeaderByHeight(height + 2)
		proof, _ := this.polySdk.GetMerkleProof(height+1, height+2)
		hp = proof.AuditPath
	}

	cnt := 0
	events, err := this.polySdk.GetSmartContractEventByBlock(height)
	for err != nil {
		log.Errorf("handleDepositEvents - get block event at height:%d error: %s", height, err.Error())
		return false
	}
	for _, event := range events {
		for _, notify := range event.Notify {
			if notify.ContractAddress == this.config.PolyConfig.EntranceContractAddress {
				states := notify.States.([]interface{})
				method, _ := states[0].(string)
				if method != "makeProof" {
					continue
				}
				if uint64(states[2].(float64)) != this.config.StarcoinConfig.SideChainId {
					continue
				}
				proof, err := this.polySdk.GetCrossStatesProof(hdr.Height-1, states[5].(string))
				if err != nil {
					log.Errorf("handleDepositEvents - failed to get proof for key %s: %v", states[5].(string), err)
					continue
				}
				auditpath, _ := hex.DecodeString(proof.AuditPath)
				value, _, _, _ := tools.ParseAuditpath(auditpath)
				param := &common2.ToMerkleValue{}
				if err := param.Deserialization(common.NewZeroCopySource(value)); err != nil {
					log.Errorf("handleDepositEvents - failed to deserialize MakeTxParam (value: %x, err: %v)", value, err)
					continue
				}
				var isTarget bool
				if len(this.config.TargetContracts) > 0 {
					toContractStr := ethcommon.BytesToAddress(param.MakeTxParam.ToContractAddress).String() //todo starcoin contract address...
					for _, v := range this.config.TargetContracts {
						toChainIdArr, ok := v[toContractStr]
						if ok {
							if len(toChainIdArr["inbound"]) == 0 {
								isTarget = true
								break
							}
							for _, id := range toChainIdArr["inbound"] {
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
						continue
					}
				}
				cnt++
				sender := this.selectSender()
				log.Infof("sender %s is handling poly tx ( hash: %s, height: %d )",
					sender.acc.Address.String(), event.TxHash, height)
				// temporarily ignore the error for tx
				sender.commitDepositEventsWithHeader(hdr, param, hp, anchor, event.TxHash, auditpath)
				//if !sender.commitDepositEventsWithHeader(hdr, param, hp, anchor, event.TxHash, auditpath) {
				//	return false
				//}
			}
		}
	}
	if cnt == 0 && isEpoch && isCurr {
		sender := this.selectSender()
		return sender.changeBookKeeper(hdr, pubkList) // commitHeader
	}

	return true
}

func (this *PolyManager) IsEpoch(hdr *polytypes.Header) (bool, []byte, error) {
	blkInfo := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(hdr.ConsensusPayload, blkInfo); err != nil {
		return false, nil, fmt.Errorf("IsEpoch - unmarshal blockInfo error: %s", err)
	}
	if hdr.NextBookkeeper == common.ADDRESS_EMPTY || blkInfo.NewChainConfig == nil {
		return false, nil, nil
	}

	eccdAddr := ethcommon.HexToAddress(this.config.StarcoinConfig.ECCDContractAddress) //todo starcoin...
	eccd, err := eccd_abi.NewEthCrossChainData(eccdAddr, this.ethClient)               //todo starcoin...
	if err != nil {
		return false, nil, fmt.Errorf("failed to new eccm: %v", err)
	}
	rawKeepers, err := eccd.GetCurEpochConPubKeyBytes(nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get current epoch keepers: %v", err)
	}

	var bookkeepers []keypair.PublicKey
	for _, peer := range blkInfo.NewChainConfig.Peers {
		keystr, _ := hex.DecodeString(peer.ID)
		key, _ := keypair.DeserializePublicKey(keystr)
		bookkeepers = append(bookkeepers, key)
	}
	bookkeepers = keypair.SortPublicKeys(bookkeepers)
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

func (this *PolyManager) findCurEpochStartHeight() uint32 {
	var address string                                                      //todo ethcommon.HexToAddress(this.config.StarcoinConfig.ECCDContractAddress)
	instance, err := eccd_abi.NewEthCrossChainData(address, this.ethClient) //todo get starcoin on-chain contract proxy
	if err != nil {
		log.Errorf("findCurEpochStartHeight - new eth cross chain failed: %s", err.Error())
		return 0
	}
	height, err := instance.GetCurEpochStartHeight(nil)
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
			log.Errorf("failed to get balance for %s: %v", v.acc.Address.String(), err)
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
	starcoinClient stcclient.StarcoinClient
	seqNumManager  tools.SeqNumManager
	keyStore       tools.StarcoinKeyStore
	acc            tools.StarcoinAccount
	config         *config.ServiceConfig
	cmap           map[string]chan *StarcoinTxInfo
}

func (this *StarcoinSender) commitDepositEventsWithHeader(header *polytypes.Header, param *common2.ToMerkleValue, headerProof string, anchorHeader *polytypes.Header, polyTxHash string, rawAuditPath []byte) bool {
	var (
		sigs       []byte
		headerData []byte
	)
	if anchorHeader != nil && headerProof != "" {
		for _, sig := range anchorHeader.SigData {
			temp := make([]byte, len(sig))
			copy(temp, sig)
			newsig, _ := signature.ConvertToEthCompatible(temp)
			sigs = append(sigs, newsig...)
		}
	} else {
		for _, sig := range header.SigData {
			temp := make([]byte, len(sig))
			copy(temp, sig)
			newsig, _ := signature.ConvertToEthCompatible(temp)
			sigs = append(sigs, newsig...)
		}
	}

	eccdAddr := ethcommon.HexToAddress(this.config.ETHConfig.ECCDContractAddress)
	eccd, err := eccd_abi.NewEthCrossChainData(eccdAddr, this.ethClient)
	if err != nil {
		panic(fmt.Errorf("failed to new CCM: %v", err))
	}
	fromTx := [32]byte{}
	copy(fromTx[:], param.TxHash[:32])
	res, _ := eccd.CheckIfFromChainTxExist(nil, param.FromChainID, fromTx)
	if res {
		log.Debugf("already relayed to starcoin: ( from_chain_id: %d, from_txhash: %x,  param.Txhash: %x)",
			param.FromChainID, param.TxHash, param.MakeTxParam.TxHash)
		return true
	}
	//log.Infof("poly proof with header, height: %d, key: %s, proof: %s", header.Height-1, string(key), proof.AuditPath)

	rawProof, _ := hex.DecodeString(headerProof)
	var rawAnchor []byte
	if anchorHeader != nil {
		rawAnchor = anchorHeader.GetMessage()
	}
	headerData = header.GetMessage()
	txData, err := this.contractAbi.Pack("verifyHeaderAndExecuteTx",
		rawAuditPath, // Poly chain tx merkle proof
		headerData,   // The header containing crossStateRoot to verify the above tx merkle proof
		rawProof,     // The header merkle proof used to verify rawHeader
		rawAnchor,    // Any header in current epoch consensus of Poly chain
		sigs)

	if err != nil {
		log.Errorf("commitDepositEventsWithHeader - err:" + err.Error())
		return false
	}

	gasPrice, err := this.ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		log.Errorf("commitDepositEventsWithHeader - get suggest sas price failed error: %s", err.Error())
		return false
	}
	contractaddr := ethcommon.HexToAddress(this.config.ETHConfig.ECCMContractAddress)
	callMsg := ethereum.CallMsg{
		From: this.acc.Address, To: &contractaddr, Gas: 0, GasPrice: gasPrice,
		Value: big.NewInt(0), Data: txData,
	}
	gasLimit, err := this.ethClient.EstimateGas(context.Background(), callMsg)
	if err != nil {
		log.Errorf("commitDepositEventsWithHeader - estimate gas limit error: %s", err.Error())
		return false
	}

	k := this.getRouter()
	c, ok := this.cmap[k]
	if !ok {
		c = make(chan *StarcoinTxInfo, ChanLen)
		this.cmap[k] = c
		go func() {
			for v := range c {
				if err = this.sendTxToEth(v); err != nil {
					log.Errorf("failed to send tx to starcoin: error: %v, txData: %s", err, hex.EncodeToString(v.txData))
				}
			}
		}()
	}
	//TODO: could be blocked
	c <- &StarcoinTxInfo{
		txData:       txData,
		contractAddr: contractaddr,
		gasPrice:     gasPrice,
		gasLimit:     gasLimit,
		polyTxHash:   polyTxHash,
	}
	return true
}

// commitHeader
func (this *StarcoinSender) changeBookKeeper(header *polytypes.Header, pubkList []byte) bool {
	headerdata := header.GetMessage()
	var (
		txData []byte
		txErr  error
		sigs   []byte
	)
	gasPrice, err := this.ethClient.SuggestGasPrice(context.Background()) //todo starcoin...
	if err != nil {
		log.Errorf("changeBookKeeper - get suggest sas price failed error: %s", err.Error())
		return false
	}
	for _, sig := range header.SigData {
		temp := make([]byte, len(sig))
		copy(temp, sig)
		newsig, _ := signature.ConvertToEthCompatible(temp) //todo starcoin...
		sigs = append(sigs, newsig...)
	}

	txData, txErr = this.contractAbi.Pack("changeBookKeeper", headerdata, pubkList, sigs)
	if txErr != nil {
		log.Errorf("changeBookKeeper - err:" + err.Error())
		return false
	}

	contractaddr := ethcommon.HexToAddress(this.config.StarcoinConfig.ECCMContractAddress)
	callMsg := ethereum.CallMsg{
		From: this.acc.Address, To: &contractaddr, Gas: 0, GasPrice: gasPrice,
		Value: big.NewInt(0), Data: txData,
	}

	gasLimit, err := this.ethClient.EstimateGas(context.Background(), callMsg) // todo gasLimit...
	if err != nil {
		log.Errorf("changeBookKeeper - estimate gas limit error: %s", err.Error())
		return false
	}

	nonce := this.seqNumManager.GetAccountSequenceNumber(this.acc.Address)
	tx := types.NewTransaction(nonce, contractaddr, big.NewInt(0), gasLimit, gasPrice, txData)
	signedtx, err := this.keyStore.SignTransaction(tx, this.acc)
	if err != nil {
		log.Errorf("changeBookKeeper - sign raw tx error: %s", err.Error())
		return false
	}
	if err = this.starcoinClient.SendTransaction(context.Background(), signedtx); err != nil {
		log.Errorf("changeBookKeeper - send transaction error:%s\n", err.Error())
		return false
	}

	hash := header.Hash()
	txhash := signedtx.Hash()
	isSuccess := this.waitTransactionConfirm(fmt.Sprintf("header: %d", header.Height), txhash)
	if isSuccess {
		log.Infof("successful to relay poly header to starcoin: (header_hash: %s, height: %d, eth_txhash: %s, nonce: %d, eth_explorer: %s)",
			hash.ToHexString(), header.Height, txhash.String(), nonce, tools.GetExplorerUrl(this.keyStore.GetChainId())+txhash.String())
	} else {
		log.Errorf("failed to relay poly header to starcoin: (header_hash: %s, height: %d, eth_txhash: %s, nonce: %d, eth_explorer: %s)",
			hash.ToHexString(), header.Height, txhash.String(), nonce, tools.GetExplorerUrl(this.keyStore.GetChainId())+txhash.String())
	}
	return true
}

func (this *StarcoinSender) sendTxToEth(txInfo *StarcoinTxInfo) error {
	nonce := this.seqNumManager.GetAccountSequenceNumber(this.acc.Address)
	tx := types.NewTransaction(nonce, txInfo.contractAddr, big.NewInt(0), txInfo.gasLimit, txInfo.gasPrice, txInfo.txData)
	signedtx, err := this.keyStore.SignTransaction(tx, this.acc)
	if err != nil {
		this.seqNumManager.ReturnSeqNum(this.acc.Address, nonce)
		return fmt.Errorf("commitDepositEventsWithHeader - sign raw tx error and return nonce %d: %v", nonce, err)
	}
	err = this.starcoinClient.SendTransaction(context.Background(), signedtx) //todo starcoin...
	if err != nil {
		this.seqNumManager.ReturnSeqNum(this.acc.Address, nonce)
		return fmt.Errorf("commitDepositEventsWithHeader - send transaction error and return nonce %d: %v", nonce, err)
	}
	hash := signedtx.Hash()

	isSuccess := this.waitTransactionConfirm(txInfo.polyTxHash, hash)
	if isSuccess {
		log.Infof("successful to relay tx to starcoin: (eth_hash: %s, nonce: %d, poly_hash: %s, eth_explorer: %s)",
			hash.String(), nonce, txInfo.polyTxHash, tools.GetExplorerUrl(this.keyStore.GetChainId())+hash.String())
	} else {
		log.Errorf("failed to relay tx to starcoin: (eth_hash: %s, nonce: %d, poly_hash: %s, eth_explorer: %s)",
			hash.String(), nonce, txInfo.polyTxHash, tools.GetExplorerUrl(this.keyStore.GetChainId())+hash.String())
	}
	return nil
}

func (this *StarcoinSender) getRouter() string {
	return strconv.FormatInt(rand.Int63n(this.config.RoutineNum), 10)
}

func (this *StarcoinSender) Balance() (*big.Int, error) {
	balance, err := this.ethClient.BalanceAt(context.Background(), this.acc.Address, nil) //todo starcoin...
	if err != nil {
		return nil, err
	}
	return balance, nil
}

// func (this *StarcoinSender) waitTransactionConfirm(polyTxHash string, hash ethcommon.Hash) bool { //todo starcoin...
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

type StarcoinTxInfo struct {
	txData       []byte
	gasLimit     uint64
	gasPrice     *big.Int
	contractAddr stctypes.AccountAddress //ethcommon.Address
	polyTxHash   string
}
