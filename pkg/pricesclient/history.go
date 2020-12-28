package pricesclient

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/akaritrading/libs/errutil"
	"github.com/akaritrading/libs/exchange"
	"github.com/akaritrading/libs/exchange/binance"
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

	symbols []string
}

type Request struct {
	c         *Client
	requestID string
	logger    *log.Logger
}

func CachePath() string {
	return "/candleCache/"
}

func CachePathExchange(exchange string) string {
	return fmt.Sprintf("%s/%s/", CachePath(), exchange)
}

func Create(host string, exchange string) (*Client, error) {

	if err := os.MkdirAll(fmt.Sprintf("%s/%s/", CachePath(), exchange), 0644); err != nil {
		return nil, err
	}

	var symbolsList []string

	if exchange == "binance" {
		binance := binance.BinanceClient{}
		if err := binance.FetchSymbols(); err != nil {
			return nil, err
		}
		symbolsList = binance.Symbols()
	} else {
		return nil, errutil.New(errutil.ErrInvalidExchange, nil)
	}

	c := &Client{
		host:        host,
		exchange:    exchange,
		mu:          &sync.Mutex{},
		appendQueue: map[string]bool{},
		candlefs:    candlefs.New(CachePathExchange(exchange)),
		symbols:     symbolsList,
	}

	return c, nil
}

func (c *Client) New(requestID string, logger *log.Logger) *Request {
	return &Request{
		c:         c,
		requestID: requestID,
		logger:    logger,
	}
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

func (r *Request) UpdateAll(stop chan int) error {

	for _, symbol := range r.c.symbols {

		r.Update(symbol)

		select {
		case <-stop:
			return nil
		default:
		}
	}

	return nil
}

func (r *Request) Update(symbol string) error {

	end := time.Now().Unix() * 1000

	var hist *exchange.History

	sh, err := r.c.candlefs.Open(symbol)
	if err != nil {
		r.logger.Error(err)
		return err
	}
	defer sh.Close()

	hist, err = r.FetchHistory(symbol, sh.End(), end)
	if err != nil {
		r.logger.Error(err)
		return err
	}

	r.c.mu.Lock()
	err = sh.Append(hist.Start, hist.Candles)
	if err != nil {
		r.logger.Error(err)
	}
	r.c.mu.Unlock()

	return err
}

func (r *Request) Read(symbol string, start int64, end int64) (*exchange.History, error) {

	sh, err := r.c.candlefs.Open(symbol)
	if err != nil {
		return nil, err
	}
	defer sh.Close()

	return sh.Read(start, end)
}
