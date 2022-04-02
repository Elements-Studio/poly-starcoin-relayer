package manager

import (
	"fmt"
	"testing"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

func TestPrintTreasuryTokenStates(t *testing.T) {
	m := getTestTreasuryManager(t)
	//
	// err := m.doMonitorTokenStates()
	// if err != nil {
	// 	fmt.Println(err)
	// 	t.FailNow()
	// }
	// return
	//fmt.Println(m)
	s, tokenStates, err := m.SprintTokenStates()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(s)
	fmt.Println(tokenStates)
	// fmt.Println(areThereAbnormalTreasuryTokenStates(tokenStates))
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
