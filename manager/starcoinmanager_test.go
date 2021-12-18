package manager

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/db"
	"github.com/elements-studio/poly-starcoin-relayer/starcoin/poly/events"
	stcpolyevts "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly/events"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	polysdk "github.com/polynetwork/poly-go-sdk"
	pcommon "github.com/polynetwork/poly/common"
	common2 "github.com/polynetwork/poly/native/service/cross_chain_manager/common"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	"github.com/starcoinorg/starcoin-go/types"
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
	fmt.Println("---------- fetchLockDepositEvents -----------")
	ok, err := starcoinManager.fetchLockDepositEvents(3434)
	fmt.Println(ok)
	fmt.Println(err)

	rl, es, err := starcoinManager.db.GetAllStarcoinTxRetry()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(len(rl))
	fmt.Println(rl)
	fmt.Println(es)

}

func TestCommitProof(t *testing.T) {
	height := 40741
	//---------------- RawData -----------------
	rawData := "100000000000000000000000000000000120a0f00e61f7aeab63429ee742e321d5783611b1a60c7c0850625b86fa4c6dc16e102d81a0427d64ff61b11ede9085efa5adda0000000000000034307832643831613034323764363466663631623131656465393038356566613561643a3a43726f7373436861696e53637269707406756e6c6f636b3f0d3078313a3a5354433a3a53544310e498d62f5d1f469d2f72eb3e9dc8f230204e000000000000000000000000000000000000000000000000000000000000"
	//---------------- Starcoin Transaction Hash -----------------
	txHash := "0xd0d79fd03a490376aea99f5a6338dec0bef054b32c939ba8c31203739a9ff8b7"
	starcoinManager := getTestStarcoinManager(t)
	fmt.Println(starcoinManager)
	dataBS, _ := tools.HexToBytes(rawData)
	txHashBS, _ := tools.HexToBytes(txHash)
	fmt.Println("----------------- commmit proof -----------------")
	proof := `{"accountProof":[]}`
	headerOrCrossChainMsg := []byte{} //`{}`
	r, err := starcoinManager.commitProof(uint32(height), []byte(proof), dataBS, txHashBS, headerOrCrossChainMsg)
	fmt.Println("---------------- poly transaction hash ------------------")
	fmt.Println(r)
	fmt.Println(err)
	if err != nil {
		t.FailNow()
	}
}

func TestGetPolySmartContractEvent(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	//fmt.Println(starcoinManager)
	k := "0a2a6502415f878d8866ae3b7d646327ce28fe3c592f7f08091c6ed6db4e55ac"
	e, err := starcoinManager.polySdk.GetSmartContractEvent(k)
	fmt.Println(e)
	fmt.Println("------------- event.State -------------")
	fmt.Println(e.State)
	fmt.Println("------------- event.Notify -------------")
	fmt.Println(e.Notify)
	fmt.Println(err)

	h, err := starcoinManager.polySdk.GetBlockHeightByTxHash(k)
	fmt.Println("------------ height(by tx. hash) ---------------")
	fmt.Println(h)
}

func TestDeserializeCrossChainEventData(t *testing.T) {
	evtData := "0x102d81a0427d64ff61b11ede9085efa5ad100000000000000000000000000000000035307832643831613034323764363466663631623131656465393038356566613561643a3a43726f7373436861696e4d616e61676572da0000000000000034307832643831613034323764363466663631623131656465393038356566613561643a3a43726f7373436861696e536372697074c7011000000000000000000000000000000000203fa1016c3440ad9c0290a4abbe24fc9e994c6879f48346ab4ddc54aec3b07219102d81a0427d64ff61b11ede9085efa5adda0000000000000034307832643831613034323764363466663631623131656465393038356566613561643a3a43726f7373436861696e53637269707406756e6c6f636b3f0d3078313a3a5354433a3a53544310e498d62f5d1f469d2f72eb3e9dc8f2301027000000000000000000000000000000000000000000000000000000000000"
	bs, _ := tools.HexToBytes(evtData)
	ccEvent, err := stcpolyevts.BcsDeserializeCrossChainEvent(bs)
	fmt.Println(err)
	if err != nil {
		t.FailNow()
	}
	fmt.Println(ccEvent)
	fmt.Println("-------------------- event.RawData ------------------")
	fmt.Println(hex.EncodeToString(ccEvent.RawData))
}

