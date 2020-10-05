package pricesclient

import (
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/akaritrading/libs/exchange"
	"github.com/akaritrading/libs/exchange/binance"
	"github.com/akaritrading/libs/util"
	"github.com/pkg/errors"
)

var client = http.Client{
	Timeout: time.Second * 10,
}

type MemoryMegabytes int

type ExchangeClient interface {
}

type PricesReader interface {
	OrderbookPrice(symbol string) (exchange.OrderbookPrice, error)
	GetHistory(symbol string, start int64, maxSize int64) (*exchange.HistoryFlat, error)
}

type Client struct {
	Host                 string
	OrderbookRefreshRate time.Duration
	ExchangeName         string

	exchange          exchange.Exchange
	orderbookPriceMap sync.Map
	streamMap         sync.Map
	streamPriceMap    sync.Map
}

func (c *Client) ToSymbol(a, b string) string {
	return c.exchange.ToSymbol(a, b)
}

func (c *Client) InitBinance() error {

	binanceClient := &binance.BinanceClient{}

	if err := binanceClient.Init(); err != nil {
		return err
	}

	c.ExchangeName = "binance"
	c.exchange = binanceClient

	return nil
}

func getRequest(url string) ([]byte, error) {

	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, util.ErrorSymbolNotFound
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("bad request or serviece error")
	}

	return ioutil.ReadAll(res.Body)
}
