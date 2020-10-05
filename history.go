package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/akaritrading/libs/exchange"
)

type ExchangeHistory struct {
	Exchange           exchange.Exchange
	HistoryPositions   map[string]*exchange.HistoryPosition
	HistorySymbolLocks map[string]*sync.RWMutex

	name string
}

func InitHistoryJob(name string, ex exchange.Exchange, stop chan int) (*ExchangeHistory, error) {

	h := &ExchangeHistory{
		Exchange:           ex,
		name:               name,
		HistoryPositions:   make(map[string]*exchange.HistoryPosition),
		HistorySymbolLocks: make(map[string]*sync.RWMutex),
	}

	if err := h.historyPositionsFromFile(); err != nil {
		return nil, err
	}

	h.syncExchangeSymbols()

	for s := range h.HistoryPositions {
		h.HistorySymbolLocks[s] = &sync.RWMutex{}
	}

	if err := h.startFetch(stop); err != nil {
		return nil, err
	}

	return h, nil
}

func (h *ExchangeHistory) startFetch(stop chan int) error {

	f, err := os.Create(h.symbolsJSONPath())
	if err != nil {
		return err
	}

	go func() {
		for {
			for symbol, save := range h.HistoryPositions {
				h.fetchWorker(symbol, save)
				if err := f.Truncate(0); err != nil {
				}
				if _, err := f.Seek(0, 0); err != nil {
				}
				if err := exchange.WriteHistoryPositions(f, h.HistoryPositions); err != nil {
					fmt.Println(err)
				}
			}

			fmt.Println("done")

			select {
			case <-stop:
				f.Close()
				return
			case <-time.After(time.Minute * 5):
			}
		}
	}()

	return nil
}

// TODO: handle error better
func (h *ExchangeHistory) fetchWorker(symbol string, save *exchange.HistoryPosition) {

	lock := h.HistorySymbolLocks[symbol]
	lock.Lock()
	defer lock.Unlock()

	now := (time.Now().Unix() - (5 * 60)) * 1000 // roll back 5 minutes
	var points []exchange.Candle

	for atomic.LoadInt64(&save.End) <= now {
		candles, err := h.Exchange.Klines(symbol, save)
		if err != nil {
			fmt.Println(err)
			break
		}
		atomic.AddInt64(&save.End, int64(len(candles)*1000*60))
		points = append(points, candles...)
	}

	if len(points) > 0 {

		f, err := os.OpenFile(h.symbolPath(symbol), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		defer f.Sync()

		exchange.WriteCandles(f, points)
	}
}

func (h *ExchangeHistory) historyPositionsFromFile() error {

	f, err := os.Open(h.symbolsJSONPath())
	if err != nil {
		err := os.RemoveAll(h.pricesDirectory())
		if err != nil {
			return err
		}
		err = os.MkdirAll(h.pricesDirectory(), 0770)
		if err != nil {
			return err
		}
	} else {
		saves, err := exchange.ReadHistoryPositions(f)
		f.Close()
		if err != nil {
			return err
		}
		h.HistoryPositions = saves
	}

	return nil
}

func (h *ExchangeHistory) symbolPath(s string) string {
	return fmt.Sprintf("/priceData/%s/prices/%s", h.name, s)
}

func (h *ExchangeHistory) pricesDirectory() string {
	return fmt.Sprintf("/priceData/%s/prices", h.name)
}

func (h *ExchangeHistory) symbolsJSONPath() string {
	return fmt.Sprintf("/priceData/%s/symbols.json", h.name)
}

var (
	SymbolSaveInitialTimestamp = time.Now().Add(-(time.Hour * 24 * 7)).Unix() * 1000 // make 0 for production
)

func (h *ExchangeHistory) syncExchangeSymbols() {

	// removes no longer existing symbols
	for s := range h.HistoryPositions {
		if err := h.Exchange.CheckSymbol(s); err != nil {
			delete(h.HistoryPositions, s)
		}
	}

	// adds missing symbols into symbol saves
	symbols := h.Exchange.Symbols()
	for _, s := range symbols {
		if _, ok := h.HistoryPositions[s]; !ok {
			h.HistoryPositions[s] = &exchange.HistoryPosition{End: SymbolSaveInitialTimestamp}
		}
	}
}

func (h *ExchangeHistory) GetSymbolHistory(symbol string, start int64) (*exchange.History, error) {

	// defer util.TimeTrack(time.Now(), "GetSymbolHistory")

	err := h.Exchange.CheckSymbol(symbol)
	if err != nil {
		return nil, err
	}

	lock := h.HistorySymbolLocks[symbol]
	lock.RLock()
	defer lock.RUnlock()

	f, err := os.Open(h.symbolPath(symbol))
	if err != nil {
		// handle better
		return nil, err
	}
	defer f.Close()

	return exchange.ReadHistoryWindow(f, h.HistoryPositions[symbol], start, 0)
}
