package binance

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

var symbolsMap map[string]bool
var symbolSaves map[string]*symbolSave
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

	symbolSaves = make(map[string]*symbolSave)

	return priceHistoryJob()
}

func priceHistoryJob() error {

	f, err := os.Open("/priceData/binance/symbols.json")
	if err != nil {
		err := os.MkdirAll("/priceData/binance/prices", 0770)
		if err != nil {
			return err
		}
	} else {
		defer f.Close()
		err := json.NewDecoder(f).Decode(&symbolSaves)
		if err != nil {
			return err
		}
	}

	syncExchangeSymbols()

	go fetchJob()

	return nil
}

func fetchJob() {

	for {
		fetchAndSaveAll()
		json, err := json.Marshal(symbolSaves)
		if err != nil {
			log.Fatal(err)
		}
		ioutil.WriteFile("/priceData/binance/symbols.json", json, 0770)

		time.Sleep(time.Hour)
	}
}

func newSymbolSaves() {
	for s := range symbolsMap {
		symbolSaves[s] = &symbolSave{End: SymbolSaveInitialTimestamp}
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
			symbolSaves[s] = &symbolSave{End: SymbolSaveInitialTimestamp}
		}
	}

}
