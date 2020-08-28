package client

import (
	"sync"
	"time"
)

const (
	SymbolNotFoundError = "SymbolNotFoundError"
)

type Exchange string

const (
	Binance Exchange = "binance"
)

type PriceClient struct {
	Host                 string
	Exchange             Exchange
	OrderbookRefreshRate time.Duration

	orderbookMap sync.Map
	streamMap    sync.Map
}
