package db

import (
	"encoding/hex"
	"encoding/json"

	"github.com/celestiaorg/smt"
	"github.com/elements-studio/poly-starcoin-relayer/poly/msg"
	rsmt "github.com/elements-studio/poly-starcoin-relayer/smt"
	pcommon "github.com/polynetwork/poly/common"
)

type ChainHeight struct {
	Key    string `gorm:"primaryKey;size:66"`
	Height uint32
}

type StarcoinTxCheck struct {
	TxHash            string `gorm:"primaryKey;size:66"` // Poly Transaction Hash
	CrossTransferData string `gorm:"size:5000"`          // CrossTransfer serialized data
	StarcoinEvent     string `gorm:"size:10000"`         // Starcoin Event
}

type StarcoinTxRetry struct {
	CrossTransferDataHash string `gorm:"primaryKey;size:66"` // CrossTransferData hash, as primary key
	CrossTransferData     string `gorm:"size:5000"`          // CrossTransfer serialized data
	StarcoinEvent         string `gorm:"size:10000"`         // Starcoin Event
}

// type CheckFeeStatus int
// const (
// 	SKIP     CheckFeeStatus = -2 // Skip since not our tx
// 	NOT_PAID CheckFeeStatus = -1 // Not paid or paid too low
// 	MISSING  CheckFeeStatus = 0  // Tx not received yet
// 	PAID     CheckFeeStatus = 1  // Paid and enough pass
// )

const (
	FEE_STATUS_SKIP     string = "-2" // Skip since not our tx
	FEE_STATUS_NOT_PAID string = "-1" // Not paid or paid too low
	FEE_STATUS_MISSING  string = "0"  // Tx not received yet
	FEE_STATUS_PAID     string = "1"  // Paid and enough pass

	STARCOIN_STATUS_NOT_ACCEPT_TOKEN string = "NOT_ACCEPT_TOKEN"
	STARCOIN_STATUS_PARSE_TX_ERROR   string = "PARSE_TX_ERROR"
)

type PolyTxRetry struct {
	FromChainID       uint64 `gorm:"size:66;primaryKey"`
	TxHash            string `gorm:"size:66;primaryKey"` // TxHash //TxData   string `gorm:"size:5000"`
	BridgeTransaction string `gorm:"size:36000"`
	PolyEvent         string `gorm:"size:5000"`
	PolyTxHash        string `gorm:"size:66"` // only use this hash for memo

	FeeStatus string `gorm:"size:20"`
	UpdatedAt int64  `gorm:"autoUpdateTime:milli"`

	CheckFeeCount int   `gorm:"default:0;NOT NULL"`
	Version       int64 `gorm:"column:version;default:0;NOT NULL"`

	StarcoinStatus       string `gorm:"size:20"`
	CheckStarcoinCount   int    `gorm:"default:0;NOT NULL"`
	CheckStarcoinMessage string `gorm:"size:200"`
}

func (o *PolyTxRetry) GetVersion() int64 {
	return o.Version
}

func (o *PolyTxRetry) SetVersion(version int64) {
	o.Version = version
}

func NewPolyTxRetry(txHash []byte, fromChainID uint64, bridgeTransaction []byte, polyEvent *msg.Tx) (*PolyTxRetry, error) {
	ej, err := json.Marshal(polyEvent)
	if err != nil {
		return nil, err
	}
	return &PolyTxRetry{
		TxHash:            hex.EncodeToString(txHash),
		FromChainID:       fromChainID,
		BridgeTransaction: hex.EncodeToString(bridgeTransaction),
		PolyEvent:         string(ej),
		PolyTxHash:        polyEvent.PolyHash,
		FeeStatus:         FEE_STATUS_NOT_PAID, //Is this ok?
	}, nil
}

func (r *PolyTxRetry) GetPolyEvent() (*msg.Tx, error) {
	e := &msg.Tx{}
	err := json.Unmarshal([]byte(r.PolyEvent), e)
	if err != nil {
		return nil, err
	}
	return e, nil
}

type PolyTxProof struct {
	Proof        string //`gorm:"size:5000"` // 	bytes memory proof,
	Header       string //`gorm:"size:5000"` // 	bytes memory rawHeader,
	HeaderProof  string //`gorm:"size:5000"` // 	bytes memory headerProof,
	AnchorHeader string //`gorm:"size:5000"` // 	bytes memory curRawHeader,
	HeaderSig    string //`gorm:"size:5000"` // 	bytes memory headerSig
}

