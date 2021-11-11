module github.com/elements-studio/poly-starcoin-relayer

go 1.16

require (
	github.com/boltdb/bolt v1.3.1
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/ethereum/go-ethereum v1.10.7
	github.com/google/uuid v1.3.0 // indirect
	github.com/novifinancial/serde-reflection/serde-generate/runtime/golang v0.0.0-20211013011333-6820d5b97d8c
	github.com/ontio/ontology v1.11.1-0.20200812075204-26cf1fa5dd47
	github.com/ontio/ontology-crypto v1.0.9
	github.com/polynetwork/bridge-common v0.0.20 // indirect
	github.com/polynetwork/poly v1.3.1
	github.com/polynetwork/poly-go-sdk v0.0.0-20210114120411-3dcba035134f
	github.com/starcoinorg/starcoin-go v0.0.0-20211022100808-4b435d900b00
	github.com/urfave/cli v1.22.4
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	gorm.io/driver/mysql v1.1.3
	gorm.io/gorm v1.21.12
)

replace github.com/starcoinorg/starcoin-go => ../../starcoinorg/starcoin-go
