package manager

import (
	"context"
	"fmt"
	"testing"

	"github.com/antihax/optional"
	"github.com/gateio/gateapi-go/v6"
)

const (
	STC_USDT_TOKEN_PAIR = "STC_USDT"
)

func TestGetStcUsdtPrice(t *testing.T) {
	client := gateapi.NewAPIClient(gateapi.NewConfiguration())
	// uncomment the next line if your are testing against testnet
	// client.ChangeBasePath("https://fx-api-testnet.gateio.ws/api/v4")
	ctx := context.Background()

	tickers, rsp, err := client.SpotApi.ListTickers(ctx, &gateapi.ListTickersOpts{optional.NewString(STC_USDT_TOKEN_PAIR)})
	if err != nil {
		if e, ok := err.(gateapi.GateAPIError); ok {
			fmt.Printf("gate api error: %s\n", e.Error())
			t.FailNow()
		} else {
			fmt.Printf("generic error: %s\n", err.Error())
			t.FailNow()
		}
	} else {
		fmt.Println(rsp)
	}
	//assert tickers.size() == 1;
	if len(tickers) != 1 {
		fmt.Println("!(tickers.size() == 1)")
		t.FailNow()
	}

	fmt.Println("--------------- " + tickers[0].CurrencyPair + " ---------------")
	lastPriceString := tickers[0].Last
	if lastPriceString == "" {
		fmt.Println("lastPriceString is empty")
	}
	fmt.Println(lastPriceString)
}
