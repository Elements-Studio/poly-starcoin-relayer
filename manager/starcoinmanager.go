package manager

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/log"
	"github.com/elements-studio/poly-starcoin-relayer/tools"

	//"github.com/ethereum/go-ethereum/accounts/abi/bind" //todo remove this
	"github.com/ontio/ontology/smartcontract/service/native/cross_chain/cross_chain_manager"
	//"github.com/polynetwork/bridge-common/abi/eccm_abi" //todo remove this

	evttypes "github.com/elements-studio/poly-starcoin-relayer/bifrost/types"
	polysdk "github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly/common"
	common2 "github.com/polynetwork/poly/native/service/cross_chain_manager/common"
	scom "github.com/polynetwork/poly/native/service/header_sync/common"
	autils "github.com/polynetwork/poly/native/service/utils"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

type StarcoinManager struct {
	client      stcclient.StarcoinClient
	polySdk     polysdk.PolySdk
	config      config.ServiceConfig
	header4sync [][]byte
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

func (this *StarcoinManager) handleBlockHeader(height uint64) bool {
	block, err := this.client.GetBlockByNumber(int(height))
	if err != nil {
		log.Errorf("handleBlockHeader - GetNodeHeader on height :%d failed", height)
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
		err = this.db.PutRetry(sink.Bytes())
		if err != nil {
			log.Errorf("fetchLockDepositEvents - this.db.PutRetry error: %s", err)
		}
		log.Infof("fetchLockDepositEvent -  height: %d", height)
	}
	return true
}
