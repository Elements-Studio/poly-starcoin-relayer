package manager

import (
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

type CrossChainManager struct {
	starcoinClient *stcclient.StarcoinClient
	module         string
}

func NewCrossChainManager(client *stcclient.StarcoinClient, module string) *CrossChainManager {
	return &CrossChainManager{
		starcoinClient: client,
		module:         module,
	}
}
