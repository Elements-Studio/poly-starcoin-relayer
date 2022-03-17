package manager

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	stcpolyevts "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly/events"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	pcommon "github.com/polynetwork/poly/common"
	common2 "github.com/polynetwork/poly/native/service/cross_chain_manager/common"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

//{"raw":"0x80969800000000000000000000000000","json":{"token":{"value":10000000}}}
type LockTreasuryResource struct {
	Raw  string `json:"raw"`
	Json struct {
		Token struct {
			Value big.Int `json:"value"`
		} `json:"token"`
	} `json:"json"`
}

func TestGetAssetLockedAmount(t *testing.T) {
	starcoinClient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	genesisAccountAddress := "0xb6d69dd935edf7f2054acf12eb884df8"
	// ---------- STC lock Resource -----------
	//resType := "0xb6d69dd935edf7f2054acf12eb884df8::LockProxy::LockTreasury<0x00000000000000000000000000000001::STC::STC>"
	// ---------- XETH lock resource ------------
	resType := "0xb6d69dd935edf7f2054acf12eb884df8::LockProxy::LockTreasury<0xb6d69dd935edf7f2054acf12eb884df8::XETH::XETH>"
	getResOption := stcclient.GetResourceOption{
		Decode: true,
	}
	lockRes := new(LockTreasuryResource) //new(map[string]interface{})
	r, err := starcoinClient.GetResource(context.Background(), genesisAccountAddress, resType, getResOption, lockRes)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	//fmt.Println(r)
	lockRes = r.(*LockTreasuryResource)
	fmt.Println("=============== Locked amount ===============")
	fmt.Println(lockRes.Json.Token.Value.String())
	//13611294676837538537534417817260828458 ETH
	//10000000 STC
}

func TestGetTransactionGas(t *testing.T) {
	starcoinClient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	//txHash := "0x08bc1d3e076c75519f1d2bdea08b980e5e2c7ec62b56136f038012e4333f541a" // GasUsed: 37982971
	//txHash := "0xe151e0e58895915b2c98c213cbe027587aba56c1d0bb27a2b11b5a5ce9f5bfe3" // GasUsed: 18271102
	txHash := "0x2f6ef17220fb4cb23bd21220ba7e52a3ce7e88375daeabd4cad0f156f1888be2"
	tx, _ := starcoinClient.GetTransactionInfoByHash(context.Background(), txHash)
	fmt.Println(string(tx.Status))
	fmt.Println(tx.GasUsed)
}

func TestFetchCrossChainEvent(t *testing.T) {
	starcoinClient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	address := "0xb6d69dd935edf7f2054acf12eb884df8"
	typeTag := "0xb6d69dd935edf7f2054acf12eb884df8::CrossChainManager::CrossChainEvent"
	height := uint64(3421831) //3422033) //3421831) //2977050)
	eventFilter := &stcclient.EventFilter{
		Address:   []string{address},
		TypeTags:  []string{typeTag},
		FromBlock: height,
		ToBlock:   &height,
	}

	events, err := starcoinClient.GetEvents(context.Background(), eventFilter)
	if err != nil {
		fmt.Printf("FetchCrossChainEvent - GetEvents error :%s", err.Error())
		t.FailNow()
	}
	if events == nil {
		fmt.Printf("FetchCrossChainEvent - no events found.")
		t.FailNow()
	}

	for _, evt := range events {
		//evt := events.Event
		//fmt.Println(evt)
		evtData, err := tools.HexToBytes(evt.Data)
		if err != nil {
			fmt.Printf("FetchCrossChainEvent - hex.DecodeString error :%s", err.Error())
			t.FailNow()
		}
		ccEvent, err := stcpolyevts.BcsDeserializeCrossChainEvent(evtData)
		// j, _ := json.Marshal(lockEvent)
		// fmt.Println(string(j))
		fmt.Println("/////////////// CrossChainEvent info. ///////////////")
		fmt.Println("--------------- TxId(TxIndex) ----------------")
		fmt.Println(hex.EncodeToString(ccEvent.TxId))
		fmt.Println("--------------- ProxyOrAssetContract ---------------")
		fmt.Println(hex.EncodeToString(ccEvent.ProxyOrAssetContract))
		fmt.Println(string(ccEvent.ProxyOrAssetContract))
		fmt.Println("--------------- Sender ----------------")
		fmt.Println(hex.EncodeToString(ccEvent.Sender))
		fmt.Println("--------------- ToChainId ----------------")
		fmt.Println(ccEvent.ToChainId)
		fmt.Println("--------------- ToContract ----------------")
		fmt.Println(hex.EncodeToString(ccEvent.ToContract))
		fmt.Println("--------------- RawData ----------------")
		fmt.Println(hex.EncodeToString(ccEvent.RawData))

		src := pcommon.NewZeroCopySource(ccEvent.RawData)
		txParam := new(common2.MakeTxParam)
		if err := txParam.Deserialization(src); err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		fmt.Println("/////////////// Decoded CrossChainEvent.RawData(MakeTxParam) ///////////////")
		fmt.Println("---------- MakeTxParam.TxHash ----------")
		fmt.Println(hex.EncodeToString(txParam.TxHash)) // []byte
		fmt.Println("---------- MakeTxParam.CrossChainId ----------")
		fmt.Println(hex.EncodeToString(txParam.CrossChainID)) // sha256(abi.encodePacked(address(this), paramTxHash))
		fmt.Println("---------- MakeTxParam.FromContractAddress ----------")
		fmt.Println(hex.EncodeToString(txParam.FromContractAddress)) // []byte
		fmt.Println("---------- MakeTxParam.ToChainId ----------")
		fmt.Println(txParam.ToChainID) // uint64
		fmt.Println("---------- MakeTxParam.ToContractAddress ----------")
		fmt.Println(hex.EncodeToString(txParam.ToContractAddress)) // []byte
		fmt.Println(string(txParam.ToContractAddress))             // []byte
		fmt.Println("---------- MakeTxParam.Method ----------")
		fmt.Println(txParam.Method) // string
		fmt.Println("---------- MakeTxParam.Args ----------")
		fmt.Println(hex.EncodeToString(txParam.Args)) // []byte
	}
}

