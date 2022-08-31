package tools

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/novifinancial/serde-reflection/serde-generate/runtime/golang/serde"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	"github.com/starcoinorg/starcoin-go/types"
)

func TestWaitTransactionConfirm(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://halley-seed.starcoin.org")
	b, err := WaitTransactionConfirm(stcclient, "0xb199fbbf9c7aeef9a0257f0d496ecd0f11ded014526965b4a294c66041272cae", time.Second*30)
	fmt.Println(b, err)
}

func TestGetTokenScalingFactor(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	tokenType := "0x416b32009fe49fcab1d5f2ba0153838f::XUSDT::XUSDT"
	sf, err := GetTokenScalingFactor(&stcclient, tokenType)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Printf("Token(%s) scaling factor: %d\n", tokenType, sf)
}

func TestGetStarcoinNodeHeight(t *testing.T) {
	restclient := NewRestClient()
	h, err := GetStarcoinNodeHeight("https://barnard-seed.starcoin.org", restclient)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(h, err)
}

func TestGetTransactionProof(t *testing.T) {
	/*
		curl --location --request POST 'https://main-seed.starcoin.org' \
		--header 'Content-Type: application/json' \
		--data-raw '{
			"id":101,
			"jsonrpc":"2.0",
			"method":"chain.get_transaction_info",
			"params":["0xa3cac3fc94d4e68de66812b3bb638e82211c26ed0e879eb368196bd849eea86a"]
		}'

		curl --location --request POST 'https://main-seed.starcoin.org/' \
		--header 'Content-Type: application/json' \
		--data-raw '{
		 "id":101,
		 "jsonrpc":"2.0",
		 "method":"chain.get_events_by_txn_hash",
		 "params":["0xa3cac3fc94d4e68de66812b3bb638e82211c26ed0e879eb368196bd849eea86a"]
		}'

		curl --location --request POST 'https://main-seed.starcoin.org/' \
		--header 'Content-Type: application/json' \
		--data-raw '{
		 "id":101,
		 "jsonrpc":"2.0",
		 "method":"chain.get_transaction_proof",
		 "params":["0x6e1412d4c8d88ae138760ec4caf2d2a8d875db9014485138833b0b785a4300d0", 8369404, 1]
		}'
	*/
	//"transaction_global_index":"8369404"
	restclient := NewRestClient()
	var eventIndex int = 1
	var txGlobalIndex uint64 = 8369404
	var blockHash = "0x6e1412d4c8d88ae138760ec4caf2d2a8d875db9014485138833b0b785a4300d0"
	p, err := GetTransactionProof("https://main-seed.starcoin.org", restclient, blockHash, txGlobalIndex, &eventIndex)
	if err != nil {
		t.FailNow()
	}
	fmt.Println("--------------- transaction proof -----------------")
	fmt.Println(p)

	transactionInfoProof := new(TransactionInfoProof)
	if err = json.Unmarshal([]byte(p), transactionInfoProof); err != nil {
		t.Errorf("unmarshal proof error:%s", err)
	}
	typesTransactionInfo, err := transactionInfoProof.TransactionInfo.ToTypesTransactionInfo()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("----------- types.TransactionInfo BCS Serialized data ------------")
	typesTxInfoBcsData, _ := typesTransactionInfo.BcsSerialize()
	fmt.Println(hex.EncodeToString(typesTxInfoBcsData))
	// ------------------------------------------------------------
	// print transaction info for create verify-accumulator unit test...
	/*
		curl --location --request POST 'https://main-seed.starcoin.org/' \
		--header 'Content-Type: application/json' \
		--data-raw '{
		"id":101,
		"jsonrpc":"2.0",
		"method":"chain.get_block_by_hash",
		"params":["0x6e1412d4c8d88ae138760ec4caf2d2a8d875db9014485138833b0b785a4300d0"]
		}'
	*/
	fmt.Println("------------ txn_accumulator_root ----------------")
	txn_accumulator_root := "0x9e9dc633087fcdeec84f6306900c76298e6667b53a743e953dbb333c74994243"
	fmt.Println(txn_accumulator_root)
	fmt.Println("------------- transaction info hash -----------------")
	txInfoHash, _ := typesTransactionInfo.CryptoHash()
	fmt.Println(hex.EncodeToString(*txInfoHash))
	fmt.Println("------------- transaction global index -----------------")
	fmt.Println(txGlobalIndex)
	fmt.Println("----------------- transaction info proof ----------------")
	for _, s := range transactionInfoProof.Proof.Siblings {
		fmt.Println(s)
	}

	// print event info for create verify-accumulator unit test...
	fmt.Println("----------------- event root hash ----------------")
	eventRootHash := hex.EncodeToString(typesTransactionInfo.EventRootHash)
	fmt.Println(eventRootHash)
	fmt.Println("----------------- event hash ----------------")
	eventData, _ := HexToBytes(transactionInfoProof.EventWithProof.Event)
	contractEventV0, _ := stcclient.EventToContractEventV0(eventData)
	contractEvent := types.ContractEvent__V0{
		Value: *contractEventV0,
	}
	eventHash, _ := contractEvent.CryptoHash()
	fmt.Println(hex.EncodeToString(*eventHash))
	fmt.Println("----------------- event index ----------------")
	fmt.Println(eventIndex)
	fmt.Println("----------------- event proof ----------------")
	for _, s := range transactionInfoProof.EventWithProof.Proof.Siblings {
		fmt.Println(s)
	}
	fmt.Println("----------------- contract event BCS data ----------------")
	contractEventBcsData, _ := contractEvent.BcsSerialize()
	fmt.Println(hex.EncodeToString(contractEventBcsData))
	eventTypeTag, _ := contractEvent.Value.TypeTag.BcsSerialize()
	fmt.Println(hex.EncodeToString(contractEvent.Value.Key))
	fmt.Println(contractEvent.Value.SequenceNumber)
	fmt.Println(hex.EncodeToString(eventTypeTag))
	fmt.Println(hex.EncodeToString(contractEvent.Value.EventData))
}

