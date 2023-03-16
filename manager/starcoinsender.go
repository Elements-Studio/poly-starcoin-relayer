package manager

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/db"
	"github.com/elements-studio/poly-starcoin-relayer/log"
	stcpoly "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology-crypto/signature"
	"github.com/polynetwork/poly/common"
	vconfig "github.com/polynetwork/poly/consensus/vbft/config"
	polytypes "github.com/polynetwork/poly/core/types"
	common2 "github.com/polynetwork/poly/native/service/cross_chain_manager/common"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	diemtypes "github.com/starcoinorg/starcoin-go/types"
	"math/big"
	"math/rand"
	"strconv"
)

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