func TestFetchLockEvent(t *testing.T) {
	starcoinClient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	address := "0xb6d69dd935edf7f2054acf12eb884df8"
	typeTag := "0xb6d69dd935edf7f2054acf12eb884df8::LockProxy::LockEvent"
	height := uint64(3422033) //3421831) //2977050)
	eventFilter := &stcclient.EventFilter{
		Address:   []string{address},
		TypeTags:  []string{typeTag},
		FromBlock: height,
		ToBlock:   &height,
	}

	events, err := starcoinClient.GetEvents(context.Background(), eventFilter)
	if err != nil {
		fmt.Printf("FetchLockEvents - GetEvents error :%s", err.Error())
		t.FailNow()
	}
	if events == nil {
		fmt.Printf("FetchLockEvents - no events found.")
		t.FailNow()
	}

	for _, evt := range events {
		//evt := events.Event
		//fmt.Println(evt)
		evtData, err := tools.HexToBytes(evt.Data)
		if err != nil {
			fmt.Printf("FetchLockEvents - hex.DecodeString error :%s", err.Error())
			t.FailNow()
		}
		lockEvent, err := stcpolyevts.BcsDeserializeLockEvent(evtData)
		// j, _ := json.Marshal(lockEvent)
		// fmt.Println(string(j))
		fmt.Println("/////////////// LockEvent info. ///////////////")
		fmt.Println("--------------- FromAssetHash ----------------")
		fmt.Println(GetTokenCodeString(&lockEvent.FromAssetHash))
		fmt.Println("--------------- FromAddress ---------------")
		fmt.Println(hex.EncodeToString(lockEvent.FromAddress))
		fmt.Println("--------------- ToChainId ----------------")
		fmt.Println(lockEvent.ToChainId)
		fmt.Println("--------------- ToAssetHash ----------------")
		fmt.Println(string(lockEvent.ToAssetHash)) // if it is Token on Starcoin, asset hash is like: "0x00000000000000000000000000000001::STC::STC"
		fmt.Println(hex.EncodeToString(lockEvent.ToAssetHash))
		fmt.Println("--------------- ToAddress ----------------")
		fmt.Println(hex.EncodeToString(lockEvent.ToAddress))
		fmt.Println("--------------- Amount ----------------")
		fmt.Println(tools.Uint128ToBigInt(lockEvent.Amount))
	}
}

