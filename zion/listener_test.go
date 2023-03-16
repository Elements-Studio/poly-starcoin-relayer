package zion

import (
	"github.com/elements-studio/poly-starcoin-relayer/zion/config"
	"github.com/polynetwork/bridge-common/base"
	"github.com/polynetwork/bridge-common/chains/zion"
	"testing"
	"time"
)

func TestListener_Init(t *testing.T) {
	listener := new(Listener)
	listener_config := new(config.ListenerConfig)

	poly, _ := zion.WithOptions(base.POLY, listener_config.Nodes, time.Minute, 1)
	listener.Init(listener_config, poly)
}

func TestListener_ScanDst(t *testing.T) {

}
