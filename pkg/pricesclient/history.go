package pricesclient

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/akaritrading/libs/exchange"
	"github.com/akaritrading/libs/exchange/candlefs"
	"github.com/akaritrading/libs/log"
	"github.com/akaritrading/libs/util"
	"github.com/pkg/errors"
)

var ErrorDateRange = errors.New("date range error")

var requestClient = http.Client{
	Timeout: time.Second * 10,
}

type Client struct {
	host     string
	exchange string

	mu          *sync.Mutex
	appendQueue map[string]bool
	candlefs    *candlefs.CandleFS
	logger      *log.Logger
}

func InitHistoryClient(host string, exchange string, logger *log.Logger) (*Client, error) {

	if err := os.MkdirAll(fmt.Sprintf("/candleCache/%s/", exchange), 0644); err != nil {
		return nil, err
	}

	c := &Client{
		host:        host,
		exchange:    exchange,
		mu:          &sync.Mutex{},
		appendQueue: map[string]bool{},
		candlefs:    candlefs.New(fmt.Sprintf("/candleCache/%s/", exchange)),
		logger:      logger,
	}

	return c, nil
}

// func (c *Client) WorkOnQueue() {

// 	c.mu.Lock()
// 	var symbols []string
// 	for k := range c.appendQueue {
// 		symbols = append(symbols, k)
// 	}
// 	for _, k := range symbols {
// 		delete(c.appendQueue, k)
// 	}
// 	c.mu.Unlock()

// 	for _, symbol := range symbols {

// 		sh, err := c.candlefs.Open(symbol)
// 		if err != nil {
// 			c.logger.Error(err)
// 		}

// 		hist, err := c.RequestHistory(symbol, sh.End(), time.Now().Unix()*1000, 0)
// 		if err != nil {
// 			c.logger.Error(err)
// 		}

// 		err = sh.Append(hist.Start, hist.ToHistory().Candles)
// 		if err != nil {
// 			c.logger.Error(err)
// 		}
// 	}

// }

func (c *Client) RequestHistory(symbol string, start int64, end int64, maxSize int64) (*exchange.HistoryFlat, error) {

	var history exchange.HistoryFlat
	_, err := util.Request(&requestClient, "GET", fmt.Sprintf("http://%s/%s/history/%s?start=%d&end=%d&maxSize=%d", c.host, c.exchange, symbol, start, end, maxSize), nil, &history)
	if err != nil {
		return nil, err
	}

	return &history, nil
}

func (c *Client) Read(symbol string, start int64, end int64) (*exchange.History, error) {

	sh, err := c.candlefs.Open(symbol)
	if err != nil {
		return nil, err
	}
	defer sh.Close()

	if sh.End() == 0 {
		c.mu.Lock()

		hist, err := c.RequestHistory(symbol, sh.End(), time.Now().Unix()*1000, 0)
		if err != nil {
			c.logger.Error(err)
		}

		err = sh.Append(hist.Start, hist.ToHistory().Candles)
		if err != nil {
			c.logger.Error(err)
		}
		c.mu.Unlock()
	}

	return sh.Read(start, end)
}
