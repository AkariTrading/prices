package binance

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/akaritrading/prices/pkg/client"
)

var symbolsMap map[string]bool
var symbolSaves map[string]*client.HistoryPosition
var symbolSavesLock map[string]*sync.Mutex

// Init -
func Init(allowedBasedAssets ...string) error {
	if err := initSymbols(allowedBasedAssets...); err != nil {
		return err
	}

	symbolSavesLock = make(map[string]*sync.Mutex)

	for s := range symbolsMap {
		symbolSavesLock[s] = &sync.Mutex{}
	}

	symbolSaves = make(map[string]*client.HistoryPosition)

	return priceHistoryJob()
}

func priceHistoryJob() error {

	f, err := os.Open("/priceData/binance/symbols.json")
	if err != nil {
		err := os.MkdirAll("/priceData/binance/prices", 0770)
		if err != nil {
			return err
		}
		symbolSaves = make(map[string]*client.HistoryPosition)
	} else {
		defer f.Close()

		saves, err := client.ReadHistoryPositions(f)
		if err != nil {
			return err
		}
		symbolSaves = saves
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
		client.WriteHistoryPositions(f, symbolSaves)
		time.Sleep(time.Hour)
	}
}

func newSymbolSaves() {
	for s := range symbolsMap {
		symbolSaves[s] = &client.HistoryPosition{End: SymbolSaveInitialTimestamp}
	}
}

func syncExchangeSymbols() {

	// removes no longer existing symbols
	for s := range symbolSaves {
		if _, ok := symbolsMap[s]; !ok {
			delete(symbolSaves, s)
		}
	}

	// adds missing symbols into symbol saves
	for s := range symbolsMap {
		if _, ok := symbolSaves[s]; !ok {
			symbolSaves[s] = &client.HistoryPosition{End: SymbolSaveInitialTimestamp}
		}
	}

}
