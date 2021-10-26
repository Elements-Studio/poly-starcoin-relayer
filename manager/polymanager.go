package manager

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/log"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology-crypto/signature"
	"github.com/polynetwork/bridge-common/abi/eccd_abi" // remove this
	polysdk "github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly/common"
	polytypes "github.com/polynetwork/poly/core/types"
	common2 "github.com/polynetwork/poly/native/service/cross_chain_manager/common"
	stcclient "github.com/starcoinorg/starcoin-go/client"
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

type StarcoinSender struct {
	starcoinClient stcclient.StarcoinClient
	seqNumManager  tools.SeqNumManager
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

	contractaddr := ethcommon.HexToAddress(this.config.ETHConfig.ECCMContractAddress)
	callMsg := ethereum.CallMsg{
		From: this.acc.Address, To: &contractaddr, Gas: 0, GasPrice: gasPrice,
		Value: big.NewInt(0), Data: txData,
	}

	gasLimit, err := this.ethClient.EstimateGas(context.Background(), callMsg)
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
	if err = this.ethClient.SendTransaction(context.Background(), signedtx); err != nil {
		log.Errorf("changeBookKeeper - send transaction error:%s\n", err.Error())
		return false
	}

	hash := header.Hash()
	txhash := signedtx.Hash()
	isSuccess := this.waitTransactionConfirm(fmt.Sprintf("header: %d", header.Height), txhash)
	if isSuccess {
		log.Infof("successful to relay poly header to ethereum: (header_hash: %s, height: %d, eth_txhash: %s, nonce: %d, eth_explorer: %s)",
			hash.ToHexString(), header.Height, txhash.String(), nonce, tools.GetExplorerUrl(this.keyStore.GetChainId())+txhash.String())
	} else {
		log.Errorf("failed to relay poly header to ethereum: (header_hash: %s, height: %d, eth_txhash: %s, nonce: %d, eth_explorer: %s)",
			hash.ToHexString(), header.Height, txhash.String(), nonce, tools.GetExplorerUrl(this.keyStore.GetChainId())+txhash.String())
	}
	return true
}
