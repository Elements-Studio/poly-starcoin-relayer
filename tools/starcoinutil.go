package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/log"
	"github.com/novifinancial/serde-reflection/serde-generate/runtime/golang/serde"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	"github.com/starcoinorg/starcoin-go/types"
)

type jsonRpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type starcoinJsonRpcReq struct {
	JsonRpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      uint          `json:"id"`
}

type starcoinChainInfoRsp struct {
	JsonRpc string `json:"jsonrpc"`
	Result  struct {
		Head struct {
			Number json.Number `json:"number"`
		} `json:"head"`
	} `json:"result,omitempty"`
	Error *jsonRpcError `json:"error,omitempty"`
	Id    uint          `json:"id"`
}

type StarcoinAccountBalanceResource struct {
	Raw  string `json:"raw"`
	Json struct {
		Token struct {
			Value big.Int `json:"value"`
		} `json:"token"`
	} `json:"json"`
}

func GetStarcoinAccountTokenBalance(starcoinClient *stcclient.StarcoinClient, account string, tokenType string) (*big.Int, error) {
	resType := "0x00000000000000000000000000000001::Account::Balance<" + tokenType + ">"
	getResOption := stcclient.GetResourceOption{
		Decode: true,
	}
	//accountBalanceRes := new(map[string]interface{}) // &map[json:map[token:map[value:8.300340036e+09]] raw:0x4423bdee010000000000000000000000]
	accountBalanceRes := new(StarcoinAccountBalanceResource)
	getResResult, err := starcoinClient.GetResource(context.Background(), account, resType, getResOption, accountBalanceRes)
	if err != nil {
		return nil, err
	}
	//fmt.Println(getResResult)
	b := getResResult.(*StarcoinAccountBalanceResource)
	return &b.Json.Token.Value, nil
}

// Get starcoin node current height.
func GetStarcoinNodeHeight(url string, restClient *RestClient) (uint64, error) {
	req := &starcoinJsonRpcReq{
		JsonRpc: "2.0",
		Method:  "chain.info", // starcoin chain info
		Params:  make([]interface{}, 0),
		Id:      1,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("GetStarcoinNodeHeight: marshal req err: %s", err.Error())
	}
	rspData, err := restClient.SendPostRequest(url, reqData)
	if err != nil {
		return 0, fmt.Errorf("GetStarcoinNodeHeight err: %s", err.Error())
	}
	rsp := &starcoinChainInfoRsp{}
	err = json.Unmarshal(rspData, rsp)
	if err != nil {
		return 0, fmt.Errorf("GetStarcoinNodeHeight, unmarshal resp err: %s", err.Error())
	}
	if rsp.Error != nil {
		return 0, fmt.Errorf("GetStarcoinNodeHeight, unmarshal resp err: %s", rsp.Error.Message)
	}
	height, err := rsp.Result.Head.Number.Int64()
	if err != nil {
		return 0, fmt.Errorf("GetStarcoinNodeHeight, parse resp height %s failed", rsp.Result)
	} else {
		return uint64(height), nil
	}
}

func IsAcceptToken(client *stcclient.StarcoinClient, accountAddress string, tokenType string) (bool, error) {
	c := stcclient.ContractCall{
		FunctionId: "0x1::Account::is_accept_token",
		TypeArgs:   []string{tokenType},
		Args:       []string{accountAddress},
	}
	r, err := client.CallContract(context.Background(), c)
	if err != nil {
		return false, err
	}
	return ToBool(ExtractSingleResult(r))
}

type starcoinTransactionProofRsp struct {
	JsonRpc string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRpcError   `json:"error,omitempty"`
	Id      uint            `json:"id"`
}

