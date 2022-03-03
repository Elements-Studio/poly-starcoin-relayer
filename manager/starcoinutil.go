package manager

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/elements-studio/poly-starcoin-relayer/log"
	stcpoly "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly"
	stcpolyevts "github.com/elements-studio/poly-starcoin-relayer/starcoin/poly/events"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	"github.com/novifinancial/serde-reflection/serde-generate/runtime/golang/serde"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	"github.com/starcoinorg/starcoin-go/types"
)

func GetTokenCodeString(tc *stcpolyevts.TokenCode) string {
	return "0x" + hex.EncodeToString(tc.Address[:]) + "::" + tc.Module + "::" + tc.Name
}

func Uint128ToBigInt(u *serde.Uint128) *big.Int {
	h := new(big.Int).SetUint64(u.High)
	l := new(big.Int).SetUint64(u.Low)
	return new(big.Int).SetBytes(append(h.Bytes(), l.Bytes()...))
}

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
	// bs, _ := txPayload.BcsSerialize()
	// fmt.Println(hex.EncodeToString(bs))
	// fmt.Println("------------------------")
	// return "", errors.New("Testing... todo: remove some lines...")
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

func LockStarcoinAssetWithStcFee(starcoinClient *stcclient.StarcoinClient, privateKeyConfig map[string]string, ccScriptModule string, from_asset_hash []byte, to_chain_id uint64, to_address []byte, amount serde.Uint128, fee serde.Uint128, id serde.Uint128) (string, error) {
	senderAddress, senderPrivateKey, err := getAccountAddressAndPrivateKey(privateKeyConfig)
	if err != nil {
		log.Errorf("LockStarcoinAssetWithFee - Convert string to AccountAddress error:%s", err.Error())
		return "", err
	}
	seqNum, err := starcoinClient.GetAccountSequenceNumber(context.Background(), tools.EncodeToHex(senderAddress[:]))
	if err != nil {
		log.Errorf("LockStarcoinAssetWithFee - GetAccountSequenceNumber error:%s", err.Error())
		return "", err
	}
	gasPrice, err := starcoinClient.GetGasUnitPrice(context.Background())
	if err != nil {
		log.Errorf("LockStarcoinAssetWithFee - GetAccountSequenceNumber error:%s", err.Error())
		return "", err
	}
	txPayload := stcpoly.EncodeLockAssetWithStcFeeTxPayload(ccScriptModule, from_asset_hash, to_chain_id, to_address, amount, fee, id)
	//bs, _ := txPayload.BcsSerialize()
	// fmt.Println("------------ Txn payload ------------")
	// fmt.Println(hex.EncodeToString(bs))
	// fmt.Println("------------------------")
	// return "", errors.New("Testing... todo: remove some lines...")
	userTx, err := starcoinClient.BuildRawUserTransaction(context.Background(), *senderAddress, txPayload, gasPrice, stcclient.DEFAULT_MAX_GAS_AMOUNT*4, seqNum)
	if err != nil {
		log.Errorf("LockStarcoinAssetWithFee - BuildRawUserTransaction error:%s", err.Error())
		return "", err
	}
	//publicKey, _ := owcrypt.GenPubkey(senderPrivateKey, owcrypt.ECC_CURVE_ED25519_NORMAL)
	//dry_run_result, err := starcoinClient.DryRunRaw(context.Background(), *userTx, publicKey)
	////fmt.Println(dry_run_result)
	//dry_run_resutl_bs, _ := json.Marshal(dry_run_result)
	// fmt.Println("------------ starcoinClient.DryRunRaw result ----------------")
	// fmt.Println(string(dry_run_resutl_bs))
	// fmt.Println("------------------------")
	// return "", errors.New("Testing... todo: remove some lines...")
	txHash, err := starcoinClient.SubmitTransaction(context.Background(), senderPrivateKey, userTx)
	if err != nil {
		log.Errorf("LockStarcoinAssetWithFee - SubmitTransaction error:%s", err.Error())
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
