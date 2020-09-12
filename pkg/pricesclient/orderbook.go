package pricesclient

import (
	"encoding/json"
	"fmt"
	"time"
)

type OrderbookPrice struct {
	Sell       float64 `json:"sell"`
	Buy        float64 `json:"buy"`
	lastUpdate time.Time
}

func (c *Client) OrderbookPrice(symbol string) (OrderbookPrice, error) {

	val, ok := c.orderbookPriceMap.Load(symbol)

	if ok {
		price := val.(OrderbookPrice)
		if time.Since(price.lastUpdate) > c.OrderbookRefreshRate {
			return c.fetchAndStorePrice(symbol)
		}
		return price, nil
	}

	return c.fetchAndStorePrice(symbol)
}

func (c *Client) fetchAndStorePrice(symbol string) (OrderbookPrice, error) {

	var price OrderbookPrice

	body, err := getRequest(fmt.Sprintf("http://%s/%s/orderbookPrice/%s", c.Host, c.Exchange, symbol))
	if err != nil {
		return price, err
	}

	err = json.Unmarshal(body, &price)
	if err != nil {
		return price, err
	}

	price.lastUpdate = time.Now()
	c.orderbookPriceMap.Store(symbol, price)
	return price, nil
}
