package manager

import (
	"github.com/elements-studio/poly-starcoin-relayer/config"
	polysdk "github.com/polynetwork/poly-go-sdk"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

type PolyManager struct {
	config         *config.ServiceConfig
	polySdk        *polysdk.PolySdk
	currentHeight  uint32
	exitChan       chan int
	starcoinClient *stcclient.StarcoinClient
	senders        []*StarcoinSender
}

type StarcoinSender struct{}
