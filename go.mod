module github.com/elements-studio/poly-starcoin-relayer

go 1.16

require (
	github.com/antihax/optional v1.0.0
	github.com/boltdb/bolt v1.3.1
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/celestiaorg/smt v0.2.1-0.20210927133715-225e28d5599a
	github.com/ethereum/go-ethereum v1.10.7
	github.com/gateio/gateapi-go/v6 v6.23.1
	github.com/go-sql-driver/mysql v1.6.0
	github.com/google/uuid v1.3.0
	github.com/jinzhu/now v1.1.4 // indirect
	github.com/joho/godotenv v1.4.0
	github.com/novifinancial/serde-reflection/serde-generate/runtime/golang v0.0.0-20211013011333-6820d5b97d8c
	github.com/ontio/ontology v1.11.1-0.20200812075204-26cf1fa5dd47
	github.com/ontio/ontology-crypto v1.0.9
	github.com/polynetwork/bridge-common v0.0.24-beta
	github.com/polynetwork/poly v1.3.1
	github.com/polynetwork/poly-go-sdk v0.0.0-20210114120411-3dcba035134f
	github.com/starcoinorg/starcoin-go v0.0.0-20220324132534-1ecb04f29836
	github.com/urfave/cli v1.22.5
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/text v0.3.7 // indirect
	gorm.io/driver/mysql v1.1.3
	gorm.io/gorm v1.22.4
)

//replace github.com/starcoinorg/starcoin-go => ../../starcoinorg/starcoin-go
//comment out above line to build docker image
