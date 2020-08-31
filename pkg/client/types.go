package client

import (
	"sync"
	"time"

	"github.com/jinzhu/gorm"
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
	DBHandle             *gorm.DB

	orderbookPriceMap sync.Map
	streamMap         sync.Map
	streamPriceMap    sync.Map
}

type PriceHistoryTick struct {
	Price  float32
	Volume float32
}

type PriceHistory struct {
	Prices   []float32
	Volumes  []float32
	Interval time.Duration
}
