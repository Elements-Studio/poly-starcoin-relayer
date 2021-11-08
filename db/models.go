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
