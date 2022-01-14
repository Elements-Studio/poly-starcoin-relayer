package manager

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/polynetwork/bridge-common/chains/bridge"
)

func TestCheckFee(t *testing.T) {
	polyManager := getTestNetPolyManager(t)
	bridgeSdk := polyManager.bridgeSdk
	//bridgeSdk := getMainNetBridgeSdkForTest(t)
	chainId := uint64(400)
	txId := "21c3e476cd9389b65bd462a61d4cca430976cf528ec403019fc73512926e29e0"
	polyHash := "dc01826756160442da9470cd137c789a28f6a98050cef7a612462342d5e9e140"

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
