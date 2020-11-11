package pricesclient

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/akaritrading/libs/exchange"
	"github.com/akaritrading/libs/exchange/candlefs"
	"github.com/akaritrading/libs/util"
	"github.com/pkg/errors"
)

var ErrorDateRange = errors.New("date range error")

var requestClient = http.Client{
	Timeout: time.Second * 10,
}

type Client struct {
	host string
}

func New(host string) *Client {
	return &Client{
		host: host,
	}
}

func (c *Client) RequestHistory(symbol string, start int64, end int64, maxSize int64) (*exchange.HistoryFlat, error) {

	var history exchange.HistoryFlat
	_, err := util.Request(&requestClient, "GET", fmt.Sprintf("%s/history/%s?start=%d&end=%d&maxSize=%d", c.host, symbol, start, end, maxSize), nil, &history)
	if err != nil {
		return nil, err
	}

	return &history, nil
}

func (c *Client) Read(symbol string, start int64, end int64, maxSize int64) (*exchange.History, error) {

	if err := os.MkdirAll("/candleCache/", 0644); err != nil {
		return nil, err
	}

	sh, err := candlefs.New("/candleCache/").Open(symbol)
	if err != nil {
		return nil, err
	}
	defer sh.Close()

	if end > sh.End() {

		hist, err := c.RequestHistory(symbol, sh.End(), time.Now().Unix()*1000, 0)
		if err != nil {
			return nil, err
		}

		err = sh.Append(hist.Start, hist.ToHistory().Candles)
		if err != nil {
			return nil, err
		}
	}

	return sh.Read(start, end)
}
