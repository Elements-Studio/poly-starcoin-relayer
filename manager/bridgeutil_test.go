package manager

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/polynetwork/bridge-common/chains/bridge"
)

func TestCheckFee(t *testing.T) {
	// polyManager := getTestNetPolyManager(t)
	// bridgeSdk := polyManager.bridgeSdk
	bridgeSdk := getMainNetBridgeSdkForTest(t)
	chainId := uint64(6) // MainNet BSC
	txId := "2244fe33767fa38717a0dba2ca413d411db0e613abd1d4ffd181c6f12eea2ab1"
	polyHash := "38ecd4268a68961acd622c7efe635edb50976582febe8a31bc3b55436def0256"

	r, err := CheckFee(bridgeSdk, chainId, txId, polyHash)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	j, _ := json.Marshal(r)
	fmt.Println(string(j))
}

func getMainNetBridgeSdkForTest(t *testing.T) *bridge.SDK {
	bridgeSdk, err := bridge.WithOptions(0, []string{
		"https://bridge.poly.network/mainnet/v1",
	}, time.Minute, 10)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	return bridgeSdk
}
