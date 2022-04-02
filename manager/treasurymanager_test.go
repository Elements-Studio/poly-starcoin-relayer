package manager

import (
	"fmt"
	"testing"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestNewTreasuryManager(t *testing.T) {
	m := getTestTreasuryManager(t)
	fmt.Println(m)
}

func getTestTreasuryManager(t *testing.T) *TreasuryManager {
	config := config.NewServiceConfig("../config-testnet.json")
	starcoinUrl := config.StarcoinConfig.RestURL
	starconClient := stcclient.NewStarcoinClient(starcoinUrl)
	m, err := NewTreasuryManager(config.TreasuriesConfig, &starconClient)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	// fmt.Println(m)
	// fmt.Println(m.treasuryTokenMaps)
	// fmt.Println(m.treasuries)
	// for k, t := range m.treasuries {
	// 	fmt.Println(k, ":")
	// 	fmt.Println(t.GetTokenList())
	// }
	return m
}
