package treasury

import (
	"math/big"

	lockproxy_abi "github.com/KSlashh/poly-abi/abi_1.10.7/lockproxy"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EthereumTreasury struct {
	BaseTreasury
	lockProxyContractAddress string
	ethClient                *ethclient.Client
	lockProxyInstance        *lockproxy_abi.LockProxy
}

func NewEthereumTreasury(ethClient *ethclient.Client, lockProxyContractAddress string) (*EthereumTreasury, error) {
	lockProxyAddress := ethcommon.HexToAddress(lockProxyContractAddress)
	lockProxyInstance, err := lockproxy_abi.NewLockProxy(lockProxyAddress, ethClient)
	if err != nil {
		return nil, err
	}
	t := &EthereumTreasury{
		lockProxyContractAddress: lockProxyContractAddress,
		ethClient:                ethClient,
		lockProxyInstance:        lockProxyInstance,
		BaseTreasury: BaseTreasury{
			openingBalanceMap: make(map[string]*big.Int),
		},
	}

	return t, nil
}

func (t *EthereumTreasury) GetBalanceFor(token string) (*big.Int, error) {
	tokenAddress := ethcommon.HexToAddress(token)
	return t.lockProxyInstance.GetBalanceFor(nil, tokenAddress)
}
