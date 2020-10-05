package pricesclient

import (
	"time"

	"github.com/akaritrading/libs/exchange"
)

type OrderbookPrice struct {
	exchange.OrderbookPrice
	lastUpdate time.Time
}

func (c *Client) OrderbookPrice(symbol string) (exchange.OrderbookPrice, error) {

	val, ok := c.orderbookPriceMap.Load(symbol)

	if ok {
		price := val.(OrderbookPrice)
		if time.Since(price.lastUpdate) > c.OrderbookRefreshRate {
			return c.fetchAndStorePrice(symbol)
		}
		return price.OrderbookPrice, nil
	}

	return c.fetchAndStorePrice(symbol)
}

func (c *Client) fetchAndStorePrice(symbol string) (exchange.OrderbookPrice, error) {

	var price OrderbookPrice

	orderbook, err := c.exchange.OrderbookPrice(symbol)
	if err != nil {
		return exchange.OrderbookPrice{}, err
	}

	price.OrderbookPrice = orderbook
	price.lastUpdate = time.Now()
	c.orderbookPriceMap.Store(symbol, price)
	return price.OrderbookPrice, nil
}