func GetTransactionProof(url string, restClient *RestClient, blockHash string, txGlobalIndex uint64, eventIndex *int) (string, error) {
	params := []interface{}{blockHash, txGlobalIndex, eventIndex}
	req := &starcoinJsonRpcReq{
		JsonRpc: "2.0",
		Method:  "chain.get_transaction_proof",
		Params:  params,
		Id:      1,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("GetTransactionProof: marshal req err: %s", err.Error())
	}
	rspData, err := restClient.SendPostRequest(url, reqData)
	if err != nil {
		return "", fmt.Errorf("GetTransactionProof err: %s", err.Error())
	}
	rsp := &starcoinTransactionProofRsp{}
	err = json.Unmarshal(rspData, rsp)
	if err != nil {
		return "", fmt.Errorf("GetTransactionProof, unmarshal resp err: %s", err.Error())
	}
	return string(rsp.Result), nil
}

type StarcoinKeyStore struct {
	privateKey types.Ed25519PrivateKey
	chainId    int
}

func NewStarcoinKeyStore(privateKey types.Ed25519PrivateKey, chainId int) *StarcoinKeyStore {
	return &StarcoinKeyStore{
		privateKey: privateKey,
		chainId:    chainId,
	}
}

func (ks *StarcoinKeyStore) GetChainId() int {
	return ks.chainId
}

func (ks *StarcoinKeyStore) GetPrivateKey() types.Ed25519PrivateKey {
	return ks.privateKey
}

type StarcoinAccount struct {
	Address types.AccountAddress //`json:"address"` // Starcoin account address derived from the key
	//URL     URL            `json:"url"`     // Optional resource locator within a backend
}

// Wait transaction to confirmed.
// Return `true, nil`` if transaction confirmed;
// return `false, {NOT-NIL-ERROR}` for known error;
// return `false, nil` for UNKNOWN ERROR or TIMED-OUT or CANNOT-GET-TX-INFO(Cannot get Transaction info on-chain).
func WaitTransactionConfirm(client stcclient.StarcoinClient, hash string, timeout time.Duration) (bool, error) {
	monitorTicker := time.NewTicker(time.Second)
	exitTicker := time.NewTicker(timeout)
	for {
		select {
		case <-monitorTicker.C:
			pendingTx, err := client.GetPendingTransactionByHash(context.Background(), hash)
			//log.Debugf("%v, %v", pendingTx, err)
			if err != nil {
				log.Debugf("GetPendingTransactionByHash error, %v", err)
				continue
			}
			if !(pendingTx == nil || pendingTx.Timestamp == 0) {
				log.Debugf("(starcoin_transaction %s) is pending", hash)
				continue
			}
			tx, err := client.GetTransactionInfoByHash(context.Background(), hash)
			if err != nil {
				log.Debugf("GetTransactionInfoByHash error, %v", err)
				continue
			}
			if tx == nil || tx.BlockNumber == "" || tx.Status == nil {
				continue // CANNOT-GET-TX-INFO, continue till get transaction's status or time-out
			}
			//log.Debug("Transaction status: " + tx.Status)
			if isStarcoinTxStatusExecuted(tx.Status) {
				return true, nil
			} else if isKnownStarcoinTxFailureStatus(tx.Status) {
				return false, fmt.Errorf("isKnownStarcoinTxFailureStatus: %s", string(tx.Status))
			} else {
				// TODO: or continue?
				return false, nil // UNKNOWN-ERROR
			}
		case <-exitTicker.C:
			//log.Debugf("WaitTransactionConfirm exceed timeout %v", timeout)
			return false, nil // TIMED-OUT
		}
	}
}

func IsStarcoinTxStatusExecutedOrKnownFailure(status []byte) (bool, bool) {
	if isStarcoinTxStatusExecuted(status) {
		return true, false
	} else if isKnownStarcoinTxFailureStatus(status) {
		return false, true
	} else {
		return false, false
	}
}

func isStarcoinTxStatusExecuted(status []byte) bool {
	return strings.EqualFold("\"Executed\"", string(status))
}

