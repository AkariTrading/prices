package binance

import (
	"strings"

	"github.com/pkg/errors"
)

// ExchangeInfo exchange info
type ExchangeInfo struct {
	Timezone        string        `json:"timezone"`
	ServerTime      int64         `json:"serverTime"`
	RateLimits      []RateLimit   `json:"rateLimits"`
	ExchangeFilters []interface{} `json:"exchangeFilters"`
	Symbols         []Symbol      `json:"symbols"`
}

// RateLimit struct
type RateLimit struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	Limit         int64  `json:"limit"`
}

// Symbol market symbol
type Symbol struct {
	Symbol                 string                   `json:"symbol"`
	Status                 string                   `json:"status"`
	BaseAsset              string                   `json:"baseAsset"`
	BaseAssetPrecision     int                      `json:"baseAssetPrecision"`
	QuoteAsset             string                   `json:"quoteAsset"`
	QuotePrecision         int                      `json:"quotePrecision"`
	OrderTypes             []string                 `json:"orderTypes"`
	IcebergAllowed         bool                     `json:"icebergAllowed"`
	OcoAllowed             bool                     `json:"ocoAllowed"`
	IsSpotTradingAllowed   bool                     `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed bool                     `json:"isMarginTradingAllowed"`
	Filters                []map[string]interface{} `json:"filters"`
}

// CheckSymbol -
func CheckSymbol(s string) bool {
	_, ok := symbolsMap[s]
	return ok
}

func initSymbols(symbolSuffixes ...string) error {
	info, err := exchangeinfo()
	if err != nil {
		return err
	}
	// for _, val := range info.Symbols {
	// 	fmt.Println(val.Status, val.BaseAsset, val.Symbol, val.IsSpotTradingAllowed, val.OrderTypes)
	// }

	symbolsMap = info.symbolNamesMap(symbolSuffixes...)

	if len(symbolsMap) == 0 {
		return errors.New("could not fetch symbols")
	}

	return nil
}

func exchangeinfo() (*ExchangeInfo, error) {

	var data ExchangeInfo
	err := httpGETJson("https://api.binance.com/api/v3/exchangeInfo", &data)

	if err != nil {
		return nil, err
	}

	return &data, nil
}

// SymbolNames -
func (ex *ExchangeInfo) symbolNamesMap(suffixes ...string) map[string]bool {
	tmp := make(map[string]bool)
	for _, s := range ex.Symbols {
		for _, suffix := range suffixes {
			if s.Status == "TRADING" && s.IsSpotTradingAllowed && s.QuoteAsset == suffix {
				tmp[strings.ToLower(s.Symbol)] = true
				break
			}
		}
	}
	return tmp
}