func TestFetchUnlockEvent(t *testing.T) {
	starcoinClient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	// tx, _ := starcoinClient.GetTransactionInfoByHash(context.Background(), "0x08bc1d3e076c75519f1d2bdea08b980e5e2c7ec62b56136f038012e4333f541a")
	address := "0xb6d69dd935edf7f2054acf12eb884df8"
	typeTag := "0xb6d69dd935edf7f2054acf12eb884df8::LockProxy::UnlockEvent"
	height := uint64(2977211)
	eventFilter := &stcclient.EventFilter{
		Address:   []string{address},
		TypeTags:  []string{typeTag},
		FromBlock: height,
		ToBlock:   &height,
	}

	events, err := starcoinClient.GetEvents(context.Background(), eventFilter)
	if err != nil {
		fmt.Printf("FetchUnlockEvent - GetEvents error :%s", err.Error())
		t.FailNow()
	}
	if events == nil {
		fmt.Printf("FetchUnlockEvent - no events found.")
		t.FailNow()
	}

	for _, evt := range events {
		//evt := events.Event
		//fmt.Println(evt)
		fmt.Println("TransactionHash: " + evt.TransactionHash)
		//return
		evtData, err := tools.HexToBytes(evt.Data)
		if err != nil {
			fmt.Printf("FetchUnlockEvent - hex.DecodeString error :%s", err.Error())
			t.FailNow()
		}
		unlockEvent, err := stcpolyevts.BcsDeserializeUnlockEvent(evtData)
		// j, _ := json.Marshal(unlockEvent)
		// fmt.Println(string(j))
		fmt.Println("/////////////// UnlockEvent info. ///////////////")
		fmt.Println("--------------- ToAssetHash ----------------")
		fmt.Println(string(unlockEvent.ToAssetHash)) // if it is Token on Starcoin, asset hash is like: "0x00000000000000000000000000000001::STC::STC"
		fmt.Println("--------------- ToAddress ----------------")
		fmt.Println(hex.EncodeToString(unlockEvent.ToAddress))
		fmt.Println("--------------- Amount ----------------")
		fmt.Println(tools.Uint128ToBigInt(unlockEvent.Amount))
	}
}

func TestFetchCrossChainFeeLockEvents(t *testing.T) {
	starcoinClient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	// tx, _ := starcoinClient.GetTransactionInfoByHash(context.Background(), "0xe2ad6bfeed3f5c96236c348556cde88d31f2336cd505be8b5d6ed1293ed7cf90")
	address := "0xb6d69dd935edf7f2054acf12eb884df8"
	typeTag := "0xb6d69dd935edf7f2054acf12eb884df8::LockProxy::CrossChainFeeLockEvent"
	height := uint64(3422033) //3421831) //3421382) //3421326) //3414132) //CrossChainFeeLockEvent on barnard
	eventFilter := &stcclient.EventFilter{
		Address:   []string{address},
		TypeTags:  []string{typeTag},
		FromBlock: height,
		ToBlock:   &height,
	}

	events, err := starcoinClient.GetEvents(context.Background(), eventFilter)
	if err != nil {
		fmt.Printf("FetchCrossChainFeeLockEvents - GetEvents error :%s", err.Error())
		t.FailNow()
	}
	if events == nil {
		fmt.Printf("FetchCrossChainFeeLockEvents - no events found.")
		t.FailNow()
	}

	for _, evt := range events {
		//evt := events.Event
		//fmt.Println(evt)
		evtData, err := tools.HexToBytes(evt.Data)
		if err != nil {
			fmt.Printf("FetchCrossChainFeeLockEvents - hex.DecodeString error :%s", err.Error())
			t.FailNow()
		}
		feeEvent, err := stcpolyevts.BcsDeserializeCrossChainFeeLockEvent(evtData)
		// j, _ := json.Marshal(lockEvent)
		// fmt.Println(string(j))
		fmt.Println("/////////////// CrossChainFeeLockEvent info. ///////////////")
		fmt.Println("--------------- FromAssetHash ----------------")
		fmt.Println(GetTokenCodeString(&feeEvent.FromAssetHash))
		fmt.Println("--------------- Sender ---------------")
		fmt.Println(hex.EncodeToString(feeEvent.Sender[:]))
		fmt.Println("--------------- ToChainId ----------------")
		fmt.Println(feeEvent.ToChainId)
		fmt.Println("--------------- ToAddress ----------------")
		fmt.Println(hex.EncodeToString(feeEvent.ToAddress))
		fmt.Println("--------------- Net ----------------")
		fmt.Println(tools.Uint128ToBigInt(feeEvent.Net))
		fmt.Println("--------------- Fee ----------------")
		fmt.Println(tools.Uint128ToBigInt(feeEvent.Fee))
		fmt.Println("--------------- Id ----------------")
		fmt.Println(tools.Uint128ToBigInt(feeEvent.Id))
	}
}

