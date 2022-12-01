package manager

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/log"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	"github.com/elements-studio/poly-starcoin-relayer/treasury"
	"github.com/ethereum/go-ethereum/ethclient"
	stcclient "github.com/starcoinorg/starcoin-go/client"
)

const (
	TREASURY_TYPE_STARCOIN                       = "STARCOIN"
	TREASURY_TYPE_ETHEREUM                       = "ETHEREUM"
	TREASURY_TOKEN_STATES_NOTIFY_INTERVAL_MILLIS = 1000 * 60 * 60 // one hour
)

type TreasuryManager struct {
	treasuriesConfig     config.TreasuriesConfig
	starcoinClient       *stcclient.StarcoinClient      // Starcoin client
	treasuries           map[string]treasury.Treasury   // Treasury Id. to treasury object mappings
	treasuryTokenMaps    map[string](map[string]string) // Treasury Id. to 'Token basic Id. to specific chain Token Id mappings' mappings
	discordWebhookClient *tools.RestClient
	lastNotifiedAt       int64
}

func NewTreasuryManager(config config.TreasuriesConfig, starcoinClient *stcclient.StarcoinClient) (*TreasuryManager, error) {
	var err error
	treasuryMap := make(map[string]treasury.Treasury)
	treasuryTokenMaps := make(map[string](map[string]string))
	for treasuryId, v := range config.Treasuries {
		var t treasury.Treasury
		if v.TreasuryType == TREASURY_TYPE_STARCOIN {
			t = treasury.NewStarcoinStarcoinTreasury(v.StarcoinConfig.AccountAddress, v.StarcoinConfig.TreasuryTypeTag,
				starcoinClient,
			)
		} else if v.TreasuryType == TREASURY_TYPE_ETHEREUM {
			ethClient, errDial := ethclient.Dial(v.EthereumConfig.EthereumClientURL)
			if errDial != nil {
				return nil, errDial
			}
			t, err = treasury.NewEthereumTreasury(ethClient, v.EthereumConfig.LockProxyContractAddress)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unknown TreasuryType: %s", v.TreasuryType)
		}
		var tokenMap map[string]string = make(map[string]string)
		for tokenBasicId, tokenConfig := range v.Tokens {
			openingBalance, ok := new(big.Int).SetString(tokenConfig.OpeningBalance, 10)
			if !ok {
				return nil, fmt.Errorf("parse tokenConfig.OpeningBalance error: %s", tokenConfig.OpeningBalance)
			}
			scalingFactor, ok := new(big.Int).SetString(tokenConfig.ScalingFactor, 10)
			if !ok {
				return nil, fmt.Errorf("parse tokenConfig.ScalingFactor error: %s", tokenConfig.ScalingFactor)
			}
			tokenMap[tokenBasicId] = tokenConfig.TokenId
			t.SetOpeningBalanceFor(tokenConfig.TokenId, openingBalance)
			t.SetScalingFactorFor(tokenConfig.TokenId, scalingFactor)
		}
		treasuryTokenMaps[treasuryId] = tokenMap
		treasuryMap[treasuryId] = t
	}
	restClient := tools.NewRestClient()
	// restClient.SetProxy("http://localhost:9999") // just for test!
	m := &TreasuryManager{
		treasuriesConfig:     config,
		treasuries:           treasuryMap,
		starcoinClient:       starcoinClient,
		treasuryTokenMaps:    treasuryTokenMaps,
		discordWebhookClient: restClient,
		lastNotifiedAt:       tools.CurrentTimeMillis(),
	}
	return m, nil
}

func (m *TreasuryManager) MonitorTokenStates() {
	monitorTicker := time.NewTicker(config.TREASURY_TOKEN_STATES_MONITOR_INTERVAL)
	for {
		select {
		case <-monitorTicker.C:
			err := m.doMonitorTokenStates()
			if err != nil {
				log.Errorf("TreasuryManager.MonitorTokenStates - doMonitorTokenStates error: %s", err.Error())
			}
		}
	}
}

