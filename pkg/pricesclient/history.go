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
	"github.com/akaritrading/libs/middleware"
	"github.com/akaritrading/libs/util"
	"github.com/pkg/errors"
)

var ErrorDateRange = errors.New("date range error")

var requestClient = http.Client{
	Timeout: time.Second * 30,
}

type Client struct {
	host     string
	exchange string

	mu          *sync.Mutex
	appendQueue map[string]bool
	candlefs    *candlefs.CandleFS
	logger      *log.Logger
}

type Request struct {
	c         *Client
	requestID string
	logger    *log.Logger
}

func Create(host string, exchange string) (*Client, error) {

	if err := os.MkdirAll(fmt.Sprintf("/candleCache/%s/", exchange), 0644); err != nil {
		return nil, err
	}

	c := &Client{
		host:        host,
		exchange:    exchange,
		mu:          &sync.Mutex{},
		appendQueue: map[string]bool{},
		candlefs:    candlefs.New(fmt.Sprintf("/candleCache/%s/", exchange)),
	}

	return c, nil
}

func (c *Client) NewRequest(r *http.Request) *Request {

	logger := middleware.GetLogger(r)
	requestID := middleware.GetRequestID(r)

	return &Request{
		c:         c,
		requestID: requestID,
		logger:    logger,
	}
}

func (r *Request) RequestHistory(symbol string, start int64, end int64, maxSize int64) (*exchange.HistoryFlat, error) {

	var history exchange.HistoryFlat

	req, err := util.NewRequest("GET", fmt.Sprintf("http://%s/%s/history/%s?start=%d&end=%d&maxSize=%d", r.c.host, r.c.exchange, symbol, start, end, maxSize), nil)
	if err != nil {
		return nil, err
	}

	_, err = util.DoRequest(&requestClient, middleware.SetRequestID(req, r.requestID), &history)
	if err != nil {
		return nil, err
	}

	return &history, nil
}

func (r *Request) Read(symbol string, start int64, end int64) (*exchange.History, error) {

	sh, err := r.c.candlefs.Open(symbol)
	if err != nil {
		return nil, err
	}
	defer sh.Close()

	// TODO: maybe have a per symbol lock, because this blocks the entire service
	if sh.End() == 0 {
		r.c.mu.Lock()

		hist, err := r.RequestHistory(symbol, sh.End(), time.Now().Unix()*1000, 0)
		if err != nil {
			r.logger.Error(err)
		}

		err = sh.Append(hist.Start, hist.ToHistory().Candles)
		if err != nil {
			r.logger.Error(err)
		}

		r.c.mu.Unlock()
	}

	return sh.Read(start, end)
}
