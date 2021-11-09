module github.com/elements-studio/poly-starcoin-relayer

go 1.16

require (
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/novifinancial/serde-reflection/serde-generate/runtime/golang v0.0.0-20211013011333-6820d5b97d8c
	github.com/polynetwork/poly-go-sdk v0.0.0-20210114120411-3dcba035134f // indirect
	github.com/starcoinorg/starcoin-go v0.0.0-20211022100808-4b435d900b00 // indirect
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	gorm.io/driver/mysql v1.1.3 // indirect
	gorm.io/gorm v1.21.12 // indirect
)

replace github.com/starcoinorg/starcoin-go => ../../starcoinorg/starcoin-go
