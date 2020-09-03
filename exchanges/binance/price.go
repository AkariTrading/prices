package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/akaritrading/libs/util"
)

type orderbook struct {
	Asks         [][]string `json:"asks"`
	Bids         [][]string `json:"bids"`
	lastUpdateId int        `json:"lastUpdateId"`
}

type pricestreamdata struct {
	P string `json:"p"`
}

type OrderbookPrice struct {
	Sell float64 `json:"sell"`
	Buy  float64 `json:"buy"`
}

var requestClient = http.Client{
	Timeout: time.Second * 10,
}

// OrderBookPrice -
func OrderBookPrice(symbol string) (OrderbookPrice, error) {

	if _, ok := symbolsMap[symbol]; !ok {
		return OrderbookPrice{}, errors.New("symbol not found")
	}

	return fetchOrderBookPrice(symbol)
}

func fetchOrderBookPrice(symbol string) (OrderbookPrice, error) {

	fmt.Println("symbol", symbol)
	var data orderbook
	err := httpGETJson(fmt.Sprintf("https://api.binance.com/api/v3/depth?symbol=%s", strings.ToUpper(symbol)), &data)
	if err != nil {
		return OrderbookPrice{}, err
	}

	sell, err := util.StrToFloat64(data.Bids[0][0])
	if err != nil {
		return OrderbookPrice{}, err
	}

	buy, err := util.StrToFloat64(data.Asks[0][0])
	if err != nil {
		return OrderbookPrice{}, err
	}

	price := OrderbookPrice{Sell: sell, Buy: buy}
	return price, nil
}

func httpGETJson(url string, obj interface{}) error {
	res, err := requestClient.Get(url)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	bdy, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return errors.New("")
	}

	return json.Unmarshal(bdy, &obj)
}
