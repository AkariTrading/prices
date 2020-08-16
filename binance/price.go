package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/scarface382/libs/util"
)

type orderbook struct {
	Asks         [][]string `json:"asks"`
	Bids         [][]string `json:"bids"`
	LastUpdateId int        `json:"lastUpdateId"`
}

type pricestreamdata struct {
	P string `json:"p"`
}

type priceCheckPoint struct {
	sell     float64
	buy      float64
	lastSave time.Time
}

var priceCheckPoints sync.Map

// PriceRate - controls refresh rate of Price
var priceRate time.Duration

func initPrice(t time.Duration) {
	priceRate = t
}

// OrderBookPrice -
func OrderBookPrice(symbol string) (float64, float64, error) {

	if _, ok := symbolsMap[symbol]; !ok {
		return 0, 0, errors.New("symbol not found")
	}

	val, ok := priceCheckPoints.Load(symbol)

	if ok {
		checkpoint := val.(priceCheckPoint)
		if time.Since(checkpoint.lastSave) > priceRate {
			fmt.Println("expired")
			return fetchAndStorePrice(symbol)
		}
		fmt.Println("cachehit")
		return checkpoint.sell, checkpoint.buy, nil
	}

	fmt.Println("cachemiss")
	return fetchAndStorePrice(symbol)
}

func fetchAndStorePrice(symbol string) (float64, float64, error) {

	var data orderbook
	err := httpGETJson(fmt.Sprintf("https://api.binance.com/api/v3/depth?symbol=%s", symbol), &data)
	if err != nil {
		return 0, 0, err
	}

	sell, err := util.StrToFloat64(data.Bids[0][0])
	if err != nil {
		return 0, 0, err
	}

	buy, err := util.StrToFloat64(data.Asks[0][0])
	if err != nil {
		return 0, 0, err
	}

	priceCheckPoints.Store(symbol, priceCheckPoint{sell: sell, buy: buy, lastSave: time.Now()})

	return sell, buy, nil
}

func httpGETJson(url string, obj interface{}) error {
	res, err := http.Get(url)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	bdy, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(bdy, &obj)
}
