package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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

type starcoinHeightReq struct {
	JsonRpc string   `json:"jsonrpc"`
	Method  string   `json:"method"`
	Params  []string `json:"params"`
	Id      uint     `json:"id"`
}

type starcoinHeightRsp struct {
	JsonRpc string        `json:"jsonrpc"`
	Result  string        `json:"result,omitempty"`
	Error   *jsonRpcError `json:"error,omitempty"`
	Id      uint          `json:"id"`
}

func GetStarcoinNodeHeight(url string, restClient *RestClient) (uint64, error) {
	req := &starcoinHeightReq{
		JsonRpc: "2.0",
		Method:  "eth_blockNumber", //todo starcoin blockNumber
		Params:  make([]string, 0),
		Id:      1,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("GetStarcoinNodeHeight: marshal req err: %s", err)
	}
	rspData, err := restClient.SendPostRequest(url, reqData)
	if err != nil {
		return 0, fmt.Errorf("GetStarcoinNodeHeight err: %s", err)
	}
	rsp := &starcoinHeightRsp{}
	err = json.Unmarshal(rspData, rsp)
	if err != nil {
		return 0, fmt.Errorf("GetStarcoinNodeHeight, unmarshal resp err: %s", err)
	}
	if rsp.Error != nil {
		return 0, fmt.Errorf("GetStarcoinNodeHeight, unmarshal resp err: %s", rsp.Error.Message)
	}
	height, err := strconv.ParseUint(rsp.Result, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("GetStarcoinNodeHeight, parse resp height %s failed", rsp.Result)
	} else {
		return height, nil
	}
}

type StarcoinKeyStore struct {
	privateKey *types.Ed25519PrivateKey
	chainId    int
}

type StarcoinAccount struct {
	Address types.AccountAddress `json:"address"` // Starcoin account address derived from the key
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
