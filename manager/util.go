package manager

import (
	"github.com/elements-studio/poly-starcoin-relayer/log"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	"github.com/starcoinorg/starcoin-go/types"
)

// type CrossChainManager struct {
// 	starcoinClient *stcclient.StarcoinClient
// 	module         string
// }

// func NewCrossChainManager(client *stcclient.StarcoinClient, module string) *CrossChainManager {
// 	return &CrossChainManager{
// 		starcoinClient: client,
// 		module:         module,
// 	}
// }

func getAccountAddressAndPrivateKey(senderpk map[string]string) (*types.AccountAddress, types.Ed25519PrivateKey, error) {
	var addressHex string
	for k := range senderpk {
		addressHex = k
		break
	}
	pk, err := tools.HexToBytes(senderpk[addressHex])
	if err != nil {
		log.Errorf("getAccountAddressAndPrivateKey - Convert hex to bytes error:%s", err.Error())
		return nil, nil, err
	}
	address, err := stcclient.ToAccountAddress(addressHex)
	if err != nil {
		return nil, nil, err
	}
	return address, pk, nil
}
