package db

type ChainHeight struct {
	Key    string `gorm:"primaryKey;size:66"`
	Height uint32
}

type StarcoinTxCheck struct {
	TxHash string `gorm:"primaryKey;size:66"`
	TxData string `gorm:"size:5000"`
}

type StarcoinTxRetry struct {
	TxHash string `gorm:"primaryKey;size:66"`
	TxData string `gorm:"size:5000"`
}

//
// alter table `poly_tx` convert to charset latin1;
//

// Poly transaction(to Starcoin)
type PolyTx struct {
	TxIndex      uint64 `gorm:"primaryKey;autoIncrement:false"`
	TxHash       string `gorm:"size:66;uniqueIndex"` // Poly Tx. hash //TxData   string `gorm:"size:5000"`
	Proof        string `gorm:"size:5000"`           // 	bytes memory proof,
	Header       string `gorm:"size:5000"`           // 	bytes memory rawHeader,
	HeaderProof  string `gorm:"size:5000"`           // 	bytes memory headerProof,
	AnchorHeader string `gorm:"size:5000"`           // 	bytes memory curRawHeader,
	HeaderSig    string `gorm:"size:5000"`           // 	bytes memory headerSig

	//------------------- for Non-Membership proof -------------------
	TxHashHash        string `gorm:"size:66;uniqueIndex"`
	SmtRootHash       string `gorm:"size:66"`
	SmtProofSideNodes string `gorm:"size:3000"`

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
}

type SmtNode struct {
	Hash string `gorm:"primaryKey;size:66"`
	Data string `gorm:"size:132"`
}
