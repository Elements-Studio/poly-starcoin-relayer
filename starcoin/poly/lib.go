package poly

import (
	"fmt"

	"github.com/novifinancial/serde-reflection/serde-generate/runtime/golang/bcs"
	"github.com/novifinancial/serde-reflection/serde-generate/runtime/golang/serde"
	diemtypes "github.com/starcoinorg/starcoin-go/types"
)

// Structured representation of a call into a known Move transaction script (legacy).
type ScriptCall interface {
	isScriptCall()
}

// Structured representation of a call into a known Move script function.
type ScriptFunctionCall interface {
	isScriptFunctionCall()
}

// Check book keeper information
type ScriptFunctionCall__ChangeBookKeeper struct {
	RawHeader  []byte
	PubKeyList []byte
	SigList    []byte
}

func (*ScriptFunctionCall__ChangeBookKeeper) isScriptFunctionCall() {}

// Initialize genesis from contract owner
type ScriptFunctionCall__InitGenesis struct {
	RawHeader  []byte
	PubKeyList []byte
}

func (*ScriptFunctionCall__InitGenesis) isScriptFunctionCall() {}

type ScriptFunctionCall__Lock struct {
	TokenType diemtypes.TypeTag
	ChainType diemtypes.TypeTag
	ToAddress []byte
	Amount    serde.Uint128
}

func (*ScriptFunctionCall__Lock) isScriptFunctionCall() {}

type ScriptFunctionCall__VerifyHeaderAndExecuteTx struct {
	Proof        []byte
	RawHeader    []byte
	HeaderProof  []byte
	CurRawHeader []byte
	HeaderSig    []byte
}

func (*ScriptFunctionCall__VerifyHeaderAndExecuteTx) isScriptFunctionCall() {}

// Build a Diem `TransactionPayload` from a structured object `ScriptFunctionCall`.
func EncodeScriptFunction(call ScriptFunctionCall) diemtypes.TransactionPayload {
	switch call := call.(type) {
	case *ScriptFunctionCall__ChangeBookKeeper:
		return EncodeChangeBookKeeperScriptFunction(call.RawHeader, call.PubKeyList, call.SigList)
	case *ScriptFunctionCall__InitGenesis:
		return EncodeInitGenesisScriptFunction(call.RawHeader, call.PubKeyList)
	case *ScriptFunctionCall__Lock:
		return EncodeLockScriptFunction(call.TokenType, call.ChainType, call.ToAddress, call.Amount)
	case *ScriptFunctionCall__VerifyHeaderAndExecuteTx:
		return EncodeVerifyHeaderAndExecuteTxScriptFunction(call.Proof, call.RawHeader, call.HeaderProof, call.CurRawHeader, call.HeaderSig)
	}
	panic("unreachable")
}

// Try to recognize a Diem `Script` and convert it into a structured object `ScriptCall`.
func DecodeScript(script *diemtypes.Script) (ScriptCall, error) {
	if helper := script_decoder_map[string(script.Code)]; helper != nil {
		val, err := helper(script)
		return val, err
	} else {
		return nil, fmt.Errorf("Unknown script bytecode: %s", string(script.Code))
	}
}

// Try to recognize a Diem `TransactionPayload` and convert it into a structured object `ScriptFunctionCall`.
func DecodeScriptFunctionPayload(script diemtypes.TransactionPayload) (ScriptFunctionCall, error) {
	switch script := script.(type) {
	case *diemtypes.TransactionPayload__ScriptFunction:
		if helper := script_function_decoder_map[string(script.Value.Module.Name)+string(script.Value.Function)]; helper != nil {
			val, err := helper(script)
			return val, err
		} else {
			return nil, fmt.Errorf("Unknown script function: %s::%s", script.Value.Module.Name, script.Value.Function)
		}
	default:
		return nil, fmt.Errorf("Unknown transaction payload encountered when decoding")
	}
}

// Check book keeper information
func EncodeChangeBookKeeperScriptFunction(raw_header []byte, pub_key_list []byte, sig_list []byte) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   diemtypes.ModuleId{Address: [16]uint8{164, 216, 175, 70, 82, 187, 53, 191, 210, 134, 211, 71, 12, 28, 90, 61}, Name: "CrossChainScript"},
			Function: "changeBookKeeper",
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u8vector_argument(raw_header), encode_u8vector_argument(pub_key_list), encode_u8vector_argument(sig_list)},
		},
	}
}

