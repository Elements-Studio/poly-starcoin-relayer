package manager

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/db"
	stcpolyevts "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly/events"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	polysdk "github.com/polynetwork/poly-go-sdk"
	pcommon "github.com/polynetwork/poly/common"
	common2 "github.com/polynetwork/poly/native/service/cross_chain_manager/common"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestFindSyncedHeight(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	fmt.Println(starcoinManager)
	h := starcoinManager.findSyncedHeight()
	fmt.Println("------------------- findSyncedHeight ------------------")
	fmt.Println(h) // 66856
}

func TestCommitHeader(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	fmt.Println(starcoinManager)
	ok := starcoinManager.handleBlockHeader(67859)
	fmt.Println(ok)
	fmt.Println("-------------------- header4sync --------------------")
	fmt.Println(starcoinManager.header4sync)
	r := starcoinManager.commitHeader()
	fmt.Println("-------------------- header4sync result --------------------")
	// 0 for ok
	fmt.Println(r)
	if r != 0 {
		t.FailNow()
	}
}

func TestFetchLockDepositEvents(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	fmt.Println(starcoinManager)
	ok := starcoinManager.fetchLockDepositEvents(70007)
	fmt.Println(ok)

	rl, err := starcoinManager.db.GetAllStarcoinTxRetry()
	fmt.Println(len(rl))
	fmt.Println(rl)
	fmt.Println(err)
}

func TestCommitProof(t *testing.T) {
	height := 68172
	//---------------- RawData -----------------
	rawData := "1000000000000000000000000000000000203fa1016c3440ad9c0290a4abbe24fc9e994c6879f48346ab4ddc54aec3b07219102d81a0427d64ff61b11ede9085efa5adda0000000000000034307832643831613034323764363466663631623131656465393038356566613561643a3a43726f7373436861696e53637269707406756e6c6f636b3f0d3078313a3a5354433a3a53544310bd7e8be8fae9f60f2f5136433e36a0911027000000000000000000000000000000000000000000000000000000000000"
	//---------------- Starcoin Transaction Hash -----------------
	txHash := "0xf54f1e2855d197f4820d1f270206287e52ccc4433bbefa21595b96fcbf18df67"
	starcoinManager := getTestStarcoinManager(t)
	fmt.Println(starcoinManager)
	dataBS, _ := tools.HexToBytes(rawData)
	txHashBS, _ := tools.HexToBytes(txHash)
	fmt.Println("----------------- commmit proof -----------------")
	proof := `{"accountProof":[]}`
	r, err := starcoinManager.commitProof(uint32(height), []byte(proof), dataBS, txHashBS)
	fmt.Println(r)
	fmt.Println(err)
	if err != nil {
		t.FailNow()
	}
}

func TestDeserializeCrossChainEventData(t *testing.T) {
	evtData := "0x102d81a0427d64ff61b11ede9085efa5ad100000000000000000000000000000000035307832643831613034323764363466663631623131656465393038356566613561643a3a43726f7373436861696e4d616e61676572da0000000000000034307832643831613034323764363466663631623131656465393038356566613561643a3a43726f7373436861696e536372697074c7011000000000000000000000000000000000203fa1016c3440ad9c0290a4abbe24fc9e994c6879f48346ab4ddc54aec3b07219102d81a0427d64ff61b11ede9085efa5adda0000000000000034307832643831613034323764363466663631623131656465393038356566613561643a3a43726f7373436861696e53637269707406756e6c6f636b3f0d3078313a3a5354433a3a53544310bd7e8be8fae9f60f2f5136433e36a0911027000000000000000000000000000000000000000000000000000000000000"
	bs, _ := tools.HexToBytes(evtData)
	ccEvent, err := stcpolyevts.BcsDeserializeCrossChainEvent(bs)
	fmt.Println(err)
	if err != nil {
		t.FailNow()
	}
	fmt.Println(ccEvent)
	fmt.Println(hex.EncodeToString(ccEvent.RawData))
}

