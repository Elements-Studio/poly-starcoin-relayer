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
	TxIndex uint64 `gorm:"primaryKey;autoIncrement:false"`
	TxHash  string `gorm:"size:66;uniqueIndex"`
	//TxData   string `gorm:"size:5000"`
	Proof        string `gorm:"size:5000"` // 	bytes memory proof,
	Header       string `gorm:"size:5000"` // 	bytes memory rawHeader,
	HeaderProof  string `gorm:"size:5000"` // 	bytes memory headerProof,
	AnchorHeader string `gorm:"size:5000"` // 	bytes memory curRawHeader,
	HeaderSig    string `gorm:"size:5000"` // 	bytes memory headerSig
	RootHash     string `gorm:"size:66"`
	//Non-inclusion proof
	NonIncProof string `gorm:"size:3000"`
}
