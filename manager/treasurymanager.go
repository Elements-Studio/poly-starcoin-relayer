package manager

import (
	"fmt"
	"math/big"
	"strings"

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

const (
	PRINT_FORMAT_TOKEN_STATE_START                           = "------------ Token: %s ------------\n"
	PRINT_FORMAT_TREASURY_ID                                 = "Treasury Id.: %s \n"
	PRINT_FORMAT_TOKEN_ID_IN_TREASURY                        = "Token Id. in treasury: %s \n"
	PRINT_FORMAT_OPENING_BALANCE_IN_TREASURY                 = "Opening balance in treasury: %d \n"
	PRINT_FORMAT_CURRENT_BALANCE_IN_TREASURY                 = "Current balance in treasury: %d \n"
	PRINT_FORMAT_LOCKED_OR_UNLOCKED_AMOUNT_AND_SCALED_AMOUNT = "Locked(positive)/unlocked(negative): %d, scaled amount: %f(%s) \n"
	PRINT_FORMAT_TREASURY_STATE_END                          = "--- above is %s state in %s treasury ---\n"
	PRINT_FORMAT_TOKEN_ON_TRANSIT_AMOUNT                     = "## %s on-transit amount(should be 0 or positive): %f(%s) \n"
)

func (m *TreasuryManager) SprintTokenStates() (string, error) {
	var msg strings.Builder
	for _, tbId := range m.treasuriesConfig.TokenBasicIds {
		msg.WriteString(fmt.Sprintf(PRINT_FORMAT_TOKEN_STATE_START, tbId))
		var tokenLockSum *big.Float = big.NewFloat(0)
		for treasuryId, tr := range m.treasuries {
			msg.WriteString(fmt.Sprintf(PRINT_FORMAT_TREASURY_ID, treasuryId))
			tokenId := m.treasuryTokenMaps[treasuryId][tbId]
			msg.WriteString(fmt.Sprintf(PRINT_FORMAT_TOKEN_ID_IN_TREASURY, tokenId))
			openingBalance := tr.GetOpeningBalanceFor(tokenId)
			msg.WriteString(fmt.Sprintf(PRINT_FORMAT_OPENING_BALANCE_IN_TREASURY, openingBalance))
			balance, err := tr.GetBalanceFor(tokenId)
			if err != nil {
				return "", err
			}
			msg.WriteString(fmt.Sprintf(PRINT_FORMAT_CURRENT_BALANCE_IN_TREASURY, balance))
			lockAmount, err := treasury.GetLockAmountFor(tr, tokenId)
			if err != nil {
				return "", err
			}
			scalingFactor := tr.GetScalingFactorFor(tokenId)
			scaledLockAmount := treasury.ScaleAmount(lockAmount, scalingFactor)
			msg.WriteString(fmt.Sprintf(PRINT_FORMAT_LOCKED_OR_UNLOCKED_AMOUNT_AND_SCALED_AMOUNT, lockAmount, scaledLockAmount, tbId))
			msg.WriteString(fmt.Sprintf(PRINT_FORMAT_TREASURY_STATE_END, tbId, treasuryId))
			tokenLockSum = new(big.Float).Add(tokenLockSum, scaledLockAmount)
		}
		msg.WriteString(fmt.Sprintf(PRINT_FORMAT_TOKEN_ON_TRANSIT_AMOUNT, tbId, tokenLockSum, tbId))
	}

	return msg.String(), nil
}
