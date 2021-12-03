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
	"github.com/elements-studio/poly-starcoin-relayer/db"
	"github.com/elements-studio/poly-starcoin-relayer/log"
	stcpoly "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly"
	"github.com/elements-studio/poly-starcoin-relayer/tools"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology-crypto/signature"

	polysdk "github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly/common"
	vconfig "github.com/polynetwork/poly/consensus/vbft/config"
	polytypes "github.com/polynetwork/poly/core/types"
	common2 "github.com/polynetwork/poly/native/service/cross_chain_manager/common"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	diemtypes "github.com/starcoinorg/starcoin-go/types"
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
	db             db.DB
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

	return &PolyManager{
		exitChan:      make(chan int),
		config:        servCfg,
		polySdk:       polySdk,
		currentHeight: startblockHeight,
		//contractAbi:   &contractabi,
		db:             db,
		starcoinClient: stcclient,
		senders:        senders,
	}, nil
}

func (this *PolyManager) init() bool {
	if this.currentHeight > 0 {
		log.Infof("PolyManager init - start height from flag: %d", this.currentHeight)
		return true
	}
	this.currentHeight, _ = this.db.GetPolyHeight() // todo handle db error...
	curEpochStart := this.findCurEpochStartHeight()
	if curEpochStart > this.currentHeight {
		this.currentHeight = curEpochStart
		log.Infof("PolyManager init - latest height from CCD: %d", this.currentHeight)
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
				if len(this.config.ProxyOrAssetContracts) > 0 {
					//proxyOrAssetContract := ethcommon.BytesToAddress(param.MakeTxParam.ToContractAddress).String()
					proxyOrAssetContract := string(param.MakeTxParam.ToContractAddress) // starcoin module(address and name)...
					for _, v := range this.config.ProxyOrAssetContracts {
						chainIdArrMap, ok := v[proxyOrAssetContract]
						if ok {
							if len(chainIdArrMap["inbound"]) == 0 {
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
						continue
					}
				}
				cnt++
				sender := this.selectSender()
				log.Infof("sender %s is handling poly tx ( hash: %s, height: %d )",
					tools.EncodeToHex(sender.acc.Address[:]), event.TxHash, height)
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
		return false, nil, fmt.Errorf("IsEpoch - unmarshal blockInfo error: %s", err.Error())
	}
	if hdr.NextBookkeeper == common.ADDRESS_EMPTY || blkInfo.NewChainConfig == nil {
		return false, nil, nil
	}

	//eccdAddr := ethcommon.HexToAddress(this.config.StarcoinConfig.CCDContractAddress)
	//eccd, err := eccd_abi.NewEthCrossChainData(eccdAddr, this.ethClient)
	ccd := NewCrossChainData(this.starcoinClient, this.config.StarcoinConfig.CCDModule)
	// if err != nil {
	// 	return false, nil, fmt.Errorf("failed to new CCD: %v", err)
	// }
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

func (this *PolyManager) InitGenesis(height *uint32) error {
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
	senderAndPK := this.config.StarcoinConfig.PrivateKeys[0]
	senderAddress, senderPrivateKey, err := getAccountAddressAndPrivateKey(senderAndPK)
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
	fmt.Println("---------------------- hdr ----------------------")
	rawHdr := hdr.GetMessage()
	//fmt.Println(len(rawHdr))
	//fmt.Println(rawHdr)
	fmt.Println(hex.EncodeToString(rawHdr))
	fmt.Println("------------------ publickeys -------------------")
	//fmt.Println(len(publickeys))
	fmt.Println(publickeys)
	fmt.Println(hex.EncodeToString(publickeys))
	fmt.Println("---------------- header.SigData ------------------")
	fmt.Println(hex.EncodeToString(encodeHeaderSigData(hdr)))
	fmt.Println("--------------------------------------------------")
	txPayload := stcpoly.EncodeInitGenesisTxPayload(this.config.StarcoinConfig.CCMModule, hdr.GetMessage(), publickeys)

	userTx, err := this.starcoinClient.BuildRawUserTransaction(context.Background(), *senderAddress, txPayload, gasPrice, stcclient.DEFAULT_MAX_GAS_AMOUNT*4, seqNum)
	if err != nil {
		log.Errorf("InitGenesis - BuildRawUserTransaction error:%s", err.Error())
		return err
	}
	txHash, err := this.starcoinClient.SubmitTransaction(context.Background(), senderPrivateKey, userTx)
	if err != nil {
		log.Errorf("InitGenesis - SubmitTransaction error:%s", err.Error())
		return err
	}
	fmt.Println("--------------- starcoin tx hash ----------------")
	fmt.Println(txHash)
	err = this.db.UpdatePolyHeight(cfgBlockNum)
	if err != nil {
		log.Errorf("InitGenesis - UpdatePolyHeight error:%s", err.Error())
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

func (this *PolyManager) findCurEpochStartHeight() uint32 {
	//ethcommon.HexToAddress(this.config.StarcoinConfig.ECCDContractAddress)
	//instance, err := eccd_abi.NewEthCrossChainData(address, this.ethClient)
	instance := NewCrossChainData(this.starcoinClient, this.config.StarcoinConfig.CCDModule)
	// if err != nil {
	// 	log.Errorf("findCurEpochStartHeight - new eth cross chain failed: %s", err.Error())
	// 	return 0
	// }
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

	// eccdAddr := ethcommon.HexToAddress(this.config.ETHConfig.ECCDContractAddress)
	// eccd, err := eccd_abi.NewEthCrossChainData(eccdAddr, this.ethClient)
	ccd := NewCrossChainData(this.starcoinClient, this.config.StarcoinConfig.CCDModule)
	// if err != nil {
	// 	panic(fmt.Errorf("failed to new CCM: %v", err))
	// }
	fromTx := [32]byte{}
	copy(fromTx[:], param.TxHash[:32])
	res, _ := ccd.checkIfFromChainTxExist(param.FromChainID, fromTx[:])
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

	polyTx := db.PolyTx{
		TxHash:       polyTxHash,
		Proof:        hex.EncodeToString(rawAuditPath),
		Header:       hex.EncodeToString(headerData),
		HeaderProof:  hex.EncodeToString(rawProof),
		AnchorHeader: hex.EncodeToString(rawAnchor),
		HeaderSig:    hex.EncodeToString(sigs),
		RootHash:     "", //todo
		NonIncProof:  "", //todo
	}
	_, err := this.db.PutPolyTx(&polyTx)
	if err != nil {
		log.Errorf("commitDepositEventsWithHeader - db.PutPolyTx error: %s", err.Error())
		return false
	}

	//todo ???
	return this.sendPolyTxToStarcoin(&polyTx)
}

func (this *StarcoinSender) sendPolyTxToStarcoin(polyTx *db.PolyTx) bool {

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
	c, ok := this.cmap[k]
	if !ok {
		c = make(chan *StarcoinTxInfo, ChanLen)
		this.cmap[k] = c
		go func() {
			for v := range c {
				if err := this.sendTxToStarcoin(v); err != nil {
					txBytes, _ := v.txPayload.BcsSerialize()
					log.Errorf("failed to send tx to starcoin: error: %v, txData: %s", err, hex.EncodeToString(txBytes))
				}
			}
		}()
	}
	//TODO: could be blocked
	c <- stcTxInfo
	return true
}

func (this *StarcoinSender) polyTxToStarcoinTxInfo(polyTx *db.PolyTx) (*StarcoinTxInfo, error) {
	polyTxHash := polyTx.TxHash
	rawAuditPath, err := hex.DecodeString(polyTx.Proof)
	if err != nil {
		log.Errorf("sendPolyTxToStarcoin - hex.DecodeString error: %s", err.Error())
		return nil, err
	}
	headerData, err := hex.DecodeString(polyTx.Header)
	if err != nil {
		log.Errorf("sendPolyTxToStarcoin - hex.DecodeString error: %s", err.Error())
		return nil, err
	}
	rawProof, err := hex.DecodeString(polyTx.HeaderProof)
	if err != nil {
		log.Errorf("sendPolyTxToStarcoin - hex.DecodeString error: %s", err.Error())
		return nil, err
	}
	rawAnchor, err := hex.DecodeString(polyTx.AnchorHeader)
	if err != nil {
		log.Errorf("sendPolyTxToStarcoin - hex.DecodeString error: %s", err.Error())
		return nil, err
	}
	sigs, err := hex.DecodeString(polyTx.HeaderSig)
	if err != nil {
		log.Errorf("sendPolyTxToStarcoin - hex.DecodeString error: %s", err.Error())
		return nil, err
	}

	txPayload := stcpoly.EncodeCCMVerifyHeaderAndExecuteTxPayload(this.config.StarcoinConfig.CCMModule,
		rawAuditPath,
		headerData,
		rawProof,
		rawAnchor,
		sigs)

	return &StarcoinTxInfo{
		txPayload: txPayload,
		//contractAddr: contractaddr,
		//gasPrice:   gasPrice,
		//gasLimit:   stcclient.DEFAULT_MAX_GAS_AMOUNT, //gasLimit,
		polyTxHash: polyTxHash,
	}, nil
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
	// gasLimit, err := this.ethClient.EstimateGas(context.Background(), callMsg) // todo gasLimit...
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
	//todo use max gas???
	if err != nil {
		log.Errorf("changeBookKeeper - BuildRawUserTransaction error: %s", err.Error())
		return false
	}
	var txhash string
	//txhash := signedtx.Hash() //todo cal txhash self???
	if txhash, err = this.starcoinClient.SubmitTransaction(context.Background(), this.keyStore.GetPrivateKey(), rawUserTx); err != nil {
		log.Errorf("changeBookKeeper - send transaction error:%s\n", err.Error())
		return false
	}

	hdrhash := header.Hash()
	isSuccess, err := tools.WaitTransactionConfirm(*this.starcoinClient, txhash, time.Second*20)
	//todo handle error???
	if isSuccess {
		log.Infof("successful to relay poly header to starcoin: (header_hash: %s, height: %d, starcoin_txhash: %s, nonce: %d, starcoin_explorer: %s)",
			hdrhash.ToHexString(), header.Height, txhash, nonce, tools.GetExplorerUrl(this.keyStore.GetChainId())+txhash)
	} else {
		log.Errorf("failed to relay poly header to starcoin: (header_hash: %s, height: %d, starcoin_txhash: %s, nonce: %d, starcoin_explorer: %s)",
			hdrhash.ToHexString(), header.Height, txhash, nonce, tools.GetExplorerUrl(this.keyStore.GetChainId())+txhash)
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
	//todo use max gas???
	if err != nil {
		log.Errorf("sendTxToStarcoin - BuildRawUserTransaction error: %s", err.Error())
		return err
	}
	var txhash string
	if txhash, err = this.starcoinClient.SubmitTransaction(context.Background(), this.keyStore.GetPrivateKey(), rawUserTx); err != nil {
		log.Errorf("sendTxToStarcoin - submit transaction error:%s\n", err.Error())
		return err
	}
	//todo cal txhash self???

	//isSuccess := this.waitTransactionConfirm(txInfo.polyTxHash, hash)
	isSuccess, err := tools.WaitTransactionConfirm(*this.starcoinClient, txhash, time.Second*20)
	//todo hanlde error
	if isSuccess {
		log.Infof("successful to relay tx to starcoin: (starcoin_hash: %s, nonce: %d, poly_hash: %s, starcoin_explorer: %s)",
			txhash, nonce, txInfo.polyTxHash, tools.GetExplorerUrl(this.keyStore.GetChainId())+txhash)
	} else {
		log.Errorf("failed to relay tx to starcoin: (starcoin_hash: %s, nonce: %d, poly_hash: %s, starcoin_explorer: %s)",
			txhash, nonce, txInfo.polyTxHash, tools.GetExplorerUrl(this.keyStore.GetChainId())+txhash)
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
	polyTxHash string
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
		keystr, _ := hex.DecodeString(peer.ID)
		key, _ := keypair.DeserializePublicKey(keystr)
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
		newsig, _ := signature.ConvertToEthCompatible(temp)
		sigs = append(sigs, newsig...)
	}
	return sigs
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
