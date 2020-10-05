package pricesclient

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/akaritrading/libs/exchange"
	"github.com/akaritrading/libs/util"
	"github.com/pkg/errors"
)

var ErrorDateRange = errors.New("date range error")

var historyPositions exchange.HistoryPositions
var historyPositionLock sync.Mutex

var symbolLocks map[string]*sync.Mutex
var symbolLock sync.Mutex

func (c *Client) InitHistoryFileCache() error {

	if symbolLocks == nil {
		symbolLocks = make(map[string]*sync.Mutex)
	}

	f, err := os.Open(c.symbolsJSONPath())
	if err != nil {
		err := os.RemoveAll(c.pricesPath())
		if err != nil {
			return err
		}
		err = os.MkdirAll(c.pricesPath(), 0770)
		if err != nil {
			return err
		}
		historyPositions = make(exchange.HistoryPositions)
	} else {
		historyPositions, err = exchange.ReadHistoryPositions(f)
		if err != nil {
			return err
		}
		f.Close()
	}

	return nil
}

func (c *Client) symbolsJSONPath() string {
	return fmt.Sprintf("/symbolscache/%s/symbols.json", c.ExchangeName)
}

func (c *Client) pricesPath() string {
	return fmt.Sprintf("/symbolscache/%s/prices", c.ExchangeName)
}

func (c *Client) symbolPath(s string) string {
	return fmt.Sprintf("/symbolscache/%s/prices/%s", c.ExchangeName, s)
}

// SymbolHistory - retrieves symbol price history between start and end unix millisecond timestamps.
// If no local cache exists, the entire history is fetched. if end is after the local cache window,
// cache is updated with the most recent available data from Prices service.
func (c *Client) SymbolHistory(symbol string, start int64, end int64) (*exchange.History, error) {

	if end <= start {
		return nil, ErrorDateRange
	}

	err := c.fetchAndCacheFile(symbol)
	if err != nil {
		if err == util.ErrorSymbolNotFound {
			delSymbolLock(symbol)
		}
		return nil, err
	}

	priceFile, err := os.Open(c.symbolPath(symbol))
	if err != nil {
		return nil, err
	}
	defer priceFile.Close()

	return exchange.ReadHistoryWindow(priceFile, getPos(symbol), start, end)
}

func (c *Client) fetchAndCacheFile(symbol string) error {

	lockSymbol(symbol)
	defer unlockSymbol(symbol)

	pos := getPos(symbol)
	if pos == nil {
		pos = &exchange.HistoryPosition{}
	}

	hist, err := c.GetHistory(symbol, pos.End, 0)
	if err != nil {
		return err
	}

	if len(hist.Candles) == 0 {
		return nil
	}

	if pos.Start != 0 {
		hist.Start = pos.Start
	}

	savePos(symbol, &hist.HistoryPosition)

	priceFile, err := os.OpenFile(c.symbolPath(symbol), os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer priceFile.Close()
	defer priceFile.Sync()

	// the next calls need to be rolled back in the case of any error :(
	err = exchange.WriteCandles(priceFile, hist.ToHistory().Candles)
	if err != nil {
		return err
	}

	historyPositionLock.Lock()
	defer historyPositionLock.Unlock()

	posFile, err := os.Create(c.symbolsJSONPath())
	if err != nil {
		return err
	}
	defer posFile.Close()
	defer posFile.Sync()

	err = exchange.WriteHistoryPositions(posFile, historyPositions)
	if err != nil {
		return err
	}

	return nil
}

func getPos(symbol string) *exchange.HistoryPosition {
	historyPositionLock.Lock()
	defer historyPositionLock.Unlock()
	return historyPositions[symbol]
}

func savePos(symbol string, pos *exchange.HistoryPosition) {
	historyPositionLock.Lock()
	defer historyPositionLock.Unlock()
	historyPositions[symbol] = pos
}

func lockSymbol(symbol string) {
	symbolLock.Lock()
	l, ok := symbolLocks[symbol]
	symbolLock.Unlock()

	if ok {
		l.Lock()
	} else {
		m := &sync.Mutex{}
		m.Lock()
		symbolLock.Lock()
		symbolLocks[symbol] = m
		symbolLock.Unlock()
	}
}

func unlockSymbol(symbol string) {
	symbolLock.Lock()
	l, ok := symbolLocks[symbol]
	symbolLock.Unlock()

	if ok {
		l.Unlock()
	}
}

func delSymbolLock(symbol string) {
	symbolLock.Lock()
	defer symbolLock.Unlock()
	delete(symbolLocks, symbol)
}

func (c *Client) GetHistory(symbol string, start int64, maxSize int64) (*exchange.HistoryFlat, error) {

	body, err := getRequest(fmt.Sprintf("http://%s/%s/history/%s?start=%d&maxSize=%d", c.Host, c.ExchangeName, symbol, start, maxSize))
	if err != nil {
		return nil, err
	}

	var history exchange.HistoryFlat
	err = json.Unmarshal(body, &history)
	if err != nil {
		return nil, err
	}

	return &history, nil
}
