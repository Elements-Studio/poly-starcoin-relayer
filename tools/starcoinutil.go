package tools

import (
	"encoding/json"
	"fmt"
	"strconv"

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
