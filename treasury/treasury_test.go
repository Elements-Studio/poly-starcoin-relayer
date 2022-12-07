package treasury

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

var (
	test_xeth_mint_amount  *big.Int
	test_xusdt_mint_amount *big.Int
	test_pstc_mint_amount  *big.Int
)

func init() {
	test_xeth_mint_amount, _ = new(big.Int).SetString("13611294676837538538534984", 10)
	test_xusdt_mint_amount, _ = new(big.Int).SetString("13611294676837538538534984", 10)
	test_pstc_mint_amount, _ = new(big.Int).SetString("5000000000000000000", 10)
}

func TestEthereumTreasuryGetBalanceFor(t *testing.T) {
	var tr Treasury = getTestEthereumTreasury(t)
	//pStcToken := "0x2e269dcdebdc5f2068dfb23972ed81ad1b0f9585"
	//ethToken := "0x0000000000000000000000000000000000000000"
	usdtToken := "0x74E9a2447De2e31C3D8c1f6BAeFBD09ed1162891"
	token := usdtToken
	///openingBalance := test_pstc_mint_amount
	openingBalance := big.NewInt(0) //ETH or USDT opening balance is zero
	b, err := tr.GetBalanceFor(token)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(token)
	fmt.Println(b.String())

	tr.SetOpeningBalanceFor(token, openingBalance)

	lockAmount, err := GetLockAmountFor(tr, token)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("Locked(negative means Unlocked) amount:")
	fmt.Println(lockAmount)
	fmt.Println(ScaleAmountFor(tr, token, lockAmount))
}

func TestMainNetStarcoinTreasuryGetBalanceForSTC(t *testing.T) {
	var tr Treasury = getMainStarcoinTreasury(t)
	stcToken := "0x00000000000000000000000000000001::STC::STC"
	//xethToken := "0x416b32009fe49fcab1d5f2ba0153838f::XETH::XETH"
	//xusdtToken := "0x416b32009fe49fcab1d5f2ba0153838f::XUSDT::XUSDT"
	//openingBalance := big.NewInt(0)
	//openingBalance := test_xeth_mint_amount
	//openingBalance := test_xusdt_mint_amount
	token := stcToken
	//token := xusdtToken
	b, err := tr.GetBalanceFor(token)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(token)
	fmt.Println(b.String())
}

func TestStarcoinTreasuryGetBalanceFor(t *testing.T) {
	var tr Treasury = getTestStarcoinTreasury(t)
	//stcToken := "0x00000000000000000000000000000001::STC::STC"
	//xethToken := "0x416b32009fe49fcab1d5f2ba0153838f::XETH::XETH"
	xusdtToken := "0x416b32009fe49fcab1d5f2ba0153838f::XUSDT::XUSDT"
	//openingBalance := big.NewInt(0)
	//openingBalance := test_xeth_mint_amount
	openingBalance := test_xusdt_mint_amount
	//token := stcToken
	token := xusdtToken
	b, err := tr.GetBalanceFor(token)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(token)
	fmt.Println(b.String())

	tr.SetOpeningBalanceFor(token, openingBalance)

	lockAmount, err := GetLockAmountFor(tr, token)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println("Locked(negative means Unlocked) amount:")
	fmt.Println(lockAmount)

	fmt.Println(ScaleAmountFor(tr, token, lockAmount))
}

func getTestStarcoinTreasury(t *testing.T) *StarcoinTreasury {
	clientUrl := "https://barnard-seed.starcoin.org"
	starcoinClient := stcclient.NewStarcoinClient(clientUrl)
	accountAddress := "0x416b32009fe49fcab1d5f2ba0153838f"
	tr := NewStarcoinStarcoinTreasury(
		accountAddress,
		"0x416b32009fe49fcab1d5f2ba0153838f::LockProxy::LockTreasury",
		&starcoinClient,
	)
	tr.SetScalingFactorMap(StarcoinTokenScalingFactorMap)
	return tr
}

func getMainStarcoinTreasury(t *testing.T) *StarcoinTreasury {
	clientUrl := "https://main-seed.starcoin.org"
	starcoinClient := stcclient.NewStarcoinClient(clientUrl)
	accountAddress := "0xe52552637c5897a2d499fbf08216f73e"
	tr := NewStarcoinStarcoinTreasury(
		accountAddress,
		"0xe52552637c5897a2d499fbf08216f73e::LockProxy::LockTreasury",
		&starcoinClient,
	)
	tr.SetScalingFactorMap(StarcoinTokenScalingFactorMap)
	return tr
}

func getTestEthereumTreasury(t *testing.T) *EthereumTreasury {
	ethClientUrl := "https://ropsten.infura.io/v3/9d5da119377e4547ad0aa8e30482a089"
	ethClient, err := ethclient.Dial(ethClientUrl)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	contractAddr := "0xfd40451429251a6dd535c4bb86a7d894409e900f"
	tr, err := NewEthereumTreasury(ethClient, contractAddr)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	tr.SetScalingFactorMap(EthereumTokenScalingFactorMap)
	return tr
}

func TestEthereumTreasuryGetBalanceForSTC(t *testing.T) {
	var tr Treasury = getMainnetEthereumTreasury(t)
	pStcToken := "0xec8614B0a68786Dc7b452e088a75Cba4F68755b8"
	token := pStcToken
	b, err := tr.GetBalanceFor(token)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(token)
	fmt.Println(b.String())
}

func getMainnetEthereumTreasury(t *testing.T) *EthereumTreasury {
	ethClientUrl := "https://mainnet.infura.io/v3/9d5da119377e4547ad0aa8e30482a089"
	ethClient, err := ethclient.Dial(ethClientUrl)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	contractAddr := "0x3Ee764C95e9d2264DE3717a4CB45BCd3c5F00035"
	tr, err := NewEthereumTreasury(ethClient, contractAddr)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	tr.SetScalingFactorMap(EthereumTokenScalingFactorMap)
	return tr
}
