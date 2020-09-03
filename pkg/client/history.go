package client

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"unsafe"
)

var ErrorDateRange = errors.New("date range error")

type Candle struct {
	Price  float64
	Volume float64
}

type HistoryPosition struct {
	Start int64
	End   int64
}

type HistoryPositions map[string]*HistoryPosition

var historyPositions HistoryPositions
var historyPositionLock sync.Mutex

var symbolLocks map[string]*sync.Mutex
var symbolLock sync.Mutex

type History struct {
	HistoryPosition
	Prices  []float64
	Volumes []float64
}

func (c *PriceClient) InitHistory() error {

	if symbolLocks == nil {
		symbolLocks = make(map[string]*sync.Mutex)
	}

	err := os.MkdirAll("/symbolscache/binance/prices", 0770)
	if err != nil {
		return err
	}

	f, err := os.Open("/symbolscache/binance/symbols.json")
	if err != nil {
		historyPositions = make(HistoryPositions)
	} else {
		defer f.Close()
		historyPositions, err = ReadHistoryPositions(f)
	}

	return err
}

// SymbolHistory - retrieves symbol price history between start and end unix millisecond timestamps.
// If no local cache exists, the entire history is fetched. if start or end is after the local cache window,
// cache is updated with the most recent available data from Prices service.
func (c *PriceClient) SymbolHistory(symbol string, start int64, end int64) (*History, error) {

	if end <= start {
		return nil, ErrorDateRange
	}

	lockSymbol(symbol)
	defer unlockSymbol(symbol)

	pos, ok := getPos(symbol)

	if !ok {
		err := c.fetchAndCacheFile(symbol, 0)
		if err != nil {
			if err == ErrorSymbolNotFound {
				delSymbolLock(symbol)
			}
			return nil, err
		}
	} else {

		if start > pos.End || end > pos.End {
			err := c.fetchAndCacheFile(symbol, pos.End)
			if err != nil {
				return nil, err
			}
		}
	}

	priceFile, err := os.Open(fmt.Sprintf("/priceData/binance/prices/%s", symbol))
	if err != nil {
		return nil, err
	}

	return HistoryWindow(priceFile, pos, start, end)
}

// assumes symbol is locked
func (c *PriceClient) fetchAndCacheFile(symbol string, start int64) error {

	hist, candles, err := c.getData(symbol, 0)
	if err != nil {
		return err
	}

	priceFile, err := os.OpenFile(fmt.Sprintf("/priceData/binance/prices/%s", symbol), os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer priceFile.Close()

	posFile, err := os.OpenFile("/priceData/binance/symbols.json", os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer posFile.Close()

	savePos(symbol, &hist.HistoryPosition)
	WriteCandles(priceFile, candles)
	WriteHistoryPositions(posFile, historyPositions)

	return nil
}

func getPos(symbol string) (*HistoryPosition, bool) {
	historyPositionLock.Lock()
	defer historyPositionLock.Unlock()
	pos, ok := historyPositions[symbol]
	return pos, ok
}

func savePos(symbol string, pos *HistoryPosition) {
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

func (c *PriceClient) getData(symbol string, start int64) (*History, []Candle, error) {

	body, err := getRequest(fmt.Sprintf("http://%s/%s/priceHistory/%s?start=%d", c.Host, c.Exchange, symbol, start))
	if err != nil {

		return nil, nil, err
	}

	var history History
	err = json.Unmarshal(body, &history)
	if err != nil {
		return nil, nil, err
	}

	candles := make([]Candle, 0, len(history.Prices))
	for i := range history.Prices {
		candles = append(candles, Candle{Price: history.Prices[i], Volume: history.Volumes[i]})
	}

	return &history, candles, nil
}

func HistoryWindow(f *os.File, save *HistoryPosition, start int64, end int64) (*History, error) {

	var millisecondsInMinute int64 = 60 * 1000

	emptyHistory := &History{HistoryPosition: HistoryPosition{Start: save.End, End: save.End}}

	if start >= save.End {
		return emptyHistory, nil
	}

	if end > 0 && (end <= start || end <= save.Start) {
		return emptyHistory, nil
	}

	total := (save.End - save.Start) / millisecondsInMinute

	offset := (start - save.Start) / millisecondsInMinute
	if offset < 0 {
		offset = 0
		start = save.Start
	}

	if offset >= total {
		return emptyHistory, nil
	}

	dataSize := int64(unsafe.Sizeof(Candle{}))
	f.Seek(offset*dataSize, 0)

	if end > save.End || end == 0 {
		end = save.End
	}

	candles := make([]Candle, (end-start)/millisecondsInMinute)
	if err := binary.Read(f, binary.LittleEndian, candles); err != nil {
		return nil, err
	}

	prices := make([]float64, 0, len(candles))
	volumes := make([]float64, 0, len(candles))

	for _, c := range candles {
		prices = append(prices, c.Price)
		volumes = append(volumes, c.Volume)
	}

	return &History{HistoryPosition: HistoryPosition{Start: start, End: end}, Prices: prices, Volumes: volumes}, nil
}

func WriteCandles(f io.Writer, candles []Candle) error {
	for _, p := range candles {
		err := binary.Write(f, binary.LittleEndian, p)
		if err != nil {
			return err
		}
	}

	return nil
}

func ReadHistoryPositions(f io.Reader) (HistoryPositions, error) {
	ret := make(HistoryPositions)
	err := json.NewDecoder(f).Decode(&ret)
	return ret, err
}

func WriteHistoryPositions(f io.Writer, position HistoryPositions) {
	json, err := json.Marshal(position)
	if err != nil {
		log.Fatal(err)
	}
	f.Write(json)
}

// if err := cache.Set(symbol, *(*[]byte)(unsafe.Pointer(history))); err != nil {
// 			// log error
// 		}

// 		return history, nil
// 	}

// 	return *(**History)(unsafe.Pointer(&data)), nil
