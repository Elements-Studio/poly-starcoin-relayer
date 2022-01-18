package manager

import (
	"context"

	"github.com/elements-studio/poly-starcoin-relayer/log"
	stcpoly "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	"github.com/novifinancial/serde-reflection/serde-generate/runtime/golang/serde"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	"github.com/starcoinorg/starcoin-go/types"
)

// type CrossChainManager struct {
// 	starcoinClient *stcclient.StarcoinClient
// 	module         string
// }

// func NewCrossChainManager(client *stcclient.StarcoinClient, module string) *CrossChainManager {
// 	return &CrossChainManager{
// 		starcoinClient: client,
// 		module:         module,
// 	}
// }

func LockStarcoinAsset(starcoinClient *stcclient.StarcoinClient, privateKeyConfig map[string]string, ccScriptModule string, from_asset_hash []byte, to_chain_id uint64, to_address []byte, amount serde.Uint128) (string, error) {
	senderAddress, senderPrivateKey, err := getAccountAddressAndPrivateKey(privateKeyConfig)
	if err != nil {
		log.Errorf("LockStarcoinAsset - Convert string to AccountAddress error:%s", err.Error())
		return "", err
	}
	seqNum, err := starcoinClient.GetAccountSequenceNumber(context.Background(), tools.EncodeToHex(senderAddress[:]))
	if err != nil {
		log.Errorf("LockStarcoinAsset - GetAccountSequenceNumber error:%s", err.Error())
		return "", err
	}
	gasPrice, err := starcoinClient.GetGasUnitPrice(context.Background())
	if err != nil {
		log.Errorf("LockStarcoinAsset - GetAccountSequenceNumber error:%s", err.Error())
		return "", err
	}
	txPayload := stcpoly.EncodeLockAssetTxPayload(ccScriptModule, from_asset_hash, to_chain_id, to_address, amount)

	userTx, err := starcoinClient.BuildRawUserTransaction(context.Background(), *senderAddress, txPayload, gasPrice, stcclient.DEFAULT_MAX_GAS_AMOUNT*4, seqNum)
	if err != nil {
		log.Errorf("LockStarcoinAsset - BuildRawUserTransaction error:%s", err.Error())
		return "", err
	}
	txHash, err := starcoinClient.SubmitTransaction(context.Background(), senderPrivateKey, userTx)
	if err != nil {
		log.Errorf("LockStarcoinAsset - SubmitTransaction error:%s", err.Error())
		return "", err
	}
	return txHash, nil
}

func submitStarcoinTransaction(starcoinClient *stcclient.StarcoinClient, privateKeyConfig map[string]string, txPayload *types.TransactionPayload) (string, error) {
	senderAddress, senderPrivateKey, err := getAccountAddressAndPrivateKey(privateKeyConfig)
	if err != nil {
		log.Errorf("submitStarcoinTransaction - Convert string to AccountAddress error:%s", err.Error())
		return "", err
	}
	seqNum, err := starcoinClient.GetAccountSequenceNumber(context.Background(), tools.EncodeToHex(senderAddress[:]))
	if err != nil {
		log.Errorf("submitStarcoinTransaction - GetAccountSequenceNumber error:%s", err.Error())
		return "", err
	}
	gasPrice, err := starcoinClient.GetGasUnitPrice(context.Background())
	if err != nil {
		log.Errorf("submitStarcoinTransaction - GetAccountSequenceNumber error:%s", err.Error())
		return "", err
	}
	userTx, err := starcoinClient.BuildRawUserTransaction(context.Background(), *senderAddress, *txPayload, gasPrice, stcclient.DEFAULT_MAX_GAS_AMOUNT*4, seqNum)
	if err != nil {
		log.Errorf("submitStarcoinTransaction - BuildRawUserTransaction error:%s", err.Error())
		return "", err
	}
	txHash, err := starcoinClient.SubmitTransaction(context.Background(), senderPrivateKey, userTx)
	if err != nil {
		log.Errorf("submitStarcoinTransaction - SubmitTransaction error:%s", err.Error())
		return "", err
	}
	return txHash, nil
}

func getAccountAddressAndPrivateKey(senderpk map[string]string) (*types.AccountAddress, types.Ed25519PrivateKey, error) {
	var addressHex string
	for k := range senderpk {
		addressHex = k
		break
	}
	pk, err := tools.HexToBytes(senderpk[addressHex])
	if err != nil {
		log.Errorf("getAccountAddressAndPrivateKey - Convert hex to bytes error:%s", err.Error())
		return nil, nil, err
	}
	address, err := types.ToAccountAddress(addressHex)
	if err != nil {
		return nil, nil, err
	}
	return address, pk, nil
}