// Initialize genesis from contract owner
func EncodeInitGenesisScriptFunction(raw_header []byte, pub_key_list []byte) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   diemtypes.ModuleId{Address: [16]uint8{164, 216, 175, 70, 82, 187, 53, 191, 210, 134, 211, 71, 12, 28, 90, 61}, Name: "CrossChainScript"},
			Function: "init_genesis",
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u8vector_argument(raw_header), encode_u8vector_argument(pub_key_list)},
		},
	}
}

func EncodeLockScriptFunction(token_type diemtypes.TypeTag, chain_type diemtypes.TypeTag, to_address []byte, amount serde.Uint128) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   diemtypes.ModuleId{Address: [16]uint8{164, 216, 175, 70, 82, 187, 53, 191, 210, 134, 211, 71, 12, 28, 90, 61}, Name: "CrossChainScript"},
			Function: "lock",
			TyArgs:   []diemtypes.TypeTag{token_type, chain_type},
			Args:     [][]byte{encode_u8vector_argument(to_address), encode_u128_argument(amount)},
		},
	}
}

func EncodeVerifyHeaderAndExecuteTxScriptFunction(proof []byte, raw_header []byte, header_proof []byte, cur_raw_header []byte, header_sig []byte) diemtypes.TransactionPayload {
	return &diemtypes.TransactionPayload__ScriptFunction{
		diemtypes.ScriptFunction{
			Module:   diemtypes.ModuleId{Address: [16]uint8{164, 216, 175, 70, 82, 187, 53, 191, 210, 134, 211, 71, 12, 28, 90, 61}, Name: "CrossChainScript"},
			Function: "verifyHeaderAndExecuteTx",
			TyArgs:   []diemtypes.TypeTag{},
			Args:     [][]byte{encode_u8vector_argument(proof), encode_u8vector_argument(raw_header), encode_u8vector_argument(header_proof), encode_u8vector_argument(cur_raw_header), encode_u8vector_argument(header_sig)},
		},
	}
}

func decode_changeBookKeeper_script_function(script diemtypes.TransactionPayload) (ScriptFunctionCall, error) {
	switch script := interface{}(script).(type) {
	case *diemtypes.TransactionPayload__ScriptFunction:
		if len(script.Value.TyArgs) < 0 {
			return nil, fmt.Errorf("Was expecting 0 type arguments")
		}
		if len(script.Value.Args) < 3 {
			return nil, fmt.Errorf("Was expecting 3 regular arguments")
		}
		var call ScriptFunctionCall__ChangeBookKeeper

		if val, err := bcs.NewDeserializer(script.Value.Args[0]).DeserializeBytes(); err == nil {
			call.RawHeader = val
		} else {
			return nil, err
		}

		if val, err := bcs.NewDeserializer(script.Value.Args[1]).DeserializeBytes(); err == nil {
			call.PubKeyList = val
		} else {
			return nil, err
		}

		if val, err := bcs.NewDeserializer(script.Value.Args[2]).DeserializeBytes(); err == nil {
			call.SigList = val
		} else {
			return nil, err
		}

		return &call, nil
	default:
		return nil, fmt.Errorf("Unexpected TransactionPayload encountered when decoding a script function")
	}
}

func decode_init_genesis_script_function(script diemtypes.TransactionPayload) (ScriptFunctionCall, error) {
	switch script := interface{}(script).(type) {
	case *diemtypes.TransactionPayload__ScriptFunction:
		if len(script.Value.TyArgs) < 0 {
			return nil, fmt.Errorf("Was expecting 0 type arguments")
		}
		if len(script.Value.Args) < 2 {
			return nil, fmt.Errorf("Was expecting 2 regular arguments")
		}
		var call ScriptFunctionCall__InitGenesis

		if val, err := bcs.NewDeserializer(script.Value.Args[0]).DeserializeBytes(); err == nil {
			call.RawHeader = val
		} else {
			return nil, err
		}

		if val, err := bcs.NewDeserializer(script.Value.Args[1]).DeserializeBytes(); err == nil {
			call.PubKeyList = val
		} else {
			return nil, err
		}

		return &call, nil
	default:
		return nil, fmt.Errorf("Unexpected TransactionPayload encountered when decoding a script function")
	}
}

