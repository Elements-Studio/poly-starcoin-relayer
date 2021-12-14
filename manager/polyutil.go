package manager

import (
	"fmt"

	polysdk "github.com/polynetwork/poly-go-sdk"
	scom "github.com/polynetwork/poly/native/service/header_sync/common"
	autils "github.com/polynetwork/poly/native/service/utils"
)

func getStarcoinHeaderInPoly(polySdk *polysdk.PolySdk, sideChainId uint64, height uint64) ([]byte, error) {
	hdrhash, err := polySdk.GetStorage(autils.HeaderSyncContractAddress.ToHexString(),
		append(append([]byte(scom.MAIN_CHAIN), autils.GetUint64Bytes(sideChainId)...), autils.GetUint64Bytes(height)...))
	if err != nil {
		return nil, err
	}
	if len(hdrhash) == 0 {
		return nil, fmt.Errorf("get empty header hash in poly")
	}

	hdrInPoly, err := polySdk.GetStorage(autils.HeaderSyncContractAddress.ToHexString(),
		append(append([]byte(scom.HEADER_INDEX), autils.GetUint64Bytes(sideChainId)...), hdrhash...))
	return hdrInPoly, err
}