func TestDeserializeCrossChainEventRawData2(t *testing.T) {
	rawData := "1000000000000000000000000000000000203fa1016c3440ad9c0290a4abbe24fc9e994c6879f48346ab4ddc54aec3b07219102d81a0427d64ff61b11ede9085efa5adda0000000000000034307832643831613034323764363466663631623131656465393038356566613561643a3a43726f7373436861696e53637269707406756e6c6f636b3f0d3078313a3a5354433a3a53544310bd7e8be8fae9f60f2f5136433e36a0911027000000000000000000000000000000000000000000000000000000000000"
	v, err := hex.DecodeString(rawData)
	//fmt.Println(err)
	if err != nil {
		t.FailNow()
	}
	data := pcommon.NewZeroCopySource(v)
	txParam := new(common2.MakeTxParam)
	if err := txParam.Deserialization(data); err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("---------- MakeTxParam ----------")
	fmt.Println(hex.EncodeToString(txParam.TxHash))              // []byte
	fmt.Println(hex.EncodeToString(txParam.CrossChainID))        // sha256(abi.encodePacked(address(this), paramTxHash))
	fmt.Println(hex.EncodeToString(txParam.FromContractAddress)) // []byte
	fmt.Println(txParam.ToChainID)                               // uint64
	fmt.Println(string(txParam.ToContractAddress))               // []byte
	fmt.Println(txParam.Method)                                  // string
	fmt.Println(hex.EncodeToString(txParam.Args))                // []byte
}
func TestDeserializeCrossChainEventRawData(t *testing.T) {
	// Solidity code:
	//    // Convert the uint256 into bytes
	//    bytes memory paramTxHash = Utils.uint256ToBytes(txHashIndex);
	//    // Construct the makeTxParam, and put the hash info storage, to help provide proof of tx existence
	//    bytes memory rawParam = abi.encodePacked(ZeroCopySink.WriteVarBytes(paramTxHash),
	// 	   ZeroCopySink.WriteVarBytes(abi.encodePacked(sha256(abi.encodePacked(address(this), paramTxHash)))),
	// 	   ZeroCopySink.WriteVarBytes(Utils.addressToBytes(msg.sender)),
	// 	   ZeroCopySink.WriteUint64(toChainId),
	// 	   ZeroCopySink.WriteVarBytes(toContract),
	// 	   ZeroCopySink.WriteVarBytes(method),
	// 	   ZeroCopySink.WriteVarBytes(txData)
	//    );
	rawData := "1000000000000000000000000000000000203fa1016c3440ad9c0290a4abbe24fc9e994c6879f48346ab4ddc54aec3b07219102d81a0427d64ff61b11ede9085efa5adda0000000000000034307832643831613034323764363466663631623131656465393038356566613561643a3a43726f7373436861696e53637269707406756e6c6f636b3f0d3078313a3a5354433a3a53544310bd7e8be8fae9f60f2f5136433e36a0911027000000000000000000000000000000000000000000000000000000000000"
	v, err := hex.DecodeString(rawData)
	//fmt.Println(err)
	if err != nil {
		t.FailNow()
	}
	s := pcommon.NewZeroCopySource(v)
	paramTxHash, b := s.NextVarBytes()
	fmt.Println("-------- paramTxHash: --------")
	fmt.Println(paramTxHash)
	hash, b := s.NextVarBytes()
	fmt.Println("-------- sha256(abi.encodePacked(address(this), paramTxHash)): --------")
	fmt.Println(hex.EncodeToString(hash))
	sender, b := s.NextVarBytes()
	fmt.Println("-------- msg.sender: --------")
	fmt.Println(hex.EncodeToString(sender))
	toChainId, b := s.NextUint64()
	fmt.Println("-------- toChainId: --------")
	fmt.Println(toChainId)
	toContract, b := s.NextVarBytes()
	fmt.Println("-------- toContract: --------")
	fmt.Println(string(toContract))
	method, b := s.NextVarBytes()
	fmt.Println("-------- method: --------")
	fmt.Println(string(method))
	txData, b := s.NextVarBytes()
	fmt.Println("-------- txData: --------")
	fmt.Println(hex.EncodeToString(txData))
	fmt.Println("-------- EOF --------")
	fmt.Println(b)

	// buff = abi.encodePacked(
	// 	ZeroCopySink.WriteVarBytes(args.toAssetHash),
	// 	ZeroCopySink.WriteVarBytes(args.toAddress),
	// 	ZeroCopySink.WriteUint255(args.amount)
	// 	);
	fmt.Println("-------- tx data(args) --------")
	s2 := pcommon.NewZeroCopySource(txData)
	toAssetHash, b := s2.NextVarBytes()
	fmt.Println(string(toAssetHash))
	toAddress, b := s2.NextVarBytes()
	fmt.Println(hex.EncodeToString(toAddress))
	// amount, b := s2.NextVarUint()
	// fmt.Println(amount)

	// Move code:
	// // Tx hash index
	// let param_tx_hash = BCS::to_bytes(&CrossChainData::get_eth_tx_hash_index());
	// raw_param = Bytes::concat(&raw_param, ZeroCopySink::write_var_bytes(&param_tx_hash));

	// let genesis_addr_byte = Address::bytify(CrossChainGlobal::genesis_account());
	// let contract_addr_serialize = Hash::sha3_256(Bytes::concat(&genesis_addr_byte, *&param_tx_hash));
	// raw_param = Bytes::concat(&raw_param, ZeroCopySink::write_var_bytes(&contract_addr_serialize));
	// raw_param = Bytes::concat(&raw_param, ZeroCopySink::write_var_bytes(&Address::bytify(account)));
	// raw_param = Bytes::concat(&raw_param, ZeroCopySink::write_var_uint(to_chain_id));
	// raw_param = Bytes::concat(&raw_param, ZeroCopySink::write_var_bytes(to_contract));
	// raw_param = Bytes::concat(&raw_param, ZeroCopySink::write_var_bytes(method));
	// raw_param = Bytes::concat(&raw_param, ZeroCopySink::write_var_bytes(tx_data));
}

func TestMisc(t *testing.T) {
	// var proof []byte = []byte("{}")
	// fmt.Print(proof)

	sink := pcommon.NewZeroCopySink(nil)
	sink.WriteUint64(1)
	fmt.Println(hex.EncodeToString(sink.Bytes()))

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