func decode_lock_script_function(script diemtypes.TransactionPayload) (ScriptFunctionCall, error) {
	switch script := interface{}(script).(type) {
	case *diemtypes.TransactionPayload__ScriptFunction:
		if len(script.Value.TyArgs) < 2 {
			return nil, fmt.Errorf("Was expecting 2 type arguments")
		}
		if len(script.Value.Args) < 2 {
			return nil, fmt.Errorf("Was expecting 2 regular arguments")
		}
		var call ScriptFunctionCall__Lock
		call.TokenType = script.Value.TyArgs[0]
		call.ChainType = script.Value.TyArgs[1]

		if val, err := bcs.NewDeserializer(script.Value.Args[0]).DeserializeBytes(); err == nil {
			call.ToAddress = val
		} else {
			return nil, err
		}

		if val, err := bcs.NewDeserializer(script.Value.Args[1]).DeserializeU128(); err == nil {
			call.Amount = val
		} else {
			return nil, err
		}

		return &call, nil
	default:
		return nil, fmt.Errorf("Unexpected TransactionPayload encountered when decoding a script function")
	}
}

func decode_verifyHeaderAndExecuteTx_script_function(script diemtypes.TransactionPayload) (ScriptFunctionCall, error) {
	switch script := interface{}(script).(type) {
	case *diemtypes.TransactionPayload__ScriptFunction:
		if len(script.Value.TyArgs) < 0 {
			return nil, fmt.Errorf("Was expecting 0 type arguments")
		}
		if len(script.Value.Args) < 5 {
			return nil, fmt.Errorf("Was expecting 5 regular arguments")
		}
		var call ScriptFunctionCall__VerifyHeaderAndExecuteTx

		if val, err := bcs.NewDeserializer(script.Value.Args[0]).DeserializeBytes(); err == nil {
			call.Proof = val
		} else {
			return nil, err
		}

		if val, err := bcs.NewDeserializer(script.Value.Args[1]).DeserializeBytes(); err == nil {
			call.RawHeader = val
		} else {
			return nil, err
		}

		if val, err := bcs.NewDeserializer(script.Value.Args[2]).DeserializeBytes(); err == nil {
			call.HeaderProof = val
		} else {
			return nil, err
		}

		if val, err := bcs.NewDeserializer(script.Value.Args[3]).DeserializeBytes(); err == nil {
			call.CurRawHeader = val
		} else {
			return nil, err
		}

		if val, err := bcs.NewDeserializer(script.Value.Args[4]).DeserializeBytes(); err == nil {
			call.HeaderSig = val
		} else {
			return nil, err
		}

		return &call, nil
	default:
		return nil, fmt.Errorf("Unexpected TransactionPayload encountered when decoding a script function")
	}
}

var script_decoder_map = map[string]func(*diemtypes.Script) (ScriptCall, error){}

var script_function_decoder_map = map[string]func(diemtypes.TransactionPayload) (ScriptFunctionCall, error){
	"CrosschainScriptchangeBookKeeper":         decode_changeBookKeeper_script_function,
	"CrosschainScriptinit_genesis":             decode_init_genesis_script_function,
	"CrosschainScriptlock":                     decode_lock_script_function,
	"CrosschainScriptverifyHeaderAndExecuteTx": decode_verifyHeaderAndExecuteTx_script_function,
}

func encode_u128_argument(arg serde.Uint128) []byte {

	s := bcs.NewSerializer()
	if err := s.SerializeU128(arg); err == nil {
		return s.GetBytes()
	}

	panic("Unable to serialize argument of type u128")
}

func encode_u8vector_argument(arg []byte) []byte {

	s := bcs.NewSerializer()
	if err := s.SerializeBytes(arg); err == nil {
		return s.GetBytes()
	}

	panic("Unable to serialize argument of type u8vector")
}

func encode_vecbytes_argument(arg diemtypes.VecBytes) []byte {

	if val, err := arg.BcsSerialize(); err == nil {
		{
			return val
		}
	}

	panic("Unable to serialize argument of type vecbytes")
}

func encode_u64_argument(arg uint64) []byte {

	s := bcs.NewSerializer()
	if err := s.SerializeU64(arg); err == nil {
		return s.GetBytes()
	}

	panic("Unable to serialize argument of type u64")
}

func encode_u8_argument(arg uint8) []byte {

	s := bcs.NewSerializer()
	if err := s.SerializeU8(arg); err == nil {
		return s.GetBytes()
	}

	panic("Unable to serialize argument of type u64")
}

func encode_bool_argument(arg bool) []byte {
	s := bcs.NewSerializer()
	if err := s.SerializeBool(arg); err == nil {
		return s.GetBytes()
	}
	panic("Unable to serialize argument of type bool")
}
