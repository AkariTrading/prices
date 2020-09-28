package pricesclient

import (
	"encoding/json"
	"fmt"
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

	body, err := getRequest(fmt.Sprintf("http://%s/%s/orderbookPrice/%s", c.Host, c.Exchange, symbol))
	if err != nil {
		return price.OrderbookPrice, err
	}

	err = json.Unmarshal(body, &price)
	if err != nil {
		return price.OrderbookPrice, err
	}

	price.lastUpdate = time.Now()
	c.orderbookPriceMap.Store(symbol, price)
	return price.OrderbookPrice, nil
}
