package binance

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/akaritrading/prices/pkg/client"
)

var symbolsMap map[string]bool
var symbolHistoryPositions map[string]*client.HistoryPosition
var symbolHistoryLocks map[string]*sync.Mutex

// Init -
func Init(allowedBasedAssets ...string) error {
	if err := initSymbols(allowedBasedAssets...); err != nil {
		return err
	}

	symbolHistoryLocks = make(map[string]*sync.Mutex)

	for s := range symbolsMap {
		symbolHistoryLocks[s] = &sync.Mutex{}
	}

	symbolHistoryPositions = make(map[string]*client.HistoryPosition)

	return priceHistoryJob()
}

func priceHistoryJob() error {

	f, err := os.Open("/priceData/binance/symbols.json")
	if err != nil {
		err := os.MkdirAll("/priceData/binance/prices", 0770)
		if err != nil {
			return err
		}
		symbolHistoryPositions = make(map[string]*client.HistoryPosition)
	} else {
		defer f.Close()

		saves, err := client.ReadHistoryPositions(f)
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

	f, err := os.OpenFile("/priceData/binance/symbols.json", os.O_CREATE|os.O_WRONLY, 0770)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	for {
		fetchAndSaveAll()
		client.WriteHistoryPositions(f, symbolHistoryPositions)
		time.Sleep(time.Hour)
	}
}

func newSymbolSaves() {
	for s := range symbolsMap {
		symbolHistoryPositions[s] = &client.HistoryPosition{End: SymbolSaveInitialTimestamp}
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
			symbolHistoryPositions[s] = &client.HistoryPosition{End: SymbolSaveInitialTimestamp}
		}
	}

}
