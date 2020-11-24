package pricesclient

import (
	"fmt"
	"io/ioutil"
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

func (r *Request) FetchHistory(symbol string, start int64, end int64) (*exchange.History, error) {

	timer := r.logger.TimerStart("FetchHistory")
	defer timer.Stop()

	req, err := util.NewRequest("GET", fmt.Sprintf("http://%s/%s/history/%s?start=%d&end=%d", r.c.host, r.c.exchange, symbol, start, end), r.requestID, nil)
	if err != nil {
		return nil, err
	}

	res, err := requestClient.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return candlefs.Decode(data)
}

func (r *Request) Read(symbol string, start int64, end int64) (*exchange.History, error) {

	sh, err := r.c.candlefs.Open(symbol)
	if err != nil {
		return nil, err
	}
	defer sh.Close()

	if sh.End() < end {

		// TODO: maybe have a per symbol lock, because this blocks the entire service
		r.c.mu.Lock()
		defer r.c.mu.Unlock()

		hist, err := r.FetchHistory(symbol, sh.End(), time.Now().Unix()*1000)
		if err != nil {
			r.logger.Error(err)
			return nil, err
		}

		fmt.Println(hist.Start, hist.End, len(hist.Candles))

		err = sh.Append(hist.Start, hist.Candles)
		if err != nil {
			r.logger.Error(err)
			return nil, err
		}

	}

	return sh.Read(start, end)
}
