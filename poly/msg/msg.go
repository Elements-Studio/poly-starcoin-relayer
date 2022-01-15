package msg

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/polynetwork/bridge-common/base"
	"github.com/polynetwork/bridge-common/chains/bridge"
	pcom "github.com/polynetwork/poly/common"
	"github.com/polynetwork/poly/core/types"
	"github.com/polynetwork/poly/native/service/cross_chain_manager/common"
)

type TxType int

const (
	SRC    TxType = 1
	POLY   TxType = 2
	HEADER TxType = 3
)

type Tx struct {
	TxType   TxType
	Attempts int

	TxId        string                `json:",omitempty"`
	MerkleValue *common.ToMerkleValue `json:"-"`
	Param       *common.MakeTxParam   `json:"-"`

	SrcHash        string `json:",omitempty"`
	SrcHeight      uint64 `json:",omitempty"`
	SrcChainId     uint64 `json:",omitempty"`
	SrcProof       []byte `json:"-"`
	SrcProofHex    string `json:",omitempty"`
	SrcEvent       []byte `json:"-"`
	SrcProofHeight uint64 `json:",omitempty"`
	SrcParam       string `json:",omitempty"`
	SrcStateRoot   []byte `json:"-"`
	SrcProxy       string `json:",omitempty"`
	SrcAddress     string `json:",omitempty"`

	PolyHash     string        `json:",omitempty"`
	PolyHeight   uint32        `json:",omitempty"`
	PolyKey      string        `json:",omitempty"`
	PolyHeader   *types.Header `json:"-"`
	AnchorHeader *types.Header `json:"-"`
	AnchorProof  string        `json:",omitempty"`
	AuditPath    string        `json:"-"`
	PolySigs     []byte        `json:"-"`

	DstAddress              string                `json:",omitempty"`
	DstHash                 string                `json:",omitempty"`
	DstHeight               uint64                `json:",omitempty"`
	DstChainId              uint64                `json:",omitempty"`
	DstGasLimit             uint64                `json:",omitempty"`
	DstGasPrice             string                `json:",omitempty"`
	DstGasPriceX            string                `json:",omitempty"`
	DstSender               interface{}           `json:"-"`
	DstPolyEpochStartHeight uint32                `json:",omitempty"`
	DstPolyKeepers          []byte                `json:"-"`
	DstData                 []byte                `json:"-"`
	DstProxy                string                `json:",omitempty"`
	SkipCheckFee            bool                  `json:",omitempty"`
	Skipped                 bool                  `json:",omitempty"`
	CheckFeeStatus          bridge.CheckFeeStatus `json:",omitempty"`
}

func (tx *Tx) Type() TxType {
	return tx.TxType
}

func (tx *Tx) Encode() string {
	if len(tx.SrcProof) > 0 && len(tx.SrcProofHex) == 0 {
		tx.SrcProofHex = hex.EncodeToString(tx.SrcProof)
	}
	bytes, _ := json.Marshal(*tx)
	return string(bytes)
}

func (tx *Tx) Decode(data string) (err error) {
	err = json.Unmarshal([]byte(data), tx)
	if err == nil {
		if len(tx.SrcParam) > 0 && tx.Param == nil {
			event, err := hex.DecodeString(tx.SrcParam)
			if err != nil {
				return fmt.Errorf("Decode src param error %v event %s", err, tx.SrcParam)
			}
			param := &common.MakeTxParam{}
			err = param.Deserialization(pcom.NewZeroCopySource(event))
			if err != nil {
				return fmt.Errorf("Decode src event error %v event %s", err, tx.SrcParam)
			}
			tx.Param = param
			tx.SrcEvent = event
		}
	}
	return
}

func (tx *Tx) CapturePatchParams(o *Tx) *Tx {
	if o != nil {
		if o.DstGasLimit > 0 {
			tx.DstGasLimit = o.DstGasLimit
		}
		if len(o.DstGasPrice) > 0 {
			tx.DstGasPrice = o.DstGasPrice
		}

		if len(o.DstGasPriceX) > 0 {
			tx.DstGasPriceX = o.DstGasPriceX
		}

		if o.SkipCheckFee {
			tx.SkipCheckFee = o.SkipCheckFee
		}
		if o.DstSender != nil {
			tx.DstSender = o.DstSender
		}
	}
	return tx
}

func (tx *Tx) SkipFee() bool {
	if tx.SkipCheckFee {
		return true
	}
	switch tx.DstChainId {
	case base.PLT, base.O3:
		return true
	}
	return false
}

func (tx *Tx) GetTxId() (id [32]byte, err error) {
	bytes, err := hex.DecodeString(tx.TxId)
	if err != nil {
		err = fmt.Errorf("GetTxId Invalid tx id hex %v", err)
		return
	}
	copy(id[:], bytes[:32])
	return
}
