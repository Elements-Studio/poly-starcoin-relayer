package manager

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"

	stcclient "github.com/starcoinorg/starcoin-go/client"
)

type CrossChainData struct {
	starcoinClient *stcclient.StarcoinClient
	module         string
}

// Get Current Epoch Start Height of Poly chain block.
// Move code:
// public(script) fun getCurEpochStartHeight(): u64
func (ccd *CrossChainData) getCurEpochStartHeight() (uint64, error) {
	c := stcclient.ContractCall{
		FunctionId: ccd.getFunctionId("getCurEpochStartHeight"),
	}
	r, err := ccd.starcoinClient.CallContract(context.Background(), c)
	if err != nil {
		return 0, err
	}
	return toUint64(r)
}

// Get Consensus book Keepers Public Key Bytes.
// Move code:
// public(script) fun getCurEpochConPubKeyBytes(): vector<u8>
func (ccd *CrossChainData) getCurEpochConPubKeyBytes() ([]byte, error) {
	c := stcclient.ContractCall{
		FunctionId: ccd.getFunctionId("getCurEpochConPubKeyBytes"),
	}
	r, err := ccd.starcoinClient.CallContract(context.Background(), c)
	if err != nil {
		return nil, err
	}
	return toBytes(r)
}

// Check if from chain tx fromChainTx has been processed before.
// Move code:
// public(script) fun checkIfFromChainTxExist(from_chain_id: u64, from_chain_tx: vector<u8>): bool
func (ccd *CrossChainData) checkIfFromChainTxExist(fromChainId uint64, fromChainTx []byte) (bool, error) {
	c := stcclient.ContractCall{
		FunctionId: ccd.getFunctionId("checkIfFromChainTxExist"),
		Args: []string{
			strconv.FormatUint(fromChainId, 10) + "u64",
			"x\"" + hex.EncodeToString(fromChainTx) + "x\"",
		},
	}
	r, err := ccd.starcoinClient.CallContract(context.Background(), c)
	if err != nil {
		return false, err
	}
	return toBool(r)
}

func (ccd *CrossChainData) getFunctionId(funcName string) string {
	return ccd.module + "::" + funcName
}

func toBool(i interface{}) (bool, error) {
	return false, nil //todo
}

func toBytes(i interface{}) ([]byte, error) {
	return nil, nil //todo
}

func toUint64(i interface{}) (uint64, error) {
	switch i.(type) {
	case string:
		return strconv.ParseUint(i.(string), 10, 64)
	}
	return 0, fmt.Errorf("unknown type to uint64")
}
