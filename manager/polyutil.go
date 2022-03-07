package manager

import (
	"fmt"
	"math/big"

	polysdk "github.com/polynetwork/poly-go-sdk"
	pcommon "github.com/polynetwork/poly/common"
	common2 "github.com/polynetwork/poly/native/service/cross_chain_manager/common"
	scom "github.com/polynetwork/poly/native/service/header_sync/common"
	autils "github.com/polynetwork/poly/native/service/utils"
)

type UnlockArgs struct {
	ToAssetHash []byte
	ToAddress   []byte
	Amount      big.Int
}

func ParseCrossChainUnlockParamsFromProof(p []byte) (*common2.ToMerkleValue, *UnlockArgs, error) {
	ps := pcommon.NewZeroCopySource(p)
	d, _ := ps.NextVarBytes()
	//fmt.Println(d)
	//fmt.Println(hex.EncodeToString(d))

	param := &common2.ToMerkleValue{}
	if err := param.Deserialization(pcommon.NewZeroCopySource(d)); err != nil {
		//log.Errorf("handleDepositEvents - failed to deserialize MakeTxParam (value: %x, err: %v)", value, err)
		//fmt.Print(err)
		return nil, nil, err
	}

	// /////////////////////////////////////
	// TxHash              []byte
	// CrossChainID        []byte
	// FromContractAddress []byte
	// ToChainID           uint64
	// ToContractAddress   []byte
	// Method              string
	// Args                []byte
	// ////////////////////////////////////
	s2 := pcommon.NewZeroCopySource(param.MakeTxParam.Args)
	toAssetHash, b := s2.NextVarBytes()
	toAddress, b := s2.NextVarBytes()

	amount_1, b := s2.NextUint64()
	amount_2, b := s2.NextUint64()
	amount_3, b := s2.NextUint64()
	amount_4, b := s2.NextUint64()
	_ = b
	amount := LittleEndianUint256ToBigInt([4]uint64{amount_1, amount_2, amount_3, amount_4})

	args := UnlockArgs{
		ToAssetHash: toAssetHash,
		ToAddress:   toAddress,
		Amount:      *amount,
	}
	return param, &args, nil
}

func LittleEndianUint256ToBigInt(u [4]uint64) *big.Int {
	b_1 := new(big.Int).SetUint64(u[0])
	b_2 := new(big.Int).SetUint64(u[1])
	b_3 := new(big.Int).SetUint64(u[2])
	b_4 := new(big.Int).SetUint64(u[3])
	i := new(big.Int).SetBytes(append(b_4.Bytes(), b_3.Bytes()...))
	i = new(big.Int).SetBytes(append(i.Bytes(), b_2.Bytes()...))
	i = new(big.Int).SetBytes(append(i.Bytes(), b_1.Bytes()...))
	return i
}

func getStarcoinHeaderInPoly(polySdk *polysdk.PolySdk, sideChainId uint64, height uint64) ([]byte, error) {
	hdrhash, err := getStarcoinHeaderHashInPoly(polySdk, sideChainId, height)
	if err != nil {
		return nil, err
	}
	if len(hdrhash) == 0 {
		return nil, fmt.Errorf("get empty header hash in poly")
	}
	return getStarcoinHeaderInPolyByHash(polySdk, sideChainId, hdrhash)
}

func getStarcoinHeaderHashInPoly(polySdk *polysdk.PolySdk, sideChainId uint64, height uint64) ([]byte, error) {
	hdrhash, err := polySdk.GetStorage(autils.HeaderSyncContractAddress.ToHexString(),
		append(append([]byte(scom.MAIN_CHAIN), autils.GetUint64Bytes(sideChainId)...), autils.GetUint64Bytes(height)...))
	return hdrhash, err
}

func getStarcoinHeaderInPolyByHash(polySdk *polysdk.PolySdk, sideChainId uint64, hdrHash []byte) ([]byte, error) {
	hdrInPoly, err := polySdk.GetStorage(autils.HeaderSyncContractAddress.ToHexString(),
		append(append([]byte(scom.HEADER_INDEX), autils.GetUint64Bytes(sideChainId)...), hdrHash...))
	return hdrInPoly, err
}
