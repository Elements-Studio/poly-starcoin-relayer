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

// Poly transaction(to Starcoin)
type PolyTx struct {
	TxIndex  uint64 `gorm:"primaryKey;autoIncrement:false"` //autoIncrement
	TxHash   string `gorm:"size:66;uniqueIndex"`
	TxData   string `gorm:"size:5000"`
	RootHash string `gorm:"size:66"`
	//Non-inclusion proof
	NonIncProof string `gorm:"size:5000"`
}