func TestGetToMerkleValueFromProof(t *testing.T) {
	p, err := hex.DecodeString("f0202d052233fd5ae70d16898ca3eb40f55adbccc3dfe34e362c4bec50ec161c3461da000000000000001000000000000000000000000000000000209b6092ccf4b2dfde71914755713ff1a550a71f1d6ef0d82c00958504c5f7d6591057aa381a5d7c0141da3965393eed9958da0000000000000034307835376161333831613564376330313431646133393635333933656564393935383a3a43726f7373436861696e53637269707406756e6c6f636b3f0d3078313a3a5354433a3a53544310e498d62f5d1f469d2f72eb3e9dc8f23087d6120000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.FailNow()
	}
	ps := pcommon.NewZeroCopySource(p)
	d, _ := ps.NextVarBytes()
	fmt.Println(d)
	fmt.Println(hex.EncodeToString(d))

	param := &common2.ToMerkleValue{}
	if err = param.Deserialization(pcommon.NewZeroCopySource(d)); err != nil {
		//log.Errorf("handleDepositEvents - failed to deserialize MakeTxParam (value: %x, err: %v)", value, err)
		fmt.Print(err)
		t.FailNow()
	}
	fmt.Println(param)
	fmt.Println(hex.EncodeToString(param.TxHash))

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
	rawData := "100000000000000000000000000000000120a0f00e61f7aeab63429ee742e321d5783611b1a60c7c0850625b86fa4c6dc16e102d81a0427d64ff61b11ede9085efa5adda0000000000000034307832643831613034323764363466663631623131656465393038356566613561643a3a43726f7373436861696e53637269707406756e6c6f636b3f0d3078313a3a5354433a3a53544310e498d62f5d1f469d2f72eb3e9dc8f230204e000000000000000000000000000000000000000000000000000000000000"
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

}