func isKnownStarcoinTxFailureStatus(status []byte) bool {
	jsonObj := make(map[string]interface{})
	//fmt.Println(string(status))
	err := json.Unmarshal(status, &jsonObj)
	if err != nil {
		return false
	}
	for k := range jsonObj {
		//fmt.Println(k)
		if strings.EqualFold("MoveAbort", k) {
			return true
		} else if strings.EqualFold("ExecutionFailure", k) {
			//{"ExecutionFailure":{"function":10,"code_offset":38,"location":{"Module":{"address":"0x18351d311d32201149a4df2a9fc2db8a","name":"LockProxy"}}}}
			return true
		}
	}
	//fmt.Println(jsonObj)
	return false
}

func ParseStructTypeTag(s string) (types.TypeTag, error) {
	ss := strings.Split(s, "::")
	if len(ss) < 3 {
		panic("Struct TypeTag string format error")
	}
	addr, err := types.ToAccountAddress(ss[0])
	if err != nil {
		return nil, err
	}
	st := types.StructTag{
		Address: *addr,
		Module:  types.Identifier(ss[1]),
		Name:    types.Identifier(ss[2]),
	}
	return &types.TypeTag__Struct{Value: st}, nil
}

func ExtractSingleResult(result interface{}) interface{} {
	r := result.([]interface{})
	if len(r) == 0 {
		return nil
	}
	return r[0]
}

func ToBool(i interface{}) (bool, error) {
	switch i := i.(type) {
	case bool:
		return i, nil
	case string:
		return strconv.ParseBool(i)
	}
	return false, fmt.Errorf("unknown type to bool %t", i)
}

func ToBytes(i interface{}) ([]byte, error) {
	switch i := i.(type) {
	case []byte:
		return i, nil
	case string:
		return HexToBytes(i)
	}
	return nil, fmt.Errorf("unknown type to []byte %t", i)
}

func ToUint64(i interface{}) (uint64, error) {
	switch i := i.(type) {
	case uint64:
		return i, nil
	case float64:
		return uint64(i), nil
	case string:
		return strconv.ParseUint(i, 10, 64)
	case json.Number:
		r, err := i.Int64()
		return uint64(r), err
	}
	return 0, fmt.Errorf("unknown type to uint64 %t", i)
}

func ToBigInt(i interface{}) (*big.Int, error) {
	switch i := i.(type) {
	case uint64:
		r := new(big.Int).SetUint64(i)
		return r, nil
	case float64:
		r, _ := big.NewFloat(i).Int(nil)
		return r, nil
	case string:
		r, ok := new(big.Int).SetString(i, 10)
		if !ok {
			fmt.Errorf("convert string to big.Int failed: %s", i)
		}
		return r, nil
	case json.Number:
		r, ok := new(big.Int).SetString(i.String(), 10)
		if !ok {
			fmt.Errorf("convert string to big.Int failed: %s", i)
		}
		return r, nil
	}
	return nil, fmt.Errorf("unknown type to big.Int %t", i)
}

// Parse module Id., return address and module name.
func ParseStarcoinModuleId(str string) (string, string, error) {
	ss := strings.Split(str, "::")
	if len(ss) < 2 {
		return "", "", fmt.Errorf("module Id string format error")
	} else if len(ss) > 2 {
		return "", "", fmt.Errorf("module Id string format error")
	}
	return ss[0], ss[1], nil
}

func Uint128ToBigInt(u serde.Uint128) *big.Int {
	h := new(big.Int).SetUint64(u.High)
	l := new(big.Int).SetUint64(u.Low)
	hbuf := make([]byte, 8)
	return new(big.Int).SetBytes(append(h.Bytes(), l.FillBytes(hbuf)...))
}

func BigIntToUint128(i *big.Int) serde.Uint128 {
	var h, l uint64
	bytesLen := len(i.Bytes())
	if bytesLen > 8 {
		h = new(big.Int).SetBytes(i.Bytes()[0 : bytesLen-8]).Uint64()
		l = new(big.Int).SetBytes(i.Bytes()[bytesLen-8 : bytesLen]).Uint64()
	} else {
		h = 0
		l = i.Uint64()
	}
	return serde.Uint128{
		High: h,
		Low:  l,
	}
}
