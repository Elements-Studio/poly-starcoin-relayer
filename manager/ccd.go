package manager

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/elements-studio/poly-starcoin-relayer/tools"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

type CrossChainData struct {
	starcoinClient *stcclient.StarcoinClient
	module         string
}

func NewCrossChainData(client *stcclient.StarcoinClient, module string) (*CrossChainData, error) {
	_, _, err := parseModuleId(module)
	if err != nil {
		return nil, err
	}
	return &CrossChainData{
		starcoinClient: client,
		module:         module,
	}, nil
}

// Get Current Epoch Start Height of Poly chain block.
// Move code:
// public(script) fun getCurEpochStartHeight(): u64
func (ccd *CrossChainData) getCurEpochStartHeight() (uint64, error) {
	c := stcclient.ContractCall{
		FunctionId: ccd.getFunctionId("getCurEpochStartHeight"),
		TypeArgs:   []string{},
		Args:       []string{},
	}
	r, err := ccd.starcoinClient.CallContract(context.Background(), c)
	if err != nil {
		return 0, err
	}
	return toUint64(extractSingleResult(r))
}

// Get Consensus book Keepers Public Key Bytes.
// Move code:
// public(script) fun getCurEpochConPubKeyBytes(): vector<u8>
func (ccd *CrossChainData) getCurEpochConPubKeyBytes() ([]byte, error) {
	c := stcclient.ContractCall{
		FunctionId: ccd.getFunctionId("getCurEpochConPubKeyBytes"),
		TypeArgs:   []string{},
		Args:       []string{},
	}
	r, err := ccd.starcoinClient.CallContract(context.Background(), c)
	if err != nil {
		return nil, err
	}
	return toBytes(extractSingleResult(r))
}

// Check if from chain tx fromChainTx has been processed before.
// Move code:
// public(script) fun checkIfFromChainTxExist(from_chain_id: u64, from_chain_tx: vector<u8>): bool
func (ccd *CrossChainData) checkIfFromChainTxExist(fromChainId uint64, fromChainTx []byte) (bool, error) {
	c := stcclient.ContractCall{
		FunctionId: ccd.getFunctionId("checkIfFromChainTxExist"),
		TypeArgs:   []string{},
		Args: []string{
			strconv.FormatUint(fromChainId, 10) + "u64",
			"x\"" + hex.EncodeToString(fromChainTx) + "x\"",
		},
	}
	r, err := ccd.starcoinClient.CallContract(context.Background(), c)
	if err != nil {
		return false, err
	}
	return toBool(extractSingleResult(r))
}

// Get on-chain cross chain txn. SMT root hash.
func (ccd *CrossChainData) getOnChainTxSparseMerkleTreeRootHash() ([]byte, error) {
	addr, _, err := parseModuleId(ccd.module)
	if err != nil {
		return nil, err
	}
	// Move code:
	// /// Query merkle root hash
	// public fun get_merkle_root_hash(): vector<u8> acquires SparseMerkleTreeRoot {
	// //...
	c := stcclient.ContractCall{
		FunctionId: addr + "::CrossChainData::get_merkle_root_hash",
		TypeArgs:   []string{}, //{addr + "::CrossChainType::Starcoin"},
		Args:       []string{},
	}
	r, err := ccd.starcoinClient.CallContract(context.Background(), c)
	if err != nil {
		return nil, err
	}
	return toBytes(extractSingleResult(r))
}

func (ccd *CrossChainData) getFunctionId(funcName string) string {
	return ccd.module + "::" + funcName
}

func extractSingleResult(result interface{}) interface{} {
	r := result.([]interface{})
	if len(r) == 0 {
		return nil
	}
	return r[0]
}

func toBool(i interface{}) (bool, error) {
	switch i := i.(type) {
	case bool:
		return i, nil
	case string:
		return strconv.ParseBool(i)
	}
	return false, fmt.Errorf("unknown type to bool %t", i)
}

func toBytes(i interface{}) ([]byte, error) {
	switch i := i.(type) {
	case []byte:
		return i, nil
	case string:
		return tools.HexToBytes(i)
	}
	return nil, fmt.Errorf("unknown type to []byte %t", i)
}

func toUint64(i interface{}) (uint64, error) {
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

// Parse module Id., return address and module name.
func parseModuleId(str string) (string, string, error) {
	ss := strings.Split(str, "::")
	if len(ss) < 2 {
		return "", "", fmt.Errorf("module Id string format error")
	} else if len(ss) > 2 {
		return "", "", fmt.Errorf("module Id string format error")
	}
	return ss[0], ss[1], nil
}
