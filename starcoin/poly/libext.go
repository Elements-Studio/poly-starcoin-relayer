package poly

import (
	"strings"

	"github.com/novifinancial/serde-reflection/serde-generate/runtime/golang/serde"
	"github.com/starcoinorg/starcoin-go/types"
	diemtypes "github.com/starcoinorg/starcoin-go/types"
)

func EncodeCCMChangeBookKeeperTxPayload(module string, raw_header []byte, pub_key_list []byte, sig_list []byte) diemtypes.TransactionPayload {
	// copy from generated code:
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module), //diemtypes.ModuleId{Address: [16]uint8{164, 216, 175, 70, 82, 187, 53, 191, 210, 134, 211, 71, 12, 28, 90, 61}, Name: "CrossChainScript"},
			Function: "changeBookKeeper",
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u8vector_argument(raw_header), encode_u8vector_argument(pub_key_list), encode_u8vector_argument(sig_list)},
		},
	}
}

// func EncodeCCMVerifyHeaderAndExecuteTxPayload(module string, proof []byte, raw_header []byte, header_proof []byte, cur_raw_header []byte, header_sig []byte) diemtypes.TransactionPayload {
// 	// copy from generated code:
// 	return &diemtypes.TransactionPayload__ScriptFunction{
// 		diemtypes.ScriptFunction{
// 			Module:   *ParseModuleId(module), //diemtypes.ModuleId{Address: [16]uint8{164, 216, 175, 70, 82, 187, 53, 191, 210, 134, 211, 71, 12, 28, 90, 61}, Name: "CrossChainScript"},
// 			Function: "verifyHeaderAndExecuteTx",
// 			TyArgs:   []diemtypes.TypeTag{},
// 			Args:     [][]byte{encode_u8vector_argument(proof), encode_u8vector_argument(raw_header), encode_u8vector_argument(header_proof), encode_u8vector_argument(cur_raw_header), encode_u8vector_argument(header_sig)},
// 		},
// 	}
// }

func EncodeCCMVerifyHeaderAndExecuteTxPayload(module string, proof []byte, raw_header []byte, header_proof []byte, cur_raw_header []byte, header_sig []byte, merkle_proof_root []byte, merkle_proof_leaf []byte, merkle_proof_siblings []byte) diemtypes.TransactionPayload {
	// copy from generated code:
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module), //diemtypes.ModuleId{Address: [16]uint8{164, 216, 175, 70, 82, 187, 53, 191, 210, 134, 211, 71, 12, 28, 90, 61}, Name: "CrossChainScript"},
			Function: "verify_header_and_execute_tx",
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u8vector_argument(proof), encode_u8vector_argument(raw_header), encode_u8vector_argument(header_proof), encode_u8vector_argument(cur_raw_header), encode_u8vector_argument(header_sig), encode_u8vector_argument(merkle_proof_root), encode_u8vector_argument(merkle_proof_leaf), encode_u8vector_argument(merkle_proof_siblings)},
		},
	}
}

func EncodeInitGenesisTxPayload(module string, raw_header []byte, pub_key_list []byte) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: "init_genesis",
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u8vector_argument(raw_header), encode_u8vector_argument(pub_key_list)},
		},
	}
}

func EncodeLockAssetTxPayload(module string, from_asset_hash []byte, to_chain_id uint64, to_address []byte, amount serde.Uint128) diemtypes.TransactionPayload {
	// public(script) fun lock(signer: signer,
	// 	from_asset_hash: vector<u8>,
	// 	to_chain_id: u64,
	// 	to_address: vector<u8>,
	// 	amount: u128) {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: "lock",
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u8vector_argument(from_asset_hash), encode_u64_argument(to_chain_id), encode_u8vector_argument(to_address), encode_u128_argument(amount)},
		},
	}
}

func EncodeLockAssetWithStcFeeTxPayload(module string, from_asset_hash []byte, to_chain_id uint64, to_address []byte, amount serde.Uint128, fee serde.Uint128, id serde.Uint128) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: "lock_with_stc_fee",
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u8vector_argument(from_asset_hash), encode_u64_argument(to_chain_id), encode_u8vector_argument(to_address), encode_u128_argument(amount), encode_u128_argument(fee), encode_u128_argument(id)},
		},
	}
}

func EncodeBindProxyHashTxPayload(module string, chain_id uint64, proxy_hash []byte) diemtypes.TransactionPayload {
	// public(script) fun bind_proxy_hash(signer: signer,
	// 	to_chain_id: u64,
	// 	target_proxy_hash: vector<u8>) {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: "bind_proxy_hash",
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u64_argument(chain_id), encode_u8vector_argument(proxy_hash)},
		},
	}
}

func EncodeBindAssetHashTxPayload(module string, from_asset_hash []byte, to_chain_id uint64, to_asset_hash []byte) diemtypes.TransactionPayload {
	// public(script) fun bind_asset_hash(signer: signer,
	// 	from_asset_hash: vector<u8>,
	// 	to_chain_id: u64,
	// 	to_asset_hash: vector<u8>) {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: "bind_asset_hash",
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u8vector_argument(from_asset_hash), encode_u64_argument(to_chain_id), encode_u8vector_argument(to_asset_hash)},
		},
	}
}

