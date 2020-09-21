package binance

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/akaritrading/prices/pkg/pricesclient"
)

var symbolsMap map[string]Symbol
var symbolHistoryPositions map[string]*pricesclient.HistoryPosition
var symbolHistoryLocks map[string]*sync.Mutex

// Init -
func Init(allowedBasedAssets ...string) error {
	if err := InitSymbols(allowedBasedAssets...); err != nil {
		return err
	}

	symbolHistoryLocks = make(map[string]*sync.Mutex)

	for s := range symbolsMap {
		symbolHistoryLocks[s] = &sync.Mutex{}
	}

	symbolHistoryPositions = make(map[string]*pricesclient.HistoryPosition)

	return priceHistoryJob()
}

func priceHistoryJob() error {

	f, err := os.Open("/priceData/binance/symbols.json")
	if err != nil {
		err := os.RemoveAll("/priceData/binance/prices")
		if err != nil {
			return err
		}
		err = os.MkdirAll("/priceData/binance/prices", 0770)
		if err != nil {
			return err
		}
		symbolHistoryPositions = make(map[string]*pricesclient.HistoryPosition)
	} else {
		saves, err := pricesclient.ReadHistoryPositions(f)
		f.Close()
		if err != nil {
			return err
		}
		symbolHistoryPositions = saves
	}

	syncExchangeSymbols()

	go fetchJob()

	return nil
}

func fetchJob() {

	f, err := os.Create("/priceData/binance/symbols.json")

	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	for {
		fetchAndSaveAll()
		pricesclient.WriteHistoryPositions(f, symbolHistoryPositions)
		time.Sleep(time.Hour)
	}
}

func newSymbolSaves() {
	for s := range symbolsMap {
		symbolHistoryPositions[s] = &pricesclient.HistoryPosition{End: SymbolSaveInitialTimestamp}
	}
}

func syncExchangeSymbols() {

	// removes no longer existing symbols
	for s := range symbolHistoryPositions {
		if _, ok := symbolsMap[s]; !ok {
			delete(symbolHistoryPositions, s)
		}
	}

	// adds missing symbols into symbol saves
	for s := range symbolsMap {
		if _, ok := symbolHistoryPositions[s]; !ok {
			symbolHistoryPositions[s] = &pricesclient.HistoryPosition{End: SymbolSaveInitialTimestamp}
		}
	}

}
