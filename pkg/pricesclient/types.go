package pricesclient

import (
	"sync"
	"time"
)

const (
	Binance string = "binance"
)

type MemoryMegabytes int

type Client struct {
	Host                 string
	Exchange             string
	OrderbookRefreshRate time.Duration

	orderbookPriceMap sync.Map
	streamMap         sync.Map
	streamPriceMap    sync.Map
}

func (c *Client) ToSymbol(symbolA string, symbolB string) string {
	if c.Exchange == Binance {
		return symbolA + symbolB
	}

	return symbolA + symbolB
}
