package manager

import (
	"bytes"
	"encoding/json"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/log"
	polysdk "github.com/polynetwork/poly-go-sdk"
	scom "github.com/polynetwork/poly/native/service/header_sync/common"
	autils "github.com/polynetwork/poly/native/service/utils"
	"github.com/starcoinorg/starcoin-go/client"
)

type StarcoinManager struct {
	client      client.StarcoinClient
	polySdk     polysdk.PolySdk
	config      config.ServiceConfig
	header4sync [][]byte
}

func (this *StarcoinManager) handleBlockHeader(height uint64) bool {
	block, err := this.client.GetBlockByNumber(int(height))
	if err != nil {
		log.Errorf("handleBlockHeader - GetNodeHeader on height :%d failed", height)
		return false
	}
	hdr := block.BlockHeader
	rawHdr, _ := json.Marshal(hdr)
	var hdrHashBytes []byte //todo hdr.Hash().Bytes()
	raw, _ := this.polySdk.GetStorage(autils.HeaderSyncContractAddress.ToHexString(),
		append(append([]byte(scom.MAIN_CHAIN), autils.GetUint64Bytes(this.config.StarcoinConfig.SideChainId)...), autils.GetUint64Bytes(height)...))
	if len(raw) == 0 || !bytes.Equal(raw, hdrHashBytes) {
		this.header4sync = append(this.header4sync, rawHdr)
	}
	return true
}
