package manager

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	stcpolyevts "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly/events"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestFetchCrossChainEvent(t *testing.T) {
	starcoinClient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	address := "0x18351d311d32201149a4df2a9fc2db8a"
	typeTag := "0x18351d311d32201149a4df2a9fc2db8a::CrossChainManager::CrossChainEvent"
	height := uint64(3422033) //3421831) //2977050)
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
		fmt.Println("--------------- Sender ----------------")
		fmt.Println(hex.EncodeToString(ccEvent.Sender))
		fmt.Println("--------------- ToChainId ----------------")
		fmt.Println(ccEvent.ToChainId)
		fmt.Println("--------------- ToContract ----------------")
		fmt.Println(hex.EncodeToString(ccEvent.ToContract))
		fmt.Println("--------------- RawData ----------------")
		fmt.Println(hex.EncodeToString(ccEvent.RawData))
	}
}

func TestFetchLockEvent(t *testing.T) {
	starcoinClient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	// tx, _ := starcoinClient.GetTransactionInfoByHash(context.Background(), "0xe2ad6bfeed3f5c96236c348556cde88d31f2336cd505be8b5d6ed1293ed7cf90")
	// fmt.Println(tx.GasUsed)
	// return
	address := "0x18351d311d32201149a4df2a9fc2db8a"
	typeTag := "0x18351d311d32201149a4df2a9fc2db8a::LockProxy::LockEvent"
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
		fmt.Println(Uint128ToBigInt(&lockEvent.Amount))
	}
}

func TestFetchUnlockEvent(t *testing.T) {
	starcoinClient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	// tx, _ := starcoinClient.GetTransactionInfoByHash(context.Background(), "0x08bc1d3e076c75519f1d2bdea08b980e5e2c7ec62b56136f038012e4333f541a")
	// fmt.Println(tx.GasUsed)
	// return
	address := "0x18351d311d32201149a4df2a9fc2db8a"
	typeTag := "0x18351d311d32201149a4df2a9fc2db8a::LockProxy::UnlockEvent"
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
		fmt.Println(Uint128ToBigInt(&unlockEvent.Amount))
	}
}

func TestFetchCrossChainFeeLockEvents(t *testing.T) {
	starcoinClient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	// tx, _ := starcoinClient.GetTransactionInfoByHash(context.Background(), "0xe2ad6bfeed3f5c96236c348556cde88d31f2336cd505be8b5d6ed1293ed7cf90")
	// fmt.Println(tx.GasUsed)
	// return
	address := "0x18351d311d32201149a4df2a9fc2db8a"
	typeTag := "0x18351d311d32201149a4df2a9fc2db8a::LockProxy::CrossChainFeeLockEvent"
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
		fmt.Println(Uint128ToBigInt(&feeEvent.Net))
		fmt.Println("--------------- Fee ----------------")
		fmt.Println(Uint128ToBigInt(&feeEvent.Fee))
		fmt.Println("--------------- Id ----------------")
		fmt.Println(Uint128ToBigInt(&feeEvent.Id))
	}
}
