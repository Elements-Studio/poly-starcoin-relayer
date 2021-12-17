package manager

import (
	"fmt"

	polysdk "github.com/polynetwork/poly-go-sdk"
	scom "github.com/polynetwork/poly/native/service/header_sync/common"
	autils "github.com/polynetwork/poly/native/service/utils"
)

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