// Poly transaction(to Starcoin)
type PolyTx struct {
	TxIndex     uint64 `gorm:"primaryKey;autoIncrement:false"`
	FromChainID uint64 `gorm:"size:66;uniqueIndex:uni_fromchainid_txhash"`
	TxHash      string `gorm:"size:66;uniqueIndex:uni_fromchainid_txhash"` // TxHash //TxData   string `gorm:"size:5000"`
	PolyTxProof string `gorm:"size:36000"`
	//------------------- for Non-Membership proof -------------------
	SmtTxPath                string `gorm:"size:66;uniqueIndex"`
	SmtNonMembershipRootHash string `gorm:"size:66"`
	SmtProofSideNodes        string `gorm:"size:18000"`

	// NonMembershipLeafData is the data of the unrelated leaf at the position
	// of the key being proven, in the case of a non-membership proof. For
	// membership proofs, is nil.
	SmtProofNonMembershipLeafData string `gorm:"size:132"`

	// SiblingData is the data of the sibling node to the leaf being proven,
	// required for updatable proofs. For unupdatable proofs, is nil.
	SmtProofSiblingData string `gorm:"size:132"`

	// BitMask, in the case of a compact proof, is a bit mask of the sidenodes
	// of the proof where an on-bit indicates that the sidenode at the bit's
	// index is a placeholder. This is only set if the proof is compact.
	SmtProofBitMask string `gorm:"size:66"`

	// NumSideNodes, in the case of a compact proof, indicates the number of
	// sidenodes in the proof when decompacted. This is only set if the proof is compact.
	SmtProofNumSideNodes int

	UpdatedAt int64 `gorm:"autoUpdateTime:milli;index"`

	Status string `gorm:"size:20;index"`

	StarcoinTxHash string `gorm:"size:66"`
	PolyTxHash     string `gorm:"size:66"`

	RetryCount int   `gorm:"default:0;NOT NULL"`
	Version    int64 `gorm:"column:version;default:0;NOT NULL"`
}

//
// note: alter table `poly_tx` convert to charset latin1;
//

func (o *PolyTx) GetVersion() int64 {
	return o.Version
}

func (o *PolyTx) SetVersion(version int64) {
	o.Version = version
}

func (p *PolyTx) GetNonMembershipProof() (*smt.SparseMerkleProof, error) {
	var proof *smt.SparseMerkleProof
	leafData, err := p.GetSmtProofNonMembershipLeafData()
	if err != nil {
		return nil, err
	}
	sns, err := DecodeSmtProofSideNodes(p.SmtProofSideNodes)
	if err != nil {
		return nil, err
	}
	siblingData, err := p.GetSmtProofSiblingData()
	if err != nil {
		return nil, err
	}
	proof = &smt.SparseMerkleProof{
		SideNodes:             sns,
		NonMembershipLeafData: leafData,
		SiblingData:           siblingData,
	}
	return proof, nil
}

func (p *PolyTx) GetSmtProofSideNodes() ([][]byte, error) {
	return DecodeSmtProofSideNodes(p.SmtProofSideNodes)
}

func (p *PolyTx) GetSmtProofNonMembershipLeafData() ([]byte, error) {
	if len(p.SmtProofNonMembershipLeafData) != 0 {
		return hex.DecodeString(p.SmtProofNonMembershipLeafData)
	}
	return nil, nil
}

func (p *PolyTx) GetSmtNonMembershipRootHash() ([]byte, error) {
	return hex.DecodeString(p.SmtNonMembershipRootHash)
}

func (p *PolyTx) GetSmtProofSiblingData() ([]byte, error) {
	if len(p.SmtProofSiblingData) != 0 {
		return hex.DecodeString(p.SmtProofSiblingData)
	}
	return nil, nil
}

func NewPolyTx(txHash []byte, fromChainID uint64, proof []byte, header []byte, headerProof []byte, anchorHeader []byte, sigs []byte, polyTxHash string) (*PolyTx, error) {
	p := &PolyTxProof{
		Proof:        hex.EncodeToString(proof),
		Header:       hex.EncodeToString(header),
		HeaderProof:  hex.EncodeToString(headerProof),
		AnchorHeader: hex.EncodeToString(anchorHeader),
		HeaderSig:    hex.EncodeToString(sigs),
	}
	j, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return &PolyTx{
		TxHash:      hex.EncodeToString(txHash),
		FromChainID: fromChainID,
		SmtTxPath:   Hash256Hex(concatFromChainIDAndTxHash(fromChainID, txHash)),
		PolyTxProof: string(j),
		PolyTxHash:  polyTxHash,
	}, nil
}

func concatFromChainIDAndTxHash(fromChainID uint64, txHash []byte) []byte {
	sink := pcommon.NewZeroCopySink(nil)
	sink.WriteUint64(fromChainID)
	sink.WriteVarBytes(txHash)
	return sink.Bytes()
}

func (p *PolyTx) GetSmtTxKey() ([]byte, error) {
	h, err := hex.DecodeString(p.TxHash)
	if err != nil {
		return nil, err
	}
	return concatFromChainIDAndTxHash(p.FromChainID, h), nil
}

func (p *PolyTx) GetPolyTxProof() (*PolyTxProof, error) {
	proof := &PolyTxProof{}
	err := json.Unmarshal([]byte(p.PolyTxProof), proof)
	if err != nil {
		return nil, err
	}
	return proof, nil
}

func (p *PolyTx) ComputePloyTxInclusionSmtRootHash() ([]byte, error) {
	path, err := hex.DecodeString(p.SmtTxPath) // Txn. path(SMT leaf hash)
	if err != nil {
		return nil, err
	}
	value := PolyTxExistsValue
	sideNodes, err := DecodeSmtProofSideNodes(p.SmtProofSideNodes)
	if err != nil {
		return nil, err
	}
	// smt_proof_non_membership_leaf_data
	oldLeafData, err := hex.DecodeString(p.SmtProofNonMembershipLeafData)
	if err != nil {
		return nil, err
	}
	r, err := rsmt.UpdateRoot(path, value, sideNodes, oldLeafData)
	if err != nil {
		return nil, err
	}
	return r, nil
}

type SmtNode struct {
	Hash string `gorm:"primaryKey;size:66"`
	Data string `gorm:"size:132"`
}
