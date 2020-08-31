package binance

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SymbolNotFoundError = "SymbolNotFoundError"
)

const (
	OPENTIME = iota // uint
	OPEN            // string
	HIGH            // string
	LOW             // string
	CLOSE           // string
	VOLUME          // string
)

const (
	SymbolSaveInitialTimestamp = 1598804620000 // make 0 for production
)

type symbolSave struct {
	Start int64
	End   int64
}

type symbolFetchJob struct {
	Symbol string
	Save   *symbolSave
}

type symbolCandle struct {
	Price  float64
	Volume float64
}

func FetchKlines(symbol string, save *symbolSave) ([]symbolCandle, error) {

	if !CheckSymbol(symbol) {
		return nil, errors.New(SymbolNotFoundError)
	}

	query := url.Values{}
	query.Set("symbol", strings.ToUpper(symbol))
	query.Set("interval", "1m")
	query.Set("startTime", strconv.FormatInt(save.End, 10))
	query.Set("limit", strconv.FormatInt(1000, 10))

	url, _ := url.Parse("https://api.binance.com/api/v3/klines")
	url.RawQuery = query.Encode()

	client := http.Client{
		Timeout: time.Second * 30,
	}

	res, err := client.Do(&http.Request{
		Method: "GET",
		URL:    url,
	})

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var data [][]interface{}
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}

	var historyPoints []symbolCandle

	for _, candle := range data {

		if save.Start == 0 {
			save.Start = int64(candle[OPENTIME].(float64))
		}

		high, _ := strconv.ParseFloat(candle[HIGH].(string), 64)
		low, _ := strconv.ParseFloat(candle[LOW].(string), 64)
		vol, _ := strconv.ParseFloat(candle[VOLUME].(string), 64)

		historyPoints = append(historyPoints, symbolCandle{Price: float64((high + low) / 2), Volume: vol})
	}

	if len(data) > 0 {
		lastCandle := data[len(data)-1]
		save.End = int64(lastCandle[OPENTIME].(float64)) + (60 * 1000) // skip one minute
	}

	return historyPoints, nil
}

func fetchAndSaveAll(symbolSaves map[string]*symbolSave) {

	jobs := make(chan symbolFetchJob, len(symbolSaves))
	workersCount := 10
	var wg sync.WaitGroup
	wg.Add(workersCount)

	for w := 0; w < workersCount; w++ {
		go fetchAndSave(jobs, &wg)
	}

	for symbol, save := range symbolSaves {
		jobs <- symbolFetchJob{Save: save, Symbol: symbol}
	}

	close(jobs)
	wg.Wait()
}

func fetchAndSave(jobs chan symbolFetchJob, wg *sync.WaitGroup) {

	var points []symbolCandle

	now := (time.Now().Unix() - 60) * 1000 // roll back a minute

	for job := range jobs {

		fmt.Println("will fetch " + job.Symbol)

		f, err := os.OpenFile(fmt.Sprintf("./priceData/binance/prices/%s", job.Symbol), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		for job.Save.End <= now {
			history, err := FetchKlines(job.Symbol, job.Save)
			if err == nil {
				points = append(points, history...)
			}
			// TODO: log error
		}

		for _, p := range points {
			binary.Write(f, binary.LittleEndian, p)
		}
	}

	wg.Done()
}

func newSymbolSaves() map[string]*symbolSave {

	symbolSaves := make(map[string]*symbolSave)
	for s := range symbolsMap {
		symbolSaves[s] = &symbolSave{End: SymbolSaveInitialTimestamp}
	}

	return symbolSaves
}

func GetData(symbol string) ([]symbolCandle, error) {

	f, err := os.Open(fmt.Sprintf("./priceData/binance/prices/%s", symbol))
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer f.Close()

	stat, _ := f.Stat()
	candles := make([]symbolCandle, stat.Size()/16)
	err = binary.Read(f, binary.LittleEndian, candles)
	return candles, err
}
