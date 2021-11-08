package db

type ChainHeight struct {
	Key    string `gorm:"primaryKey"`
	Height uint32
}

type StarcoinTxCheck struct {
	TxHash string `gorm:"primaryKey"` //todo lenght...
	TxData string
}