type TransactionInfoProof struct {
	TransactionInfo stcclient.TransactionInfo `json:"transaction_info"`
	Proof           AccumulatorProof          `json:"proof"`
	EventWithProof  EventWithProof            `json:"event_proof"`
	StateWithProof  StateWithProofJson        `json:"state_proof"`
	AccessPath      *string                   `json:"access_path,omitempty"`
	EventIndex      *int                      `json:"event_index,omitempty"`
}

type EventWithProof struct {
	Event string           `json:"event"`
	Proof AccumulatorProof `json:"proof"`
}

type AccumulatorProof struct {
	Siblings []string `json:"siblings"`
}

type StateProofJson struct {
	AccountState      []byte                `json:"account_state"`
	AccountProof      SparseMerkleProofJson `json:"account_proof"`
	AccountStateProof SparseMerkleProofJson `json:"account_state_proof"`
}

type StateWithProofJson struct {
	State []byte         `json:"state"`
	Proof StateProofJson `json:"proof"`
}

type SparseMerkleProofJson struct {
	Leaf     []string `json:"leaf"`
	Siblings []string `json:"siblings"`
}

func TestGetTransactionInfoByHash(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://halley-seed.starcoin.org")
	txInfo, err := stcclient.GetTransactionInfoByHash(context.Background(), "0xb199fbbf9c7aeef9a0257f0d496ecd0f11ded014526965b4a294c66041272cae")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(txInfo)
	fmt.Println(strings.EqualFold("\"Executed\"", string(txInfo.Status)))
	fmt.Println(txInfo.Status)
	fmt.Println(isKnownStarcoinTxFailureStatus(txInfo.Status))
}

func TestIsAcceptToken(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	accountAddr := "0xdb9d6f70922c8deb4c9c6500633f425d"
	// accountAddr := "0xd117638e105403784bf6A92AA1276Ec1"
	tokenType := "0x00000000000000000000000000000001::STC::STC"
	// tokenType := "0x416b32009fe49fcab1d5f2ba0153838f::XETH::XETH"
	a, err := IsAcceptToken(&stcclient, accountAddr, tokenType)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Printf("Is account '%s' accept token '%s': %v\n", accountAddr, tokenType, a)
}

func TestAccountExistsAt(t *testing.T) {
	stcclient := stcclient.NewStarcoinClient("https://barnard-seed.starcoin.org")
	//accountAddr := "0xdb9d6f70922c8deb4c9c6500633f425d"
	accountAddr := "0x4afb6a3ED1e2ff212586fc6BcDb8DdAF"
	a, err := AccountExistsAt(&stcclient, accountAddr)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Printf("Account Exists At '%s': %v\n", accountAddr, a)
}

func TestUint128AndBigIntConvert(t *testing.T) {
	u1 := serde.Uint128{
		High: 0,
		Low:  2423531242,
	}
	b1 := Uint128ToBigInt(u1)
	if u1.Low != b1.Uint64() {
		t.FailNow()
	}
	fmt.Println(b1.String())
	u1_c := BigIntToUint128(b1)
	fmt.Println(u1_c.Low)
	if u1.High != u1_c.High || u1.Low != u1_c.Low {
		t.FailNow()
	}

	// /////////////////////
	u2 := serde.Uint128{
		High: 1124,
		Low:  2423531242,
	}
	b2 := Uint128ToBigInt(u2)
	fmt.Println(b2.String())
	b2_expected := new(big.Int).Add(new(big.Int).Mul(new(big.Int).Add(new(big.Int).SetUint64(math.MaxUint64), new(big.Int).SetUint64(1)), new(big.Int).SetUint64(u2.High)), new(big.Int).SetUint64(u2.Low))
	fmt.Println(b2_expected.String())
	if b2.String() != b2_expected.String() {
		t.FailNow()
	}
	// ////////////////////
	u2_c := BigIntToUint128(b2)
	if u2.High != u2_c.High || u2.Low != u2_c.Low {
		t.FailNow()
	}
}