func (m *TreasuryManager) doMonitorTokenStates() error {
	s, tokenStates, err := m.SprintTokenStates()
	if err != nil {
		return err
	}
	c, isAbnormal := m.buildWebhookMessageContent(s, tokenStates)
	var sendRightNow bool = isAbnormal
	if !isAbnormal {
		now := tools.CurrentTimeMillis()
		if now > m.lastNotifiedAt+TREASURY_TOKEN_STATES_NOTIFY_INTERVAL_MILLIS {
			sendRightNow = true
			m.lastNotifiedAt = now
		}
	}
	if sendRightNow {
		// fmt.Println(c)
		// return nil // print then return
		return m.sendDiscordWebhookMessage(c)
	}
	return nil
}

const (
	PRINT_FORMAT_ALARM_HEADING                               = "# Poly Bridge ALARM !!!"
	PRINT_FORMAT_NOTIFICATION_HEADING                        = "# Poly Bridge Notification"
	PRINT_FORMAT_TOKEN_STATE_START                           = "------------ Token: %s ------------\n"
	PRINT_FORMAT_TREASURY_ID                                 = "Treasury Id.: %s \n"
	PRINT_FORMAT_TOKEN_ID_IN_TREASURY                        = "Token Id.: %s \n"       // in treasury
	PRINT_FORMAT_OPENING_BALANCE_IN_TREASURY                 = "Opening balance: %d \n" // in treasury
	PRINT_FORMAT_CURRENT_BALANCE_IN_TREASURY                 = "Current balance: %d \n" // in treasury
	PRINT_FORMAT_LOCKED_OR_UNLOCKED_AMOUNT_AND_SCALED_AMOUNT = "Locked(positive)/unlocked(negative) amount: %d, scaled amount: %f(%s) \n"
	PRINT_FORMAT_TREASURY_STATE_END                          = "\n" //"--- above is %s state in %s treasury ---\n"
	PRINT_FORMAT_TOKEN_ON_TRANSIT_AMOUNT                     = "## %s on-transit amount(should be 0 or positive): %f(%s) \n"

	WEBHOOK_CONTENT_MAX_LENGTH = 2000 //Must be 2000 or fewer in length.
)

// If there are abnormal states, return message content and true.
func (m *TreasuryManager) buildWebhookMessageContent(s string, tokenStates map[string]*big.Float) (string, bool) {
	var msg strings.Builder
	isAbnormal, abnormalTokenStates := areThereAbnormalTreasuryTokenStates(tokenStates)
	if isAbnormal {
		msg.WriteString(fmt.Sprintf(m.get_print_format_alarm_heading()))
		msg.WriteString("\n")
	} else {
		msg.WriteString(fmt.Sprintf(m.get_print_format_notification_heading()))
		msg.WriteString("\n")
	}
	msg.WriteString(s)
	msg.WriteString("\n")
	var conclusion string
	if isAbnormal {
		conclusion = formatAbnormalTreasuryTokenStates(abnormalTokenStates)
	} else {
		conclusion = "Seem like ok."
	}
	detail := msg.String()
	if len(detail)+len(conclusion) > WEBHOOK_CONTENT_MAX_LENGTH {
		// truncate message
		msg.Reset()
		msg.WriteString(detail[0 : WEBHOOK_CONTENT_MAX_LENGTH-len("...\n")-len(conclusion)])
		msg.WriteString("...\n")
		msg.WriteString(conclusion)
	} else {
		msg.WriteString(conclusion)
	}
	return msg.String(), isAbnormal
}

func (m *TreasuryManager) sendDiscordWebhookMessage(content string) error {
	j, err := json.Marshal(NewDiscordWebhookMessage(content))
	if err != nil {
		return err
	}
	r, err := m.discordWebhookClient.SendPostRequest(m.treasuriesConfig.AlertDiscordWebhookUrl, j)
	if err != nil {
		return err
	}
	log.Infof("TreasuryManager.sendDiscordWebhookMessage - return: %s", string(r))
	return nil
}

