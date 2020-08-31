package binance

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// Init -
func Init(allowedBasedAssets ...string) error {
	if err := initSymbols(allowedBasedAssets...); err != nil {
		return err
	}

	initPriceHistoryJob()

	return nil
}

func initPriceHistoryJob() {

	var symbolsToFetch map[string]*symbolSave

	f, err := os.Open("./priceData/binance/symbols.json")
	if err != nil {
		err := os.MkdirAll("./priceData/binance/prices", 0770)
		if err != nil {
			log.Fatal("cannot create folders")
		}
		symbolsToFetch = newSymbolSaves()
	} else {
		defer f.Close()
		var symbolSaves map[string]*symbolSave
		err := json.NewDecoder(f).Decode(&symbolSaves)
		if err != nil {
			log.Fatal(err)
		}
		addNewSymbols(symbolSaves)
		symbolsToFetch = symbolSaves
	}

	go fetchJob(symbolsToFetch)
}

func fetchJob(symbols map[string]*symbolSave) {

	for {
		fetchAndSaveAll(symbols)
		json, err := json.Marshal(symbols)
		if err != nil {
			log.Fatal(err)
		}
		ioutil.WriteFile("./priceData/binance/symbols.json", json, 0770)

		time.Sleep(time.Hour)
	}

}

func addNewSymbols(symbolSaves map[string]*symbolSave) {
	for s := range symbolsMap {
		if _, ok := symbolSaves[s]; !ok {
			symbolSaves[s] = &symbolSave{Start: 0, End: SymbolSaveInitialTimestamp}
		}
	}
}
