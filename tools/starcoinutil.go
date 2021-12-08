package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/log"
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
		Method:  "chain.get_transaction_proof", // starcoin chain info
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
				continue
			}
			//log.Debug("Transaction status: " + tx.Status)
			if strings.EqualFold("Executed", tx.Status) {
				return true, nil
			} else {
				continue //todo return false on some statuses???
			}
		case <-exitTicker.C:
			return false, fmt.Errorf("WaitTransactionConfirm exceed timeout %v", timeout)
		}
	}
}
