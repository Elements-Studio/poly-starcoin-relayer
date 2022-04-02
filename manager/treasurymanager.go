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