func EncodeSetChainIdTxPayload(module string, chainType diemtypes.TypeTag, chain_id uint64) diemtypes.TransactionPayload {
	//public(script) fun set_chain_id<ChainType: store>(signer: signer, chain_id: u64) {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: "set_chain_id",
			TyArgs:   []diemtypes.TypeTag{chainType},
			Args:     [][]byte{encode_u64_argument(chain_id)},
		},
	}
}

func EncodeEmptyArgsTxPaylaod(module string, function string) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: diemtypes.Identifier(function),
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{},
		},
	}
}

func EncodeU128TxPaylaod(module string, function string, u serde.Uint128) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: diemtypes.Identifier(function),
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u128_argument(u)},
		},
	}
}

func EncodeU64AndU8TxPaylaod(module string, function string, u1 uint64, u2 uint8) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: diemtypes.Identifier(function),
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u64_argument(u1), encode_u8_argument(u2)},
		},
	}
}

func EncodeOneTypeArgAndU128TxPaylaod(module string, function string, t1 diemtypes.TypeTag, u serde.Uint128) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: diemtypes.Identifier(function),
			TyArgs:   []diemtypes.TypeTag{t1},
			Args:     [][]byte{encode_u128_argument(u)},
		},
	}
}

func EncodeTwoTypeArgsAndU128TxPaylaod(module string, function string, t1, t2 diemtypes.TypeTag, u serde.Uint128) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: diemtypes.Identifier(function),
			TyArgs:   []diemtypes.TypeTag{t1, t2},
			Args:     [][]byte{encode_u128_argument(u)},
		},
	}
}

func EncodeTwoTypeArgsAndTwoU128TxPaylaod(module string, function string, t1, t2 diemtypes.TypeTag, u1, u2 serde.Uint128) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: diemtypes.Identifier(function),
			TyArgs:   []diemtypes.TypeTag{t1, t2},
			Args:     [][]byte{encode_u128_argument(u1), encode_u128_argument(u2)},
		},
	}
}

func EncodeTwoTypeArgsAndThreeU128TxPaylaod(module string, function string, t1, t2 diemtypes.TypeTag, u1, u2, u3 serde.Uint128) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: diemtypes.Identifier(function),
			TyArgs:   []diemtypes.TypeTag{t1, t2},
			Args:     [][]byte{encode_u128_argument(u1), encode_u128_argument(u2), encode_u128_argument(u3)},
		},
	}
}

func EncodeTwoTypeArgsAndFourU128TxPaylaod(module string, function string, t1, t2 diemtypes.TypeTag, u1, u2, u3, u4 serde.Uint128) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: diemtypes.Identifier(function),
			TyArgs:   []diemtypes.TypeTag{t1, t2},
			Args:     [][]byte{encode_u128_argument(u1), encode_u128_argument(u2), encode_u128_argument(u3), encode_u128_argument(u4)},
		},
	}
}

func EncodeAccountAddressTxPaylaod(module string, function string, a diemtypes.AccountAddress) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: diemtypes.Identifier(function),
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_address_argument(a)},
		},
	}
}

func EncodeBoolTxPaylaod(module string, function string, b bool) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module),
			Function: diemtypes.Identifier(function),
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_bool_argument(b)},
		},
	}
}

func EncodeTransferStcTxPayload(payee types.AccountAddress, amount serde.Uint128) types.TransactionPayload {
	coinType := types.StructTag{
		Address: [16]uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		Module:  types.Identifier("STC"),
		Name:    types.Identifier("STC"),
	}
	return EncodePeerToPeerV2ScriptFunction(&types.TypeTag__Struct{Value: coinType}, payee, amount)
}

func EncodePeerToPeerV2ScriptFunction(currency types.TypeTag, payee types.AccountAddress, amount serde.Uint128) types.TransactionPayload {
	return &types.TransactionPayload__ScriptFunction{
		types.ScriptFunction{
			Module:   types.ModuleId{Address: [16]uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, Name: "TransferScripts"},
			Function: "peer_to_peer_v2",
			TyArgs:   []types.TypeTag{currency},
			Args:     [][]byte{encode_address_argument(payee), encode_u128_argument(amount)},
		},
	}
}

func ParseModuleId(str string) *diemtypes.ModuleId {
	ss := strings.Split(str, "::")
	if len(ss) < 2 {
		panic("module Id string format error")
	}
	addr, err := diemtypes.ToAccountAddress(ss[0])
	if err != nil {
		panic("module Id string address format error")
	}
	return &diemtypes.ModuleId{
		Address: *addr,                       //[16]uint8{164, 216, 175, 70, 82, 187, 53, 191, 210, 134, 211, 71, 12, 28, 90, 61},
		Name:    diemtypes.Identifier(ss[1]), //"CrossChainScript",
	}
}

func encode_address_argument(arg types.AccountAddress) []byte {
	if val, err := arg.BcsSerialize(); err == nil {
		{
			return val
		}
	}
	panic("Unable to serialize argument of type address")
}
