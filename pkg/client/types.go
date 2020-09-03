package client

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrorSymbolNotFound = errors.New("symbol does not exist")
)

type Exchange string

const (
	Binance Exchange = "binance"
)

type MemoryMegabytes int

type PriceClient struct {
	Host                 string
	Exchange             Exchange
	OrderbookRefreshRate time.Duration
	MaxCacheSize         MemoryMegabytes

	orderbookPriceMap sync.Map
	streamMap         sync.Map
	streamPriceMap    sync.Map
}
