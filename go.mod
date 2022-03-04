module github.com/elements-studio/poly-starcoin-relayer

go 1.16

require (
	github.com/antihax/optional v1.0.0 // indirect
	github.com/blocktree/go-owcrypt v1.1.10 // indirect
	github.com/boltdb/bolt v1.3.1
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/celestiaorg/smt v0.2.1-0.20210927133715-225e28d5599a // indirect
	github.com/ethereum/go-ethereum v1.10.7
	github.com/gateio/gateapi-go/v6 v6.23.1 // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/jinzhu/now v1.1.4 // indirect
	github.com/joho/godotenv v1.4.0 // indirect
	github.com/klauspost/compress v1.4.1 // indirect
	github.com/klauspost/cpuid v1.2.0 // indirect
	github.com/novifinancial/serde-reflection/serde-generate/runtime/golang v0.0.0-20211013011333-6820d5b97d8c
	github.com/ontio/ontology v1.11.1-0.20200812075204-26cf1fa5dd47
	github.com/ontio/ontology-crypto v1.0.9
	github.com/polynetwork/bridge-common v0.0.24-beta // indirect
	github.com/polynetwork/poly v1.3.1
	github.com/polynetwork/poly-go-sdk v0.0.0-20210114120411-3dcba035134f
	github.com/starcoinorg/starcoin-go v0.0.0-20220105024102-530daedc128b
	github.com/urfave/cli v1.22.4
	github.com/valyala/fasthttp v1.4.0 // indirect
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	gorm.io/driver/mysql v1.1.3
	gorm.io/gorm v1.22.4
)

//replace github.com/starcoinorg/starcoin-go => ../../starcoinorg/starcoin-go
//comment out above line to build docker image

replace github.com/lazyledger/smt => github.com/celestiaorg/smt v0.2.1
