package treasury

import (
	"context"
	"fmt"
	"math/big"

	stcclient "github.com/starcoinorg/starcoin-go/client"
)

type StarcoinTreasury struct {
	BaseTreasury
	accountAddress  string                    //Treasury account address
	treasuryTypeTag string                    //Treasury resource type tag
	starcoinClient  *stcclient.StarcoinClient // Starcoin client
	//openingBalanceMap map[string]*big.Int
}

func NewStarcoinStarcoinTreasury(accountAddress string, treasuryTypeTag string, starcoinClient *stcclient.StarcoinClient) *StarcoinTreasury {
	s := &StarcoinTreasury{
		accountAddress:  accountAddress,
		treasuryTypeTag: treasuryTypeTag,
		starcoinClient:  starcoinClient,
		BaseTreasury: BaseTreasury{
			openingBalanceMap: make(map[string]*big.Int),
		},
	}
	return s
}

func (t *StarcoinTreasury) GetBalanceFor(token string) (*big.Int, error) {
	resType := t.treasuryTypeTag + "<" + token + ">"
	getResOption := stcclient.GetResourceOption{
		Decode: true,
	}
	lockTrRes := new(LockTreasuryResource) //new(map[string]interface{})
	r, err := t.starcoinClient.GetResource(context.Background(), t.accountAddress, resType, getResOption, lockTrRes)
	if err != nil {
		return nil, err
	}
	lockTrRes, ok := r.(*LockTreasuryResource)
	if !ok {
		return nil, fmt.Errorf("GetResource return not LockTreasuryResource")
	}
	return &lockTrRes.Json.Token.Value, nil
}

// Starcoin LockProxy Treasury resource struct.
type LockTreasuryResource struct {
	Raw  string `json:"raw"`
	Json struct {
		Token struct {
			Value big.Int `json:"value"`
		} `json:"token"`
	} `json:"json"`
}