func TestDeserializeCrossChainEvent(t *testing.T) {
	ehex := "0x00180500000000000000e498d62f5d1f469d2f72eb3e9dc8f230020000000000000007e498d62f5d1f469d2f72eb3e9dc8f2301143726f7373436861696e4d616e616765720f43726f7373436861696e4576656e7400de0210e498d62f5d1f469d2f72eb3e9dc8f230100000000000000000000000000000000235307865343938643632663564316634363964326637326562336539646338663233303a3a43726f7373436861696e4d616e61676572da0000000000000034307865343938643632663564316634363964326637326562336539646338663233303a3a43726f7373436861696e536372697074c7011000000000000000000000000000000002208f5e5f785723333b2ab129a7928d1d47129a4df840da708d25086f1a361e0f6910e498d62f5d1f469d2f72eb3e9dc8f230da0000000000000034307865343938643632663564316634363964326637326562336539646338663233303a3a43726f7373436861696e53637269707406756e6c6f636b3f0d3078313a3a5354433a3a535443102d81a0427d64ff61b11ede9085efa5ad1027000000000000000000000000000000000000000000000000000000000000"
	ebs, err := tools.HexToBytes(ehex)
	if err != nil {
		t.FailNow()
	}
	evt, err := types.BcsDeserializeContractEvent(ebs)
	if err != nil {
		t.FailNow()
	}
	fmt.Println(evt)
	var ev0 types.ContractEventV0
	switch evt.(type) {
	case *types.ContractEvent__V0:
		ev0 = evt.(*types.ContractEvent__V0).Value
	default:
		t.FailNow()
	}
	evtData, err := events.BcsDeserializeCrossChainEvent(ev0.EventData)
	if err != nil {
		t.FailNow()
	}
	fmt.Println(evtData)
	rawDataSrc := pcommon.NewZeroCopySource(evtData.RawData)
	txParam := new(common2.MakeTxParam)
	if err := txParam.Deserialization(rawDataSrc); err != nil {
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

func TestMisc(t *testing.T) {
	// var proof []byte = []byte("{}")
	// fmt.Print(proof)

	sink := pcommon.NewZeroCopySink(nil)
	sink.WriteUint64(1)
	fmt.Println(hex.EncodeToString(sink.Bytes()))

}

func TestGetStarcoinHeaderInPoly(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	var height uint64 = 222623
	blockCount := 1
	for i := 0; i < blockCount; i++ {

		hdrOnChain, err := starcoinManager.client.HeaderWithDifficultyInfoByNumber(context.Background(), height)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		j, err := json.Marshal(hdrOnChain)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		fmt.Println("--------------- get starcoin block header on-chain ----------------")
		fmt.Println(string(j))
		hdrhash, err := hdrOnChain.BlockHeader.Hash()
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		fmt.Println("Calculated Hash in starcoin: " + hex.EncodeToString(hdrhash))

		// ////////////////////////////////////////

		//fmt.Println("--------------- get starcoin block header in poly (by height) ---------------")
		//hdr, err := getStarcoinHeaderInPoly(starcoinManager.polySdk, starcoinManager.config.StarcoinConfig.SideChainId, height)
		// get by hash
		fmt.Println("--------------- get starcoin block header in poly (by hash) ---------------")
		hdr, err := getStarcoinHeaderInPolyByHash(starcoinManager.polySdk, starcoinManager.config.StarcoinConfig.SideChainId, hdrhash)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		//fmt.Println(hex.EncodeToString(hdr))
		h, err := types.BcsDeserializeBlockHeader(hdr)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		fmt.Printf("Height(number): %d\n", h.Number)
		fmt.Printf("Timestamp: %d\n", h.Timestamp)
		fmt.Printf("ParentHash: %s\n", tools.EncodeToHex(h.ParentHash[:]))
		fmt.Printf("Difficulty: %s\n", tools.EncodeToHex(h.Difficulty[:]))
		fmt.Printf("StateRoot: %s\n", tools.EncodeToHex(h.StateRoot[:]))
		fmt.Println("--------------- get starcoin block hash in poly (by height) ---------------")
		hdrhashInPoly, err := getStarcoinHeaderHashInPoly(starcoinManager.polySdk, starcoinManager.config.StarcoinConfig.SideChainId, height)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		fmt.Printf("HeaderHash: %s\n", tools.EncodeToHex(hdrhashInPoly))

		height++
	}
}

func TestGetBlockHeaderInPolyByHash(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	hdrhash, err := tools.HexToBytes("0x3b6f3a5bb470a45e4870d13ada0947dec4504259cf3aa0eeebadf48f66d74995")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	hdr, err := getStarcoinHeaderInPolyByHash(starcoinManager.polySdk, starcoinManager.config.StarcoinConfig.SideChainId, hdrhash)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	//fmt.Println(hex.EncodeToString(hdr))
	h, err := types.BcsDeserializeBlockHeader(hdr)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("--------------- get starcoin block header in poly (by hash) ---------------")
	fmt.Println(hex.EncodeToString(hdr))
	fmt.Printf("Height(number): %d\n", h.Number)
	fmt.Printf("Timestamp: %d\n", h.Timestamp)
	fmt.Printf("ParentHash: %s\n", tools.EncodeToHex(h.ParentHash[:]))
	fmt.Printf("Difficulty: %s\n", tools.EncodeToHex(h.Difficulty[:]))
	fmt.Printf("StateRoot: %s\n", tools.EncodeToHex(h.StateRoot[:]))
}

func TestGetBlockHeaders(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	var height uint64 = 222625
	blockCount := 25
	var hdrs = make([]*stcclient.BlockHeader, 0, blockCount)
	for i := 0; i < blockCount; i++ {
		h := height - uint64(i)
		hdr, err := starcoinManager.client.HeaderByNumber(context.Background(), h)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		//fmt.Println(hdr)
		hdrs = append(hdrs, hdr)
	}
	j, err := json.Marshal(hdrs)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("--------------- starcoin block headers ----------------")
	fmt.Println(string(j))

	filePath := "testjson.json"
	writeTextFile(filePath, string(j), t)
}

func TestGetBlockHeaderAndBlockInfoByNumber(t *testing.T) {
	starcoinManager := getTestStarcoinManager(t)
	var height uint64 = 291946
	h, err := starcoinManager.client.GetBlockHeaderAndBlockInfoByNumber(context.Background(), height)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	j, err := json.Marshal(h)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(string(j))
	//fmt.Println(hex.EncodeToString(j))
	// /////////////////////////////////////////////////////
	//note: poly may use this hex to init genesis...
	filePath := fmt.Sprintf("blockHeaderAndBlockInfoHex-%d.txt", height)
	writeTextFile(filePath, hex.EncodeToString(j), t)
	// /////////////////////////////////////////////////////
}

func writeTextFile(filePath string, content string, t *testing.T) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	defer file.Close()
	_, err = file.WriteString(content)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
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

func TestMisc2(t *testing.T) {
	s := "{\"header\":{\"timestamp\":\"1639375200198\",\"author\":\"0x00e4ea282432073992bc04ab278ddd60\",\"author_auth_key\":null,\"block_accumulator_root\":\"0xfa55091e7f19023cd70d55bc147c194d09649585ac90cade4898302530c50bda\",\"block_hash\":\"0xb6c0a3c14df4133e5ce8b89f7adff3add41e1df10b818da39c8eab54f26225cb\",\"body_hash\":\"0xc01e0329de6d899348a8ef4bd51db56175b3fa0988e57c3dcec8eaf13a164d97\",\"chain_id\":253,\"difficulty\":\"0x80\",\"difficulty_number\":0,\"extra\":\"0x00000000\",\"gas_used\":\"0\",\"Nonce\":3108099670,\"number\":\"222625\",\"parent_hash\":\"0xf976fea99030c3442508b6deac2596b338d9dc9d3a2bcc886ebed1bcd70b1fce\",\"state_root\":\"0xa0f7a539ecaeabe08e47ba2a11e698684f75db18e623cacbd4dd83724bf4a945\",\"txn_accumulator_root\":\"0x0b4bbaefcb7a509b32ae41681b39ad6e4917e79220aa2883d6b995b7f94b55c0\"},\"block_time_target\":5000,\"block_difficulty_window\":24,\"block_info\":{\"block_id\":\"0xb6c0a3c14df4133e5ce8b89f7adff3add41e1df10b818da39c8eab54f26225cb\",\"total_difficulty\":\"0x029bb161\",\"txn_accumulator_info\":{\"accumulator_root\":\"0x0b4bbaefcb7a509b32ae41681b39ad6e4917e79220aa2883d6b995b7f94b55c0\",\"frozen_subtree_roots\":[\"0x0e475fde7a9b246667cb2959040806f7fc1c3b838bc57ac7fb7ffdcf2cd83e09\",\"0xb8430591e9bc195ba37f3fe547bf17c811329ba4502c7026b23dd90412cc8d20\",\"0x8483fe396477fabde168d2fc7157f4da104b1b0bdb24546106e2431394e440cf\",\"0x568c93a6d640e8914cd84e34bf503cc9b44f13bff570c97046676560b4a33643\",\"0x3b9537dcce9b09f0f86a3bb53c850e9bdfc9cc7e319ab03dd78073730b5aea4c\",\"0x460e665c61bec4e9d82c793c5fbe16f442fb81c8938e63519450b419eaedd271\",\"0xe7ce04f5e738da78c33cdd1ea85b0b2af31cf3b1bf153b047114fb0ac6d88228\",\"0x33a7a75916d27fc243a0192b1840c9ebf490c03c3b86606d670e751c43934f05\",\"0x11f81290ce50e7adfd1558c93c98609d631e9a4b97e670b46e15056543080f83\",\"0x2e9bff7bb711c4ef7037d9ff4daf9068a4789b0b74c0c94e92a71dd732e945f3\",\"0xc3e977dafca6a6b1070abb034ba07bcc4f43f8a117706d55108a7f88ed12073f\",\"0xba183643e4de7f9b39253967c7b93bd60e609f7969d12cb107267e8313b4753c\"],\"num_leaves\":254271,\"num_nodes\":508530},\"block_accumulator_info\":{\"accumulator_root\":\"0x1d2d1802b1468edf403fced476ee2b97349bfec24dcd05822909acba2b49d3f4\",\"frozen_subtree_roots\":[\"0xb30a1da75cb78d9a842d9deaa43c9a3262cf0744ac5ccf23e84880da2de84df0\",\"0xda5f9b05b4e56cbd6fe53395ea2f195fc5f6ede7050dfad22d4e723d31c9add5\",\"0xbb503e3c2c6aa00b146ae282080e5072a4c98048242e8c40636a3b0d7009f511\",\"0x6c10758b358dd4d1ede5e626c4fd1ac2722cb5adf23532eb1f582f44acddfa39\",\"0x7de9f8440ff2ad23242fb36dde4de0c2158f2abdd32052f7e61e71d9a90696a2\",\"0x1b57796a2df27f33adc2e97e1263e041d19ddb1a36be8a62100e36c5a3eadab4\",\"0x46f68f4e616c94dedad1a5050f78982ac0e0792b4c7669cabc0a07d6762267f2\",\"0xa22e7d51a7352eec7246ce6441b215ab0d3cabcaea247c19a28ff587a4b1541a\",\"0xca68f6f0740fd4a13cb3ebba15fc74c907f4c813d779652ef3019f49d66ee71d\"],\"num_leaves\":222626,\"num_nodes\":445243}}}"
	fmt.Println(s)
}
