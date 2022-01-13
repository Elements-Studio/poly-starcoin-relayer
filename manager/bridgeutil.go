package manager

import (
	"github.com/polynetwork/bridge-common/chains/bridge"
)

func CheckFee(sdk *bridge.SDK, chainId uint64, txId string, polyHash string) (res *bridge.CheckFeeRequest, err error) {
	state := map[string]*bridge.CheckFeeRequest{}
	state[polyHash] = &bridge.CheckFeeRequest{
		ChainId:  chainId,
		TxId:     txId,
		PolyHash: polyHash,
	}
	if state[polyHash] == nil {
		state[polyHash] = new(bridge.CheckFeeRequest)
	}
	return state[polyHash], nil
}
