package binance

import "time"

// HOST - url for binance REST api
var host = ""

// WEBSOCKETHOST -url for binance websocket api
var websockethost = "wss://stream.binance.com:9443/ws"

// Init -
func Init(priceRefreshRate time.Duration, allowedBasedAssets ...string) error {
	if err := initSymbols(allowedBasedAssets...); err != nil {
		return err
	}

	initPrice(priceRefreshRate)
	return nil
}
