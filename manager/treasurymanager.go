package manager

import (
	"fmt"
	"math/big"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/treasury"
	"github.com/ethereum/go-ethereum/ethclient"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

const (
	TREASURY_TYPE_STARCOIN = "STARCOIN"
	TREASURY_TYPE_ETHEREUM = "ETHEREUM"
)

type TreasuryManager struct {
	treasuriesConfig  config.TreasuriesConfig
	starcoinClient    *stcclient.StarcoinClient      // Starcoin client
	treasuries        map[string]treasury.Treasury   // Treasury Id. to treasury object mappings
	treasuryTokenMaps map[string](map[string]string) // Treasury Id. to 'Token basic Id. to specific chain Token Id mappings' mappings
}

func NewTreasuryManager(config config.TreasuriesConfig, starcoinClient *stcclient.StarcoinClient) (*TreasuryManager, error) {
	var err error
	treasuryMap := make(map[string]treasury.Treasury)
	treasuryTokenMaps := make(map[string](map[string]string))
	for treasuryId, v := range config.Treasuries {
		var t treasury.Treasury
		if v.TreasuryType == TREASURY_TYPE_STARCOIN {
			t = treasury.NewStarcoinStarcoinTreasury(v.StarcoinConfig.AccountAddress, v.StarcoinConfig.TreasuryTypeTag,
				starcoinClient,
			)
		} else if v.TreasuryType == TREASURY_TYPE_ETHEREUM {
			ethClient, errDial := ethclient.Dial(v.EthereumConfig.EthereumClientURL)
			if errDial != nil {
				return nil, errDial
			}
			t, err = treasury.NewEthereumTreasury(ethClient, v.EthereumConfig.LockProxyContractAddress)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unknown TreasuryType: %s", v.TreasuryType)
		}
		var tokenMap map[string]string = make(map[string]string)
		for tokenBasicId, tokenConfig := range v.Tokens {
			openingBalance, ok := new(big.Int).SetString(tokenConfig.OpeningBalance, 10)
			if !ok {
				return nil, fmt.Errorf("parse tokenConfig.OpeningBalance error: %s", tokenConfig.OpeningBalance)
			}
			scalingFactor, ok := new(big.Int).SetString(tokenConfig.ScalingFactor, 10)
			if !ok {
				return nil, fmt.Errorf("parse tokenConfig.ScalingFactor error: %s", tokenConfig.ScalingFactor)
			}
			tokenMap[tokenBasicId] = tokenConfig.TokenId
			t.SetOpeningBalanceFor(tokenConfig.TokenId, openingBalance)
			t.SetScalingFactorFor(tokenConfig.TokenId, scalingFactor)
		}
		treasuryTokenMaps[treasuryId] = tokenMap
		treasuryMap[treasuryId] = t
	}
	m := &TreasuryManager{
		treasuriesConfig:  config,
		treasuries:        treasuryMap,
		starcoinClient:    starcoinClient,
		treasuryTokenMaps: treasuryTokenMaps,
	}
	return m, nil
}

func (m *TreasuryManager) PrintTokenStates() error {
	for _, tbId := range m.treasuriesConfig.TokenBasicIds {
		fmt.Printf("------------ %s ------------\n", tbId)
		var tokenLockSum *big.Float = big.NewFloat(0)
		for treasuryId, tr := range m.treasuries {
			fmt.Printf("Treasury Id.: %s \n", treasuryId)
			tokenId := m.treasuryTokenMaps[treasuryId][tbId]
			fmt.Printf("Token Id. in treasury: %s \n", tokenId)
			openingBalance := tr.GetOpeningBalanceFor(tokenId)
			fmt.Printf("Opening balance in treasury: %d \n", openingBalance)
			balance, err := tr.GetBalanceFor(tokenId)
			if err != nil {
				return err
			}
			fmt.Printf("Current balance in treasury: %d \n", balance)
			lockAmount, err := treasury.GetLockAmountFor(tr, tokenId)
			if err != nil {
				return err
			}
			scalingFactor := tr.GetScalingFactorFor(tokenId)
			scaledLockAmount := treasury.ScaleAmount(lockAmount, scalingFactor)
			fmt.Printf("Locked(positive)/unlocked(negative) amount: %d, scaled amount: %f \n", lockAmount, scaledLockAmount)
			fmt.Println("---") //fmt.Printf("--- %s. \n", treasuryId)
			tokenLockSum = new(big.Float).Add(tokenLockSum, scaledLockAmount)
		}
		fmt.Printf("## On-transit amount: %f(%s) \n", tokenLockSum, tbId)
	}
	return nil
}
