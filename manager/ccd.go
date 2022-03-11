package manager

import (
	"context"

	"github.com/elements-studio/poly-starcoin-relayer/tools"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

type CrossChainData struct {
	starcoinClient *stcclient.StarcoinClient
	module         string
}

func NewCrossChainData(client *stcclient.StarcoinClient, module string) (*CrossChainData, error) {
	_, _, err := tools.ParseStarcoinModuleId(module)
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
		FunctionId: ccd.getFunctionId("get_cur_epoch_start_height"),
		TypeArgs:   []string{},
		Args:       []string{},
	}
	r, err := ccd.starcoinClient.CallContract(context.Background(), c)
	if err != nil {
		return 0, err
	}
	return tools.ToUint64(tools.ExtractSingleResult(r))
}

// Get Consensus book Keepers Public Key Bytes.
// Move code:
// public(script) fun getCurEpochConPubKeyBytes(): vector<u8>
func (ccd *CrossChainData) getCurEpochConPubKeyBytes() ([]byte, error) {
	c := stcclient.ContractCall{
		FunctionId: ccd.getFunctionId("get_cur_epoch_con_pubkey_bytes"),
		TypeArgs:   []string{},
		Args:       []string{},
	}
	r, err := ccd.starcoinClient.CallContract(context.Background(), c)
	if err != nil {
		return nil, err
	}
	return tools.ToBytes(tools.ExtractSingleResult(r))
}

// // Check if from chain tx fromChainTx has been processed before.
// // Move code:
// // public(script) fun checkIfFromChainTxExist(from_chain_id: u64, from_chain_tx: vector<u8>): bool
// func (ccd *CrossChainData) checkIfFromChainTxExist(fromChainId uint64, fromChainTx []byte) (bool, error) {
// 	c := stcclient.ContractCall{
// 		FunctionId: ccd.getFunctionId("checkIfFromChainTxExist"),
// 		TypeArgs:   []string{},
// 		Args: []string{
// 			strconv.FormatUint(fromChainId, 10) + "u64",
// 			"x\"" + hex.EncodeToString(fromChainTx) + "x\"",
// 		},
// 	}
// 	r, err := ccd.starcoinClient.CallContract(context.Background(), c)
// 	if err != nil {
// 		return false, err
// 	}
// 	return toBool(extractSingleResult(r))
// }

// Get on-chain cross chain txn. SMT root hash.
func (ccd *CrossChainData) getOnChainTxSparseMerkleTreeRootHash() ([]byte, error) {
	addr, _, err := tools.ParseStarcoinModuleId(ccd.module)
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
	return tools.ToBytes(tools.ExtractSingleResult(r))
}

func (ccd *CrossChainData) getFunctionId(funcName string) string {
	return ccd.module + "::" + funcName
}

// func ExtractSingleResult(result interface{}) interface{} {
// 	r := result.([]interface{})
// 	if len(r) == 0 {
// 		return nil
// 	}
// 	return r[0]
// }

// func ToBool(i interface{}) (bool, error) {
// 	switch i := i.(type) {
// 	case bool:
// 		return i, nil
// 	case string:
// 		return strconv.ParseBool(i)
// 	}
// 	return false, fmt.Errorf("unknown type to bool %t", i)
// }
