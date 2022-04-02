package treasury

import "math/big"

var (
	EthereumTokenScalingFactorMap map[string]*big.Int
	StarcoinTokenScalingFactorMap map[string]*big.Int
)

func init() {
	e := map[string]*big.Int{
		"0x0000000000000000000000000000000000000000": big.NewInt(1_000_000_000_000_000_000), // ETH
		"0x2e269dcdebdc5f2068dfb23972ed81ad1b0f9585": big.NewInt(1_000_000_000),             // pSTC
		"0x74E9a2447De2e31C3D8c1f6BAeFBD09ed1162891": big.NewInt(1_000_000),                 // USDT
	}
	EthereumTokenScalingFactorMap = e

	s := map[string]*big.Int{
		"0x416b32009fe49fcab1d5f2ba0153838f::XETH::XETH":   big.NewInt(1_000_000_000_000_000_000),
		"0x00000000000000000000000000000001::STC::STC":     big.NewInt(1_000_000_000),
		"0x416b32009fe49fcab1d5f2ba0153838f::XUSDT::XUSDT": big.NewInt(1_000_000),
	}
	StarcoinTokenScalingFactorMap = s
}

type Treasury interface {
	GetBalanceFor(token string) (*big.Int, error)
	SetOpeningBalanceFor(token string, balance *big.Int)
	GetOpeningBalanceFor(token string) *big.Int
	GetScalingFactorFor(token string) *big.Int
	SetScalingFactorFor(token string, balance *big.Int)
	GetTokenList() []string
}

type BaseTreasury struct {
	openingBalanceMap map[string]*big.Int
	scalingFactorMap  map[string]*big.Int
}

func NewBaseTreasury() BaseTreasury {
	return BaseTreasury{
		openingBalanceMap: make(map[string]*big.Int),
		scalingFactorMap:  make(map[string]*big.Int),
	}
}

func (t *BaseTreasury) GetTokenList() []string {
	tl := make([]string, 0, len(t.openingBalanceMap))
	for k := range t.openingBalanceMap {
		tl = append(tl, k)
	}
	return tl
}

func (t *BaseTreasury) SetOpeningBalanceMap(m map[string]*big.Int) {
	t.openingBalanceMap = m
}

func (t *BaseTreasury) SetScalingFactorMap(m map[string]*big.Int) {
	t.scalingFactorMap = m
}

func (t *BaseTreasury) SetOpeningBalanceFor(token string, balance *big.Int) {
	tokenAmountMapSet(t.openingBalanceMap, token, balance)
}

func (t *BaseTreasury) GetOpeningBalanceFor(token string) *big.Int {
	return tokenAmountMapGet(t.openingBalanceMap, token)
}

func (t *BaseTreasury) SetScalingFactorFor(token string, f *big.Int) {
	tokenAmountMapSet(t.scalingFactorMap, token, f)
}

func (t *BaseTreasury) GetScalingFactorFor(token string) *big.Int {
	return tokenAmountMapGet(t.scalingFactorMap, token)
}

// Get treasury locked/unlocked amount, negative result means UNLOCKED amount.
func GetLockAmountFor(tr Treasury, token string) (*big.Int, error) {
	opening := tr.GetOpeningBalanceFor(token)
	balance, err := tr.GetBalanceFor(token)
	if err != nil {
		return nil, err
	}
	return new(big.Int).Sub(balance, opening), nil
}

func ScaleAmountFor(tr Treasury, token string, amount *big.Int) *big.Float {
	sf := tr.GetScalingFactorFor(token)
	return ScaleAmount(amount, sf)
}

func ScaleAmount(amount *big.Int, sf *big.Int) *big.Float {
	if sf == nil {
		return new(big.Float).SetInt(amount)
	}
	return new(big.Float).Quo(new(big.Float).SetInt(amount), new(big.Float).SetInt(sf))
}

func tokenAmountMapSet(m map[string]*big.Int, token string, balance *big.Int) {
	m[token] = balance
}

func tokenAmountMapGet(m map[string]*big.Int, token string) *big.Int {
	return m[token]
}
