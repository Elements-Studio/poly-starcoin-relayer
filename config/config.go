/*
* Copyright (C) 2020 The poly network Authors
* This file is part of The poly network library.
*
* The poly network is free software: you can redistribute it and/or modify
* it under the terms of the GNU Lesser General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* The poly network is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
* GNU Lesser General Public License for more details.
* You should have received a copy of the GNU Lesser General Public License
* along with The poly network . If not, see <http://www.gnu.org/licenses/>.
 */
package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/log"
)

const (
	STARCOIN_MONITOR_INTERVAL              = 2 * time.Second
	POLY_MONITOR_INTERVAL                  = 2 * time.Second
	GAS_SUBSIDY_MONITOR_INTERVAL           = 10 * time.Second
	TREASURY_TOKEN_STATES_MONITOR_INTERVAL = 2 * time.Minute

	STARCOIN_USEFUL_BLOCK_NUM    = 9  // eth relayer : 3
	STARCOIN_PROOF_USERFUL_BLOCK = 36 // eth relayer : 12
	ONT_USEFUL_BLOCK_NUM         = 1
	DEFAULT_CONFIG_FILE_NAME     = "./config.json"
	Version                      = "1.0"

	DEFAULT_LOG_LEVEL = log.InfoLog
)

type ServiceConfig struct {
	PolyConfig            *PolyConfig
	StarcoinConfig        *StarcoinConfig
	BridgeURLs            []string
	BoltDbPath            string
	MySqlDSN              string
	RoutineNum            int64
	CheckFee              bool
	ProxyOrAssetContracts []map[string]map[string][]uint64
	TreasuriesConfig      TreasuriesConfig
}

type PolyConfig struct {
	RestURL                 string
	EntranceContractAddress string
	WalletFile              string
	WalletPwd               string
}

type StarcoinConfig struct {
	SideChainId            uint64
	ChainId                int // starcoin chain Id.
	RestURL                string
	CCScriptModule         string
	CCMModule              string //Cross Chain Data Module Id.
	CCDModule              string //Cross Chain Manager Module Id.
	CrossChainEventAddress string //Cross Chain Event Address
	CrossChainEventTypeTag string
	CCSMTRootResourceType  string
	GenesisAccountAddress  string
	// KeyStorePath       string
	// KeyStorePwdSet     map[string]string
	PrivateKeys        []map[string]string
	BlockConfirmations uint64 // safe block confirmations
	HeadersPerBatch    int
	MonitorInterval    uint64
	MaxGasAmount       uint64
	GasSubsidyConfig   GasSubsidyConfig
}

type TreasuriesConfig struct {
	TokenBasicIds            []string                  // token basic Id., 'ETH', 'USDT', 'STC'...
	Treasuries               map[string]TreasuryConfig // treasury Id. to config object mappings
	AlertDiscordWebhookUrl   string
	AlertNotificationHeading string
	AlertAlarmHeading        string
}

type TreasuryConfig struct {
	TreasuryType   string // Starcoin or Ethereum
	StarcoinConfig *TreasuryStarcoinConfig
	EthereumConfig *EthereumTreasuryConfig
	Tokens         map[string]TreasuryTokenConfig // token basic Id. to specific chain token mappings
}

type EthereumTreasuryConfig struct {
	LockProxyContractAddress string
	EthereumClientURL        string
}

type TreasuryStarcoinConfig struct {
	AccountAddress  string // Treasury account address
	TreasuryTypeTag string // Treasury resource type tag
}

type TreasuryTokenConfig struct {
	TokenId        string // specific chain token
	OpeningBalance string
	ScalingFactor  string
}

type GasSubsidyConfig struct {
	FromChainIds                []int                                // active from(source) chain IDs.
	FromChains                  map[string]GasSubsidyFromChainConfig // from(source) chain configs, key is source chain ID
	SenderPrivateKeys           []map[string]string
	ToAddressBlacklist          string
	SubsidizableMinAssetAmounts map[string]uint64 // minimun transffered asset amounts of subsidizable transactions, key is asset hash of target chain
}

type GasSubsidyFromChainConfig struct {
	SubsidyAmount uint64
}

func ReadFile(fileName string) ([]byte, error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("ReadFile: open file %s error %s", fileName, err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Errorf("ReadFile: File %s close error %s", fileName, err)
		}
	}()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("ReadFile: ioutil.ReadAll %s error %s", fileName, err)
	}
	return data, nil
}

func NewServiceConfig(configFilePath string) *ServiceConfig {
	fileContent, err := ReadFile(configFilePath)
	if err != nil {
		log.Errorf("NewServiceConfig: failed, err: %s", err.Error())
		return nil
	}
	// /////////////////////////////////////////
	fileContent = replaceJsonEnvs(fileContent)
	// /////////////////////////////////////////
	servConfig := &ServiceConfig{}
	err = json.Unmarshal(fileContent, servConfig)
	if err != nil {
		log.Errorf("NewServiceConfig: failed, err: %s", err.Error())
		return nil
	}

	// for k, v := range servConfig.StarcoinConfig.KeyStorePwdSet {
	// 	delete(servConfig.StarcoinConfig.KeyStorePwdSet, k)
	// 	servConfig.StarcoinConfig.KeyStorePwdSet[strings.ToLower(k)] = v
	// }

	return servConfig
}