func TestFetchVerifyHeaderAndExecuteTxEvent(t *testing.T) {
	starcoinClient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	// tx, _ := starcoinClient.GetTransactionInfoByHash(context.Background(), "0xe2ad6bfeed3f5c96236c348556cde88d31f2336cd505be8b5d6ed1293ed7cf90")
	address := "0xb6d69dd935edf7f2054acf12eb884df8"
	typeTag := "0xb6d69dd935edf7f2054acf12eb884df8::CrossChainManager::VerifyHeaderAndExecuteTxEvent"
	height := uint64(2977211)
	eventFilter := &stcclient.EventFilter{
		Address:   []string{address},
		TypeTags:  []string{typeTag},
		FromBlock: height,
		ToBlock:   &height,
	}

	events, err := starcoinClient.GetEvents(context.Background(), eventFilter)
	if err != nil {
		fmt.Printf("FetchCrossChainFeeLockEvents - GetEvents error :%s", err.Error())
		t.FailNow()
	}
	if events == nil {
		fmt.Printf("FetchCrossChainFeeLockEvents - no events found.")
		t.FailNow()
	}

	for _, evt := range events {
		//evt := events.Event
		//fmt.Println(evt)
		evtData, err := tools.HexToBytes(evt.Data)
		if err != nil {
			fmt.Printf("FetchCrossChainFeeLockEvents - hex.DecodeString error :%s", err.Error())
			t.FailNow()
		}
		veEvent, err := stcpolyevts.BcsDeserializeVerifyHeaderAndExecuteTxEvent(evtData)
		// j, _ := json.Marshal(lockEvent)
		// fmt.Println(string(j))
		fmt.Println("/////////////// VerifyHeaderAndExecuteTxEvent info. ///////////////")
		fmt.Println("--------------- CrossChainTxHash ----------------")
		fmt.Println(hex.EncodeToString(veEvent.CrossChainTxHash))
		fmt.Println("--------------- FromChainId ---------------")
		fmt.Println(veEvent.FromChainId)
		fmt.Println("--------------- FromChainTxHash ----------------")
		fmt.Println(hex.EncodeToString(veEvent.FromChainTxHash))
		fmt.Println("--------------- ToContract ----------------")
		fmt.Println(hex.EncodeToString(veEvent.ToContract))
		fmt.Println("--------------- ToContract(as string) ----------------")
		fmt.Println(string(veEvent.ToContract)) //to Starcoin, it is like this: 0xb6d69dd935edf7f2054acf12eb884df8::CrossChainScript
	}
}

func TestDeserializeCrossChainFeeLockEvent(t *testing.T) {
	bs := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 3, 83, 84, 67, 3, 83, 84, 67, 24, 53, 29, 49, 29, 50, 32, 17, 73, 164, 223, 42, 159, 194, 219, 138, 11, 0, 0, 0, 0, 0, 0, 0, 16, 24, 53, 29, 49, 29, 50, 32, 17, 73, 164, 223, 42, 159, 194, 219, 138, 111, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 222, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 77, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	feeEvent, err := stcpolyevts.BcsDeserializeCrossChainFeeLockEvent(bs)
	if err != nil {
		t.FailNow()
	}
	fmt.Println("/////////////// CrossChainFeeLockEvent info. ///////////////")
	fmt.Println("--------------- FromAssetHash ----------------")
	fmt.Println(GetTokenCodeString(&feeEvent.FromAssetHash))
	fmt.Println("--------------- Sender ---------------")
	fmt.Println(hex.EncodeToString(feeEvent.Sender[:]))
	fmt.Println("--------------- ToChainId ----------------")
	fmt.Println(feeEvent.ToChainId)
	fmt.Println("--------------- ToAddress ----------------")
	fmt.Println(hex.EncodeToString(feeEvent.ToAddress))
	fmt.Println("--------------- Net ----------------")
	fmt.Println(tools.Uint128ToBigInt(feeEvent.Net))
	fmt.Println("--------------- Fee ----------------")
	fmt.Println(tools.Uint128ToBigInt(feeEvent.Fee))
	fmt.Println("--------------- Id ----------------")
	fmt.Println(tools.Uint128ToBigInt(feeEvent.Id))
}
