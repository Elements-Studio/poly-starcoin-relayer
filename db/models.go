package db

import (
	"encoding/hex"
	"encoding/json"

	"github.com/celestiaorg/smt"
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
	CroosTransferDataHash string `gorm:"primaryKey;size:66"` // CrossTransfer data hash, as primary key
	CrossTransferData     string `gorm:"size:5000"`          // CrossTransfer serialized data
	StarcoinEvent         string `gorm:"size:10000"`         // Starcoin Event
}

//
// alter table `poly_tx` convert to charset latin1;
//

// Poly transaction(to Starcoin)
type PolyTx struct {
	TxIndex     uint64 `gorm:"primaryKey;autoIncrement:false"`
	TxHash      string `gorm:"size:66;uniqueIndex"` // Poly Tx. hash //TxData   string `gorm:"size:5000"`
	PolyTxProof string `gorm:"size:36000"`
	//------------------- for Non-Membership proof -------------------
	TxHashHash               string `gorm:"size:66;uniqueIndex"`
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

	UpdatedAt int64 `gorm:"autoUpdateTime:milli"`

	Status string `gorm:"size:20"`
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

func NewPolyTx(polyTxHash string, proof []byte, header []byte, headerProof []byte, anchorHeader []byte, sigs []byte) (*PolyTx, error) {
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
		TxHash:      polyTxHash,
		TxHashHash:  Hash256Hex([]byte(polyTxHash)), //todo is this ok
		PolyTxProof: string(j),
	}, nil
}

func (p *PolyTx) GetPolyTxProof() (*PolyTxProof, error) {
	proof := &PolyTxProof{}
	err := json.Unmarshal([]byte(p.PolyTxProof), proof)
	if err != nil {
		return nil, err
	}
	return proof, nil
}

type PolyTxProof struct {
	Proof        string //`gorm:"size:5000"` // 	bytes memory proof,
	Header       string //`gorm:"size:5000"` // 	bytes memory rawHeader,
	HeaderProof  string //`gorm:"size:5000"` // 	bytes memory headerProof,
	AnchorHeader string //`gorm:"size:5000"` // 	bytes memory curRawHeader,
	HeaderSig    string //`gorm:"size:5000"` // 	bytes memory headerSig
}

type SmtNode struct {
	Hash string `gorm:"primaryKey;size:66"`
	Data string `gorm:"size:132"`
}