func (m *TreasuryManager) get_print_format_alarm_heading() string {
	if m.treasuriesConfig.AlertAlarmHeading == "" {
		return PRINT_FORMAT_ALARM_HEADING
	}
	return m.treasuriesConfig.AlertAlarmHeading
}

func (m *TreasuryManager) get_print_format_notification_heading() string {
	if m.treasuriesConfig.AlertNotificationHeading == "" {
		return PRINT_FORMAT_NOTIFICATION_HEADING
	}
	return m.treasuriesConfig.AlertNotificationHeading
}

func (m *TreasuryManager) SprintTokenStates() (string, map[string]*big.Float, error) {
	var msg strings.Builder
	var tokenStates = make(map[string]*big.Float)
	for _, tbId := range m.treasuriesConfig.TokenBasicIds {
		msg.WriteString(fmt.Sprintf(PRINT_FORMAT_TOKEN_STATE_START, tbId))
		var tokenLockSum *big.Float = big.NewFloat(0)
		for treasuryId, tr := range m.treasuries {
			msg.WriteString(fmt.Sprintf(PRINT_FORMAT_TREASURY_ID, treasuryId))
			tokenId := m.treasuryTokenMaps[treasuryId][tbId]
			msg.WriteString(fmt.Sprintf(PRINT_FORMAT_TOKEN_ID_IN_TREASURY, tokenId))
			openingBalance := tr.GetOpeningBalanceFor(tokenId)
			msg.WriteString(fmt.Sprintf(PRINT_FORMAT_OPENING_BALANCE_IN_TREASURY, openingBalance))
			balance, err := tr.GetBalanceFor(tokenId)
			if err != nil {
				return "", nil, err
			}
			msg.WriteString(fmt.Sprintf(PRINT_FORMAT_CURRENT_BALANCE_IN_TREASURY, balance))
			lockAmount, err := treasury.GetLockAmountFor(tr, tokenId)
			if err != nil {
				return "", nil, err
			}
			scalingFactor := tr.GetScalingFactorFor(tokenId)
			scaledLockAmount := treasury.ScaleAmount(lockAmount, scalingFactor)
			msg.WriteString(fmt.Sprintf(PRINT_FORMAT_LOCKED_OR_UNLOCKED_AMOUNT_AND_SCALED_AMOUNT, lockAmount, scaledLockAmount, tbId))
			msg.WriteString(fmt.Sprintf(PRINT_FORMAT_TREASURY_STATE_END)) //, tbId, treasuryId))
			tokenLockSum = new(big.Float).Add(tokenLockSum, scaledLockAmount)
			tokenStates[tbId] = tokenLockSum
		}
		msg.WriteString(fmt.Sprintf(PRINT_FORMAT_TOKEN_ON_TRANSIT_AMOUNT, tbId, tokenLockSum, tbId))
	}

	return msg.String(), tokenStates, nil
}

func areThereAbnormalTreasuryTokenStates(tokenStates map[string]*big.Float) (bool, map[string]*big.Float) {
	var abnormalStates = make(map[string]*big.Float)
	var isAbnormal = false
	for id, sum := range tokenStates {
		if sum.Cmp(big.NewFloat(0)) < 0 {
			abnormalStates[id] = sum
			log.Errorf("areThereAbnormalTreasuryTokenStates, abnormal token found, Id: %s, locked/unlocked amount: %f", id, sum)
			isAbnormal = true
		}
	}
	return isAbnormal, abnormalStates
}

func formatAbnormalTreasuryTokenStates(abnormalTokenStates map[string]*big.Float) string {
	if len(abnormalTokenStates) == 0 {
		return ""
	}
	var msg strings.Builder
	for tbId, tokenLockSum := range abnormalTokenStates {
		msg.WriteString("!!! ")
		msg.WriteString(fmt.Sprintf(PRINT_FORMAT_TOKEN_ON_TRANSIT_AMOUNT, tbId, tokenLockSum, tbId))
	}
	return msg.String()
}

type DiscordWebhookMessage struct {
	Content string `json:"content"`
}

func NewDiscordWebhookMessage(content string) *DiscordWebhookMessage {
	return &DiscordWebhookMessage{
		Content: content,
	}
}
