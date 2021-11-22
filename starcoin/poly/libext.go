package poly

import (
	"strings"

	stcclient "github.com/starcoinorg/starcoin-go/client"
	diemtypes "github.com/starcoinorg/starcoin-go/types"
)

func ParseModuleId(str string) *diemtypes.ModuleId {
	ss := strings.Split(str, "::")
	if len(ss) < 2 {
		panic("module Id string format error")
	}
	addr, err := stcclient.ToAccountAddress(ss[0])
	if err != nil {
		panic("module Id string address format error")
	}
	return &diemtypes.ModuleId{
		Address: *addr,                       //[16]uint8{164, 216, 175, 70, 82, 187, 53, 191, 210, 134, 211, 71, 12, 28, 90, 61},
		Name:    diemtypes.Identifier(ss[1]), //"CrossChainScript",
	}
}

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

func EncodeCCMVerifyHeaderAndExecuteTxPayload(module string, proof []byte, raw_header []byte, header_proof []byte, cur_raw_header []byte, header_sig []byte) diemtypes.TransactionPayload {
	// copy from generated code:
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   *ParseModuleId(module), //diemtypes.ModuleId{Address: [16]uint8{164, 216, 175, 70, 82, 187, 53, 191, 210, 134, 211, 71, 12, 28, 90, 61}, Name: "CrossChainScript"},
			Function: "verifyHeaderAndExecuteTx",
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u8vector_argument(proof), encode_u8vector_argument(raw_header), encode_u8vector_argument(header_proof), encode_u8vector_argument(cur_raw_header), encode_u8vector_argument(header_sig)},
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
