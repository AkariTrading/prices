package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type OrderbookPrice struct {
	Sell       float64 `json:"sell"`
	Buy        float64 `json:"buy"`
	lastUpdate time.Time
}

var client = http.Client{
	Timeout: time.Second * 10,
}

func (c *PriceClient) OrderbookPrice(symbol string) (OrderbookPrice, error) {

	val, ok := c.orderbookMap.Load(symbol)

	if ok {
		price := val.(OrderbookPrice)
		if time.Since(price.lastUpdate) > c.OrderbookRefreshRate {
			return c.fetchAndStorePrice(symbol)
		}
		fmt.Println("cache hit")
		return price, nil
	}

	return c.fetchAndStorePrice(symbol)
}

func (c *PriceClient) fetchAndStorePrice(symbol string) (OrderbookPrice, error) {

	var price OrderbookPrice

	err := httpGETJson(fmt.Sprintf("http://%s/%s/orderbookPrice/%s", c.Host, c.Exchange, symbol), &price)
	if err != nil {
		return price, err
	}

	price.lastUpdate = time.Now()
	c.orderbookMap.Store(symbol, price)
	return price, nil
}

func httpGETJson(url string, price *OrderbookPrice) error {

	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return errors.New(SymbolNotFoundError)
	}

	if res.StatusCode != http.StatusOK {
		fmt.Println(res.StatusCode)
		return errors.New("bad request")
	}

	bdy, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bdy, price)
	return err
}
