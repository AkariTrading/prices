package pricesclient

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"unsafe"

	"github.com/akaritrading/libs/util"
	"github.com/pkg/errors"
)

var ErrorDateRange = errors.New("date range error")

func (c *Client) InitHistoryFileCache() error {

	if symbolLocks == nil {
		symbolLocks = make(map[string]*sync.Mutex)
	}

	f, err := os.Open("/symbolscache/binance/symbols.json")
	if err != nil {
		err := os.RemoveAll("/symbolscache/binance/prices")
		if err != nil {
			return err
		}
		err = os.MkdirAll("/symbolscache/binance/prices", 0770)
		if err != nil {
			return err
		}
		historyPositions = make(HistoryPositions)
	} else {
		historyPositions, err = ReadHistoryPositions(f)
		if err != nil {
			return err
		}
		f.Close()
	}

	return nil
}

// SymbolHistory - retrieves symbol price history between start and end unix millisecond timestamps.
// If no local cache exists, the entire history is fetched. if end is after the local cache window,
// cache is updated with the most recent available data from Prices service.
func (c *Client) SymbolHistory(symbol string, start int64, end int64) (*History, error) {

	if end <= start {
		return nil, ErrorDateRange
	}

	symbol = strings.ToLower(symbol)

	lockSymbol(symbol)
	defer unlockSymbol(symbol)

	pos, ok := getPos(symbol)

	if !ok {
		updatedPos, err := c.fetchAndCacheFile(symbol, &HistoryPosition{})
		if err != nil {
			if err == util.ErrorSymbolNotFound {
				delSymbolLock(symbol)
			}
			return nil, err
		}
		pos = updatedPos
	} else {
		if end > pos.End {
			updatedPos, err := c.fetchAndCacheFile(symbol, pos)
			if err != nil {
				return nil, err
			}
			pos = updatedPos
		}
	}

	priceFile, err := os.Open(fmt.Sprintf("/symbolscache/binance/prices/%s", symbol))
	if err != nil {
		return nil, err
	}

	return ReadHistoryWindow(priceFile, pos, start, end)
}

// warning - assumes symbol is locked
func (c *Client) fetchAndCacheFile(symbol string, pos *HistoryPosition) (*HistoryPosition, error) {

	hist, err := c.GetHistory(symbol, pos.End, 0)
	if err != nil {
		return nil, err
	}

	if len(hist.Candles) == 0 {
		return &hist.HistoryPosition, nil
	}

	fmt.Println("hist.Start, hist.End ", hist.Start, hist.End)

	priceFile, err := os.OpenFile(fmt.Sprintf("/symbolscache/binance/prices/%s", symbol), os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer priceFile.Close()
	defer priceFile.Sync()

	if pos.Start != 0 {
		hist.Start = pos.Start
	}

	savePos(symbol, &hist.HistoryPosition)

	// the next calls need to be rolled back in the case of any error :(
	err = WriteCandles(priceFile, hist.ToHistory().Candles)
	if err != nil {
		return nil, err
	}

	historyPositionLock.Lock()
	defer historyPositionLock.Unlock()

	posFile, err := os.Create("/symbolscache/binance/symbols.json")
	if err != nil {
		return nil, err
	}
	defer posFile.Close()
	defer posFile.Sync()

	err = WriteHistoryPositions(posFile, historyPositions)
	if err != nil {
		return nil, err
	}

	return &hist.HistoryPosition, nil
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

func (c *Client) GetHistory(symbol string, start int64, maxSize int64) (*HistoryResponse, error) {

	body, err := getRequest(fmt.Sprintf("http://%s/%s/history/%s?start=%d&maxSize=%d", c.Host, c.Exchange, symbol, start, maxSize))
	if err != nil {
		return nil, err
	}

	var history HistoryResponse
	err = json.Unmarshal(body, &history)
	if err != nil {
		return nil, err
	}

	return &history, nil
}

func ReadHistoryWindow(f *os.File, pos *HistoryPosition, start int64, end int64) (*History, error) {

	// defer util.TimeTrack(time.Now(), "HistoryWindow")

	var millisecondsInMinute int64 = 60 * 1000

	emptyHistory := &History{HistoryPosition: HistoryPosition{Start: pos.End, End: pos.End}}

	if start >= pos.End {
		return emptyHistory, nil
	}

	if end > 0 && (end <= start || end <= pos.Start) {
		return emptyHistory, nil
	}

	total := (pos.End - pos.Start) / millisecondsInMinute

	offset := (start - pos.Start) / millisecondsInMinute
	if offset < 0 {
		offset = 0
		start = pos.Start
	}

	if offset >= total {
		return emptyHistory, nil
	}

	dataSize := int64(unsafe.Sizeof(Candle{}))
	f.Seek(offset*dataSize, 0)

	if end > pos.End || end == 0 {
		end = pos.End
	}

	candles := make([]Candle, (end-start)/millisecondsInMinute)
	if err := binary.Read(f, binary.LittleEndian, candles); err != nil {
		return nil, err
	}

	return &History{HistoryPosition: HistoryPosition{Start: start, End: end}, Candles: candles}, nil
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

func WriteHistoryPositions(f io.Writer, position HistoryPositions) error {
	json, err := json.Marshal(position)
	if err != nil {
		return err
	}
	_, err = f.Write(json)
	return err
}

// if err := cache.Set(symbol, *(*[]byte)(unsafe.Pointer(history))); err != nil {
// 			// log error
// 		}

// 		return history, nil
// 	}

// 	return *(**History)(unsafe.Pointer(&data)), nil
